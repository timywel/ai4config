package desktopui

import (
	"image"
	"image/color"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
)

// Card 圆角卡片（surface 填充 + hover 态 + 可选点击 + 手型光标）。
// Gio 无 box-shadow，用圆角矩形填充 + 描边表达层次（DESKTOP-UI-DESIGN §卡片）。
func Card(gtx layout.Context, cs Colors, clickable *widget.Clickable, content layout.Widget) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			r := gtx.Dp(unit.Dp(RadiusL))
			rr := clip.RRect{Rect: image.Rectangle{Max: gtx.Constraints.Min}, NE: r, NW: r, SE: r, SW: r}
			bg := cs.Surface
			if clickable != nil && clickable.Hovered() {
				bg = cs.SurfaceHover
			}
			paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
			// 描边
			paint.FillShape(gtx.Ops, cs.Border, clip.Stroke{Path: rr.Path(gtx.Ops), Width: 1}.Op())
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if clickable != nil {
				return clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					pointer.CursorPointer.Add(gtx.Ops)
					return layout.UniformInset(unit.Dp(SpaceL)).Layout(gtx, content)
				})
			}
			return layout.UniformInset(unit.Dp(SpaceL)).Layout(gtx, content)
		}),
	)
}

// Badge 徽标（kind/状态小标签）。
func Badge(gtx layout.Context, cs Colors, text string, bg color.NRGBA, content layout.Widget) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			r := gtx.Dp(unit.Dp(RadiusM))
			rr := clip.RRect{Rect: image.Rectangle{Max: gtx.Constraints.Min}, NE: r, NW: r, SE: r, SW: r}
			paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(SpaceS)).Layout(gtx, content)
		}),
	)
}
