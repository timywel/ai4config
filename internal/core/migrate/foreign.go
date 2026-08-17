package migrate

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/atomicfile"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/store"
)

// filterForeign 外来内容检查（W2[7]）：计划路径的目标现状 vs 导出清单三态判定。
// 目标文件不存在（新建）→ 直接允许；存在且清单一致 → ours 直接覆盖；
// modified/foreign → 确认（--force=backup-overwrite）。
func (e *Engine) filterForeign(req ExportRequest, files []adapters.WrittenFile) ([]adapters.WrittenFile, error) {
	m, err := e.Repo.LoadExportManifest(string(req.To), "global")
	if err != nil {
		m = &store.ExportManifest{Tool: string(req.To), Scope: "global"}
	}

	var out []adapters.WrittenFile
	for _, f := range files {
		content, err := os.ReadFile(f.Path)
		if os.IsNotExist(err) {
			out = append(out, f) // 新建文件（目标尚无内容被覆盖风险）→ 允许
			continue
		}
		if err != nil {
			return nil, err
		}
		status := m.Classify(f.Path, content)
		switch status {
		case store.StatusOurs:
			out = append(out, f) // 本工具产物，直接覆盖
		case store.StatusModified, store.StatusForeign:
			choice := "skip"
			if req.Force {
				choice = "backup-overwrite"
			} else if e.Hooks.ConfirmForeign != nil {
				c, err := e.Hooks.ConfirmForeign(f.Path, status)
				if err != nil {
					return nil, err
				}
				choice = c
			}
			switch choice {
			case "overwrite", "backup-overwrite":
				out = append(out, f)
			case "skip", "view-diff":
				// skip / 暂不写
			}
		}
	}
	return out, nil
}

// writePlanned 引擎统一写盘（写入协议 atomicfile，目录 0700/文件 0600）。
func (e *Engine) writePlanned(files []adapters.WrittenFile) error {
	for _, f := range files {
		if err := os.MkdirAll(filepath.Dir(f.Path), 0o700); err != nil {
			return err
		}
		if err := atomicfile.WriteFile(f.Path, f.Content, 0o600); err != nil {
			return err
		}
	}
	return nil
}

// updateExportManifest 更新导出清单（W2[10]）。scope 预留按层分清单。
func (e *Engine) updateExportManifest(to adapters.ToolID, scope ir.Scope, files []adapters.WrittenFile) error {
	_ = scope // P0 统一记 global 清单；分层清单后续按需
	m, err := e.Repo.LoadExportManifest(string(to), "global")
	if err != nil {
		m = &store.ExportManifest{Tool: string(to), Scope: "global"}
	}
	for _, f := range files {
		content, err := os.ReadFile(f.Path)
		if err != nil {
			continue
		}
		m.Record(f.Path, content)
	}
	return e.Repo.SaveExportManifest(m)
}

// slugifyPath 项目路径 → profile 目录名（项目关联骨架；完整指纹见 registry）。
func slugifyPath(p string) string {
	p = filepath.ToSlash(p)
	p = strings.Trim(p, "/")
	p = strings.ReplaceAll(p, ":", "")
	var b strings.Builder
	for _, r := range p {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// osIsNotExist 包装（便于 engine 判断）。
func osIsNotExist(err error) bool { return os.IsNotExist(err) }
