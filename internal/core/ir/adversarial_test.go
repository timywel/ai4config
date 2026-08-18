package ir

import (
	"strings"
	"testing"
)

// 对抗用例回归（research/adversarial-cases.md）。命名 TestAdversarial_<编号>。

// AC-A1 import-cycle-direct：CLAUDE.md ↔ docs/a.md 直接环，DFS 应检出
func TestAdversarial_A1_ImportCycleDirect(t *testing.T) {
	refs := map[string][]string{
		"CLAUDE.md":    {"./docs/a.md"},
		"./docs/a.md":  {"../CLAUDE.md"},
		"../CLAUDE.md": {"./docs/a.md"}, // 归一化前近似
	}
	// 归一化节点后成环
	refs2 := map[string][]string{
		"CLAUDE.md": {"docs/a.md"},
		"docs/a.md": {"CLAUDE.md"},
	}
	cycle := DetectImportCycle(refs2)
	if cycle == nil {
		t.Fatal("直接互相引用应检出环")
	}
	_ = refs
}

// AC-A1 无环场景不误报
func TestAdversarial_A1_NoCycle(t *testing.T) {
	refs := map[string][]string{
		"CLAUDE.md": {"docs/a.md", "docs/b.md"},
		"docs/a.md": {"docs/c.md"},
		"docs/b.md": {},
		"docs/c.md": {},
	}
	if cycle := DetectImportCycle(refs); cycle != nil {
		t.Errorf("无环不应误报，实际环 %v", cycle)
	}
}

// AC-E3 cross-tool-import-loop：CLAUDE.md（claude 系）与 AGENTS.md（codex 系）互相 @import
// 图论域为全局引用图 → 跨条目环必须检出
func TestAdversarial_E3_CrossToolImportLoop(t *testing.T) {
	instructions := []Instruction{
		{
			Header:  Header{ID: "instruction.claude", IRVersion: 1, Origin: &Origin{Tool: "claude-code", Path: "CLAUDE.md"}},
			Imports: []Import{{Path: "AGENTS.md", Resolved: true}},
		},
		{
			Header:  Header{ID: "instruction.agents", IRVersion: 1, Origin: &Origin{Tool: "codex", Path: "AGENTS.md"}},
			Imports: []Import{{Path: "CLAUDE.md", Resolved: true}},
		},
	}
	refs := InstructionImportRefs(instructions)
	cycle := DetectImportCycle(refs)
	if cycle == nil {
		t.Fatal("跨工具互相引用（CLAUDE.md↔AGENTS.md）应检出环")
	}
	joined := strings.Join(cycle, "->")
	if !strings.Contains(joined, "CLAUDE.md") || !strings.Contains(joined, "AGENTS.md") {
		t.Errorf("环应含两个文件: %v", cycle)
	}
}

// AC-A5 setting-key-with-dot：VS Code 风格点号 key 的三段式解析
func TestAdversarial_A5_SettingKeyWithDot(t *testing.T) {
	tool, key, err := ParseSettingID("setting.copilot.editor.fontSize")
	if err != nil {
		t.Fatalf("点号 key 应可解析: %v", err)
	}
	if tool != "copilot" || key != "editor.fontSize" {
		t.Errorf("tool=%q key=%q，期望 copilot/editor.fontSize", tool, key)
	}
	// 更深嵌套
	_, key2, _ := ParseSettingID("setting.copilot.chat.mcp.access")
	if key2 != "chat.mcp.access" {
		t.Errorf("嵌套 key 应完整保留: %q", key2)
	}
}

// AC-A3 skill-name-case-dot：id 放行点号与大写（规范化由适配器 sanitizeIDName 负责）
func TestAdversarial_A3_IDAllowsDotAndCase(t *testing.T) {
	// 规范化后的 id（MyHelper.v2 → myhelper.v2）应通过校验
	if _, _, err := ParseID("skill.myhelper.v2"); err != nil {
		t.Errorf("含点号 id 应合法: %v", err)
	}
	if _, _, err := ParseID("skill.MyHelper"); err != nil {
		t.Errorf("含大写 id 应合法: %v", err)
	}
	// 但空格等仍非法
	if _, _, err := ParseID("skill.my helper"); err == nil {
		t.Error("含空格 id 应非法")
	}
}
