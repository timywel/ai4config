package codex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/atomicfile"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
)

// exportBundle 把 merged Bundle 物化为 Codex 配置（ADAPTERS §3.2 导出布局）。
func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	project := opts.ProjectRoot != ""

	var codexDir, agentsBase string
	if project {
		codexDir = filepath.Join(opts.ProjectRoot, ".codex")
		agentsBase = opts.ProjectRoot
	} else {
		dir := os.Getenv("CODEX_HOME")
		if dir == "" {
			home, err := paths.Home()
			if err != nil {
				return nil, err
			}
			dir = filepath.Join(home, ".codex")
		}
		codexDir = dir
		agentsBase = dir
	}

	var files []adapters.WrittenFile
	var warnings []ir.Warning
	add := func(path string, content []byte) error {
		wf, err := writeOne(path, content, opts.DryRun)
		if err != nil {
			return err
		}
		files = append(files, wf)
		return nil
	}

	// config.toml（整块重写；项目级跳过机器级键）
	if len(b.Settings) > 0 || len(b.MCPServers) > 0 || len(b.Hooks) > 0 {
		data, w, err := renderConfigTOML(b.Settings, b.MCPServers, b.Hooks, project)
		if err != nil {
			return nil, err
		}
		warnings = append(warnings, w...)
		if err := add(filepath.Join(codexDir, "config.toml"), data); err != nil {
			return nil, err
		}
	}

	// instructions → AGENTS.md（按 subtree 还原目录层级）
	for _, inst := range b.Instructions {
		target := filepath.Join(agentsBase, "AGENTS.md")
		if inst.Subtree != "" && inst.Subtree != "." {
			target = filepath.Join(agentsBase, filepath.FromSlash(inst.Subtree), "AGENTS.md")
		}
		if err := add(target, []byte(inst.Body)); err != nil {
			return nil, err
		}
	}

	// skills → skills/<name>/SKILL.md
	for _, p := range b.Skills {
		data := renderSkillMD(p)
		if err := add(filepath.Join(codexDir, "skills", p.Name, "SKILL.md"), data); err != nil {
			return nil, err
		}
	}

	_ = warnings // 降级/跳过 Warning 由引擎层收集（此处并入 files 返回前由调用方处理）
	return files, nil
}

// renderSkillMD PromptPack → SKILL.md（frontmatter + 正文）。
func renderSkillMD(p ir.PromptPack) []byte {
	var sb strings.Builder
	sb.WriteString("---\nname: " + p.Name + "\n")
	if p.Description != "" {
		sb.WriteString("description: " + p.Description + "\n")
	}
	sb.WriteString("---\n")
	sb.WriteString(p.Body)
	return []byte(sb.String())
}

// writeOne 写单文件（dryRun 时只计算 hash 不落盘）。
func writeOne(path string, content []byte, dryRun bool) (adapters.WrittenFile, error) {
	sum := sha256.Sum256(content)
	wf := adapters.WrittenFile{Path: path, Hash: hex.EncodeToString(sum[:])}
	if dryRun {
		return wf, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return wf, err
	}
	if err := atomicfile.WriteFile(path, content, 0o600); err != nil {
		return wf, err
	}
	return wf, nil
}
