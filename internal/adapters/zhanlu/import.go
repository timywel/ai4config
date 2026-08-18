package zhanlu

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
)

// importLocation 按 scope 分发采集（Zhanlu Import 主流程）。
func (a *adapter) importLocation(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	b := &ir.Bundle{Scope: loc.Scope, IRVersion: 1}
	switch loc.Scope {
	case ir.ScopeGlobal:
		a.importGlobal(loc, b)
	case ir.ScopeProject:
		a.importProject(loc, b)
	}
	return b, nil
}

// importGlobal 采集全局层：zhanlu.json + 全局 AGENTS.md + ~/.agents/skills/。
func (a *adapter) importGlobal(loc adapters.Location, b *ir.Bundle) {
	dir := loc.Root // ~/.config/zhanlu
	// zhanlu.json / zhanlu.jsonc
	for _, name := range []string{"zhanlu.json", "zhanlu.jsonc"} {
		if data, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			settings, mcps, err := parseZhanluJSON(data, ir.ScopeGlobal, "~/.config/zhanlu/"+name)
			if err == nil {
				b.Settings = append(b.Settings, settings...)
				b.MCPServers = append(b.MCPServers, mcps...)
				for _, s := range settings {
					b.Add(ir.KindSetting, s.ID)
				}
				for _, m := range mcps {
					b.Add(ir.KindMCP, m.ID)
				}
			}
			break
		}
	}
	// 全局 AGENTS.md（~/.agents/AGENTS.md 或 ~/.config/zhanlu/AGENTS.md）
	if home, err := paths.Home(); err == nil {
		for _, p := range []string{filepath.Join(home, ".agents", "AGENTS.md"), filepath.Join(dir, "AGENTS.md")} {
			if data, err := os.ReadFile(p); err == nil {
				inst := parseAgentsMD(string(data), ir.ScopeGlobal, "~/.agents/AGENTS.md", "")
				b.Instructions = append(b.Instructions, inst)
				b.Add(ir.KindInstruction, inst.ID)
				break
			}
		}
		// ~/.agents/skills/<name>/SKILL.md
		a.readSkillsDir(b, filepath.Join(home, ".agents", "skills"), ir.ScopeGlobal, "~/.agents/skills")
	}
}

// importProject 采集项目层：AGENTS.md、.kilo/agent|command、kilo.json。
func (a *adapter) importProject(loc adapters.Location, b *ir.Bundle) {
	root := loc.Root
	// AGENTS.md
	if data, err := os.ReadFile(filepath.Join(root, "AGENTS.md")); err == nil {
		inst := parseAgentsMD(string(data), ir.ScopeProject, "AGENTS.md", "")
		b.Instructions = append(b.Instructions, inst)
		b.Add(ir.KindInstruction, inst.ID)
	}
	// .kilo/agent/*.md、.kilo/command/*.md
	a.readKiloDir(b, filepath.Join(root, ".kilo", "agent"), ir.KindAgent, ".kilo/agent")
	a.readKiloDir(b, filepath.Join(root, ".kilo", "command"), ir.KindCommand, ".kilo/command")
	// kilo.json（项目主配置）
	if data, err := os.ReadFile(filepath.Join(root, "kilo.json")); err == nil {
		settings, mcps, err := parseZhanluJSON(data, ir.ScopeProject, "kilo.json")
		if err == nil {
			b.Settings = append(b.Settings, settings...)
			b.MCPServers = append(b.MCPServers, mcps...)
		}
	}
	// .zhanlu/ 下多为运行时锁文件（agent-manager.json 等），不采集
}

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
		p := parsePackMD(data, ir.KindSkill, e.Name(), scope, rawBase+"/"+e.Name()+"/SKILL.md")
		b.Skills = append(b.Skills, p)
		b.Add(ir.KindSkill, p.ID)
	}
}

func (a *adapter) readKiloDir(b *ir.Bundle, dir string, kind ir.EntityKind, rawBase string) {
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
		p := parsePackMD(data, kind, name, ir.ScopeProject, rawBase+"/"+e.Name())
		if kind == ir.KindAgent {
			b.Agents = append(b.Agents, p)
		} else {
			b.Commands = append(b.Commands, p)
		}
		b.Add(kind, p.ID)
	}
}
