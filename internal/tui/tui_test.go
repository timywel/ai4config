package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBrowserCursorMove(t *testing.T) {
	items := []Item{
		{Kind: "mcp", ID: "mcp.a"},
		{Kind: "skill", ID: "skill.b"},
	}
	m := model{title: "t", items: items}
	// 下移
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	mm := m2.(model)
	if mm.cursor != 1 {
		t.Errorf("j 应下移光标到 1，实际 %d", mm.cursor)
	}
	// 越界不下移
	m3, _ := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m3.(model).cursor != 1 {
		t.Error("光标不应越界")
	}
	// 上移
	m4, _ := m3.(model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m4.(model).cursor != 0 {
		t.Error("k 应上移光标到 0")
	}
}

func TestBrowserView(t *testing.T) {
	items := []Item{{Kind: "mcp", ID: "mcp.a", Note: "npx"}}
	m := model{title: "标题", items: items}
	v := m.View()
	if !strings.Contains(v, "mcp.a") || !strings.Contains(v, "标题") {
		t.Errorf("View 应含标题与条目: %s", v)
	}
	if !strings.Contains(v, "退出") {
		t.Error("View 应含操作提示")
	}
}

func TestBrowserQuit(t *testing.T) {
	m := model{title: "t", items: []Item{{Kind: "mcp", ID: "x"}}}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("q 应返回 Quit 命令")
	}
}
