// Package opencode 是 OpenCode（开源终端 agent）的适配器。配置：opencode.json、AGENTS.md。
package opencode

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func init() { adapters.Register(&adapter{}) }

type adapter struct{}

func (a *adapter) Meta() adapters.ToolMeta {
	return adapters.ToolMeta{
		ID: "opencode", DisplayName: "OpenCode", MinVersion: "0.5", MaxVersion: "0.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportFull, Note: "AGENTS.md"},
			ir.KindMCP:         {Level: adapters.SupportFull, Note: "opencode.json mcp 段"},
			ir.KindSkill:       {Level: adapters.SupportNone}, ir.KindAgent: {Level: adapters.SupportPartial},
			ir.KindCommand: {Level: adapters.SupportPartial}, ir.KindWorkflow: {Level: adapters.SupportNone},
			ir.KindHook: {Level: adapters.SupportNone}, ir.KindSetting: {Level: adapters.SupportFull},
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
		if isDir(filepath.Join(home, ".config", "opencode")) {
			locs = append(locs, adapters.Location{Scope: ir.ScopeGlobal, Root: filepath.Join(home, ".config", "opencode")})
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
		if isFile(filepath.Join(abs, "opencode.json")) || isFile(filepath.Join(abs, "AGENTS.md")) || isDir(filepath.Join(abs, ".git")) {
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
	// AGENTS.md
	if data, err := os.ReadFile(filepath.Join(loc.Root, "AGENTS.md")); err == nil {
		b.Instructions = append(b.Instructions, ir.Instruction{
			Header:     ir.Header{ID: "instruction.agents", IRVersion: 1, Origin: &ir.Origin{Tool: "opencode", Path: "AGENTS.md", Scope: loc.Scope}},
			Activation: ir.ActivationAlways, AppliesTo: []string{"opencode"}, Priority: 200, Body: string(data),
		})
	}
	// opencode.json
	cfgPath := filepath.Join(loc.Root, "opencode.json")
	if loc.Scope == ir.ScopeGlobal {
		cfgPath = filepath.Join(loc.Root, "opencode.json")
	}
	if data, err := os.ReadFile(cfgPath); err == nil {
		var root map[string]any
		if json.Unmarshal(data, &root) == nil {
			for k, v := range root {
				if k == "mcp" || k == "mcpServers" {
					if m, ok := v.(map[string]any); ok {
						for name, confAny := range m {
							if conf, ok := confAny.(map[string]any); ok {
								s := ir.MCPServer{
									Header:    ir.Header{ID: "mcp." + name, IRVersion: 1, Origin: &ir.Origin{Tool: "opencode", Path: "opencode.json", Scope: loc.Scope}},
									Name:      name,
									Transport: strOf(conf, "type", "stdio"), Command: strOf(conf, "command", ""), URL: strOf(conf, "url", ""),
								}
								b.MCPServers = append(b.MCPServers, s)
							}
						}
					}
					continue
				}
				b.Settings = append(b.Settings, ir.SettingEntry{
					Header: ir.Header{ID: "setting.opencode." + k, IRVersion: 1, Origin: &ir.Origin{Tool: "opencode", Path: "opencode.json", Scope: loc.Scope}},
					Key:    k, Value: v, ToolScope: []string{"opencode"},
				})
			}
		}
	}
	return b, nil
}

func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	root := opts.ProjectRoot
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".config", "opencode")
	}
	var files []adapters.WrittenFile
	add := func(p string, c []byte) error {
		sum := sha256.Sum256(c)
		files = append(files, adapters.WrittenFile{Path: p, Hash: hex.EncodeToString(sum[:]), Content: c})
		return nil
	}
	if len(b.Instructions) > 0 {
		var bodies []byte
		for _, inst := range b.Instructions {
			bodies = append(bodies, inst.Body...)
			bodies = append(bodies, '\n')
		}
		add(filepath.Join(root, "AGENTS.md"), bodies)
	}
	if len(b.Settings) > 0 || len(b.MCPServers) > 0 {
		root2 := map[string]any{}
		for _, s := range b.Settings {
			root2[s.Key] = s.Value
		}
		if len(b.MCPServers) > 0 {
			servers := map[string]any{}
			for _, s := range b.MCPServers {
				conf := map[string]any{}
				if s.Command != "" {
					conf["command"] = s.Command
				}
				if s.URL != "" {
					conf["url"] = s.URL
				}
				name := s.Name
				if name == "" {
					name = ir.NameTail(s.ID)
				}
				servers[name] = conf
			}
			root2["mcp"] = servers
		}
		data, _ := json.MarshalIndent(root2, "", "  ")
		add(filepath.Join(root, "opencode.json"), data)
	}
	return files, nil
}

func strOf(m map[string]any, k, def string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return def
}
