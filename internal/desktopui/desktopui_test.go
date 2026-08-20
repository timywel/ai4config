package desktopui

import "testing"

func TestThemeStoreToggle(t *testing.T) {
	ts := NewThemeStore(true)
	if !ts.Dark || ts.Colors != DarkColors() {
		t.Error("默认深色")
	}
	ts.Toggle()
	if ts.Dark || ts.Colors != LightColors() {
		t.Error("切换后应浅色")
	}
	if ts.Theme == nil {
		t.Error("Theme 不应为 nil")
	}
	if ts.Theme.Palette.Bg != ts.Colors.Bg {
		t.Error("Palette.Bg 应与 Colors.Bg 一致")
	}
}

func TestIcons(t *testing.T) {
	icons := MustIcons()
	if icons.Dashboard == nil || icons.Check == nil || icons.Key == nil {
		t.Error("图标应为非 nil")
	}
}

func TestConfirmModalShowHide(t *testing.T) {
	m := NewConfirmModal()
	if m.visible {
		t.Error("初始不可见")
	}
	called := false
	m.Show("标题", "内容", func() { called = true })
	if !m.visible {
		t.Error("Show 后可见")
	}
	_ = called
}
