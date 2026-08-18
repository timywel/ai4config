package opencode

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
	os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# 规范\n"), 0o644)
	os.WriteFile(filepath.Join(root, "opencode.json"), []byte(`{"model":"x","mcp":{"fs":{"command":"npx"}}}`), 0o644)
	a := &adapter{}
	b, _ := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: root})
	if len(b.Instructions) != 1 {
		t.Errorf("AGENTS.md 未解析: %d", len(b.Instructions))
	}
	if len(b.MCPServers) != 1 || b.MCPServers[0].Name != "fs" {
		t.Errorf("mcp 段错误: %+v", b.MCPServers)
	}
	var found bool
	for _, s := range b.Settings {
		if s.Key == "model" {
			found = true
		}
	}
	if !found {
		t.Error("model setting 未解析")
	}
	dst := t.TempDir()
	files, _ := a.Export(context.Background(), b, adapters.ExportOpts{ProjectRoot: dst})
	for _, f := range files {
		os.MkdirAll(filepath.Dir(f.Path), 0o755)
		os.WriteFile(f.Path, f.Content, 0o644)
	}
	if _, err := os.Stat(filepath.Join(dst, "AGENTS.md")); err != nil {
		t.Errorf("AGENTS.md 未生成: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "opencode.json")); err != nil {
		t.Errorf("opencode.json 未生成: %v", err)
	}
	_ = ir.ScopeProject
}
