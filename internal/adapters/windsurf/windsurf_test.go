package windsurf

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
	os.MkdirAll(filepath.Join(root, ".windsurf", "rules"), 0o755)
	os.WriteFile(filepath.Join(root, ".windsurf", "rules", "style.md"), []byte("# 风格规范\n"), 0o644)
	os.WriteFile(filepath.Join(root, ".windsurf", "mcp_config.json"), []byte(`{"mcpServers":{"fs":{"command":"npx"}}}`), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: root})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(b.Instructions) != 1 {
		t.Errorf("应有 1 条 rule，实际 %d", len(b.Instructions))
	}
	if len(b.MCPServers) != 1 || b.MCPServers[0].Name != "fs" {
		t.Errorf("mcp 错误: %+v", b.MCPServers)
	}

	dst := t.TempDir()
	files, err := a.Export(context.Background(), b, adapters.ExportOpts{ProjectRoot: dst})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	for _, f := range files {
		os.MkdirAll(filepath.Dir(f.Path), 0o755)
		os.WriteFile(f.Path, f.Content, 0o644)
	}
	if _, err := os.Stat(filepath.Join(dst, ".windsurf", "rules", "style.md")); err != nil {
		t.Errorf("style.md 未生成: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, ".windsurf", "mcp_config.json")); err != nil {
		t.Errorf("mcp_config.json 未生成: %v", err)
	}
}
