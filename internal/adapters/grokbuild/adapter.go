package grokbuild

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
)

// Detect 探测 ~/.grok（全局）与项目 .grok/。
func (a *adapter) Detect(ctx context.Context) ([]adapters.Location, error) {
	var locs []adapters.Location
	if home, err := paths.Home(); err == nil {
		gd := filepath.Join(home, ".grok")
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
		if isDir(filepath.Join(abs, ".grok")) || isDir(filepath.Join(abs, ".git")) {
			return abs
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return ""
		}
		abs = parent
	}
}

// importLocation 按 scope 采集。
func (a *adapter) importLocation(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	b := &ir.Bundle{Scope: loc.Scope, IRVersion: 1}
	dir := loc.Root
	if loc.Scope == ir.ScopeProject {
		dir = filepath.Join(loc.Root, ".grok")
	}
	// config.toml
	a.readConfig(b, filepath.Join(dir, "config.toml"), loc.Scope)
	// AGENTS.md（项目层）/ ~/.grok 指令
	for _, name := range []string{"AGENTS.md", "GROK.md"} {
		if data, err := os.ReadFile(filepath.Join(loc.Root, name)); err == nil {
			b.Instructions = append(b.Instructions, ir.Instruction{
				Header:     ir.Header{ID: "instruction." + strings.ToLower(strings.TrimSuffix(name, ".md")), IRVersion: 1, Origin: &ir.Origin{Tool: "grokbuild", Path: name, Scope: loc.Scope}},
				Activation: ir.ActivationAlways,
				AppliesTo:  []string{"grokbuild"},
				Body:       string(data),
			})
			break
		}
	}
	// skills/<name>/SKILL.md
	a.readSkills(b, filepath.Join(dir, "skills"), loc.Scope)
	// subagents/<name>.md
	a.readPacks(b, filepath.Join(dir, "subagents"), ir.KindAgent, loc.Scope)
	return b, nil
}

func (a *adapter) readConfig(b *ir.Bundle, path string, scope ir.Scope) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var root map[string]any
	if err := toml.Unmarshal(data, &root); err != nil {
		return
	}
	for key, val := range root {
		if key == "mcp_servers" || key == "mcpServers" {
			if m, ok := val.(map[string]any); ok {
				for name, confAny := range m {
					if conf, ok := confAny.(map[string]any); ok {
						s := ir.MCPServer{
							Header:    ir.Header{ID: "mcp." + sanitize(name), IRVersion: 1, Origin: &ir.Origin{Tool: "grokbuild", Path: "config.toml", Scope: scope}},
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
			continue
		}
		b.Settings = append(b.Settings, ir.SettingEntry{
			Header:    ir.Header{ID: "setting.grokbuild." + key, IRVersion: 1, Origin: &ir.Origin{Tool: "grokbuild", Path: "config.toml", Scope: scope}},
			Key:       key,
			Value:     val,
			ToolScope: []string{"grokbuild"},
		})
	}
}

func (a *adapter) readSkills(b *ir.Bundle, dir string, scope ir.Scope) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		var p ir.PromptPack
		body, ext, err := ir.UnmarshalMarkdownDoc(data, &p)
		if err != nil {
			continue
		}
		p.Extensions = ext
		p.Body = body
		p.Kind = ir.KindSkill
		if p.Name == "" {
			p.Name = e.Name()
		}
		if p.ID == "" {
			p.ID = "skill." + sanitize(p.Name)
		}
		p.Origin = &ir.Origin{Tool: "grokbuild", Path: "skills/" + e.Name() + "/SKILL.md", Scope: scope}
		b.Skills = append(b.Skills, p)
		b.Add(ir.KindSkill, p.ID)
	}
}

func (a *adapter) readPacks(b *ir.Bundle, dir string, kind ir.EntityKind, scope ir.Scope) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var p ir.PromptPack
		body, ext, err := ir.UnmarshalMarkdownDoc(data, &p)
		if err != nil {
			continue
		}
		p.Extensions = ext
		p.Body = body
		p.Kind = kind
		if p.Name == "" {
			p.Name = strings.TrimSuffix(e.Name(), ".md")
		}
		if p.ID == "" {
			p.ID = string(kind) + "." + sanitize(p.Name)
		}
		p.Origin = &ir.Origin{Tool: "grokbuild", Path: "subagents/" + e.Name(), Scope: scope}
		b.Agents = append(b.Agents, p)
		b.Add(kind, p.ID)
	}
}

// exportBundle 物化到 .grok/。
func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	project := opts.ProjectRoot != ""
	var dir, agentsBase string
	if project {
		dir = filepath.Join(opts.ProjectRoot, ".grok")
		agentsBase = opts.ProjectRoot
	} else {
		home, err := paths.Home()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, ".grok")
		agentsBase = dir
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

	// config.toml（settings + mcp_servers）
	if len(b.Settings) > 0 || len(b.MCPServers) > 0 {
		root := map[string]any{}
		for _, s := range b.Settings {
			root[s.Key] = s.Value
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
				if s.Transport != "" && s.Transport != "stdio" {
					conf["type"] = s.Transport
				}
				name := s.Name
				if name == "" {
					name = ir.NameTail(s.ID)
				}
				servers[name] = conf
			}
			root["mcp_servers"] = servers
		}
		data, err := toml.Marshal(root)
		if err != nil {
			return nil, err
		}
		if err := add(filepath.Join(dir, "config.toml"), data); err != nil {
			return nil, err
		}
	}
	// instructions → AGENTS.md
	for _, inst := range b.Instructions {
		if err := add(filepath.Join(agentsBase, "AGENTS.md"), []byte(inst.Body)); err != nil {
			return nil, err
		}
	}
	// skills
	for _, p := range b.Skills {
		var sb strings.Builder
		sb.WriteString("---\nname: " + p.Name + "\n")
		if p.Description != "" {
			sb.WriteString("description: " + p.Description + "\n")
		}
		sb.WriteString("---\n" + p.Body)
		if err := add(filepath.Join(dir, "skills", p.Name, "SKILL.md"), []byte(sb.String())); err != nil {
			return nil, err
		}
	}
	// subagents
	for _, p := range b.Agents {
		var sb strings.Builder
		sb.WriteString("---\nname: " + p.Name + "\n---\n" + p.Body)
		if err := add(filepath.Join(dir, "subagents", p.Name+".md"), []byte(sb.String())); err != nil {
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
