// Package desktopui 是 cfg4ai 桌面应用的设计系统基座。
// 设计依据：docs/DESKTOP-UI-DESIGN.md（双主题色板/字体/间距/圆角 token）。
package desktopui

import (
	"image/color"

	"gioui.org/text"
	"gioui.org/widget/material"
)

// Colors 语义色板（material.Palette 仅 4 色，精致界面需要扩展色表）。
type Colors struct {
	Bg            color.NRGBA // 应用底色（最深）
	Surface       color.NRGBA // 卡片/面板
	SurfaceHover  color.NRGBA // 悬停态
	Border        color.NRGBA // 描边
	Text          color.NRGBA // 主文字
	TextSecondary color.NRGBA // 次级文字
	Accent        color.NRGBA // 主强调（Linear 靛蓝）
	Success       color.NRGBA
	Danger        color.NRGBA
}

// DarkColors 深色主题（默认，Linear 式）。
func DarkColors() Colors {
	return Colors{
		Bg:            nrgba(0x0B0C0E),
		Surface:       nrgba(0x171A21),
		SurfaceHover:  nrgba(0x1F242E),
		Border:        nrgba(0x2A303C),
		Text:          nrgba(0xE6E9EF),
		TextSecondary: nrgba(0x8B93A1),
		Accent:        nrgba(0x5E6AD2),
		Success:       nrgba(0x4CB782),
		Danger:        nrgba(0xEB5757),
	}
}

// LightColors 浅色主题（GitHub Primer 原值）。
func LightColors() Colors {
	return Colors{
		Bg:            nrgba(0xFFFFFF),
		Surface:       nrgba(0xF6F8FA),
		SurfaceHover:  nrgba(0xEAEEF2),
		Border:        nrgba(0xD1D9E0),
		Text:          nrgba(0x1F2328),
		TextSecondary: nrgba(0x59636E),
		Accent:        nrgba(0x0969DA),
		Success:       nrgba(0x1A7F37),
		Danger:        nrgba(0xD1242F),
	}
}

func nrgba(hex uint32) color.NRGBA {
	return color.NRGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 0xff}
}

// ThemeStore 主题持有者（支持运行时切换）。
type ThemeStore struct {
	Dark   bool
	Theme  *material.Theme
	Colors Colors
}

// NewThemeStore 构造主题（dark=true 深色）。
func NewThemeStore(dark bool) *ThemeStore {
	ts := &ThemeStore{Dark: dark}
	ts.apply()
	return ts
}

// SetDark 切换主题（调用方需随后 w.Invalidate()）。
func (ts *ThemeStore) SetDark(dark bool) {
	ts.Dark = dark
	ts.apply()
}

// Toggle 切换深浅主题。
func (ts *ThemeStore) Toggle() { ts.SetDark(!ts.Dark) }

func (ts *ThemeStore) apply() {
	ts.Theme = material.NewTheme()
	if ts.Dark {
		ts.Colors = DarkColors()
	} else {
		ts.Colors = LightColors()
	}
	ts.Theme.Palette = material.Palette{
		Bg:         ts.Colors.Bg,
		Fg:         ts.Colors.Text,
		ContrastBg: ts.Colors.Accent,
		ContrastFg: nrgba(0xFFFFFF),
	}
	// CJK 字体兜底：embed 子集字体（麒麟等精简系统防豆腐块）。
	// 字体文件就位后取消注释：
	// if face, err := opentype.Parse(notoSCSubset); err == nil {
	// 	ts.Theme.Shaper = text.NewShaper(text.WithCollection(
	// 		[]text.FontFace{{Font: font.Font{Typeface: "NotoSansSC"}, Face: face}}))
	// 	ts.Theme.Face = "NotoSansSC"
	// }
	_ = text.NewShaper // 占位保留依赖
}

// 间距 token（DESKTOP-UI-DESIGN §间距系统）。
const (
	SpaceXS  = 4
	SpaceS   = 8
	SpaceM   = 12
	SpaceL   = 16
	SpaceXL  = 20
	SpaceXXL = 24
	Space3XL = 32
	Space4XL = 40
)

// 圆角 token。
const (
	RadiusS   = 4
	RadiusM   = 6
	RadiusL   = 10
	RadiusXL  = 14
	Radius2XL = 20
)
