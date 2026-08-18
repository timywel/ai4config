package copilot

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

// importLocation 按 scope 分发采集（Copilot Import 主流程）。
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

// importGlobal 采集全局（VS Code user profile）：mcp.json、settings.json、instructions/、prompts/。
func (a *adapter) importGlobal(loc adapters.Location, b *ir.Bundle) {
	dir := loc.Root // <Code User>
	a.readMCPFile(b, filepath.Join(dir, "mcp.json"), ir.ScopeGlobal, "%APPDATA%/Code/User/mcp.json")
	a.readSettingsFile(b, filepath.Join(dir, "settings.json"), ir.ScopeGlobal, "%APPDATA%/Code/User/settings.json")
	a.readInstructionsDir(b, filepath.Join(dir, "instructions"), ir.ScopeGlobal, "%APPDATA%/Code/User/instructions")
	a.readPacksDir(b, filepath.Join(dir, "prompts"), ir.KindCommand, ir.ScopeGlobal, "%APPDATA%/Code/User/prompts")
}

// importProject 采集项目层：.github/ 三件套 + .vscode/。
func (a *adapter) importProject(loc adapters.Location, b *ir.Bundle) {
	root := loc.Root
	gh := filepath.Join(root, ".github")
	vs := filepath.Join(root, ".vscode")

	// .github/copilot-instructions.md（always）
	if data, err := os.ReadFile(filepath.Join(gh, "copilot-instructions.md")); err == nil {
		inst := parsePlainInstruction("instruction.copilot-instructions", string(data), ir.ScopeProject, ".github/copilot-instructions.md")
		b.Instructions = append(b.Instructions, inst)
		b.Add(ir.KindInstruction, inst.ID)
	}
	// .github/instructions/*.instructions.md（applyTo）
	a.readInstructionsDir(b, filepath.Join(gh, "instructions"), ir.ScopeProject, ".github/instructions")
	// .github/prompts/*.prompt.md → command
	a.readPacksDir(b, filepath.Join(gh, "prompts"), ir.KindCommand, ir.ScopeProject, ".github/prompts")
	// .github/agents/*.agent.md → agent
	a.readPacksDir(b, filepath.Join(gh, "agents"), ir.KindAgent, ir.ScopeProject, ".github/agents")
	// .vscode/mcp.json、settings.json
	a.readMCPFile(b, filepath.Join(vs, "mcp.json"), ir.ScopeProject, ".vscode/mcp.json")
	a.readSettingsFile(b, filepath.Join(vs, "settings.json"), ir.ScopeProject, ".vscode/settings.json")
}

// ---------- 文件级读取器 ----------

func (a *adapter) readMCPFile(b *ir.Bundle, path string, scope ir.Scope, rawPath string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	mcps, fileExt, err := parseMCPJSON(data, scope, rawPath)
	if err != nil {
		return
	}
	b.MCPServers = append(b.MCPServers, mcps...)
	if len(fileExt) > 0 {
		if b.MCPFileExtensions == nil {
			b.MCPFileExtensions = map[string]any{}
		}
		for k, v := range fileExt {
			b.MCPFileExtensions[k] = v
		}
	}
	for _, m := range mcps {
		b.Add(ir.KindMCP, m.ID)
	}
}

func (a *adapter) readSettingsFile(b *ir.Bundle, path string, scope ir.Scope, rawPath string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, s := range parseSettingsJSON(data, scope, rawPath) {
		b.Settings = append(b.Settings, s)
		b.Add(ir.KindSetting, s.ID)
	}
}

func (a *adapter) readInstructionsDir(b *ir.Bundle, dir string, scope ir.Scope, rawBase string) {
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
		name = strings.TrimSuffix(name, ".instructions")
		inst := parseInstructionFile(data, name, scope, rawBase+"/"+e.Name())
		b.Instructions = append(b.Instructions, inst)
		b.Add(ir.KindInstruction, inst.ID)
	}
}

func (a *adapter) readPacksDir(b *ir.Bundle, dir string, kind ir.EntityKind, scope ir.Scope, rawBase string) {
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
		name := e.Name()
		name = strings.TrimSuffix(name, ".md")
		name = strings.TrimSuffix(name, ".prompt")
		name = strings.TrimSuffix(name, ".agent")
		p := parsePackMD(data, kind, name, scope, rawBase+"/"+e.Name())
		switch kind {
		case ir.KindCommand:
			b.Commands = append(b.Commands, p)
		case ir.KindAgent:
			b.Agents = append(b.Agents, p)
		}
		b.Add(kind, p.ID)
	}
}
