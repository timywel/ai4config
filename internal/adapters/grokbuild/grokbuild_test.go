package grokbuild

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
	gdir := filepath.Join(root, ".grok")
	os.MkdirAll(filepath.Join(gdir, "skills", "review"), 0o755)
	os.WriteFile(filepath.Join(gdir, "config.toml"), []byte("model = \"grok-code-fast\"\n\n[mcp_servers.fs]\ncommand = \"npx\"\n"), 0o644)
	os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Grok 规范\n"), 0o644)
	os.WriteFile(filepath.Join(gdir, "skills", "review", "SKILL.md"), []byte("---\nname: review\ndescription: 评审\n---\n正文\n"), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: root})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	// config.toml：model setting + mcp_servers
	var model *ir.SettingEntry
	for i := range b.Settings {
		if b.Settings[i].Key == "model" {
			model = &b.Settings[i]
		}
	}
	if model == nil || model.Value != "grok-code-fast" {
		t.Errorf("model 错误: %+v", model)
	}
	if len(b.MCPServers) != 1 || b.MCPServers[0].Name != "fs" {
		t.Errorf("mcp_servers 错误: %+v", b.MCPServers)
	}
	if len(b.Skills) != 1 || b.Skills[0].Name != "review" {
		t.Errorf("skill 错误: %+v", b.Skills)
	}

	// Export round-trip
	dst := t.TempDir()
	files, err := a.Export(context.Background(), b, adapters.ExportOpts{ProjectRoot: dst})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	for _, f := range files {
		os.MkdirAll(filepath.Dir(f.Path), 0o755)
		os.WriteFile(f.Path, f.Content, 0o644)
	}
	data, _ := os.ReadFile(filepath.Join(dst, ".grok", "config.toml"))
	if !strings.Contains(string(data), "mcp_servers") || !strings.Contains(string(data), "fs") {
		t.Errorf("config.toml 应含 mcp_servers.fs:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(dst, ".grok", "skills", "review", "SKILL.md")); err != nil {
		t.Errorf("SKILL.md 未生成: %v", err)
	}
}
