package cline

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
	os.MkdirAll(filepath.Join(root, ".clinerules"), 0o755)
	os.WriteFile(filepath.Join(root, ".clinerules", "style.md"), []byte("# 风格\n"), 0o644)
	os.WriteFile(filepath.Join(root, "cline_mcp_settings.json"), []byte(`{"mcpServers":{"fs":{"command":"npx"}}}`), 0o644)
	a := &adapter{}
	b, _ := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: root})
	if len(b.Instructions) != 1 {
		t.Errorf(".clinerules 未解析: %d", len(b.Instructions))
	}
	if len(b.MCPServers) != 1 || b.MCPServers[0].Name != "fs" {
		t.Errorf("mcp 错误: %+v", b.MCPServers)
	}
	dst := t.TempDir()
	files, _ := a.Export(context.Background(), b, adapters.ExportOpts{ProjectRoot: dst})
	for _, f := range files {
		os.MkdirAll(filepath.Dir(f.Path), 0o755)
		os.WriteFile(f.Path, f.Content, 0o644)
	}
	if _, err := os.Stat(filepath.Join(dst, ".clinerules", "style.md")); err != nil {
		t.Errorf("style.md 未生成: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "cline_mcp_settings.json")); err != nil {
		t.Errorf("cline_mcp_settings.json 未生成: %v", err)
	}
	_ = ir.ScopeProject
}
