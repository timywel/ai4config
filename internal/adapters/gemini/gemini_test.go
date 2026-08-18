package gemini

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func TestImportGlobal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{
  "general": {"preferredEditor": "vscode"},
  "mcpServers": {
    "fs": {"command": "npx", "args": ["-y","fs"], "trust": true, "includeTools": ["read"]},
    "db": {"type": "http", "url": "https://db.x.com"}
  }
}`), 0o644)
	os.WriteFile(filepath.Join(dir, "GEMINI.md"), []byte("# Gemini 规范\n"), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeGlobal, Root: dir})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	// 顶层非 mcp 键 → 不透明 setting
	var general *ir.SettingEntry
	for i := range b.Settings {
		if b.Settings[i].Key == "general" {
			general = &b.Settings[i]
		}
	}
	if general == nil {
		t.Fatal("general 键应作不透明 setting")
	}
	if gv, ok := general.Value.(map[string]any); !ok || gv["preferredEditor"] != "vscode" {
		t.Errorf("general 嵌套 value 错误: %+v", general.Value)
	}

	// mcpServers：trust / includeTools / http
	byName := map[string]ir.MCPServer{}
	for _, s := range b.MCPServers {
		byName[s.Name] = s
	}
	fs := byName["fs"]
	if fs.Trust == nil || !*fs.Trust {
		t.Errorf("gemini trust 应解析为 true: %+v", fs.Trust)
	}
	if len(fs.EnabledTools) != 1 || fs.EnabledTools[0] != "read" {
		t.Errorf("includeTools 应映射 enabledTools: %v", fs.EnabledTools)
	}
	if byName["db"].Transport != "http" {
		t.Errorf("db 应 http transport: %+v", byName["db"])
	}
}

func TestExportRoundTrip(t *testing.T) {
	root := t.TempDir()
	b := &ir.Bundle{
		Scope: ir.ScopeMerged,
		Instructions: []ir.Instruction{
			{Header: ir.Header{ID: "instruction.g", IRVersion: 1}, Body: "Gemini 规范\n"},
		},
		MCPServers: []ir.MCPServer{
			{Header: ir.Header{ID: "mcp.fs", IRVersion: 1}, Name: "fs", Transport: "stdio", Command: "npx"},
		},
		Settings: []ir.SettingEntry{
			{Header: ir.Header{ID: "setting.gemini.general", IRVersion: 1}, Key: "general", Value: map[string]any{"preferredEditor": "vscode"}},
		},
	}
	a := &adapter{}
	files, err := a.Export(context.Background(), b, adapters.ExportOpts{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	for _, f := range files {
		os.MkdirAll(filepath.Dir(f.Path), 0o755)
		os.WriteFile(f.Path, f.Content, 0o644)
	}
	// 项目 GEMINI.md 在项目根
	if _, err := os.Stat(filepath.Join(root, "GEMINI.md")); err != nil {
		t.Errorf("项目 GEMINI.md 未生成: %v", err)
	}
	// .gemini/settings.json 含 mcpServers + general
	data, _ := os.ReadFile(filepath.Join(root, ".gemini", "settings.json"))
	if !strings.Contains(string(data), "mcpServers") || !strings.Contains(string(data), "preferredEditor") {
		t.Errorf("settings.json 应含 mcpServers 与嵌套 general:\n%s", data)
	}
}
