// Command cfg4ai-desktop 是 cfg4ai 的原生桌面壳（Gio 即时模式 GUI，纯 Go 无 CGO）。
// 独立编译产物 cfg4ai-desktop.exe，与主 CLI（cfg4ai.exe）分离。
// 界面：标题 + 已采集实体列表 + 退出。
package main

import (
	"image/color"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/platform/paths"
	"github.com/timywel/ai4config/internal/store"
)

type entityItem struct {
	kind, id, note string
}

func main() {
	items := loadEntities()
	go func() {
		w := new(app.Window)
		w.Option(app.Title("cfg4ai"), app.Size(unit.Dp(900), unit.Dp(640)))
		if err := loop(w, items); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

// loadEntities 读 global profile 的实体。
func loadEntities() []entityItem {
	var items []entityItem
	root, err := paths.DataHome()
	if err != nil {
		return items
	}
	repo, err := store.Open(root)
	if err != nil {
		return items
	}
	sb, err := profile.Load(repo.Path(store.DirProfiles, "global"), ir.ScopeGlobal)
	if err != nil {
		return items
	}
	b := sb.Bundle
	add := func(kind, id, note string) { items = append(items, entityItem{kind, id, note}) }
	for _, x := range b.Instructions {
		add("指令", x.ID, x.Description)
	}
	for _, x := range b.MCPServers {
		add("MCP", x.ID, x.Command+x.URL)
	}
	for _, x := range b.Skills {
		add("技能", x.ID, x.Description)
	}
	for _, x := range b.Agents {
		add("Agent", x.ID, x.Description)
	}
	for _, x := range b.Hooks {
		add("Hook", x.ID, string(x.Event))
	}
	for _, x := range b.Settings {
		add("设置", x.ID, x.Key)
	}
	return items
}

func loop(w *app.Window, items []entityItem) error {
	th := material.NewTheme()
	var ops op.Ops
	list := &widget.List{List: layout.List{Axis: layout.Vertical}}
	quitBtn := new(widget.Clickable)
	titleColor := color.NRGBA{R: 0x1a, G: 0x5f, B: 0xb4, A: 0xff}

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			if quitBtn.Clicked(gtx) {
				return nil
			}

			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.H5(th, "cfg4ai — 已采集实体")
								lbl.Color = titleColor
								return lbl.Layout(gtx)
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								btn := material.Button(th, quitBtn, "退出")
								return btn.Layout(gtx)
							}),
						)
					})
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						if len(items) == 0 {
							return material.Body1(th, "无数据（先 cfg4ai collect 采集）").Layout(gtx)
						}
						return list.Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
							it := items[i]
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										lbl := material.Label(th, unit.Sp(13), "["+it.kind+"]")
										lbl.Color = titleColor
										lbl.Font.Weight = font.Bold
										return lbl.Layout(gtx)
									}),
									layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
									layout.Rigid(material.Label(th, unit.Sp(13), it.id).Layout),
								)
							})
						})
					})
				}),
			)
			e.Frame(gtx.Ops)
		}
	}
}