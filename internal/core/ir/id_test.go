package ir

import "testing"

func TestParseID(t *testing.T) {
	cases := []struct {
		id       string
		wantKind EntityKind
		wantErr  bool
	}{
		{"mcp.filesystem", KindMCP, false},
		{"instruction.coding-style", KindInstruction, false},
		{"skill.code-review", KindSkill, false},
		{"hook.pre-tool-guard", KindHook, false},
		// D2：name 段放行点号与大写
		{"setting.copilot.chat.mcp.access", KindSetting, false},
		{"mcp.My-Server.v2", KindMCP, false},
		// setting 必须三段式
		{"setting.model", KindSetting, true},
		// 未知 type
		{"prompt.x", "", true},
		// 缺段
		{"mcp", "", true},
		{".filesystem", "", true},
		{"mcp.", "", true},
		// 非法字符
		{"mcp.bad name", "", true},
		{"mcp.-lead", "", true},
		{"mcp.中文", "", true},
	}
	for _, tc := range cases {
		kind, _, err := ParseID(tc.id)
		if tc.wantErr && err == nil {
			t.Errorf("ParseID(%q) 期望报错却成功（kind=%s）", tc.id, kind)
		}
		if !tc.wantErr {
			if err != nil {
				t.Errorf("ParseID(%q) 意外报错: %v", tc.id, err)
			} else if kind != tc.wantKind {
				t.Errorf("ParseID(%q) kind = %s, 期望 %s", tc.id, kind, tc.wantKind)
			}
		}
	}
}

func TestParseSettingID(t *testing.T) {
	tool, key, err := ParseSettingID("setting.copilot.chat.mcp.access")
	if err != nil {
		t.Fatalf("意外报错: %v", err)
	}
	if tool != "copilot" || key != "chat.mcp.access" {
		t.Fatalf("tool=%q key=%q，期望 copilot / chat.mcp.access", tool, key)
	}
	if _, _, err := ParseSettingID("mcp.filesystem"); err == nil {
		t.Fatal("非 setting id 应报错")
	}
}

func TestNameTail(t *testing.T) {
	if got := NameTail("setting.copilot.chat.mcp.access"); got != "access" {
		t.Fatalf("NameTail = %q，期望 access", got)
	}
	if got := NameTail("mcp.filesystem"); got != "filesystem" {
		t.Fatalf("NameTail = %q，期望 filesystem", got)
	}
}
