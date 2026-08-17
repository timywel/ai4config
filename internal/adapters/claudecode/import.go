package claudecode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
)

// importLocation 按 scope 分发采集（Import 主流程）。
func (a *adapter) importLocation(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	b := &ir.Bundle{Scope: loc.Scope, IRVersion: 1}
	var err error
	switch loc.Scope {
	case ir.ScopeGlobal:
		err = a.importGlobal(loc, b)
	case ir.ScopeProject:
		err = a.importProject(loc, b)
	case ir.ScopeManaged:
		err = a.importManaged(loc, b)
	}
	if err != nil {
		return b, err
	}
	indexBundle(b)
	return b, nil
}

// importGlobal 采集全局层：~/.claude/ 全部 + ~/.claude.json 的 mcpServers。
func (a *adapter) importGlobal(loc adapters.Location, b *ir.Bundle) error {
	dir := loc.Root // ~/.claude
	home, _ := paths.Home()

	// CLAUDE.md（全局指令）
	a.readInstructionFile(b, filepath.Join(dir, "CLAUDE.md"), ir.ScopeGlobal, "~/.claude/CLAUDE.md", "")
	// rules/*.md（含 frontmatter paths → file_patterns）
	a.readRulesDir(b, filepath.Join(dir, "rules"), ir.ScopeGlobal, "~/.claude/rules")
	// agents/*.md、commands/*.md（legacy）
	a.readPackDir(b, filepath.Join(dir, "agents"), ir.KindAgent, ir.ScopeGlobal, "~/.claude/agents")
	a.readPackDir(b, filepath.Join(dir, "commands"), ir.KindCommand, ir.ScopeGlobal, "~/.claude/commands")
	// skills/<name>/SKILL.md
	a.readSkillsDir(b, filepath.Join(dir, "skills"), ir.ScopeGlobal, "~/.claude/skills")
	// settings.json → settings + hooks
	a.readSettingsFile(b, filepath.Join(dir, "settings.json"), ir.ScopeGlobal, "~/.claude/settings.json")
	// ~/.claude.json 的 mcpServers（user scope；该文件含运行时状态，仅取 mcpServers 键——局部 patch 原则）
	a.readClaudeJSONMCP(b, filepath.Join(home, ".claude.json"), ir.ScopeGlobal, "~/.claude.json")
	return nil
}

// importProject 采集项目层：<root>/CLAUDE.md、.claude/、.mcp.json、CLAUDE.local.md（local）。
func (a *adapter) importProject(loc adapters.Location, b *ir.Bundle) error {
	root := loc.Root
	claudeDir := filepath.Join(root, ".claude")

	// <root>/CLAUDE.md 与 .claude/CLAUDE.md（向上逐级拼接的叶子；此处只采本层）
	a.readInstructionFile(b, filepath.Join(root, "CLAUDE.md"), ir.ScopeProject, "CLAUDE.md", "")
	a.readInstructionFile(b, filepath.Join(claudeDir, "CLAUDE.md"), ir.ScopeProject, ".claude/CLAUDE.md", "")
	// CLAUDE.local.md（项目内私人层 → scope=local）
	a.readInstructionFile(b, filepath.Join(root, "CLAUDE.local.md"), ir.ScopeLocal, "CLAUDE.local.md", "")
	// .mcp.json
	a.readMCPFile(b, filepath.Join(root, ".mcp.json"), ir.ScopeProject, ".mcp.json")
	// .claude/rules、agents、commands、skills
	a.readRulesDir(b, filepath.Join(claudeDir, "rules"), ir.ScopeProject, ".claude/rules")
	a.readPackDir(b, filepath.Join(claudeDir, "agents"), ir.KindAgent, ir.ScopeProject, ".claude/agents")
	a.readPackDir(b, filepath.Join(claudeDir, "commands"), ir.KindCommand, ir.ScopeProject, ".claude/commands")
	a.readSkillsDir(b, filepath.Join(claudeDir, "skills"), ir.ScopeProject, ".claude/skills")
	// .claude/settings.json（项目）、settings.local.json（local 层）
	a.readSettingsFile(b, filepath.Join(claudeDir, "settings.json"), ir.ScopeProject, ".claude/settings.json")
	a.readSettingsFile(b, filepath.Join(claudeDir, "settings.local.json"), ir.ScopeLocal, ".claude/settings.local.json")
	return nil
}

// importManaged 采集企业管理层（只读，不物化）：仅 settings。
func (a *adapter) importManaged(loc adapters.Location, b *ir.Bundle) error {
	a.readSettingsFile(b, filepath.Join(loc.Root, "settings.json"), ir.ScopeManaged, managedSettingsRaw(loc.Root))
	return nil
}

// ---------- 文件级读取器（存在才读，缺失跳过） ----------

func (a *adapter) readInstructionFile(b *ir.Bundle, path string, scope ir.Scope, rawPath string, subtree string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	id := "instruction." + sanitizeIDName(strings.TrimSuffix(filepath.Base(path), ".md"))
	if scope == ir.ScopeLocal {
		id = "instruction." + sanitizeIDName(strings.TrimSuffix(filepath.Base(path), ".md"))
	}
	inst := parseInstructionMD(id, string(data), scope, rawPath, nil)
	inst.Subtree = subtree
	b.Instructions = append(b.Instructions, inst)
	b.Add(ir.KindInstruction, inst.ID)
}

