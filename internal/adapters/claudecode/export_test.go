package claudecode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func TestExportProject(t *testing.T) {
	root := t.TempDir()
	b := &ir.Bundle{
		Scope: ir.ScopeMerged,
		Instructions: []ir.Instruction{
			{Header: ir.Header{ID: "instruction.a", IRVersion: 1}, Body: "规则 A\n"},
			{Header: ir.Header{ID: "instruction.b", IRVersion: 1}, Body: "规则 B\n"},
		},
		MCPServers: []ir.MCPServer{
			{Header: ir.Header{ID: "mcp.fs", IRVersion: 1}, Name: "fs", Transport: "stdio", Command: "npx", Args: []string{"-y", "fs"}},
		},
		Settings: []ir.SettingEntry{
			{Header: ir.Header{ID: "setting.claude-code.model", IRVersion: 1}, Key: "model", Value: "opus"},
		},
		Hooks: []ir.Hook{
			{Header: ir.Header{ID: "hook.g", IRVersion: 1}, Event: ir.HookPreToolUse,
				Matcher: ir.HookMatcher{Tool: "Bash"}, Handler: ir.HookHandler{Type: "command", Command: "./g.sh"}},
		},
		Skills: []ir.PromptPack{
			{Header: ir.Header{ID: "skill.review", IRVersion: 1}, Kind: ir.KindSkill, Name: "review", Description: "评审", Body: "正文\n"},
		},
	}

	a := &adapter{}
	files, err := a.Export(context.Background(), b, adapters.ExportOpts{ProjectRoot: root})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("应产生写入文件")
	}

	// CLAUDE.md 含边界注释 + 两条正文
	data, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md 未生成: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "<!-- cfg4ai:begin instruction.a -->") {
		t.Error("CLAUDE.md 缺边界注释 begin")
	}
	if !strings.Contains(content, "规则 A") || !strings.Contains(content, "规则 B") {
		t.Error("CLAUDE.md 缺正文")
	}

	// .mcp.json 结构
	mcpData, _ := os.ReadFile(filepath.Join(root, ".mcp.json"))
	var mcpFile struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(mcpData, &mcpFile); err != nil {
		t.Fatalf(".mcp.json 解析: %v", err)
	}
	fs, ok := mcpFile.MCPServers["fs"]
	if !ok || fs.Command != "npx" {
		t.Errorf(".mcp.json 应用原始键名 fs: %+v", mcpFile.MCPServers)
	}

	// settings.json 含 model + hooks
	setData, _ := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
	var settings map[string]any
	json.Unmarshal(setData, &settings)
	if settings["model"] != "opus" {
		t.Errorf("settings.json model 错误: %v", settings["model"])
	}
	if hooks, ok := settings["hooks"].(map[string]any); !ok || hooks["PreToolUse"] == nil {
		t.Errorf("settings.json 应含 PreToolUse hook（Claude 事件名）: %v", settings["hooks"])
	}

	// skills/review/SKILL.md
	skillData, err := os.ReadFile(filepath.Join(root, ".claude", "skills", "review", "SKILL.md"))
	if err != nil {
		t.Fatalf("SKILL.md 未生成: %v", err)
	}
	if !strings.Contains(string(skillData), "name: review") || !strings.Contains(string(skillData), "正文") {
		t.Errorf("SKILL.md 内容错误:\n%s", skillData)
	}
}

// round-trip：构造目录 → Import → Export 到新目录 → 再 Import → 关键字段一致
func TestRoundTrip(t *testing.T) {
	// 源目录
	src := t.TempDir()
	srcClaude := filepath.Join(src, ".claude")
	os.MkdirAll(filepath.Join(srcClaude, "agents"), 0o755)
	os.WriteFile(filepath.Join(src, "CLAUDE.md"), []byte("# 规范\n遵循 @docs/x.md\n"), 0o644)
	os.WriteFile(filepath.Join(src, ".mcp.json"), []byte(`{"mcpServers":{"db":{"type":"http","url":"https://db.x.com"}}}`), 0o644)
	os.WriteFile(filepath.Join(srcClaude, "settings.json"), []byte(`{"model":"sonnet"}`), 0o644)
	os.WriteFile(filepath.Join(srcClaude, "agents", "a.md"), []byte("---\nname: a\ndescription: 助手\n---\n正文\n"), 0o644)

	a := &adapter{}
	// 第一次 Import
	b1, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: src})
	if err != nil {
		t.Fatalf("Import1: %v", err)
	}

	// Export 到新目录
	dst := t.TempDir()
	if _, err := a.Export(context.Background(), b1, adapters.ExportOpts{ProjectRoot: dst}); err != nil {
		t.Fatalf("Export: %v", err)
	}

	// 再 Import
	b2, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: dst})
	if err != nil {
		t.Fatalf("Import2: %v", err)
	}

	// 关键字段一致性（字段级 round-trip，IR-SCHEMA §1.3 强承诺）
	if len(b1.MCPServers) != len(b2.MCPServers) {
		t.Errorf("MCP 数量 round-trip 不一致: %d vs %d", len(b1.MCPServers), len(b2.MCPServers))
	}
	if len(b2.MCPServers) > 0 {
		if b2.MCPServers[0].Name != "db" || b2.MCPServers[0].URL != "https://db.x.com" {
			t.Errorf("MCP 字段丢失: %+v", b2.MCPServers[0])
		}
	}
	// agent 名称与正文
	if len(b2.Agents) != 1 || b2.Agents[0].Name != "a" || strings.TrimSpace(b2.Agents[0].Body) != "正文" {
		t.Errorf("agent round-trip 不一致: %+v", b2.Agents)
	}
	// settings model
	var model string
	for _, s := range b2.Settings {
		if s.Key == "model" {
			model, _ = s.Value.(string)
		}
	}
	if model != "sonnet" {
		t.Errorf("settings model round-trip 不一致: %q", model)
	}
}

// dry-run：不产生文件但返回清单
func TestExportDryRun(t *testing.T) {
	root := t.TempDir()
	b := &ir.Bundle{
		Scope:        ir.ScopeMerged,
		Instructions: []ir.Instruction{{Header: ir.Header{ID: "instruction.a", IRVersion: 1}, Body: "x\n"}},
	}
	a := &adapter{}
	files, err := a.Export(context.Background(), b, adapters.ExportOpts{ProjectRoot: root, DryRun: true})
	if err != nil {
		t.Fatalf("Export dry-run: %v", err)
	}
	if len(files) == 0 || files[0].Hash == "" {
		t.Error("dry-run 应返回文件清单含 hash")
	}
	if _, err := os.Stat(filepath.Join(root, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Error("dry-run 不应落盘")
	}
}
