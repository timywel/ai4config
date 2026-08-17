package migrate

import (
	"context"
	"path/filepath"
	"strconv"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

// roundTripCheck Verify 第二级（ARCHITECTURE §5.1）：导出后重新 Import 写入产物，
// 与导出前 Bundle 做语义对比（忽略异构 x- 与白名单降级项），差异记 Warning。
// 这是"原理四：正确性靠证伪逼近"的即时落地——每次导出后尝试证伪它。
func (e *Engine) roundTripCheck(ctx context.Context, adapter adapters.Adapter, exported *ir.Bundle, files []adapters.WrittenFile, projectPath string) []ir.Warning {
	if len(files) == 0 {
		return nil
	}

	// 重新 Import 写入产物（按目标位置）
	loc := adapters.Location{Scope: ir.ScopeProject, Root: projectPath}
	if projectPath == "" {
		// 全局导出：用第一个写入文件的目录作为重读根（best-effort）
		loc = adapters.Location{Scope: ir.ScopeGlobal, Root: filepath.Dir(files[0].Path)}
	}
	reimported, err := adapter.Import(ctx, loc)
	if err != nil {
		return []ir.Warning{{Kind: "verify", Message: "round-trip 自检无法重新导入: " + err.Error()}}
	}

	// 语义对比：关键实体计数与关键字段（忽略 x- 与已知降级项）
	var warnings []ir.Warning
	compareCount := func(kind string, before, after int) {
		if before != after {
			warnings = append(warnings, ir.Warning{
				Kind:    "verify",
				Message: kind + " 数量 round-trip 不一致（导出前 " + strconv.Itoa(before) + " / 重导入 " + strconv.Itoa(after) + "）",
			})
		}
	}
	compareCount("mcp", len(exported.MCPServers), len(reimported.MCPServers))
	compareCount("skill", len(exported.Skills), len(reimported.Skills))
	compareCount("agent", len(exported.Agents), len(reimported.Agents))
	compareCount("hook", len(exported.Hooks), len(reimported.Hooks))
	compareCount("instruction", len(exported.Instructions), len(reimported.Instructions))

	// MCP 关键字段比对（name/transport/url/command）
	if len(exported.MCPServers) > 0 && len(reimported.MCPServers) > 0 {
		before := map[string]ir.MCPServer{}
		for _, s := range exported.MCPServers {
			before[s.Name] = s
		}
		for _, s := range reimported.MCPServers {
			b, ok := before[s.Name]
			if !ok {
				continue
			}
			if b.Transport != s.Transport || b.URL != s.URL || b.Command != s.Command {
				warnings = append(warnings, ir.Warning{
					Kind:    "verify",
					Entity:  s.ID,
					Message: "MCP " + s.Name + " 关键字段 round-trip 不一致",
				})
			}
		}
	}

	return warnings
}
