package gemini

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
)

// exportBundle 物化为 Gemini 配置（settings.json + GEMINI.md）。
func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	project := opts.ProjectRoot != ""
	var geminiDir string
	if project {
		geminiDir = filepath.Join(opts.ProjectRoot, ".gemini")
	} else {
		home, err := paths.Home()
		if err != nil {
			return nil, err
		}
		geminiDir = filepath.Join(home, ".gemini")
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

	// settings.json（不透明 setting 键 + mcpServers）
	if len(b.Settings) > 0 || len(b.MCPServers) > 0 {
		data, err := renderSettingsJSON(b.Settings, b.MCPServers)
		if err != nil {
			return nil, err
		}
		if err := add(filepath.Join(geminiDir, "settings.json"), data); err != nil {
			return nil, err
		}
	}
	// GEMINI.md（instructions 拼接）
	if len(b.Instructions) > 0 {
		target := filepath.Join(geminiDir, "GEMINI.md")
		if project {
			target = filepath.Join(opts.ProjectRoot, "GEMINI.md")
		}
		if err := add(target, []byte(renderGeminiMD(b.Instructions))); err != nil {
			return nil, err
		}
	}
	return files, nil
}

// renderSettingsJSON settings + mcp → settings.json。
func renderSettingsJSON(settings []ir.SettingEntry, mcps []ir.MCPServer) ([]byte, error) {
	root := map[string]any{}
	for _, s := range settings {
		root[s.Key] = s.Value
	}
	if len(mcps) > 0 {
		servers := map[string]any{}
		for _, s := range mcps {
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
			if s.Trust != nil {
				conf["trust"] = *s.Trust
			}
			if len(s.EnabledTools) > 0 {
				conf["includeTools"] = s.EnabledTools
			}
			if len(s.DisabledTools) > 0 {
				conf["excludeTools"] = s.DisabledTools
			}
			name := s.Name
			if name == "" {
				name = ir.NameTail(s.ID)
			}
			servers[name] = conf
		}
		root["mcpServers"] = servers
	}
	return json.MarshalIndent(root, "", "  ")
}

// renderGeminiMD instructions 拼接为 GEMINI.md。
func renderGeminiMD(instructions []ir.Instruction) string {
	var sb strings.Builder
	for i, inst := range instructions {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("<!-- cfg4ai:begin %s -->\n", inst.ID))
		sb.WriteString(strings.TrimRight(inst.Body, "\n"))
		sb.WriteString(fmt.Sprintf("\n<!-- cfg4ai:end %s -->\n", inst.ID))
	}
	return sb.String()
}

// writeOne 生成渲染计划（不落盘）。
func writeOne(path string, content []byte, dryRun bool) (adapters.WrittenFile, error) {
	_ = dryRun
	sum := sha256.Sum256(content)
	return adapters.WrittenFile{Path: path, Hash: hex.EncodeToString(sum[:]), Content: content}, nil
}
