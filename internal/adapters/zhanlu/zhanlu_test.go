package zhanlu

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
	home := t.TempDir()
	zdir := filepath.Join(home, ".config", "zhanlu")
	os.MkdirAll(zdir, 0o755)
	os.MkdirAll(filepath.Join(home, ".agents", "skills", "review"), 0o755)
	os.WriteFile(filepath.Join(zdir, "zhanlu.json"), []byte(`{"model":"kimi-k3","mcpServers":{"fs":{"command":"npx","args":["-y","fs"]}},"providers":{"p1":{"api_key":"x"}}}`), 0o644)
	os.WriteFile(filepath.Join(home, ".agents", "skills", "review", "SKILL.md"), []byte("---\nname: review\ndescription: 评审\n---\n技能正文\n"), 0o644)

	// 设置 XDG_CONFIG_HOME 使 globalDir 指向临时 home
	t.Setenv("USERPROFILE", home) // Windows：paths.Home() 重定向
	t.Setenv("HOME", home)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeGlobal, Root: zdir})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	// settings：model + providers（不透明）
	var model *ir.SettingEntry
	for i := range b.Settings {
		if b.Settings[i].Key == "model" {
			model = &b.Settings[i]
		}
	}
	if model == nil || model.Value != "kimi-k3" {
		t.Errorf("model setting 错误: %+v", model)
	}
	// mcpServers 段 → MCPServer
	if len(b.MCPServers) != 1 || b.MCPServers[0].Name != "fs" {
		t.Errorf("mcpServers 解析错误: %+v", b.MCPServers)
	}
	// skill：语义路由 activation
	if len(b.Skills) != 1 {
		t.Fatalf("应有 1 个 skill，实际 %d", len(b.Skills))
	}
	if b.Skills[0].Activation != ir.ActivationModelDecision {
		t.Errorf("zhanlu skill 应 model-decision 激活，实际 %s", b.Skills[0].Activation)
	}
}

func TestImportProject(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".kilo", "agent"), 0o755)
	os.MkdirAll(filepath.Join(root, ".kilo", "command"), 0o755)
	os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# 项目规范\n"), 0o644)
	os.WriteFile(filepath.Join(root, ".kilo", "agent", "helper.md"), []byte("---\nname: helper\n---\n助手正文\n"), 0o644)
	os.WriteFile(filepath.Join(root, ".kilo", "command", "build.md"), []byte("---\nname: build\n---\n构建正文\n"), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: root})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(b.Instructions) != 1 {
		t.Errorf("应有 1 条 instruction，实际 %d", len(b.Instructions))
	}
	if len(b.Agents) != 1 || b.Agents[0].Name != "helper" {
		t.Errorf(".kilo/agent 解析错误: %+v", b.Agents)
	}
	if len(b.Commands) != 1 || b.Commands[0].Name != "build" {
		t.Errorf(".kilo/command 解析错误: %+v", b.Commands)
	}
	if b.Commands[0].Activation != ir.ActivationManual {
		t.Errorf("command 应 manual 激活，实际 %s", b.Commands[0].Activation)
	}
}

func TestExportRoundTrip(t *testing.T) {
	root := t.TempDir()
	b := &ir.Bundle{
		Scope: ir.ScopeMerged,
		Instructions: []ir.Instruction{
			{Header: ir.Header{ID: "instruction.main", IRVersion: 1}, Body: "项目规范\n"},
		},
		MCPServers: []ir.MCPServer{
			{Header: ir.Header{ID: "mcp.fs", IRVersion: 1}, Name: "fs", Transport: "stdio", Command: "npx"},
		},
		Agents: []ir.PromptPack{
			{Header: ir.Header{ID: "agent.helper", IRVersion: 1}, Kind: ir.KindAgent, Name: "helper", Body: "助手\n"},
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
	// AGENTS.md
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Errorf("AGENTS.md 未生成: %v", err)
	}
	// .kilo/agent/helper.md
	agentData, err := os.ReadFile(filepath.Join(root, ".kilo", "agent", "helper.md"))
	if err != nil {
		t.Fatalf(".kilo/agent/helper.md 未生成: %v", err)
	}
	if !strings.Contains(string(agentData), "name: helper") {
		t.Errorf("agent 文件内容错误:\n%s", agentData)
	}
	// kilo.json 含 mcpServers
	kiloData, _ := os.ReadFile(filepath.Join(root, "kilo.json"))
	if !strings.Contains(string(kiloData), "mcpServers") || !strings.Contains(string(kiloData), "fs") {
		t.Errorf("kilo.json 应含 mcpServers.fs:\n%s", kiloData)
	}
}
