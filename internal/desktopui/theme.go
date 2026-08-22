// Package desktopui 是 cfg4ai 桌面应用的设计系统基座。
// 设计依据：docs/DESKTOP-UI-DESIGN.md + docs/design-demos（A/B/D 三稿，D 默认）。
package desktopui

import (
	"image/color"

	"gioui.org/text"
	"gioui.org/widget/material"
)

// Colors 语义色板（material.Palette 仅 4 色，精致界面需要扩展色表）。
type Colors struct {
	Bg            color.NRGBA // 应用底色
	Surface       color.NRGBA // 卡片/面板
	SurfaceHover  color.NRGBA // 悬停态
	Border        color.NRGBA // 描边
	Text          color.NRGBA // 主文字
	TextSecondary color.NRGBA // 次级文字
	Accent        color.NRGBA // 主强调
	AccentSoft    color.NRGBA // 柔和强调（tag/badge 底色）
	TextInverse   color.NRGBA // 反转文字（强调色/成功/危险底上的文字，恒白）
	Success       color.NRGBA
	Danger        color.NRGBA
	Warn          color.NRGBA

	// D 玻璃拟态专用（其他主题忽略）
	IsGlass     bool        // 玻璃风标记（背景渐变+半透明卡片）
	GradientTop color.NRGBA // 背景渐变三段
	GradientMid color.NRGBA
	GradientBot color.NRGBA
	GlowA       color.NRGBA // 光斑（右上紫）
	GlowB       color.NRGBA // 光斑（左下青）
	Accent2     color.NRGBA // 次强调（渐变青）
}

// DarkProColors A 稿：深色专业风（Catppuccin Mocha 系）。
func DarkProColors() Colors {
	return Colors{
		Bg:            nrgba(0x1E1E2E),
		Surface:       nrgba(0x181825),
		SurfaceHover:  nrgba(0x313244),
		Border:        nrgba(0x313244),
		Text:          nrgba(0xCDD6F4),
		TextSecondary: nrgba(0xB8C0DA),
		Accent:        nrgba(0x89B4FA),
		AccentSoft:    nrgba(0x313244),
		TextInverse:   nrgba(0xFFFFFF),
		Success:       nrgba(0xA6E3A1),
		Danger:        nrgba(0xF38BA8),
		Warn:          nrgba(0xF9E2AF),
	}
}

// LightCleanColors B 稿：浅色清爽风（Linear/Notion 系）。
func LightCleanColors() Colors {
	return Colors{
		Bg:            nrgba(0xF7F7F5),
		Surface:       nrgba(0xFFFFFF),
		SurfaceHover:  nrgba(0xF1F1EF),
		Border:        nrgba(0xE9E9E7),
		Text:          nrgba(0x37352F),
		TextSecondary: nrgba(0x9B9A97),
		Accent:        nrgba(0x5B6CFF),
		AccentSoft:    nrgba(0xEEF0FF),
		TextInverse:   nrgba(0xFFFFFF),
		Success:       nrgba(0x2E9E5B),
		Danger:        nrgba(0xE03E3E),
		Warn:          nrgba(0xD9730D),
	}
}

// GlassColors D 稿：现代玻璃拟态风（深色渐变底 + 半透明卡片）。
func GlassColors() Colors {
	return Colors{
		Bg:            nrgba(0x191735),
		Surface:       color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x14}, // 8% 白（玻璃感）
		SurfaceHover:  color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x22}, // 13% 白
		Border:        color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x2E}, // 18% 白描边
		Text:          nrgba(0xE8E8F0),
		TextSecondary: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x80},
		Accent:        nrgba(0xBFA6FF),
		Accent2:       nrgba(0x60D5FA),
		AccentSoft:    color.NRGBA{R: 0xA7, G: 0x8B, B: 0xFA, A: 0x26}, // 15% 紫
		TextInverse:   nrgba(0xFFFFFF),
		Success:       nrgba(0x34D399),
		Danger:        nrgba(0xF87171),
		Warn:          nrgba(0xFBBF24),
		IsGlass:       true,
		GradientTop:   nrgba(0x0F0C29),
		GradientMid:   nrgba(0x302B63),
		GradientBot:   nrgba(0x24243E),
		GlowA:         color.NRGBA{R: 0x78, G: 0x50, B: 0xFF, A: 0x40}, // 25% 紫
		GlowB:         color.NRGBA{R: 0x00, G: 0xD2, B: 0xFF, A: 0x30}, // 19% 青
	}
}

func nrgba(hex uint32) color.NRGBA {
	return color.NRGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 0xff}
}

// 主题模式。
const (
	ModeDarkPro    = "A" // 深色专业
	ModeLightClean = "B" // 浅色清爽
	ModeGlass      = "D" // 玻璃拟态（默认）
)

// ModeNames 主题显示名（设置 UI 用）。
var ModeNames = map[string]string{
	ModeDarkPro:    "A 深色专业",
	ModeLightClean: "B 浅色清爽",
	ModeGlass:      "D 玻璃拟态",
}

// ThemeStore 主题持有者（支持运行时切换）。
type ThemeStore struct {
	Mode   string
	Theme  *material.Theme
	Colors Colors
}

// NewThemeStore 构造主题（mode: "A"/"B"/"D"，空或非法值默认 D）。
func NewThemeStore(mode string) *ThemeStore {
	ts := &ThemeStore{}
	ts.SetMode(mode)
	return ts
}

// SetMode 切换主题（调用方需随后 w.Invalidate()）。
func (ts *ThemeStore) SetMode(mode string) {
	switch mode {
	case ModeDarkPro, ModeLightClean, ModeGlass:
		ts.Mode = mode
	default:
		ts.Mode = ModeGlass // 默认 D
	}
	ts.apply()
}

func (ts *ThemeStore) apply() {
	ts.Theme = material.NewTheme()
	switch ts.Mode {
	case ModeDarkPro:
		ts.Colors = DarkProColors()
	case ModeLightClean:
		ts.Colors = LightCleanColors()
	default:
		ts.Colors = GlassColors()
	}
	ts.Theme.Palette = material.Palette{
		Bg:         ts.Colors.Bg,
		Fg:         ts.Colors.Text,
		ContrastBg: ts.Colors.Accent,
		ContrastFg: nrgba(0xFFFFFF),
	}
	_ = text.NewShaper // CJK 字体兜底占位（麒麟等精简系统防豆腐块，后续接入 embed 子集字体）
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
