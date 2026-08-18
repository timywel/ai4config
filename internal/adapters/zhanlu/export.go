package zhanlu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
)

// exportBundle 把 merged Bundle 物化为 Zhanlu 配置（ADAPTERS §3.4 导出布局）。
func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	project := opts.ProjectRoot != ""

	var configDir, skillsDir, agentsBase, kiloBase string
	if project {
		configDir = opts.ProjectRoot
		agentsBase = opts.ProjectRoot
		kiloBase = filepath.Join(opts.ProjectRoot, ".kilo")
	} else {
		configDir = globalDir()
		agentsBase = configDir
		if home, err := paths.Home(); err == nil {
			skillsDir = filepath.Join(home, ".agents", "skills")
		}
	}

	var files []adapters.WrittenFile
	add := func(path string, content []byte) error {
		wf, err := writeOne(path, content, opts.DryRun)
		if err != nil {
			return err
		}
		files = append(files, wf)
		return nil
	}

	// instructions → AGENTS.md
	if len(b.Instructions) > 0 {
		if err := add(filepath.Join(agentsBase, "AGENTS.md"), []byte(renderAgentsMD(b.Instructions))); err != nil {
			return nil, err
		}
	}
	// skills → skills/<name>/SKILL.md
	for _, p := range b.Skills {
		dir := skillsDir
		if project {
			dir = filepath.Join(agentsBase, ".agents", "skills")
		}
		if err := add(filepath.Join(dir, p.Name, "SKILL.md"), renderPackMD(p)); err != nil {
			return nil, err
		}
	}
	// agents/commands → .kilo/agent|command/<name>.md（仅项目层有 .kilo）
	if project {
		for _, p := range b.Agents {
			if err := add(filepath.Join(kiloBase, "agent", p.Name+".md"), renderPackMD(p)); err != nil {
				return nil, err
			}
		}
		for _, p := range b.Commands {
			if err := add(filepath.Join(kiloBase, "command", p.Name+".md"), renderPackMD(p)); err != nil {
				return nil, err
			}
		}
	}
	// settings + mcp → zhanlu.json
	if len(b.Settings) > 0 || len(b.MCPServers) > 0 {
		data, err := renderZhanluJSON(b.Settings, b.MCPServers)
		if err != nil {
			return nil, err
		}
		target := filepath.Join(configDir, "zhanlu.json")
		if project {
			target = filepath.Join(configDir, "kilo.json")
		}
		if err := add(target, data); err != nil {
			return nil, err
		}
	}

	return files, nil
}

// writeOne 生成渲染计划（不落盘；统一由引擎经 atomicfile 写盘）。
func writeOne(path string, content []byte, dryRun bool) (adapters.WrittenFile, error) {
	_ = dryRun
	sum := sha256.Sum256(content)
	return adapters.WrittenFile{Path: path, Hash: hex.EncodeToString(sum[:]), Content: content}, nil
}

var _ = os.Getenv // 占位（globalDir 在 detect.go）
