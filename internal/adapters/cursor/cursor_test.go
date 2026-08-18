package cursor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func TestImportRules(t *testing.T) {
	root := t.TempDir()
	rulesDir := filepath.Join(root, ".cursor", "rules")
	os.MkdirAll(rulesDir, 0o755)
	os.WriteFile(filepath.Join(rulesDir, "style.mdc"), []byte("---\ndescription: 代码风格\nglobs: \"**/*.go\"\nalwaysApply: false\n---\nGo 规范正文\n"), 0o644)
	os.WriteFile(filepath.Join(rulesDir, "always.mdc"), []byte("---\ndescription: 全局\nalwaysApply: true\n---\n全局规范\n"), 0o644)
	os.WriteFile(filepath.Join(root, ".cursor", "mcp.json"), []byte(`{"mcpServers":{"fs":{"command":"npx"}}}`), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: root})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	var globInst, alwaysInst *ir.Instruction
	for i := range b.Instructions {
		if b.Instructions[i].ID == "instruction.style" {
			globInst = &b.Instructions[i]
		}
		if b.Instructions[i].ID == "instruction.always" {
			alwaysInst = &b.Instructions[i]
		}
	}
	// globs → file_patterns + glob 激活
	if globInst == nil || globInst.Activation != ir.ActivationGlob {
		t.Errorf("globs 应 glob 激活: %+v", globInst)
	}
	if globInst != nil && len(globInst.FilePatterns) > 0 && globInst.FilePatterns[0] != "**/*.go" {
		t.Errorf("globs 应映射 file_patterns: %v", globInst.FilePatterns)
	}
	// alwaysApply → always
	if alwaysInst == nil || alwaysInst.Activation != ir.ActivationAlways {
		t.Errorf("alwaysApply 应 always 激活: %+v", alwaysInst)
	}
	// mcp
	if len(b.MCPServers) != 1 || b.MCPServers[0].Name != "fs" {
		t.Errorf("mcp 错误: %+v", b.MCPServers)
	}
}

func TestExportRoundTrip(t *testing.T) {
	root := t.TempDir()
	b := &ir.Bundle{
		Scope: ir.ScopeMerged,
		Instructions: []ir.Instruction{
			{Header: ir.Header{ID: "instruction.style", IRVersion: 1}, Description: "风格", Activation: ir.ActivationGlob, FilePatterns: []string{"**/*.go"}, Body: "Go 规范\n"},
		},
		MCPServers: []ir.MCPServer{
			{Header: ir.Header{ID: "mcp.fs", IRVersion: 1}, Name: "fs", Transport: "stdio", Command: "npx"},
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
	// rules/style.mdc 含 globs
	data, err := os.ReadFile(filepath.Join(root, ".cursor", "rules", "style.mdc"))
	if err != nil {
		t.Fatalf("style.mdc 未生成: %v", err)
	}
	if !strings.Contains(string(data), "globs") || !strings.Contains(string(data), "**/*.go") {
		t.Errorf("mdc 应含 globs:\n%s", data)
	}
	// mcp.json mcpServers
	mcpData, _ := os.ReadFile(filepath.Join(root, ".cursor", "mcp.json"))
	if !strings.Contains(string(mcpData), "mcpServers") {
		t.Errorf("mcp.json 应含 mcpServers:\n%s", mcpData)
	}
}
