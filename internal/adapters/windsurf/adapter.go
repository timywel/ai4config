package windsurf

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

// Detect 探测 .windsurf/ 与 .devin/（双读，品牌迁移兼容）。
func (a *adapter) Detect(ctx context.Context) ([]adapters.Location, error) {
	var locs []adapters.Location
	if home, err := os.UserHomeDir(); err == nil {
		for _, d := range []string{".windsurf", ".devin"} {
			if gd := filepath.Join(home, d); isDir(gd) {
				locs = append(locs, adapters.Location{Scope: ir.ScopeGlobal, Root: gd})
				break
			}
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
	abs, _ := filepath.Abs(dir)
	for {
		if isDir(filepath.Join(abs, ".windsurf")) || isDir(filepath.Join(abs, ".devin")) || isFile(filepath.Join(abs, ".windsurfrules")) || isDir(filepath.Join(abs, ".git")) {
			return abs
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return ""
		}
		abs = parent
	}
}

// configDir 返回配置目录（.devin 优先，回退 .windsurf）。
func configDir(root string) string {
	if isDir(filepath.Join(root, ".devin")) {
		return filepath.Join(root, ".devin")
	}
	return filepath.Join(root, ".windsurf")
}

func (a *adapter) importLocation(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	b := &ir.Bundle{Scope: loc.Scope, IRVersion: 1}
	var base string
	if loc.Scope == ir.ScopeGlobal {
		base = loc.Root
	} else {
		base = configDir(loc.Root)
	}
	// rules/*.md
	entries, err := os.ReadDir(filepath.Join(base, "rules"))
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(base, "rules", e.Name()))
			if err != nil {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".md")
			var p ir.PromptPack
			body, ext, _ := ir.UnmarshalMarkdownDoc(data, &p)
			p.Extensions = ext
			p.Body = body
			b.Instructions = append(b.Instructions, ir.Instruction{
				Header:     ir.Header{ID: "instruction." + sanitize(name), IRVersion: 1, Origin: &ir.Origin{Tool: "windsurf", Path: "rules/" + e.Name(), Scope: loc.Scope}},
				Activation: ir.ActivationAlways,
				AppliesTo:  []string{"windsurf"},
				Priority:   200,
				Body:       body,
			})
			b.Add(ir.KindInstruction, b.Instructions[len(b.Instructions)-1].ID)
		}
	}
	// .windsurfrules（legacy 单文件，项目层）
	if loc.Scope == ir.ScopeProject {
		if data, err := os.ReadFile(filepath.Join(loc.Root, ".windsurfrules")); err == nil {
			b.Instructions = append(b.Instructions, ir.Instruction{
				Header:     ir.Header{ID: "instruction.windsurfrules", IRVersion: 1, Origin: &ir.Origin{Tool: "windsurf", Path: ".windsurfrules", Scope: loc.Scope}},
				Activation: ir.ActivationAlways,
				AppliesTo:  []string{"windsurf"},
				Priority:   200,
				Body:       string(data),
			})
		}
		// workflows/*.md
		wfEntries, _ := os.ReadDir(filepath.Join(base, "workflows"))
		for _, e := range wfEntries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(base, "workflows", e.Name()))
			if err != nil {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".md")
			b.Workflows = append(b.Workflows, ir.PromptPack{
				Header: ir.Header{ID: "workflow." + sanitize(name), IRVersion: 1, Origin: &ir.Origin{Tool: "windsurf", Path: "workflows/" + e.Name(), Scope: loc.Scope}},
				Kind:   ir.KindWorkflow,
				Name:   name,
				Body:   string(data),
			})
		}
	}
	// mcp_config.json
	if data, err := os.ReadFile(filepath.Join(base, "mcp_config.json")); err == nil {
		var f struct {
			MCPServers map[string]map[string]any `json:"mcpServers"`
		}
		if json.Unmarshal(data, &f) == nil {
			for name, conf := range f.MCPServers {
				s := ir.MCPServer{
					Header:    ir.Header{ID: "mcp." + sanitize(name), IRVersion: 1, Origin: &ir.Origin{Tool: "windsurf", Path: "mcp_config.json", Scope: loc.Scope}},
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
	return b, nil
}

// exportBundle 物化 .windsurf/（或 .devin/）。
func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	project := opts.ProjectRoot != ""
	var base string
	if project {
		base = configDir(opts.ProjectRoot)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		base = filepath.Join(home, ".windsurf")
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
	for _, inst := range b.Instructions {
		name := ir.NameTail(inst.ID)
		if err := add(filepath.Join(base, "rules", name+".md"), []byte(inst.Body)); err != nil {
			return nil, err
		}
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
		if err := add(filepath.Join(base, "mcp_config.json"), data); err != nil {
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
