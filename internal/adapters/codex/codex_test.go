package codex

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func TestImportConfigTOML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
model = "gpt-5-codex"
approval_policy = "on-request"

[mcp_servers.fs]
command = "npx"
args = ["-y", "fs"]
enabled = false
startup_timeout_sec = 5
tool_timeout_sec = 60

[mcp_servers.db]
type = "http"
url = "https://db.x.com"

[[hooks.SessionStart]]
[[hooks.SessionStart.hooks]]
type = "command"
command = "./init.sh"
command_windows = "./init.ps1"
timeout = 30
`), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeGlobal, Root: dir})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	// 顶层不透明 setting
	var model *ir.SettingEntry
	for i := range b.Settings {
		if b.Settings[i].Key == "model" {
			model = &b.Settings[i]
		}
	}
	if model == nil || model.Value != "gpt-5-codex" {
		t.Errorf("model setting 错误: %+v", model)
	}

	// MCP：极性取反 + timeout 换算
	byName := map[string]ir.MCPServer{}
	for _, s := range b.MCPServers {
		byName[s.Name] = s
	}
	fs := byName["fs"]
	if !fs.Disabled {
		t.Error("enabled=false 应取反为 disabled=true（极性）")
	}
	if fs.Timeout == nil || fs.Timeout.StartupMs != 5000 || fs.Timeout.ToolSec != 60 {
		t.Errorf("timeout 换算错误（5s→5000ms）: %+v", fs.Timeout)
	}
	db := byName["db"]
	if db.Disabled {
		t.Error("enabled 缺省应为 disabled=false")
	}
	if db.Transport != "http" || db.URL != "https://db.x.com" {
		t.Errorf("db server 解析错误: %+v", db)
	}

	// hooks：SessionStart → session-start；command_windows 别名
	var hook *ir.Hook
	for i := range b.Hooks {
		if b.Hooks[i].Event == ir.HookSessionStart {
			hook = &b.Hooks[i]
		}
	}
	if hook == nil {
		t.Fatal("缺少 session-start hook")
	}
	if hook.Handler.CommandWindows != "./init.ps1" {
		t.Errorf("command_windows 别名未解析: %+v", hook.Handler)
	}
}

func TestImportAgentsOverride(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("普通版"), 0o644)
	os.WriteFile(filepath.Join(root, "AGENTS.override.md"), []byte("覆盖版"), 0o644)
	// 子目录 AGENTS.md
	sub := filepath.Join(root, "sub")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "AGENTS.md"), []byte("子目录版"), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: root})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	var rootInst, subInst *ir.Instruction
	for i := range b.Instructions {
		if b.Instructions[i].Subtree == "" {
			rootInst = &b.Instructions[i]
		}
		if b.Instructions[i].Subtree == "sub" {
			subInst = &b.Instructions[i]
		}
	}
	if rootInst == nil || rootInst.Body != "覆盖版" {
		t.Errorf("AGENTS.override.md 应优先于 AGENTS.md: %+v", rootInst)
	}
	if rootInst != nil && rootInst.Extensions["x-codex"] == nil {
		t.Error("override 应记 x-codex.override")
	}
	if subInst == nil || subInst.Subtree != "sub" {
		t.Errorf("子目录 AGENTS.md 应记 subtree=sub: %+v", subInst)
	}
}

func TestExportPolarityAndTimeout(t *testing.T) {
	root := t.TempDir()
	b := &ir.Bundle{
		Scope: ir.ScopeMerged,
		Settings: []ir.SettingEntry{
			{Header: ir.Header{ID: "setting.codex.model", IRVersion: 1}, Key: "model", Value: "gpt-5"},
		},
		MCPServers: []ir.MCPServer{
			{
				Header:    ir.Header{ID: "mcp.fs", IRVersion: 1},
				Name:      "fs",
				Transport: "stdio",
				Command:   "npx",
				Disabled:  true, // IR disabled=true → codex enabled=false
				Timeout:   &ir.Timeout{StartupMs: 5000, ToolSec: 60},
			},
		},
	}

	a := &adapter{}
	if _, err := a.Export(context.Background(), b, adapters.ExportOpts{ProjectRoot: root}); err != nil {
		t.Fatalf("Export: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("config.toml 未生成: %v", err)
	}
	var cfg map[string]any
	if err := toml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("config.toml 解析: %v", err)
	}
	if cfg["model"] != "gpt-5" {
		t.Errorf("model 写回错误: %v", cfg["model"])
	}
	servers := cfg["mcp_servers"].(map[string]any)
	fs := servers["fs"].(map[string]any)
	if fs["enabled"] != false {
		t.Errorf("disabled=true 应取反回 enabled=false: %v", fs["enabled"])
	}
	if fs["startup_timeout_sec"] != int64(5) {
		t.Errorf("5000ms 应换算回 5s: %v", fs["startup_timeout_sec"])
	}
	if fs["tool_timeout_sec"] != int64(60) {
		t.Errorf("tool_timeout_sec 错误: %v", fs["tool_timeout_sec"])
	}
}

// 机器级键项目级导出时跳过
func TestExportProjectSkipsMachineKeys(t *testing.T) {
	root := t.TempDir()
	b := &ir.Bundle{
		Scope: ir.ScopeMerged,
		Settings: []ir.SettingEntry{
			{Header: ir.Header{ID: "setting.codex.model", IRVersion: 1}, Key: "model", Value: "gpt-5"},
			{Header: ir.Header{ID: "setting.codex.notify", IRVersion: 1}, Key: "notify", Value: []any{"x"}}, // 机器级
		},
	}
	a := &adapter{}
	if _, err := a.Export(context.Background(), b, adapters.ExportOpts{ProjectRoot: root}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(root, ".codex", "config.toml"))
	var cfg map[string]any
	toml.Unmarshal(data, &cfg)
	if _, exists := cfg["notify"]; exists {
		t.Error("机器级键 notify 项目级应跳过")
	}
	if cfg["model"] != "gpt-5" {
		t.Error("普通键 model 应写入")
	}
}

// round-trip：config.toml Import→Export→Import 字段一致
func TestRoundTrip(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "config.toml"), []byte(`
model = "gpt-5"
[mcp_servers.fs]
command = "npx"
enabled = false
startup_timeout_sec = 8
`), 0o644)

	a := &adapter{}
	b1, _ := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeGlobal, Root: src})
	dst := t.TempDir()
	os.Setenv("CODEX_HOME", dst)
	defer os.Unsetenv("CODEX_HOME")
	if _, err := a.Export(context.Background(), b1, adapters.ExportOpts{}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	b2, _ := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeGlobal, Root: dst})

	if len(b1.MCPServers) != len(b2.MCPServers) {
		t.Fatalf("MCP 数量不一致: %d vs %d", len(b1.MCPServers), len(b2.MCPServers))
	}
	s1, s2 := b1.MCPServers[0], b2.MCPServers[0]
	if s1.Disabled != s2.Disabled {
		t.Errorf("极性 round-trip 不一致: %v vs %v", s1.Disabled, s2.Disabled)
	}
	if s1.Timeout.StartupMs != s2.Timeout.StartupMs {
		t.Errorf("timeout round-trip 不一致: %d vs %d", s1.Timeout.StartupMs, s2.Timeout.StartupMs)
	}
}
