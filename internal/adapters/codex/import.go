package codex

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

// importLocation 按 scope 分发采集（Codex Import 主流程）。
func (a *adapter) importLocation(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	b := &ir.Bundle{Scope: loc.Scope, IRVersion: 1}
	var err error
	switch loc.Scope {
	case ir.ScopeGlobal:
		err = a.importGlobal(loc, b)
	case ir.ScopeProject:
		err = a.importProject(loc, b)
	}
	return b, err
}

// importGlobal 采集全局层：~/.codex/（config.toml、AGENTS.md/override、skills/）。
func (a *adapter) importGlobal(loc adapters.Location, b *ir.Bundle) error {
	dir := loc.Root // ~/.codex（或 CODEX_HOME）

	// config.toml → settings + mcp + hooks
	a.readConfigTOML(b, filepath.Join(dir, "config.toml"), ir.ScopeGlobal, "~/.codex/config.toml")
	// AGENTS.md（AGENTS.override.md 优先）
	a.readAgentsMD(b, dir, ir.ScopeGlobal, "~/.codex", "")
	// skills/<name>/SKILL.md
	a.readSkillsDir(b, filepath.Join(dir, "skills"), ir.ScopeGlobal, "~/.codex/skills")
	// agents/<name>.md
	a.readPackDir(b, filepath.Join(dir, "agents"), ir.KindAgent, ir.ScopeGlobal, "~/.codex/agents")
	// auth.json：仅扫描敏感值（不读明文进 IR；抽取为 secretref 由脱敏管线处理）
	a.noteAuthFile(b, filepath.Join(dir, "auth.json"))
	return nil
}

// importProject 采集项目层：AGENTS.md 逐目录（含 override 优先）+ .codex/config.toml（trusted-gate）。
func (a *adapter) importProject(loc adapters.Location, b *ir.Bundle) error {
	root := loc.Root

	// trusted-gate：项目级 .codex/config.toml 仅 trusted 项目加载
	trusted := a.checkTrusted(root)
	if !trusted {
		b.Warnings = append(b.Warnings, ir.Warning{
			Kind:    "trust-gate",
			Message: "项目未被 trust（~/.codex/config.toml projects.<path>.trust_level），跳过项目级 .codex/ 配置",
		})
	}

	// AGENTS.md 逐目录（含 AGENTS.override.md 优先）
	a.readAgentsTree(b, root)

	// .codex/config.toml（trusted 才读）
	if trusted {
		a.readConfigTOML(b, filepath.Join(root, ".codex", "config.toml"), ir.ScopeProject, ".codex/config.toml")
	}
	return nil
}

// ---------- 文件级读取器 ----------

// readConfigTOML 读取 config.toml。
func (a *adapter) readConfigTOML(b *ir.Bundle, path string, scope ir.Scope, rawPath string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	settings, mcps, hooks, err := parseConfigTOML(data, scope, rawPath)
	if err != nil {
		b.Warnings = append(b.Warnings, ir.Warning{Kind: "parse", Message: "config.toml 解析失败: " + err.Error()})
		return
	}
	b.Settings = append(b.Settings, settings...)
	b.MCPServers = append(b.MCPServers, mcps...)
	b.Hooks = append(b.Hooks, hooks...)
	for _, s := range settings {
		b.Add(ir.KindSetting, s.ID)
	}
	for _, m := range mcps {
		b.Add(ir.KindMCP, m.ID)
	}
	for _, h := range hooks {
		b.Add(ir.KindHook, h.ID)
	}
}

// readAgentsMD 读取单目录的 AGENTS.md（AGENTS.override.md 优先替代）。
func (a *adapter) readAgentsMD(b *ir.Bundle, dir string, scope ir.Scope, rawBase string, subtree string) {
	path := filepath.Join(dir, "AGENTS.md")
	rawPath := joinRaw(rawBase, "AGENTS.md")
	// override 优先
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.override.md")); err == nil {
		path = filepath.Join(dir, "AGENTS.override.md")
		rawPath = joinRaw(rawBase, "AGENTS.override.md")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	id := "instruction.agents"
	if subtree != "" {
		id = "instruction.agents." + sanitizeIDName(strings.ReplaceAll(subtree, "/", "-"))
	}
	inst := ir.Instruction{
		Header: ir.Header{
			ID:        id,
			IRVersion: 1,
			Origin:    &ir.Origin{Tool: "codex", Path: rawPath, Scope: scope},
		},
		Activation: ir.ActivationAlways,
		AppliesTo:  []string{"codex"},
		Priority:   defaultPriority(scope),
		Subtree:    subtree,
		Body:       string(data),
	}
	if strings.HasSuffix(path, "override.md") {
		inst.Extensions = map[string]any{"x-codex": map[string]any{"override": true}}
	}
	b.Instructions = append(b.Instructions, inst)
	b.Add(ir.KindInstruction, inst.ID)
}

