// Package cline 是 Cline（VS Code 扩展）的适配器。配置：.clinerules/、cline_mcp_settings.json。
package cline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func init() { adapters.Register(&adapter{}) }

type adapter struct{}

func (a *adapter) Meta() adapters.ToolMeta {
	return adapters.ToolMeta{
		ID: "cline", DisplayName: "Cline", MinVersion: "3.0", MaxVersion: "3.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportFull, Note: ".clinerules/ 目录"},
			ir.KindMCP:         {Level: adapters.SupportFull, Note: "cline_mcp_settings.json"},
			ir.KindSkill:       {Level: adapters.SupportNone}, ir.KindAgent: {Level: adapters.SupportNone},
			ir.KindCommand: {Level: adapters.SupportNone}, ir.KindWorkflow: {Level: adapters.SupportNone},
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
		if isDir(filepath.Join(abs, ".clinerules")) || isDir(filepath.Join(abs, ".git")) {
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
	// .clinerules/*.md
	entries, _ := os.ReadDir(filepath.Join(loc.Root, ".clinerules"))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(loc.Root, ".clinerules", e.Name()))
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		b.Instructions = append(b.Instructions, ir.Instruction{
			Header:     ir.Header{ID: "instruction." + name, IRVersion: 1, Origin: &ir.Origin{Tool: "cline", Path: ".clinerules/" + e.Name(), Scope: loc.Scope}},
			Activation: ir.ActivationAlways, AppliesTo: []string{"cline"}, Priority: 200, Body: string(data),
		})
	}
	// cline_mcp_settings.json
	if data, err := os.ReadFile(filepath.Join(loc.Root, "cline_mcp_settings.json")); err == nil {
		var f struct {
			MCPServers map[string]map[string]any `json:"mcpServers"`
		}
		if json.Unmarshal(data, &f) == nil {
			for name, conf := range f.MCPServers {
				s := ir.MCPServer{
					Header:    ir.Header{ID: "mcp." + name, IRVersion: 1, Origin: &ir.Origin{Tool: "cline", Path: "cline_mcp_settings.json", Scope: loc.Scope}},
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
		add(filepath.Join(root, ".clinerules", ir.NameTail(inst.ID)+".md"), []byte(inst.Body))
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
		add(filepath.Join(root, "cline_mcp_settings.json"), data)
	}
	return files, nil
}

func strOf(m map[string]any, k, def string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return def
}
