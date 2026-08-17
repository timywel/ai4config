// Package migrate 是迁移引擎：把 Load→Merge→Map→Render→Verify→Write 编排为导出管线。
// 权威规范：docs/ARCHITECTURE.md §5（迁移引擎）、docs/WORKFLOWS.md W2（导出流程）。
// 职责切分：引擎负责 Merge/Map/Assist/Verify；适配器只做 Import 与 Render/Write（不接触 AI）。
package migrate

import (
	"context"
	"fmt"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/store"
)

// Engine 迁移引擎。
type Engine struct {
	Repo  *store.Repo
	Hooks Hooks
}

// Hooks 引擎与调用方（CLI/TUI）的交互回调。
type Hooks struct {
	// ConfirmForeign 外来内容确认（W2[7]）：返回 overwrite|skip|view-diff|backup-overwrite。
	// 为 nil 时 foreign/modified 一律 skip（安全默认）。
	ConfirmForeign func(path string, status store.ForeignStatus) (string, error)
	// Snapshot 写入前快照（W2[9]）；为 nil 时引擎自调 Repo.CreateSnapshot。
	Snapshot func(note string) (string, error)
}

// ExportRequest 导出请求。
type ExportRequest struct {
	To             adapters.ToolID
	ProjectPath    string // 非空则合并项目 profile（scope=project）
	DryRun         bool
	Force          bool            // 全部按 backup-overwrite（仍需快照）
	IncludeForeign bool            // 纳入异构来源条目（applies_to 不含目标时）
	Only           []ir.EntityKind // 限定实体类型（--only）
}

// ExportResult 导出结果。
type ExportResult struct {
	Written    []adapters.WrittenFile // 实际/计划写入清单（dry-run 为计划）
	Warnings   []ir.Warning
	SnapshotID string
	DryRun     bool
}

// Export 导出主管线（W2 全流程）。
func (e *Engine) Export(ctx context.Context, req ExportRequest) (*ExportResult, error) {
	adapter, ok := adapters.Get(req.To)
	if !ok {
		return nil, fmt.Errorf("migrate: 未注册的目标适配器 %q", req.To)
	}

	if err := e.Repo.Lock(); err != nil { // W1[1]/W2[1] 写锁
		return nil, err
	}
	defer e.Repo.Unlock()

	// [1] Load 全局 + 项目 profile
	layers, err := e.loadLayers(req.ProjectPath)
	if err != nil {
		return nil, err
	}

	// [2] Merge 五层物化（含墓碑遮蔽）
	merged := profile.MergeBundles(layers...)

	// [3] 空集保护（红队 T-01）：merged 为空且目标已有文件 → 拒绝（--force 也需确认，CLI 层处理）
	if isEmptyBundle(merged) && !req.Force {
		return nil, fmt.Errorf("migrate: 合并后有效配置为空（可能是误删/盘掉线），已拒绝导出；确认无误请加 --force")
	}

	// [4] 过滤：--only 限定 + applies_to 过滤（--include-foreign 纳入异构）
	filtered, filterWarn := e.filterForTarget(merged, adapter.Meta().ID, req)
	var warnings []ir.Warning
	warnings = append(warnings, filterWarn...)

	// [5] Map：能力矩阵降级（ADAPTERS §5 两级规则）
	mapped, degradeWarn := e.applyCapabilities(filtered, adapter.Meta().Capabilities)
	warnings = append(warnings, degradeWarn...)

	// [6] 写入前快照（W2[9]；dry-run 也先计划但不落盘）
	var snapshotID string
	if !req.DryRun {
		if e.Hooks.Snapshot != nil {
			snapshotID, err = e.Hooks.Snapshot("export to " + string(req.To))
		} else {
			snapshotID, err = e.Repo.CreateSnapshot("export to " + string(req.To))
		}
		if err != nil {
			return nil, fmt.Errorf("migrate: 写入前快照失败: %w", err)
		}
	}

	// [7] Render（适配器只渲染计划不落盘——写盘统一收归引擎，ARCHITECTURE §5.1/§5.3）
	exportOpts := adapters.ExportOpts{
		ProjectRoot: req.ProjectPath,
		DryRun:      req.DryRun,
		Force:       req.Force,
	}
	files, err := adapter.Export(ctx, mapped, exportOpts)
	if err != nil {
		return nil, fmt.Errorf("migrate: 适配器渲染失败: %w", err)
	}

	// [8] 外来内容检查（计划路径 vs 目标现状 + exports 清单三态；W2[7]）
	if !req.DryRun {
		files, err = e.filterForeign(req, files)
		if err != nil {
			return nil, err
		}
	}

	// [9] 引擎统一写盘（写入协议 atomicfile；W2[9]）
	if !req.DryRun {
		if err := e.writePlanned(files); err != nil {
			return nil, fmt.Errorf("migrate: 写入失败: %w", err)
		}
	}

	// [10] Verify 两级：格式校验（适配器内）+ round-trip 自检
	if !req.DryRun {
		rtWarn := e.roundTripCheck(ctx, adapter, mapped, files, req.ProjectPath)
		warnings = append(warnings, rtWarn...)
	}

	// [11] 更新导出清单（W2[10]）
	if !req.DryRun {
		if err := e.updateExportManifest(req.To, merged.Scope, files); err != nil {
			warnings = append(warnings, ir.Warning{Kind: "manifest", Message: "更新导出清单失败: " + err.Error()})
		}
	}

	return &ExportResult{
		Written:    files,
		Warnings:   warnings,
		SnapshotID: snapshotID,
		DryRun:     req.DryRun,
	}, nil
}

// loadLayers 加载全局 + 项目 profile 为物化层。
func (e *Engine) loadLayers(projectPath string) ([]*profile.ScopedBundle, error) {
	var layers []*profile.ScopedBundle

	// 全局层
	globalDir := e.Repo.Path(store.DirProfiles, "global")
	if sb, err := profile.Load(globalDir, ir.ScopeGlobal); err == nil {
		layers = append(layers, sb)
	} else if !isNotExist(err) {
		return nil, fmt.Errorf("migrate: 加载全局 profile 失败: %w", err)
	}

	// 项目层（按项目路径查注册表找 pid → profiles/projects/<pid>）
	if projectPath != "" {
		projDir, err := e.projectProfileDir(projectPath)
		if err == nil && projDir != "" {
			if sb, err := profile.Load(projDir, ir.ScopeProject); err == nil {
				layers = append(layers, sb)
			}
		}
	}

	if len(layers) == 0 {
		return nil, fmt.Errorf("migrate: 无任何 profile（请先 cfg4ai collect 采集）")
	}
	return layers, nil
}

// projectProfileDir 按项目路径查注册表定位项目 profile 目录。
// 骨架阶段：用路径的 slug 作为项目 id 目录（注册表 registry 完整实现见 T-后续）。
func (e *Engine) projectProfileDir(projectPath string) (string, error) {
	slug := slugifyPath(projectPath)
	dir := e.Repo.Path(store.DirProfiles, "projects", slug)
	if _, err := profile.LoadManifest(dir); err != nil {
		return "", err
	}
	return dir, nil
}

// isEmptyBundle 判断 merged 是否无任何实体（空集保护用）。
func isEmptyBundle(b *ir.Bundle) bool {
	return len(b.Instructions) == 0 && len(b.MCPServers) == 0 &&
		len(b.Skills) == 0 && len(b.Agents) == 0 && len(b.Commands) == 0 &&
		len(b.Workflows) == 0 && len(b.Hooks) == 0 && len(b.Settings) == 0
}

func isNotExist(err error) bool {
	return err != nil && (osIsNotExist(err))
}
