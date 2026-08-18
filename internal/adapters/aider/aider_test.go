package aider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func TestImportExport(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "CONVENTIONS.md"), []byte("# 约定\n"), 0o644)
	os.WriteFile(filepath.Join(root, ".aider.conf.yml"), []byte("model: gpt-4o\n"), 0o644)
	a := &adapter{}
	b, _ := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: root})
	if len(b.Instructions) != 1 {
		t.Errorf("CONVENTIONS.md 未解析: %d", len(b.Instructions))
	}
	if len(b.Settings) != 1 || b.Settings[0].Value != "gpt-4o" {
		t.Errorf(".aider.conf.yml 解析错误: %+v", b.Settings)
	}
	dst := t.TempDir()
	files, _ := a.Export(context.Background(), b, adapters.ExportOpts{ProjectRoot: dst})
	for _, f := range files {
		os.MkdirAll(filepath.Dir(f.Path), 0o755)
		os.WriteFile(f.Path, f.Content, 0o644)
	}
	if _, err := os.Stat(filepath.Join(dst, "CONVENTIONS.md")); err != nil {
		t.Errorf("CONVENTIONS.md 未生成: %v", err)
	}
	d, _ := os.ReadFile(filepath.Join(dst, ".aider.conf.yml"))
	if !strings.Contains(string(d), "model") {
		t.Errorf(".aider.conf.yml 应含 model: %s", d)
	}
	_ = ir.ScopeProject
}
