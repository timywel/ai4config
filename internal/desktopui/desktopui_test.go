package desktopui

import "testing"

func TestThemeStoreModes(t *testing.T) {
	ts := NewThemeStore("") // 空 → 默认 D
	if ts.Mode != ModeGlass {
		t.Error("默认应 D 玻璃拟态")
	}
	if ts.Colors != GlassColors() {
		t.Error("默认应为 Glass 色板")
	}
	ts.SetMode(ModeDarkPro)
	if ts.Colors != DarkProColors() {
		t.Error("A 应为 DarkPro 色板")
	}
	ts.SetMode(ModeLightClean)
	if ts.Colors != LightCleanColors() {
		t.Error("B 应为 LightClean 色板")
	}
	ts.SetMode("invalid")
	if ts.Mode != ModeGlass {
		t.Error("非法 mode 应回落 D")
	}
	if ts.Theme == nil {
		t.Error("Theme 不应为 nil")
	}
	if ts.Theme.Palette.Bg != ts.Colors.Bg {
		t.Error("Palette.Bg 应与 Colors.Bg 一致")
	}
}

func TestTextInverseToken(t *testing.T) {
	// 三主题都必须有反转文字令牌（强调色底上的文字，D 主题 Surface 半透明不能复用）
	for _, cs := range []Colors{DarkProColors(), LightCleanColors(), GlassColors()} {
		if cs.TextInverse.A != 0xFF {
			t.Error("TextInverse 必须不透明（玻璃主题 Surface 半透明不可作文字色）")
		}
	}
	if GlassColors().Surface.A == 0xFF {
		t.Error("D 玻璃主题 Surface 应半透明")
	}
}

func TestIcons(t *testing.T) {
	icons := MustIcons()
	if icons.Dashboard == nil || icons.Check == nil || icons.Key == nil || icons.Explore == nil || icons.Renew == nil {
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
