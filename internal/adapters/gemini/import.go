package gemini

import (
	"context"
	"os"
	"path/filepath"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

// importLocation 按 scope 分发采集（Gemini Import 主流程）。
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

// importGlobal 采集 ~/.gemini（settings.json、GEMINI.md）。
func (a *adapter) importGlobal(loc adapters.Location, b *ir.Bundle) {
	dir := loc.Root
	a.readSettings(b, filepath.Join(dir, "settings.json"), ir.ScopeGlobal, "~/.gemini/settings.json")
	if data, err := os.ReadFile(filepath.Join(dir, "GEMINI.md")); err == nil {
		inst := parseGeminiMD(string(data), ir.ScopeGlobal, "~/.gemini/GEMINI.md")
		b.Instructions = append(b.Instructions, inst)
		b.Add(ir.KindInstruction, inst.ID)
	}
}

// importProject 采集项目层（GEMINI.md、.gemini/settings.json）。
func (a *adapter) importProject(loc adapters.Location, b *ir.Bundle) {
	root := loc.Root
	if data, err := os.ReadFile(filepath.Join(root, "GEMINI.md")); err == nil {
		inst := parseGeminiMD(string(data), ir.ScopeProject, "GEMINI.md")
		b.Instructions = append(b.Instructions, inst)
		b.Add(ir.KindInstruction, inst.ID)
	}
	a.readSettings(b, filepath.Join(root, ".gemini", "settings.json"), ir.ScopeProject, ".gemini/settings.json")
}

func (a *adapter) readSettings(b *ir.Bundle, path string, scope ir.Scope, rawPath string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	settings, mcps, err := parseSettingsJSON(data, scope, rawPath)
	if err != nil {
		return
	}
	b.Settings = append(b.Settings, settings...)
	b.MCPServers = append(b.MCPServers, mcps...)
	for _, s := range settings {
		b.Add(ir.KindSetting, s.ID)
	}
	for _, m := range mcps {
		b.Add(ir.KindMCP, m.ID)
	}
}
