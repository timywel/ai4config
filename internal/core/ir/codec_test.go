package ir

import (
	"strings"
	"testing"
	"time"
)

func TestSplitFrontmatter(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantFM   string // "" 表示无 frontmatter
		wantBody string
		wantErr  bool
	}{
		{"标准", "---\nid: mcp.x\n---\n正文", "id: mcp.x", "正文", false},
		{"无 frontmatter", "id: mcp.x\n正文", "", "id: mcp.x\n正文", false},
		{"CRLF", "---\r\nid: mcp.x\r\n---\r\n正文", "id: mcp.x", "正文", false},
		{"正文为空", "---\nid: mcp.x\n---\n", "id: mcp.x", "", false},
		{"未闭合", "---\nid: mcp.x\n正文", "", "", true},
		{"结尾紧邻分隔线", "---\nid: mcp.x\n\n---", "id: mcp.x", "", false},
	}
	for _, tc := range cases {
		fm, body, err := SplitFrontmatter([]byte(tc.input))
		if tc.wantErr {
			if err == nil {
				t.Errorf("%s: 期望报错", tc.name)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: 意外报错: %v", tc.name, err)
			continue
		}
		if tc.wantFM == "" && fm != nil {
			t.Errorf("%s: 期望无 frontmatter，实际 %q", tc.name, fm)
		}
		if tc.wantFM != "" && !strings.Contains(string(fm), tc.wantFM) {
			t.Errorf("%s: frontmatter %q 不含 %q", tc.name, fm, tc.wantFM)
		}
		if tc.name != "无 frontmatter" && string(body) != tc.wantBody {
			t.Errorf("%s: body = %q，期望 %q", tc.name, body, tc.wantBody)
		}
	}
}

func TestEntityExtensionsRoundTrip(t *testing.T) {
	orig := &MCPServer{
		Header: Header{
			ID:        "mcp.filesystem",
			IRVersion: 1,
			Origin: &Origin{
				Tool: "claude-code", Path: ".mcp.json", Scope: ScopeProject,
				CollectedAt: time.Date(2026, 8, 16, 10, 0, 0, 0, time.FixedZone("CST", 8*3600)),
				RawHash:     "sha256:aaa", StoredHash: "sha256:bbb",
			},
		},
		Name:      "filesystem",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"-y", "server-filesystem"},
	}
	ext := map[string]any{
		"x-vscode": map[string]any{"inputs": []any{map[string]any{"id": "token", "type": "promptString"}}},
	}

	data, err := MarshalEntity(orig, ext)
	if err != nil {
		t.Fatalf("MarshalEntity: %v", err)
	}
	if !strings.Contains(string(data), "x-vscode:") {
		t.Fatalf("序列化结果缺少 x- 键:\n%s", data)
	}

	var back MCPServer
	gotExt, err := UnmarshalEntity(data, &back)
	if err != nil {
		t.Fatalf("UnmarshalEntity: %v", err)
	}
	if back.ID != orig.ID || back.Name != "filesystem" || back.Transport != "stdio" {
		t.Fatalf("round-trip 字段不一致: %+v", back)
	}
	if back.Origin == nil || back.Origin.Tool != "claude-code" {
		t.Fatalf("origin 丢失: %+v", back.Origin)
	}
	vs, ok := gotExt["x-vscode"].(map[string]any)
	if !ok {
		t.Fatalf("x-vscode 未收拢: %+v", gotExt)
	}
	if _, ok := vs["inputs"]; !ok {
		t.Fatalf("x-vscode.inputs 丢失: %+v", vs)
	}
}

func TestMarkdownDocRoundTrip(t *testing.T) {
	inst := &Instruction{
		Header:   Header{ID: "instruction.coding-style", IRVersion: 1},
		Priority: 100,
		Language: "zh",
	}
	body := "# 编码规范\n\n- 所有回复使用中文\n"
	data, err := MarshalMarkdownDoc(inst, nil, body)
	if err != nil {
		t.Fatalf("MarshalMarkdownDoc: %v", err)
	}

	var back Instruction
	gotBody, _, err := UnmarshalMarkdownDoc(data, &back)
	if err != nil {
		t.Fatalf("UnmarshalMarkdownDoc: %v", err)
	}
	if back.ID != inst.ID || back.Priority != 100 {
		t.Fatalf("frontmatter 不一致: %+v", back)
	}
	if gotBody != body {
		t.Fatalf("正文不一致:\n%q\n期望:\n%q", gotBody, body)
	}
}

func TestMarshalEntityRejectsNonXKey(t *testing.T) {
	_, err := MarshalEntity(&MCPServer{Header: Header{ID: "mcp.x", IRVersion: 1}, Name: "x", Transport: "stdio", Command: "c"},
		map[string]any{"vscode": 1})
	if err == nil {
		t.Fatal("非 x- 前缀扩展键应被拒绝")
	}
}
