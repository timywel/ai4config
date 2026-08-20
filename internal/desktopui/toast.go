package desktopui

import (
	"image"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

// Toast 右下角滑入通知（自动消失）。
type Toast struct {
	text     string
	isErr    bool
	anim     component.VisibilityAnimation
	deadline time.Time
}

// Show 弹出 toast（isErr 决定是否用危险色）。
func (t *Toast) Show(now time.Time, text string, isErr bool) {
	t.text = text
	t.isErr = isErr
	t.anim = component.VisibilityAnimation{State: component.Invisible, Duration: 160 * time.Millisecond}
	t.anim.Appear(now)
	t.deadline = now.Add(3 * time.Second)
}

// Layout 渲染 toast（叠加在右下角）。
func (t *Toast) Layout(gtx layout.Context, th *material.Theme, cs Colors) layout.Dimensions {
	if t.deadline.IsZero() {
		return layout.Dimensions{}
	}
	if gtx.Now.After(t.deadline) && t.anim.State == component.Visible {
		t.anim.Disappear(gtx.Now)
	}
	rev := t.anim.Revealed(gtx)
	if rev <= 0 {
		return layout.Dimensions{}
	}
	return layout.SE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(SpaceL)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			bg := cs.Success
			if t.isErr {
				bg = cs.Danger
			}
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					r := gtx.Dp(unit.Dp(RadiusM))
					rr := clip.RRect{Rect: image.Rectangle{Max: gtx.Constraints.Min}, NE: r, NW: r, SE: r, SW: r}
					paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
					return layout.Dimensions{Size: gtx.Constraints.Min}
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(SpaceM)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(th, t.text)
						lbl.Color = cs.Surface
						return lbl.Layout(gtx)
					})
				}),
			)
		})
	})
}

// 确保 op 被引用（动画帧驱动在 VisibilityAnimation 内部）。
var _ = op.InvalidateCmd{}
