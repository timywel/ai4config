package cursor

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

// Detect 探测 .cursor/（项目级为主）+ ~/.cursor/rules（全局）。
func (a *adapter) Detect(ctx context.Context) ([]adapters.Location, error) {
	var locs []adapters.Location
	if home, err := os.UserHomeDir(); err == nil {
		gd := filepath.Join(home, ".cursor")
		if isDir(gd) {
			locs = append(locs, adapters.Location{Scope: ir.ScopeGlobal, Root: gd})
		}
	}
	if projRoot := findProjectRoot("."); projRoot != "" {
		locs = append(locs, adapters.Location{Scope: ir.ScopeProject, Root: projRoot})
	}
	return locs, nil
}

func isDir(p string) bool  { i, e := os.Stat(p); return e == nil && i.IsDir() }
func isFile(p string) bool { i, e := os.Stat(p); return e == nil && !i.IsDir() }

func findProjectRoot(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		if isDir(filepath.Join(abs, ".cursor")) || isFile(filepath.Join(abs, ".cursorrules")) || isDir(filepath.Join(abs, ".git")) {
			return abs
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return ""
		}
		abs = parent
	}
}

// ruleFrontmatter .mdc frontmatter（description/globs/alwaysApply）。
type ruleFrontmatter struct {
	Description string `yaml:"description"`
	Globs       string `yaml:"globs"`
	AlwaysApply bool   `yaml:"alwaysApply"`
}

func (a *adapter) importLocation(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	b := &ir.Bundle{Scope: loc.Scope, IRVersion: 1}
	rulesDir := filepath.Join(loc.Root, ".cursor", "rules")
	if loc.Scope == ir.ScopeGlobal {
		rulesDir = filepath.Join(loc.Root, "rules")
	}
	// rules/*.mdc
	entries, err := os.ReadDir(rulesDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".mdc") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(rulesDir, e.Name()))
			if err != nil {
				continue
			}
			var fm ruleFrontmatter
			body, ext, _ := ir.UnmarshalMarkdownDoc(data, &fm)
			name := strings.TrimSuffix(e.Name(), ".mdc")
			inst := ir.Instruction{
				Header: ir.Header{
					ID:         "instruction." + sanitize(name),
					IRVersion:  1,
					Origin:     &ir.Origin{Tool: "cursor", Path: ".cursor/rules/" + e.Name(), Scope: loc.Scope},
					Extensions: ext,
				},
				Description: fm.Description,
				AppliesTo:   []string{"cursor"},
				Priority:    defaultPriority(loc.Scope),
				Body:        body,
			}
			if fm.AlwaysApply {
				inst.Activation = ir.ActivationAlways
			} else if fm.Globs != "" {
				inst.Activation = ir.ActivationGlob
				inst.FilePatterns = strings.Split(fm.Globs, ",")
			} else {
				inst.Activation = ir.ActivationModelDecision
			}
			b.Instructions = append(b.Instructions, inst)
			b.Add(ir.KindInstruction, inst.ID)
		}
	}
	// .cursor/mcp.json
	mcpPath := filepath.Join(loc.Root, ".cursor", "mcp.json")
	if loc.Scope == ir.ScopeGlobal {
		mcpPath = filepath.Join(loc.Root, "mcp.json")
	}
	if data, err := os.ReadFile(mcpPath); err == nil {
		var f struct {
			MCPServers map[string]map[string]any `json:"mcpServers"`
		}
		if json.Unmarshal(data, &f) == nil {
			for name, conf := range f.MCPServers {
				s := ir.MCPServer{
					Header:    ir.Header{ID: "mcp." + sanitize(name), IRVersion: 1, Origin: &ir.Origin{Tool: "cursor", Path: ".cursor/mcp.json", Scope: loc.Scope}},
					Name:      name,
					Transport: getStr(conf, "type", "stdio"),
					Command:   getStr(conf, "command", ""),
					URL:       getStr(conf, "url", ""),
				}
				b.MCPServers = append(b.MCPServers, s)
				b.Add(ir.KindMCP, s.ID)
			}
		}
	}
	// .cursorrules（legacy 单文件）
	if loc.Scope == ir.ScopeProject {
		if data, err := os.ReadFile(filepath.Join(loc.Root, ".cursorrules")); err == nil {
			inst := ir.Instruction{
				Header:     ir.Header{ID: "instruction.cursorrules", IRVersion: 1, Origin: &ir.Origin{Tool: "cursor", Path: ".cursorrules", Scope: loc.Scope}},
				Activation: ir.ActivationAlways,
				AppliesTo:  []string{"cursor"},
				Priority:   200,
				Body:       string(data),
			}
			b.Instructions = append(b.Instructions, inst)
			b.Add(ir.KindInstruction, inst.ID)
		}
	}
	return b, nil
}

// exportBundle 物化 .cursor/。
func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	project := opts.ProjectRoot != ""
	var base string
	if project {
		base = filepath.Join(opts.ProjectRoot, ".cursor")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		base = filepath.Join(home, ".cursor")
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
	// instructions → rules/<name>.mdc
	for _, inst := range b.Instructions {
		var fm strings.Builder
		fm.WriteString("---\n")
		if inst.Description != "" {
			fm.WriteString("description: " + inst.Description + "\n")
		}
		if len(inst.FilePatterns) > 0 {
			fm.WriteString("globs: \"" + strings.Join(inst.FilePatterns, ",") + "\"\n")
		}
		fm.WriteString("alwaysApply: " + boolStr(inst.Activation == ir.ActivationAlways) + "\n---\n")
		name := ir.NameTail(inst.ID)
		if err := add(filepath.Join(base, "rules", name+".mdc"), []byte(fm.String()+inst.Body)); err != nil {
			return nil, err
		}
	}
	// mcp → mcp.json
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
		data, err := json.MarshalIndent(map[string]any{"mcpServers": servers}, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := add(filepath.Join(base, "mcp.json"), data); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func writeOne(path string, content []byte, dryRun bool) (adapters.WrittenFile, error) {
	_ = dryRun
	sum := sha256.Sum256(content)
	return adapters.WrittenFile{Path: path, Hash: hex.EncodeToString(sum[:]), Content: content}, nil
}

func defaultPriority(s ir.Scope) int {
	if s == ir.ScopeProject || s == ir.ScopeLocal {
		return 200
	}
	if s == ir.ScopeRemote {
		return 150
	}
	return 100
}
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
func sanitize(s string) string {
	var b []rune
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			b = append(b, r)
		} else {
			b = append(b, '-')
		}
	}
	return string(b)
}
func getStr(m map[string]any, k, def string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return def
}
