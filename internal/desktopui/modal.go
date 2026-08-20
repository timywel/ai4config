package desktopui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// ConfirmModal 危险操作二次确认对话框（自绘模态：遮罩 + 居中卡片）。
type ConfirmModal struct {
	visible   bool
	title     string
	body      string
	onOK      func()
	okBtn     *widget.Clickable
	cancelBtn *widget.Clickable
}

// NewConfirmModal 构造确认框。
func NewConfirmModal() *ConfirmModal {
	return &ConfirmModal{
		okBtn:     new(widget.Clickable),
		cancelBtn: new(widget.Clickable),
	}
}

// Show 显示确认框。
func (m *ConfirmModal) Show(title, body string, onOK func()) {
	m.title = title
	m.body = body
	m.onOK = onOK
	m.visible = true
}

// Layout 渲染模态（在根布局末尾以 Stack 叠加调用）。
func (m *ConfirmModal) Layout(gtx layout.Context, th *material.Theme, cs Colors) layout.Dimensions {
	if !m.visible {
		return layout.Dimensions{}
	}
	if m.okBtn.Clicked(gtx) {
		m.visible = false
		if m.onOK != nil {
			m.onOK()
		}
	}
	if m.cancelBtn.Clicked(gtx) {
		m.visible = false
	}
	return layout.Stack{}.Layout(gtx,
		// 半透明遮罩
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paint.Fill(gtx.Ops, color.NRGBA{A: 0x99})
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		// 居中确认卡片
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.H6(th, m.title).Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(SpaceM)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body1(th, m.body)
							lbl.Color = cs.TextSecondary
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(SpaceL)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(th, m.okBtn, "确认")
									btn.Background = cs.Danger
									return btn.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(SpaceM)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return material.Button(th, m.cancelBtn, "取消").Layout(gtx)
								}),
							)
						}),
					)
				})
			})
		}),
	)
}

// 确保 image 被引用。
var _ = image.Rectangle{}
