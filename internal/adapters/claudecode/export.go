package claudecode

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/atomicfile"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
)

// exportBundle 把 merged Bundle 物化为 Claude Code 配置（ADAPTERS §3.1 导出布局）。
// 职责边界：适配器只做 Render/Write（Map/Merge/降级由引擎层完成）。
func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	project := opts.ProjectRoot != ""

	// 目标根目录
	var claudeDir string // agents/skills/commands/rules 所在
	var instrTarget string
	if project {
		claudeDir = filepath.Join(opts.ProjectRoot, ".claude")
		instrTarget = filepath.Join(opts.ProjectRoot, "CLAUDE.md")
	} else {
		home, err := paths.Home()
		if err != nil {
			return nil, err
		}
		claudeDir = filepath.Join(home, ".claude")
		instrTarget = filepath.Join(home, ".claude", "CLAUDE.md")
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

	// instructions → CLAUDE.md（边界注释拼接）
	if len(b.Instructions) > 0 {
		if err := add(instrTarget, []byte(renderClaudeMD(b.Instructions))); err != nil {
			return nil, err
		}
	}
	// settings + hooks → settings.json
	if len(b.Settings) > 0 || len(b.Hooks) > 0 {
		data, err := renderSettingsJSON(b.Settings, b.Hooks)
		if err != nil {
			return nil, err
		}
		if err := add(filepath.Join(claudeDir, "settings.json"), data); err != nil {
			return nil, err
		}
	}
	// MCP
	if len(b.MCPServers) > 0 {
		if project {
			data, err := renderMCPJSON(b.MCPServers)
			if err != nil {
				return nil, err
			}
			if err := add(filepath.Join(opts.ProjectRoot, ".mcp.json"), data); err != nil {
				return nil, err
			}
		} else {
			// 全局 user scope → ~/.claude.json 局部 patch（保留运行时状态键）
			wf, err := a.patchGlobalMCP(b.MCPServers, opts.DryRun)
			if err != nil {
				return nil, err
			}
			files = append(files, wf)
		}
	}
	// agents / skills / commands
	for _, p := range b.Agents {
		data, err := renderPackMD(p)
		if err != nil {
			return nil, err
		}
		if err := add(filepath.Join(claudeDir, "agents", p.Name+".md"), data); err != nil {
			return nil, err
		}
	}
	for _, p := range b.Skills {
		data, err := renderPackMD(p)
		if err != nil {
			return nil, err
		}
		if err := add(filepath.Join(claudeDir, "skills", p.Name, "SKILL.md"), data); err != nil {
			return nil, err
		}
	}
	for _, p := range b.Commands {
		data, err := renderPackMD(p)
		if err != nil {
			return nil, err
		}
		if err := add(filepath.Join(claudeDir, "commands", p.Name+".md"), data); err != nil {
			return nil, err
		}
	}

	return files, nil
}

// patchGlobalMCP 把全局 MCP 局部 patch 进 ~/.claude.json（只改 mcpServers 键，保留运行时状态）。
func (a *adapter) patchGlobalMCP(servers []ir.MCPServer, dryRun bool) (adapters.WrittenFile, error) {
	home, err := paths.Home()
	if err != nil {
		return adapters.WrittenFile{}, err
	}
	path := filepath.Join(home, ".claude.json")

	root := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &root) // 保留全部既有键（含运行时状态）
	}
	// 只替换 mcpServers 键
	mcp := map[string]any{}
	for _, s := range servers {
		conf := map[string]any{}
		if s.Transport != "" && s.Transport != "stdio" {
			conf["type"] = s.Transport
		}
		if s.Command != "" {
			conf["command"] = s.Command
		}
		if len(s.Args) > 0 {
			conf["args"] = s.Args
		}
		if len(s.Env) > 0 {
			conf["env"] = s.Env
		}
		if s.URL != "" {
			conf["url"] = s.URL
		}
		if len(s.Headers) > 0 {
			conf["headers"] = s.Headers
		}
		name := s.Name
		if name == "" {
			name = ir.NameTail(s.ID)
		}
		mcp[name] = conf
	}
	root["mcpServers"] = mcp

	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return adapters.WrittenFile{}, err
	}
	return writeOne(path, data, dryRun)
}

// writeOne 写单文件（dryRun 时只计算 hash 不落盘，供引擎预览）。
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