func (a *adapter) readRulesDir(b *ir.Bundle, dir string, scope ir.Scope, rawPath string) {
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
		name := strings.TrimSuffix(e.Name(), ".md")
		inst := ir.Instruction{}
		body, ext, err := ir.UnmarshalMarkdownDoc(data, &inst)
		if err != nil {
			continue
		}
		inst.Extensions = ext
		inst.Body = body
		inst.ID = "instruction." + sanitizeIDName(name)
		inst.Activation = ir.ActivationGlob // rules 默认按 glob 激活
		inst.AppliesTo = []string{"claude-code"}
		inst.Priority = defaultPriority(scope)
		inst.Origin = &ir.Origin{Tool: "claude-code", Path: rawPath + "/" + e.Name(), Scope: scope}
		b.Instructions = append(b.Instructions, inst)
		b.Add(ir.KindInstruction, inst.ID)
	}
}

func (a *adapter) readPackDir(b *ir.Bundle, dir string, kind ir.EntityKind, scope ir.Scope, rawPath string) {
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
		name := strings.TrimSuffix(e.Name(), ".md")
		p, err := parsePromptPackMD(data, kind, name)
		if err != nil {
			continue
		}
		p.Origin = &ir.Origin{Tool: "claude-code", Path: rawPath + "/" + e.Name(), Scope: scope}
		appendPack(b, kind, p)
	}
}

func (a *adapter) readSkillsDir(b *ir.Bundle, dir string, scope ir.Scope, rawPath string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillFile := filepath.Join(dir, e.Name(), "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}
		p, err := parsePromptPackMD(data, ir.KindSkill, e.Name())
		if err != nil {
			continue
		}
		p.Origin = &ir.Origin{Tool: "claude-code", Path: rawPath + "/" + e.Name() + "/SKILL.md", Scope: scope}
		appendPack(b, ir.KindSkill, p)
	}
}

func (a *adapter) readSettingsFile(b *ir.Bundle, path string, scope ir.Scope, rawPath string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	settings, hooks, mcps, err := parseSettingsJSON(data, scope, rawPath)
	if err != nil {
		return
	}
	b.Settings = append(b.Settings, settings...)
	b.Hooks = append(b.Hooks, hooks...)
	b.MCPServers = append(b.MCPServers, mcps...)
	for _, s := range settings {
		b.Add(ir.KindSetting, s.ID)
	}
	for _, h := range hooks {
		b.Add(ir.KindHook, h.ID)
	}
	for _, m := range mcps {
		b.Add(ir.KindMCP, m.ID)
	}
}

func (a *adapter) readMCPFile(b *ir.Bundle, path string, scope ir.Scope, rawPath string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	mcps, err := parseMCPJSON(data, scope, rawPath)
	if err != nil {
		return
	}
	b.MCPServers = append(b.MCPServers, mcps...)
	for _, m := range mcps {
		b.Add(ir.KindMCP, m.ID)
	}
}

// readClaudeJSONMCP 从 ~/.claude.json 局部读取 mcpServers（该文件含运行时状态，只取所需键——局部 patch 原则）。
func (a *adapter) readClaudeJSONMCP(b *ir.Bundle, path string, scope ir.Scope, rawPath string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var f struct {
		MCPServers map[string]mcpServerConf `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &f); err != nil || len(f.MCPServers) == 0 {
		return
	}
	for name, conf := range f.MCPServers {
		transport := conf.Type
		if transport == "" {
			transport = "stdio"
		}
		s := ir.MCPServer{
			Header: ir.Header{
				ID:        "mcp." + sanitizeIDName(name),
				IRVersion: 1,
				Origin:    &ir.Origin{Tool: "claude-code", Path: rawPath, Scope: scope},
			},
			Name:      name,
			Transport: transport,
			Command:   conf.Command,
			Args:      conf.Args,
			Env:       conf.Env,
			URL:       conf.URL,
			Headers:   conf.Headers,
		}
		b.MCPServers = append(b.MCPServers, s)
		b.Add(ir.KindMCP, s.ID)
	}
}

// appendPack 按 kind 把 PromptPack 追加到对应切片并登记索引。
func appendPack(b *ir.Bundle, kind ir.EntityKind, p ir.PromptPack) {
	switch kind {
	case ir.KindSkill:
		b.Skills = append(b.Skills, p)
	case ir.KindAgent:
		b.Agents = append(b.Agents, p)
	case ir.KindCommand:
		b.Commands = append(b.Commands, p)
	case ir.KindWorkflow:
		b.Workflows = append(b.Workflows, p)
	}
	b.Add(kind, p.ID)
}

// indexBundle 确保索引完整（兜底）。
func indexBundle(b *ir.Bundle) {}

// managedSettingsRaw managed settings 的 raw 形态路径。
func managedSettingsRaw(root string) string {
	return filepath.Join(root, "settings.json")
}