// readAgentsTree 遍历项目树，采集各目录的 AGENTS.md（subtree 记录相对路径）。
func (a *adapter) readAgentsTree(b *ir.Bundle, root string) {
	// 项目根本层
	a.readAgentsMD(b, root, ir.ScopeProject, "AGENTS.md", "")
	// 子目录（跳过 .git/.codex 等隐藏与依赖目录）
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name != "." && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "AGENTS.md" && d.Name() != "AGENTS.override.md" {
			return nil
		}
		dir := filepath.Dir(path)
		if dir == root {
			return nil // 根目录已处理
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return nil
		}
		a.readAgentsMD(b, dir, ir.ScopeProject, filepath.ToSlash(rel), filepath.ToSlash(rel))
		return nil
	})
}

// readSkillsDir 读取 skills/<name>/SKILL.md。
func (a *adapter) readSkillsDir(b *ir.Bundle, dir string, scope ir.Scope, rawBase string) {
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
			p.ID = "skill." + sanitizeIDName(p.Name)
		}
		p.Origin = &ir.Origin{Tool: "codex", Path: joinRaw(rawBase, e.Name()+"/SKILL.md"), Scope: scope}
		b.Skills = append(b.Skills, p)
		b.Add(ir.KindSkill, p.ID)
	}
}

// noteAuthFile 检测 auth.json 存在性并记录（敏感文件，不读明文进 IR）。
func (a *adapter) noteAuthFile(b *ir.Bundle, path string) {
	if _, err := os.Stat(path); err != nil {
		return
	}
	b.Warnings = append(b.Warnings, ir.Warning{
		Kind:    "secret-file",
		Message: "检测到 auth.json（认证凭据），不采集明文；如需迁移请凭 secretref 另行处理",
	})
}

// checkTrusted 检查项目是否被 trust（读全局 config.toml 的 projects.<path>.trust_level）。
func (a *adapter) checkTrusted(projectRoot string) bool {
	globalDir := os.Getenv("CODEX_HOME")
	if globalDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return true // 无法判定时默认 trust（不阻塞采集）
		}
		globalDir = filepath.Join(home, ".codex")
	}
	data, err := os.ReadFile(filepath.Join(globalDir, "config.toml"))
	if err != nil {
		return true // 无全局 config 默认 trust
	}
	var root map[string]any
	if err := toml.Unmarshal(data, &root); err != nil {
		return true
	}
	projects, ok := root["projects"].(map[string]any)
	if !ok {
		return true // 无 projects 键默认 trust
	}
	proj, ok := projects[projectRoot].(map[string]any)
	if !ok {
		return true // 未登记默认按 codex 行为（首次询问）；采集侧不阻塞
	}
	level, _ := proj["trust_level"].(string)
	return level != "untrusted"
}

// defaultPriority 按 scope 给指令默认优先级（IR-SCHEMA §3.1）。
func defaultPriority(scope ir.Scope) int {
	switch scope {
	case ir.ScopeProject, ir.ScopeLocal:
		return 200
	case ir.ScopeRemote:
		return 150
	default:
		return 100
	}
}

// joinRaw 拼接 raw 形态路径（统一 / 分隔）。
func joinRaw(base, name string) string {
	if base == "" {
		return name
	}
	return strings.TrimSuffix(base, "/") + "/" + name
}

// readPackDir reads <dir>/*.md as PromptPack entries (agents etc.).
func (a *adapter) readPackDir(b *ir.Bundle, dir string, kind ir.EntityKind, scope ir.Scope, rawBase string) {
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
			p.ID = string(kind) + "." + sanitizeIDName(p.Name)
		}
		p.Origin = &ir.Origin{Tool: "codex", Path: joinRaw(rawBase, e.Name()), Scope: scope}
		if kind == ir.KindAgent {
			b.Agents = append(b.Agents, p)
		}
		b.Add(kind, p.ID)
	}
}
