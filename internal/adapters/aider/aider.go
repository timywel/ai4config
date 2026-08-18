// Package aider 是 Aider（终端 AI 编码）的适配器。配置：.aider.conf.yml、CONVENTIONS.md。无 MCP。
package aider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func init() { adapters.Register(&adapter{}) }

type adapter struct{}

func (a *adapter) Meta() adapters.ToolMeta {
	return adapters.ToolMeta{
		ID:          "aider",
		DisplayName: "Aider",
		MinVersion:  "0.50",
		MaxVersion:  "0.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportFull, Note: "CONVENTIONS.md"},
			ir.KindMCP:         {Level: adapters.SupportNone, Note: "无 MCP"},
			ir.KindSkill:       {Level: adapters.SupportNone},
			ir.KindAgent:       {Level: adapters.SupportNone},
			ir.KindCommand:     {Level: adapters.SupportNone},
			ir.KindWorkflow:    {Level: adapters.SupportNone},
			ir.KindHook:        {Level: adapters.SupportNone},
			ir.KindSetting:     {Level: adapters.SupportFull, Note: ".aider.conf.yml"},
		},
	}
}

func (a *adapter) Import(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	return a.importLocation(ctx, loc)
}
func (a *adapter) Export(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	return a.exportBundle(ctx, b, opts)
}

func (a *adapter) Detect(ctx context.Context) ([]adapters.Location, error) {
	var locs []adapters.Location
	if home, err := os.UserHomeDir(); err == nil {
		if isFile(filepath.Join(home, ".aider.conf.yml")) {
			locs = append(locs, adapters.Location{Scope: ir.ScopeGlobal, Root: home})
		}
	}
	if pr := findRoot("."); pr != "" {
		locs = append(locs, adapters.Location{Scope: ir.ScopeProject, Root: pr})
	}
	return locs, nil
}

func isDir(p string) bool  { i, e := os.Stat(p); return e == nil && i.IsDir() }
func isFile(p string) bool { i, e := os.Stat(p); return e == nil && !i.IsDir() }
func findRoot(dir string) string {
	abs, _ := filepath.Abs(dir)
	for {
		if isFile(filepath.Join(abs, ".aider.conf.yml")) || isFile(filepath.Join(abs, "CONVENTIONS.md")) || isDir(filepath.Join(abs, ".git")) {
			return abs
		}
		if p := filepath.Dir(abs); p != abs {
			abs = p
		} else {
			return ""
		}
	}
}

func (a *adapter) importLocation(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	b := &ir.Bundle{Scope: loc.Scope, IRVersion: 1}
	// CONVENTIONS.md → instruction
	if data, err := os.ReadFile(filepath.Join(loc.Root, "CONVENTIONS.md")); err == nil {
		b.Instructions = append(b.Instructions, ir.Instruction{
			Header:     ir.Header{ID: "instruction.conventions", IRVersion: 1, Origin: &ir.Origin{Tool: "aider", Path: "CONVENTIONS.md", Scope: loc.Scope}},
			Activation: ir.ActivationAlways, AppliesTo: []string{"aider"}, Priority: 200, Body: string(data),
		})
	}
	// .aider.conf.yml → settings
	if data, err := os.ReadFile(filepath.Join(loc.Root, ".aider.conf.yml")); err == nil {
		var cfg map[string]any
		if yaml.Unmarshal(data, &cfg) == nil {
			for k, v := range cfg {
				b.Settings = append(b.Settings, ir.SettingEntry{
					Header: ir.Header{ID: "setting.aider." + k, IRVersion: 1, Origin: &ir.Origin{Tool: "aider", Path: ".aider.conf.yml", Scope: loc.Scope}},
					Key:    k, Value: v, ToolScope: []string{"aider"},
				})
			}
		}
	}
	return b, nil
}

func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	root := opts.ProjectRoot
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		root = home
	}
	var files []adapters.WrittenFile
	add := func(path string, content []byte) error {
		sum := sha256.Sum256(content)
		files = append(files, adapters.WrittenFile{Path: path, Hash: hex.EncodeToString(sum[:]), Content: content})
		return nil
	}
	if len(b.Instructions) > 0 {
		var sb strings.Builder
		for _, inst := range b.Instructions {
			sb.WriteString(inst.Body + "\n")
		}
		add(filepath.Join(root, "CONVENTIONS.md"), []byte(sb.String()))
	}
	if len(b.Settings) > 0 {
		cfg := map[string]any{}
		for _, s := range b.Settings {
			cfg[s.Key] = s.Value
		}
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return nil, err
		}
		add(filepath.Join(root, ".aider.conf.yml"), data)
	}
	return files, nil
}
