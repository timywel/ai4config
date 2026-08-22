package desktopui

import (
	"image"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/outlay"
)

// ChipGroup 一组可单选的 chip（替代竖排 RadioButton）。
type ChipGroup struct {
	Value string
	btns  map[string]*widget.Clickable
}

// NewChipGroup 构造。
func NewChipGroup(initial string) *ChipGroup {
	return &ChipGroup{Value: initial, btns: map[string]*widget.Clickable{}}
}

func (g *ChipGroup) btn(opt string) *widget.Clickable {
	if g.btns[opt] == nil {
		g.btns[opt] = new(widget.Clickable)
	}
	return g.btns[opt]
}

// Layout 流式 chip 布局（自动换行；点选设置 Value）。
func (g *ChipGroup) Layout(gtx layout.Context, th *material.Theme, cs Colors, options []string) layout.Dimensions {
	for _, opt := range options {
		if g.btn(opt).Clicked(gtx) {
			g.Value = opt
		}
	}
	return outlay.FlowWrap{}.Layout(gtx, len(options), func(gtx layout.Context, i int) layout.Dimensions {
		opt := options[i]
		return chip(gtx, th, cs, opt, g.Value == opt, g.btn(opt))
	})
}

// chip 单个圆角标签（选中 accent 填充）。
func chip(gtx layout.Context, th *material.Theme, cs Colors, label string, selected bool, btn *widget.Clickable) layout.Dimensions {
	bg := cs.Surface
	fg := cs.Text
	if selected {
		bg = cs.AccentSoft // 柔和选中：AccentSoft 底 + Accent 文字（对比度达标 + 与导航选中语义一致）
		fg = cs.Accent
	} else if btn.Hovered() {
		bg = cs.SurfaceHover
	}
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		pointer.CursorPointer.Add(gtx.Ops)
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				r := gtx.Dp(unit.Dp(Radius2XL)) // 半圆角（chip 胶囊形）
				rr := clip.RRect{Rect: image.Rectangle{Max: gtx.Constraints.Min}, NE: r, NW: r, SE: r, SW: r}
				paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(SpaceM)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body2(th, label)
					lbl.Color = fg
					return lbl.Layout(gtx)
				})
			}),
		)
	})
}
