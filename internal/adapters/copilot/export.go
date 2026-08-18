package copilot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

// exportBundle 把 merged Bundle 物化为 Copilot 配置（ADAPTERS §3.3 导出布局）。
func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	project := opts.ProjectRoot != ""

	var ghDir, vsDir string
	var globalDir string
	if project {
		ghDir = filepath.Join(opts.ProjectRoot, ".github")
		vsDir = filepath.Join(opts.ProjectRoot, ".vscode")
	} else {
		globalDir = userProfileDir()
		if globalDir == "" {
			return nil, errString("copilot: 无法定位 VS Code user profile 目录")
		}
		vsDir = globalDir
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

	// instructions：always → copilot-instructions.md；glob → instructions/<name>.instructions.md
	var always, globs []ir.Instruction
	for _, inst := range b.Instructions {
		if inst.Activation == ir.ActivationGlob || len(inst.FilePatterns) > 0 {
			globs = append(globs, inst)
		} else {
			always = append(always, inst)
		}
	}
	if len(always) > 0 {
		target := filepath.Join(ghDir, "copilot-instructions.md")
		if !project {
			target = filepath.Join(globalDir, "instructions", "copilot-instructions.md")
		}
		if err := add(target, []byte(renderCopilotInstructionsMD(always))); err != nil {
			return nil, err
		}
	}
	for _, inst := range globs {
		name := ir.NameTail(inst.ID)
		target := filepath.Join(ghDir, "instructions", name+".instructions.md")
		if !project {
			target = filepath.Join(globalDir, "instructions", name+".instructions.md")
		}
		if err := add(target, renderInstructionFile(inst)); err != nil {
			return nil, err
		}
	}

	// commands → prompts/<name>.prompt.md；agents → agents/<name>.agent.md（仅项目）
	for _, p := range b.Commands {
		target := filepath.Join(ghDir, "prompts", p.Name+".prompt.md")
		if !project {
			target = filepath.Join(globalDir, "prompts", p.Name+".prompt.md")
		}
		if err := add(target, renderPackMD(p)); err != nil {
			return nil, err
		}
	}
	if project {
		for _, p := range b.Agents {
			if err := add(filepath.Join(ghDir, "agents", p.Name+".agent.md"), renderPackMD(p)); err != nil {
				return nil, err
			}
		}
	}

	// mcp → mcp.json（servers + inputs/sandbox）
	if len(b.MCPServers) > 0 {
		data, err := renderMCPJSON(b.MCPServers, b.MCPFileExtensions)
		if err != nil {
			return nil, err
		}
		target := filepath.Join(vsDir, "mcp.json")
		if !project {
			target = filepath.Join(globalDir, "mcp.json")
		}
		if err := add(target, data); err != nil {
			return nil, err
		}
	}

	// settings → settings.json
	if len(b.Settings) > 0 {
		data, err := renderSettingsJSON(b.Settings)
		if err != nil {
			return nil, err
		}
		target := filepath.Join(vsDir, "settings.json")
		if !project {
			target = filepath.Join(globalDir, "settings.json")
		}
		if err := add(target, data); err != nil {
			return nil, err
		}
	}

	return files, nil
}

type errString string

func (e errString) Error() string { return string(e) }

// writeOne 生成渲染计划（不落盘；统一由引擎经 atomicfile 写盘）。
func writeOne(path string, content []byte, dryRun bool) (adapters.WrittenFile, error) {
	_ = dryRun
	sum := sha256.Sum256(content)
	return adapters.WrittenFile{Path: path, Hash: hex.EncodeToString(sum[:]), Content: content}, nil
}
