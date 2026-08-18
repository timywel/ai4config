package roo

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func TestImportExport(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".roo", "rules"), 0o755)
	os.WriteFile(filepath.Join(root, ".roo", "rules", "style.md"), []byte("# 风格\n"), 0o644)
	os.WriteFile(filepath.Join(root, ".roomodes"), []byte("customModes:\n  - slug: reviewer\n    name: reviewer\n    roleDefinition: 评审\n    customInstructions: 评审正文\n"), 0o644)
	os.WriteFile(filepath.Join(root, ".roo", "mcp.json"), []byte(`{"mcpServers":{"fs":{"command":"npx"}}}`), 0o644)
	a := &adapter{}
	b, _ := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: root})
	if len(b.Instructions) != 1 {
		t.Errorf(".roo/rules 未解析: %d", len(b.Instructions))
	}
	if len(b.Agents) != 1 || b.Agents[0].Name != "reviewer" {
		t.Errorf(".roomodes 未解析为 agent: %+v", b.Agents)
	}
	if len(b.MCPServers) != 1 {
		t.Errorf("mcp 错误: %+v", b.MCPServers)
	}
	dst := t.TempDir()
	files, _ := a.Export(context.Background(), b, adapters.ExportOpts{ProjectRoot: dst})
	for _, f := range files {
		os.MkdirAll(filepath.Dir(f.Path), 0o755)
		os.WriteFile(f.Path, f.Content, 0o644)
	}
	if _, err := os.Stat(filepath.Join(dst, ".roo", "rules", "style.md")); err != nil {
		t.Errorf("style.md 未生成: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, ".roomodes")); err != nil {
		t.Errorf(".roomodes 未生成: %v", err)
	}
	_ = ir.ScopeProject
}
