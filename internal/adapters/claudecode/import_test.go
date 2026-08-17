package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

// 构造临时 Claude Code 全局目录并 Import 验证。
func TestImportGlobal(t *testing.T) {
	home := t.TempDir()
	claudeDir := filepath.Join(home, ".claude")
	os.MkdirAll(filepath.Join(claudeDir, "agents"), 0o755)
	os.MkdirAll(filepath.Join(claudeDir, "skills", "review"), 0o755)

	// CLAUDE.md（含 @import）
	os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"),
		[]byte("# 规范\n\n遵循 @docs/style.md 与 @./team.md\n"), 0o644)
	// settings.json（model + hooks）
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{
  "model": "claude-opus",
  "hooks": {
    "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "./guard.sh", "timeout": 30}]}]
  }
}`), 0o644)
	// agents/reviewer.md（frontmatter + 正文）
	os.WriteFile(filepath.Join(claudeDir, "agents", "reviewer.md"),
		[]byte("---\nname: reviewer\ndescription: 代码评审\ntools: [bash, read]\n---\n评审正文\n"), 0o644)
	// skills/review/SKILL.md
	os.WriteFile(filepath.Join(claudeDir, "skills", "review", "SKILL.md"),
		[]byte("---\nname: review\ndescription: 评审技能\n---\n技能正文\n"), 0o644)
	// ~/.claude.json（含 mcpServers + 运行时状态）
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{
  "mcpServers": {"fs": {"command": "npx", "args": ["-y", "server-fs"], "env": {"TOKEN": "x"}}},
  "projects": {"/some/path": {"trust_level": "trusted"}}
}`), 0o644)

	// 直接调用 Import（绕过 paths.Home()，直接给 Location）
	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{
		Scope: ir.ScopeGlobal,
		Root:  claudeDir,
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	// CLAUDE.md → instruction + imports
	var inst *ir.Instruction
	for i := range b.Instructions {
		if b.Instructions[i].ID == "instruction.claude" {
			inst = &b.Instructions[i]
		}
	}
	if inst == nil {
		t.Fatal("缺少 CLAUDE.md instruction")
	}
	if len(inst.Imports) != 2 {
		t.Errorf("@import 应提取 2 个引用，实际 %v", inst.Imports)
	}
	if inst.Activation != ir.ActivationAlways {
		t.Errorf("CLAUDE.md activation 应为 always，实际 %s", inst.Activation)
	}

	// settings model
	var model *ir.SettingEntry
	for i := range b.Settings {
		if b.Settings[i].Key == "model" {
			model = &b.Settings[i]
		}
	}
	if model == nil || model.Value != "claude-opus" {
		t.Errorf("model setting 解析错误: %+v", model)
	}

	// hooks（PreToolUse → pre-tool-use）
	var hook *ir.Hook
	for i := range b.Hooks {
		if b.Hooks[i].Event == ir.HookPreToolUse {
			hook = &b.Hooks[i]
		}
	}
	if hook == nil {
		t.Fatal("缺少 pre-tool-use hook")
	}
	if hook.Handler.Command != "./guard.sh" || hook.Matcher.Tool != "Bash" {
		t.Errorf("hook 解析错误: %+v", hook.Handler)
	}

	// agent
	if len(b.Agents) != 1 || b.Agents[0].Name != "reviewer" {
		t.Fatalf("agent 解析错误: %+v", b.Agents)
	}
	if b.Agents[0].Body != "评审正文\n" {
		t.Errorf("agent 正文错误: %q", b.Agents[0].Body)
	}

	// skill
	if len(b.Skills) != 1 || b.Skills[0].Name != "review" {
		t.Fatalf("skill 解析错误: %+v", b.Skills)
	}

	// 注意：~/.claude.json 的 mcpServers 由 readClaudeJSONMCP 读取 home/.claude.json，
	// 本测试 Root 指向 .claude 目录，home 路径不在其中，故 MCP 单独在项目层测试。
}

func TestImportProject(t *testing.T) {
	root := t.TempDir()
	claudeDir := filepath.Join(root, ".claude")
	os.MkdirAll(claudeDir, 0o755)

	// 项目 CLAUDE.md
	os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# 项目规范\n"), 0o644)
	// .mcp.json
	os.WriteFile(filepath.Join(root, ".mcp.json"), []byte(`{
  "mcpServers": {
    "db": {"type": "http", "url": "https://db.example.com", "headers": {"Auth": "Bearer x"}},
    "fs": {"command": "npx", "args": ["-y", "fs"]}
  }
}`), 0o644)
	// .claude/settings.json
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{"permissions": {"allow": ["Bash(npm *)"]}}`), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{
		Scope: ir.ScopeProject,
		Root:  root,
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	// .mcp.json → 2 个 MCP server
	if len(b.MCPServers) != 2 {
		t.Fatalf("应解析 2 个 MCP server，实际 %d", len(b.MCPServers))
	}
	byName := map[string]ir.MCPServer{}
	for _, s := range b.MCPServers {
		byName[s.Name] = s
	}
	if db, ok := byName["db"]; !ok || db.Transport != "http" || db.URL != "https://db.example.com" {
		t.Errorf("http server 解析错误: %+v", db)
	}
	if fs, ok := byName["fs"]; !ok || fs.Transport != "stdio" || fs.Command != "npx" {
		t.Errorf("stdio server 解析错误（默认 transport 应为 stdio）: %+v", fs)
	}
	// 原始键名保留（导出还原依据）
	for _, s := range b.MCPServers {
		if s.Name == "" {
			t.Errorf("MCP server 缺原始键名: %+v", s)
		}
	}

	// permissions → 不透明 setting
	var perm *ir.SettingEntry
	for i := range b.Settings {
		if b.Settings[i].Key == "permissions" {
			perm = &b.Settings[i]
		}
	}
	if perm == nil {
		t.Fatal("缺少 permissions setting")
	}
	if pv, ok := perm.Value.(map[string]any); !ok || pv["allow"] == nil {
		t.Errorf("permissions 应为不透明嵌套 value: %+v", perm.Value)
	}
}

// 项目层 CLAUDE.local.md → scope=local
func TestImportLocalScope(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "CLAUDE.local.md"), []byte("# 私人备忘\n"), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeProject, Root: root})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	var found bool
	for _, inst := range b.Instructions {
		if inst.Origin != nil && inst.Origin.Scope == ir.ScopeLocal {
			found = true
		}
	}
	if !found {
		t.Error("CLAUDE.local.md 应标记 scope=local")
	}
}
