// Package roo 是 Roo Code（VS Code 扩展）的适配器。配置：.roo/rules/、.roomodes（custom modes）、.roo/mcp.json。
package roo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
		ID: "roo", DisplayName: "Roo Code", MinVersion: "3.0", MaxVersion: "3.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportFull, Note: ".roo/rules/"},
			ir.KindMCP:         {Level: adapters.SupportFull, Note: ".roo/mcp.json"},
			ir.KindSkill:       {Level: adapters.SupportNone},
			ir.KindAgent:       {Level: adapters.SupportPartial, Note: ".roomodes custom modes（groups+fileRegex 权限进 x-roo）"},
			ir.KindCommand:     {Level: adapters.SupportNone}, ir.KindWorkflow: {Level: adapters.SupportNone},
			ir.KindHook: {Level: adapters.SupportNone}, ir.KindSetting: {Level: adapters.SupportPartial},
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
	if pr := findRoot("."); pr != "" {
		locs = append(locs, adapters.Location{Scope: ir.ScopeProject, Root: pr})
	}
	return locs, nil
}

func isDir(p string) bool { i, e := os.Stat(p); return e == nil && i.IsDir() }
func findRoot(dir string) string {
	abs, _ := filepath.Abs(dir)
	for {
		if isDir(filepath.Join(abs, ".roo")) || isDir(filepath.Join(abs, ".git")) {
			return abs
		}
		if p := filepath.Dir(abs); p != abs {
			abs = p
		} else {
			return ""
		}
	}
}

// roomode .roomodes 的 custom mode（权限维度 groups+fileRegex 进 x-roo）。
type roomode struct {
	Slug               string `yaml:"slug"`
	Name               string `yaml:"name"`
	RoleDefinition     string `yaml:"roleDefinition"`
	CustomInstructions string `yaml:"customInstructions"`
	Groups             []any  `yaml:"groups"`
}

func (a *adapter) importLocation(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	b := &ir.Bundle{Scope: loc.Scope, IRVersion: 1}
	// .roo/rules/*.md
	entries, _ := os.ReadDir(filepath.Join(loc.Root, ".roo", "rules"))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(loc.Root, ".roo", "rules", e.Name()))
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		b.Instructions = append(b.Instructions, ir.Instruction{
			Header:     ir.Header{ID: "instruction." + name, IRVersion: 1, Origin: &ir.Origin{Tool: "roo", Path: ".roo/rules/" + e.Name(), Scope: loc.Scope}},
			Activation: ir.ActivationAlways, AppliesTo: []string{"roo"}, Priority: 200, Body: string(data),
		})
	}
	// .roomodes（custom modes → agent，权限进 x-roo）
	if data, err := os.ReadFile(filepath.Join(loc.Root, ".roomodes")); err == nil {
		var f struct {
			CustomModes []roomode `yaml:"customModes"`
		}
		if yaml.Unmarshal(data, &f) == nil {
			for _, m := range f.CustomModes {
				p := ir.PromptPack{
					Header: ir.Header{
						ID:         "agent." + m.Slug,
						IRVersion:  1,
						Origin:     &ir.Origin{Tool: "roo", Path: ".roomodes", Scope: loc.Scope},
						Extensions: map[string]any{"x-roo": map[string]any{"groups": m.Groups}},
					},
					Kind:        ir.KindAgent,
					Name:        m.Slug,
					Description: m.RoleDefinition,
					Body:        m.CustomInstructions,
				}
				b.Agents = append(b.Agents, p)
			}
		}
	}
	// .roo/mcp.json
	if data, err := os.ReadFile(filepath.Join(loc.Root, ".roo", "mcp.json")); err == nil {
		var f struct {
			MCPServers map[string]map[string]any `json:"mcpServers"`
		}
		if json.Unmarshal(data, &f) == nil {
			for name, conf := range f.MCPServers {
				s := ir.MCPServer{
					Header:    ir.Header{ID: "mcp." + name, IRVersion: 1, Origin: &ir.Origin{Tool: "roo", Path: ".roo/mcp.json", Scope: loc.Scope}},
					Name:      name,
					Transport: strOf(conf, "type", "stdio"), Command: strOf(conf, "command", ""), URL: strOf(conf, "url", ""),
				}
				b.MCPServers = append(b.MCPServers, s)
			}
		}
	}
	return b, nil
}

func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	root := opts.ProjectRoot
	if root == "" {
		home, _ := os.UserHomeDir()
		root = home
	}
	var files []adapters.WrittenFile
	add := func(p string, c []byte) error {
		sum := sha256.Sum256(c)
		files = append(files, adapters.WrittenFile{Path: p, Hash: hex.EncodeToString(sum[:]), Content: c})
		return nil
	}
	for _, inst := range b.Instructions {
		add(filepath.Join(root, ".roo", "rules", ir.NameTail(inst.ID)+".md"), []byte(inst.Body))
	}
	// agents → .roomodes
	if len(b.Agents) > 0 {
		modes := map[string]any{"customModes": []any{}}
		var list []any
		for _, p := range b.Agents {
			m := map[string]any{"slug": p.Name, "name": p.Name, "roleDefinition": p.Description, "customInstructions": p.Body}
			if ext, ok := p.Extensions["x-roo"].(map[string]any); ok {
				m["groups"] = ext["groups"]
			}
			list = append(list, m)
		}
		modes["customModes"] = list
		data, _ := yaml.Marshal(modes)
		add(filepath.Join(root, ".roomodes"), data)
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
		data, _ := json.MarshalIndent(map[string]any{"mcpServers": servers}, "", "  ")
		add(filepath.Join(root, ".roo", "mcp.json"), data)
	}
	return files, nil
}

func strOf(m map[string]any, k, def string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return def
}
