package copilot

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func TestImportProject(t *testing.T) {
	root := t.TempDir()
	gh := filepath.Join(root, ".github")
	vs := filepath.Join(root, ".vscode")
	os.MkdirAll(filepath.Join(gh, "instructions"), 0o755)
	os.MkdirAll(filepath.Join(gh, "prompts"), 0o755)
	os.MkdirAll(filepath.Join(gh, "agents"), 0o755)
	os.MkdirAll(vs, 0o755)

	os.WriteFile(filepath.Join(gh, "copilot-instructions.md"), []byte("# 项目规范\n用中文\n"), 0o644)
	os.WriteFile(filepath.Join(gh, "instructions", "python.md"), []byte("---\ndescription: Python 规范\napplyTo: \"**/*.py\"\n---\nPython 正文\n"), 0o644)
	os.WriteFile(filepath.Join(gh, "prompts", "review.prompt.md"), []byte("---\nname: review\ndescription: 评审\n---\n评审正文\n"), 0o644)
	os.WriteFile(filepath.Join(gh, "agents", "helper.agent.md"), []byte("---\nname: helper\ndescription: 助手\n---\n助手正文\n"), 0o644)
	os.WriteFile(filepath.Join(vs, "mcp.json"), []byte(`{"servers":{"fs":{"command":"npx","args":["-y","fs"]},"db":{"type":"http","url":"https://db.x.com"}},"inputs":[{"id":"tok","type":"promptString"}]}`), 0o644)
	os.WriteFile(filepath.Join(vs, "settings.json"), []byte(`{"editor.fontSize":14,"chat.mcp.access":"registry"}`), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: root})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	// copilot-instructions.md → always instruction
	var main *ir.Instruction
	for i := range b.Instructions {
		if b.Instructions[i].ID == "instruction.copilot-instructions" {
			main = &b.Instructions[i]
		}
	}
	if main == nil || main.Activation != ir.ActivationAlways {
		t.Errorf("copilot-instructions 应 always: %+v", main)
	}

	// .instructions.md → applyTo → file_patterns + glob 激活
	var pyInst *ir.Instruction
	for i := range b.Instructions {
		if len(b.Instructions[i].FilePatterns) > 0 {
			pyInst = &b.Instructions[i]
		}
	}
	if pyInst == nil {
		t.Fatal("缺少带 applyTo 的 instruction")
	}
	if pyInst.FilePatterns[0] != "**/*.py" || pyInst.Activation != ir.ActivationGlob {
		t.Errorf("applyTo 未映射为 file_patterns/glob: %+v", pyInst)
	}

	// prompt → command；agent → agent
	if len(b.Commands) != 1 || b.Commands[0].Name != "review" {
		t.Errorf("prompt→command 错误: %+v", b.Commands)
	}
	if len(b.Agents) != 1 || b.Agents[0].Name != "helper" {
		t.Errorf("agent 错误: %+v", b.Agents)
	}

	// mcp：servers 键；inputs 进 file_extensions
	if len(b.MCPServers) != 2 {
		t.Fatalf("应解析 2 个 MCP，实际 %d", len(b.MCPServers))
	}
	if _, ok := b.MCPFileExtensions["inputs"]; !ok {
		t.Errorf("inputs 应进 MCPFileExtensions: %+v", b.MCPFileExtensions)
	}

	// settings：点号 key
	var found bool
	for _, s := range b.Settings {
		if s.Key == "chat.mcp.access" {
			found = true
		}
	}
	if !found {
		t.Error("点号 key chat.mcp.access 未解析")
	}
}

func TestExportRoundTrip(t *testing.T) {
	root := t.TempDir()
	b := &ir.Bundle{
		Scope: ir.ScopeMerged,
		Instructions: []ir.Instruction{
			{Header: ir.Header{ID: "instruction.main", IRVersion: 1}, Body: "主规范\n", Activation: ir.ActivationAlways},
			{Header: ir.Header{ID: "instruction.python", IRVersion: 1}, Name: "python", Body: "Py 规范\n", Activation: ir.ActivationGlob, FilePatterns: []string{"**/*.py"}},
		},
		MCPServers: []ir.MCPServer{
			{Header: ir.Header{ID: "mcp.fs", IRVersion: 1}, Name: "fs", Transport: "stdio", Command: "npx"},
		},
		Settings: []ir.SettingEntry{
			{Header: ir.Header{ID: "setting.copilot.editor.fontSize", IRVersion: 1}, Key: "editor.fontSize", Value: float64(14)},
		},
	}

	a := &adapter{}
	files, err := a.Export(context.Background(), b, adapters.ExportOpts{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	// 落盘后重导（round-trip）
	for _, f := range files {
		os.MkdirAll(filepath.Dir(f.Path), 0o755)
		os.WriteFile(f.Path, f.Content, 0o644)
	}

	// .vscode/mcp.json 用 servers 键
	mcpData, _ := os.ReadFile(filepath.Join(root, ".vscode", "mcp.json"))
	if !strings.Contains(string(mcpData), `"servers"`) || !strings.Contains(string(mcpData), `"fs"`) {
		t.Errorf("mcp.json 应用 servers 键含 fs:\n%s", mcpData)
	}
	// .github/copilot-instructions.md（always）
	if _, err := os.Stat(filepath.Join(root, ".github", "copilot-instructions.md")); err != nil {
		t.Errorf("copilot-instructions.md 未生成: %v", err)
	}
	// glob instruction 独立成文件
	pyFile := filepath.Join(root, ".github", "instructions", "python.instructions.md")
	pyData, err := os.ReadFile(pyFile)
	if err != nil {
		t.Fatalf("python.instructions.md 未生成: %v", err)
	}
	if !strings.Contains(string(pyData), "applyTo") || !strings.Contains(string(pyData), "**/*.py") {
		t.Errorf("glob instruction 应含 applyTo:\n%s", pyData)
	}
	// settings 点号 key
	setData, _ := os.ReadFile(filepath.Join(root, ".vscode", "settings.json"))
	if !strings.Contains(string(setData), "editor.fontSize") {
		t.Errorf("settings.json 应含点号 key:\n%s", setData)
	}
}
