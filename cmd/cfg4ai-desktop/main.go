// Command cfg4ai-desktop 是 cfg4ai 的原生桌面应用（Gio 即时模式 GUI，纯 Go 无 CGO）。
// 双击即用：全部功能（采集/迁移/快照/浏览）在窗口内点按钮完成，无需命令行。
package main

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"os"
	"strings"

	"gioui.org/app"
	"gioui.org/font"
	"image"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/timywel/ai4config/internal/adapters"
	_ "github.com/timywel/ai4config/internal/adapters/all"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/migrate"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/core/secrets"
	"github.com/timywel/ai4config/internal/desktopui"
	"github.com/timywel/ai4config/internal/platform/paths"
	"github.com/timywel/ai4config/internal/store"
)

type entityItem struct{ kind, id, note string }

// pages
const (
	pageDashboard = iota
	pageEntities
	pageCollect
	pageMigrate
	pageSnapshot
)

var pageNames = []string{"仪表盘", "实体", "采集", "迁移", "快照"}

// toolOptions 采集/迁移可选工具。
var toolOptions = []string{"claude-code", "codex", "copilot", "zhanlu", "gemini", "claude-desktop", "grokbuild", "cursor", "windsurf", "aider", "cline", "roo", "opencode"}

// kindOptions 实体类型过滤 chip 选项。
var kindOptions = []string{"全部", "指令", "MCP", "技能", "Agent", "Hook", "设置"}

type desktopApp struct {
	repo *store.Repo

	page    int
	items   []entityItem
	stats   struct{ tools, entities, snapshots int }
	msg     string
	msgErr  bool
	loading bool

	// 控件
	navBtns    []*widget.Clickable
	refreshBtn *widget.Clickable

	collectTool  *desktopui.ChipGroup
	collectBtn   *widget.Clickable
	collectScope *desktopui.ChipGroup

	migrateFrom *desktopui.ChipGroup
	migrateTo   *desktopui.ChipGroup
	migrateBtn  *widget.Clickable
	migrateDry  *widget.Bool

	snapNote   *widget.Editor
	snapCreate *widget.Clickable
	snapList   []snapItem

	entityList *widget.List
	snapWidget *widget.List
	win        *app.Window // 用于 goroutine 完成后触发重绘

	ts    *desktopui.ThemeStore // 主题（深色默认）
	icons desktopui.IconSet     // 图标集

	bundle      *ir.Bundle                // 完整实体数据（详情用）
	selKind     string                    // 选中实体类型
	selID       string                    // 选中实体 id
	detailBtns  map[int]*widget.Clickable // 实体行点击
	closeDetail *widget.Clickable         // 关闭详情

	searchEd   *widget.Editor       // 搜索框（F02）
	kindFilter *desktopui.ChipGroup // 类型过滤 chip（F02）
	filtered   []entityItem         // 过滤后的实体
}

type snapItem struct {
	id, note string
	files    int
	restore  *widget.Clickable
}

func newDesktopApp() *desktopApp {
	d := &desktopApp{
		entityList:   &widget.List{List: layout.List{Axis: layout.Vertical}},
		snapWidget:   &widget.List{List: layout.List{Axis: layout.Vertical}},
		collectTool:  desktopui.NewChipGroup(""),
		collectScope: desktopui.NewChipGroup("all"),
		migrateFrom:  desktopui.NewChipGroup("claude-code"),
		migrateTo:    desktopui.NewChipGroup("codex"),
		snapNote:     &widget.Editor{},
		refreshBtn:   new(widget.Clickable),
		collectBtn:   new(widget.Clickable),
		migrateBtn:   new(widget.Clickable),
		snapCreate:   new(widget.Clickable),
	}
	for range pageNames {
		d.navBtns = append(d.navBtns, new(widget.Clickable))
	}
	d.collectTool.Value = ""
	d.collectScope.Value = "all"
	d.ts = desktopui.NewThemeStore(true) // 深色默认
	d.icons = desktopui.MustIcons()
	d.detailBtns = map[int]*widget.Clickable{}
	d.closeDetail = new(widget.Clickable)
	d.searchEd = &widget.Editor{}
	d.kindFilter = desktopui.NewChipGroup("全部")
	d.migrateFrom.Value = "claude-code"
	d.migrateTo.Value = "codex"

	root, err := paths.DataHome()
	if err == nil {
		d.repo, _ = store.Open(root)
	}
	d.reload()
	return d
}

func main() {
	d := newDesktopApp()
	go func() {
		w := new(app.Window)
		w.Option(app.Title("cfg4ai"), app.Size(unit.Dp(960), unit.Dp(680)))
		if err := d.loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func (d *desktopApp) reload() {
	d.items = nil
	d.bundle = nil
	if d.repo == nil {
		return
	}
	sb, err := profile.Load(d.repo.Path(store.DirProfiles, "global"), ir.ScopeGlobal)
	if err == nil {
		d.bundle = sb.Bundle
		b := sb.Bundle
		add := func(kind, id, note string) { d.items = append(d.items, entityItem{kind, id, note}) }
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
	}
	d.stats.tools = len(adapters.List())
	d.stats.entities = len(d.items)
	if snaps, err := d.repo.ListSnapshots(); err == nil {
		d.stats.snapshots = len(snaps)
		d.snapList = nil
		for _, s := range snaps {
			d.snapList = append(d.snapList, snapItem{id: s.ID, note: s.Note, files: len(s.Files), restore: new(widget.Clickable)})
		}
	}
}

// loop 主事件循环 + 布局。
func (d *desktopApp) loop(w *app.Window) error {
	d.win = w // 持有窗口引用供异步重绘
	th := d.ts.Theme
	cs := d.ts.Colors
	var ops op.Ops
	titleColor := cs.Accent
	errColor := cs.Danger
	okColor := cs.Success

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			paint.Fill(gtx.Ops, cs.Bg) // 主题底色

			// 导航点击
			for i, btn := range d.navBtns {
				if btn.Clicked(gtx) {
					d.page = i
					if i == pageDashboard || i == pageSnapshot {
						d.reload()
					}
				}
			}
			if d.refreshBtn.Clicked(gtx) {
				d.reload()
			}
			if d.closeDetail.Clicked(gtx) {
				d.selID = ""
			}
			// 采集
			if d.collectBtn.Clicked(gtx) && !d.loading {
				d.loading = true
				d.msg = "采集中…"
				d.msgErr = false
				go d.doCollect()
			}
			// 迁移
			if d.migrateBtn.Clicked(gtx) && !d.loading {
				d.loading = true
				d.msg = "迁移中…"
				d.msgErr = false
				go d.doMigrate()
			}
			// 快照创建
			if d.snapCreate.Clicked(gtx) && !d.loading {
				d.loading = true
				d.msg = "创建快照中…"
				d.msgErr = false
				go d.doSnapshotCreate()
			}
			// 快照恢复
			for _, s := range d.snapList {
				if s.restore.Clicked(gtx) && !d.loading {
					d.loading = true
					d.msg = "恢复中…"
					d.msgErr = false
					go d.doSnapshotRestore(s.id)
				}
			}

			// 布局：左侧导航 + 右侧内容
			layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.H6(th, "cfg4ai")
								lbl.Color = titleColor
								lbl.Font.Weight = font.Bold
								return lbl.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return d.navLayout(gtx, th)
							}),
						)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return d.pageLayout(gtx, th, titleColor, errColor, okColor)
							}),
						)
					})
				}),
			)
			e.Frame(gtx.Ops)
		}
	}
}

// navLayout 左侧图标导航（图标+文字，选中高亮卡片式）。
func (d *desktopApp) navLayout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	cs := d.ts.Colors
	navIcons := []*widget.Icon{d.icons.Dashboard, d.icons.List, d.icons.Download, d.icons.Sync, d.icons.Snapshot}
	var children []layout.FlexChild
	for i, name := range pageNames {
		i := i
		var ic *widget.Icon
		if i < len(navIcons) {
			ic = navIcons[i]
		}
		selected := d.page == i
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return d.navItem(gtx, th, cs, ic, name, selected, d.navBtns[i])
		}))
	}
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return d.navItem(gtx, th, cs, d.icons.History, "刷新", false, d.refreshBtn)
	}))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// navItem 单个导航项：图标+文字，选中/hover 态背景。
func (d *desktopApp) navItem(gtx layout.Context, th *material.Theme, cs desktopui.Colors, icon *widget.Icon, name string, selected bool, btn *widget.Clickable) layout.Dimensions {
	bg := color.NRGBA{}
	fg := cs.Text
	if selected {
		bg = cs.Accent
		fg = cs.Surface
	} else if btn.Hovered() {
		bg = cs.SurfaceHover
	}
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		pointer.CursorPointer.Add(gtx.Ops)
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				if selected || btn.Hovered() {
					r := gtx.Dp(unit.Dp(desktopui.RadiusM))
					rr := clip.RRect{Rect: image.Rectangle{Max: gtx.Constraints.Min}, NE: r, NW: r, SE: r, SW: r}
					paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
				}
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(desktopui.SpaceM)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if icon != nil {
								return icon.Layout(gtx, fg)
							}
							return layout.Dimensions{}
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceM)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Body2(th, name)
							lbl.Color = fg
							return lbl.Layout(gtx)
						}),
					)
				})
			}),
		)
	})
}

// pageLayout 右侧内容区（按当前页切换）。
func (d *desktopApp) pageLayout(gtx layout.Context, th *material.Theme, titleColor, errColor, okColor color.NRGBA) layout.Dimensions {
	var children []layout.FlexChild

	// 状态消息条
	if d.msg != "" {
		msg := d.msg
		isErr := d.msgErr
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body2(th, msg)
			if isErr {
				lbl.Color = errColor
			} else {
				lbl.Color = okColor
			}
			return layout.UniformInset(unit.Dp(6)).Layout(gtx, lbl.Layout)
		}))
	}

	switch d.page {
	case pageDashboard:
		children = append(children, d.dashboardPage(th)...)
	case pageEntities:
		children = append(children, d.entitiesPage(th)...)
	case pageCollect:
		children = append(children, d.collectPage(th, titleColor)...)
	case pageMigrate:
		children = append(children, d.migratePage(th, titleColor)...)
	case pageSnapshot:
		children = append(children, d.snapshotPage(th, titleColor)...)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// dashboardPage 仪表盘。
func (d *desktopApp) dashboardPage(th *material.Theme) []layout.FlexChild {
	return []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H5(th, "概览").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(statCard(th, fmt.Sprintf("%d", d.stats.tools), "已接入工具")),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(statCard(th, fmt.Sprintf("%d", d.stats.entities), "已采集实体")),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(statCard(th, fmt.Sprintf("%d", d.stats.snapshots), "快照数")),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if d.repo != nil {
				return material.Body2(th, "仓库："+d.repo.Root).Layout(gtx)
			}
			return layout.Dimensions{}
		}),
	}
}

func statCard(th *material.Theme, num, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.H4(th, num)
				l.Alignment = 1
				return l.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Body2(th, label)
				l.Alignment = 1
				return l.Layout(gtx)
			}),
		)
	}
}

// entitiesPage 实体列表（可点行）+ 详情面板（按类型渲染）。
func (d *desktopApp) entitiesPage(th *material.Theme) []layout.FlexChild {
	cs := d.ts.Colors
	filtered := d.filterItems()
	var children []layout.FlexChild
	// 顶部：搜索框 + 类型过滤 chip
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return d.icons.Search.Layout(gtx, cs.TextSecondary)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceS)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return material.Editor(th, d.searchEd, "搜索 id / 备注 / 类型…").Layout(gtx)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceS)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return d.kindFilter.Layout(gtx, th, cs, kindOptions)
			}),
		)
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return material.H6(th, fmt.Sprintf("实体（%d/%d）", len(filtered), len(d.items))).Layout(gtx)
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
	// 详情面板（有选中时）
	if d.selID != "" {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return d.detailPanel(gtx, th, cs)
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
	}
	children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
		if len(d.items) == 0 {
			return material.Body1(th, "无数据——请到「采集」页采集").Layout(gtx)
		}
		return d.entityList.Layout(gtx, len(d.items), func(gtx layout.Context, i int) layout.Dimensions {
			it := d.items[i]
			if d.detailBtns[i] == nil {
				d.detailBtns[i] = new(widget.Clickable)
			}
			btn := d.detailBtns[i]
			if btn.Clicked(gtx) {
				d.selKind = it.kind
				d.selID = it.id
			}
			selected := d.selID == it.id
			return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				pointer.CursorPointer.Add(gtx.Ops)
				bg := color.NRGBA{}
				if selected {
					bg = cs.SurfaceHover
				} else if btn.Hovered() {
					bg = cs.SurfaceHover
				}
				return layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx layout.Context) layout.Dimensions {
						if selected || btn.Hovered() {
							r := gtx.Dp(unit.Dp(desktopui.RadiusM))
							rr := clip.RRect{Rect: image.Rectangle{Max: gtx.Constraints.Min}, NE: r, NW: r, SE: r, SW: r}
							paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
						}
						return layout.Dimensions{Size: gtx.Constraints.Min}
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(desktopui.SpaceM)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return desktopui.Badge(gtx, cs, it.kind, cs.Accent, func(gtx layout.Context) layout.Dimensions {
										lbl := material.Caption(th, it.kind)
										lbl.Color = cs.Surface
										return lbl.Layout(gtx)
									})
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceM)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									lbl := material.Body2(th, it.id)
									lbl.Color = cs.Text
									lbl.Font.Weight = font.Medium
									return lbl.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceS)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									lbl := material.Caption(th, it.note)
									lbl.Color = cs.TextSecondary
									return lbl.Layout(gtx)
								}),
							)
						})
					}),
				)
			})
		})
	}))
	return children
}

// detailPanel 详情面板（按类型渲染完整内容）。
func (d *desktopApp) detailPanel(gtx layout.Context, th *material.Theme, cs desktopui.Colors) layout.Dimensions {
	return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
		var rows []layout.FlexChild
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					lbl := material.H6(th, d.selID)
					lbl.Color = cs.Text
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.IconButton(th, d.closeDetail, d.icons.Close, "关闭")
					return btn.Layout(gtx)
				}),
			)
		}))
		rows = append(rows, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout))
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return d.detailContent(gtx, th, cs)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
	})
}

// collectPage 采集页。
func (d *desktopApp) collectPage(th *material.Theme, titleColor color.NRGBA) []layout.FlexChild {
	return []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H6(th, "采集配置").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Body2(th, "选择工具（全部留空=所有）：").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return d.collectTool.Layout(gtx, th, d.ts.Colors, toolOptions)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Button(th, d.collectBtn, "开始采集").Layout(gtx)
		}),
	}
}

// migratePage 迁移页。
func (d *desktopApp) migratePage(th *material.Theme, titleColor color.NRGBA) []layout.FlexChild {
	return []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H6(th, "迁移 / 导出").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body2(th, "从：").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return d.migrateFrom.Layout(gtx, th, d.ts.Colors, toolOptions)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body2(th, "到：").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return d.migrateTo.Layout(gtx, th, d.ts.Colors, toolOptions)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(material.CheckBox(th, d.migrateDry, "先预览（dry-run）").Layout),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(material.Button(th, d.migrateBtn, "开始迁移").Layout),
			)
		}),
	}
}

// snapshotPage 快照页。
func (d *desktopApp) snapshotPage(th *material.Theme, titleColor color.NRGBA) []layout.FlexChild {
	var children []layout.FlexChild
	children = append(children,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H6(th, "快照").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					ed := material.Editor(th, d.snapNote, "备注（可选）")
					return ed.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(material.Button(th, d.snapCreate, "创建快照").Layout),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
	)
	for _, s := range d.snapList {
		s := s
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return material.Body2(th, s.id+"  "+s.note+"  ("+fmt.Sprintf("%d", s.files)+" 文件)").Layout(gtx)
					}),
					layout.Rigid(material.Button(th, s.restore, "恢复").Layout),
				)
			})
		}))
	}
	return children
}

// ---- 操作（goroutine 异步，避免阻塞 UI） ----

func (d *desktopApp) doCollect() {
	defer func() {
		d.loading = false
		if d.win != nil {
			d.win.Invalidate()
		}
	}()
	if d.repo == nil {
		d.setMsg("仓库未就绪", true)
		return
	}
	scanner := secrets.DefaultScanner()
	backend, _ := secrets.ResolveBackend("", d.repo.Root, nil)
	totalNew := 0
	for _, a := range adapters.List() {
		tool := d.collectTool.Value
		if tool != "" && string(a.Meta().ID) != tool {
			continue
		}
		locs, err := a.Detect(context.Background())
		if err != nil {
			continue
		}
		for _, loc := range locs {
			b, err := a.Import(context.Background(), loc)
			if err != nil {
				continue
			}
			sanitizeBundleDesktop(b, scanner, backend)
			n, _, _, err := reconcileIntoDesktop(d.repo, loc.Scope, b)
			if err == nil {
				totalNew += n
			}
		}
	}
	d.setMsg(fmt.Sprintf("采集完成，新增 %d 条", totalNew), false)
	d.reload()
}

func (d *desktopApp) doMigrate() {
	defer func() {
		d.loading = false
		if d.win != nil {
			d.win.Invalidate()
		}
	}()
	if d.repo == nil {
		d.setMsg("仓库未就绪", true)
		return
	}
	from := d.migrateFrom.Value
	to := d.migrateTo.Value
	if from == to {
		d.setMsg("源与目标不能相同", true)
		return
	}
	// collect from
	scanner := secrets.DefaultScanner()
	backend, _ := secrets.ResolveBackend("", d.repo.Root, nil)
	for _, a := range adapters.List() {
		if string(a.Meta().ID) != from {
			continue
		}
		locs, _ := a.Detect(context.Background())
		for _, loc := range locs {
			b, err := a.Import(context.Background(), loc)
			if err != nil {
				continue
			}
			sanitizeBundleDesktop(b, scanner, backend)
			reconcileIntoDesktop(d.repo, loc.Scope, b)
		}
	}
	// export to
	e := &migrate.Engine{Repo: d.repo}
	res, err := e.Export(context.Background(), migrate.ExportRequest{
		To:             adapters.ToolID(to),
		DryRun:         d.migrateDry.Value,
		IncludeForeign: true,
	})
	if err != nil {
		d.setMsg("迁移失败: "+err.Error(), true)
		return
	}
	verb := "已写入"
	if d.migrateDry.Value {
		verb = "预览（未落盘）"
	}
	d.setMsg(fmt.Sprintf("迁移到 %s：%s %d 个文件", to, verb, len(res.Written)), false)
	d.reload()
}

func (d *desktopApp) doSnapshotCreate() {
	defer func() {
		d.loading = false
		if d.win != nil {
			d.win.Invalidate()
		}
	}()
	if d.repo == nil {
		d.setMsg("仓库未就绪", true)
		return
	}
	note := d.snapNote.Text()
	id, err := d.repo.CreateSnapshot(note)
	if err != nil {
		d.setMsg("快照失败: "+err.Error(), true)
		return
	}
	d.setMsg("已创建快照 "+id, false)
	d.reload()
}

func (d *desktopApp) doSnapshotRestore(id string) {
	defer func() {
		d.loading = false
		if d.win != nil {
			d.win.Invalidate()
		}
	}()
	if d.repo == nil {
		d.setMsg("仓库未就绪", true)
		return
	}
	if _, err := d.repo.CreateSnapshot("before-restore-" + id); err != nil {
		d.setMsg("反向快照失败: "+err.Error(), true)
		return
	}
	if err := d.repo.RestoreSnapshot(id); err != nil {
		d.setMsg("恢复失败: "+err.Error(), true)
		return
	}
	d.setMsg("已恢复快照 "+id, false)
	d.reload()
}

func (d *desktopApp) setMsg(s string, isErr bool) {
	d.msg = s
	d.msgErr = isErr
}

// 桌面端复用 cmd 的采集辅助（简化内联版）。
func sanitizeBundleDesktop(b *ir.Bundle, scanner *secrets.Scanner, backend secrets.Backend) {
	profileName := "global"
	if b.Scope == ir.ScopeProject {
		profileName = "project"
	}
	for i := range b.MCPServers {
		s := &b.MCPServers[i]
		s.Env, _, _ = secrets.SanitizeMap(backend, scanner, profileName, s.ID, "env", s.Env)
		s.Headers, _, _ = secrets.SanitizeMap(backend, scanner, profileName, s.ID, "headers", s.Headers)
	}
}

func reconcileIntoDesktop(repo *store.Repo, scope ir.Scope, b *ir.Bundle) (int, int, int, error) {
	dir := repo.Path(store.DirProfiles, "global")
	if scope != ir.ScopeGlobal {
		dir = repo.Path(store.DirProfiles, "projects", "default")
	}
	var existing *ir.Bundle
	var man *profile.Manifest
	if sb, err := profile.Load(dir, scope); err == nil {
		existing = sb.Bundle
		man = sb.Manifest
	} else {
		existing = &ir.Bundle{IRVersion: profile.CurrentIRVersion, Scope: scope}
		man = &profile.Manifest{IRVersion: profile.CurrentIRVersion, Profile: profile.Meta{Name: "global", Kind: "global"}}
	}
	merged, n, u, t := reconcileBundlesDesktop(existing, b)
	if err := profile.Save(dir, merged, man); err != nil {
		return 0, 0, 0, err
	}
	return n, u, t, nil
}

func reconcileBundlesDesktop(existing, fresh *ir.Bundle) (*ir.Bundle, int, int, int) {
	// 简化：按 id 覆盖更新（桌面端采集复用）
	result := *existing
	added := 0
	for _, f := range fresh.MCPServers {
		found := false
		for i := range result.MCPServers {
			if result.MCPServers[i].ID == f.ID {
				result.MCPServers[i] = f
				found = true
			}
		}
		if !found {
			result.MCPServers = append(result.MCPServers, f)
			added++
		}
	}
	for _, f := range fresh.Instructions {
		found := false
		for i := range result.Instructions {
			if result.Instructions[i].ID == f.ID {
				result.Instructions[i] = f
				found = true
			}
		}
		if !found {
			result.Instructions = append(result.Instructions, f)
			added++
		}
	}
	for _, f := range fresh.Skills {
		found := false
		for i := range result.Skills {
			if result.Skills[i].ID == f.ID {
				result.Skills[i] = f
				found = true
			}
		}
		if !found {
			result.Skills = append(result.Skills, f)
			added++
		}
	}
	return &result, added, 0, 0
}

// detailContent 按类型渲染选中实体的详情内容（F01）。
func (d *desktopApp) detailContent(gtx layout.Context, th *material.Theme, cs desktopui.Colors) layout.Dimensions {
	if d.bundle == nil {
		return material.Body2(th, "无数据").Layout(gtx)
	}
	var lines []string
	switch d.selKind {
	case "指令":
		for _, x := range d.bundle.Instructions {
			if x.ID == d.selID {
				lines = append(lines, "激活: "+string(x.Activation), "优先级: "+fmt.Sprintf("%d", x.Priority))
				if len(x.FilePatterns) > 0 {
					lines = append(lines, "文件作用域: "+strings.Join(x.FilePatterns, ", "))
				}
				if x.Origin != nil {
					lines = append(lines, "来源: "+x.Origin.Tool+" @ "+x.Origin.Path)
				}
				lines = append(lines, "", x.Body)
			}
		}
	case "MCP":
		for _, x := range d.bundle.MCPServers {
			if x.ID == d.selID {
				lines = append(lines, "名称: "+x.Name, "传输: "+x.Transport, "命令: "+x.Command, "URL: "+x.URL)
				if x.Disabled {
					lines = append(lines, "状态: 已禁用")
				}
				if len(x.Env) > 0 {
					lines = append(lines, "环境变量: "+fmt.Sprintf("%d 项（secretref 已遮罩）", len(x.Env)))
				}
			}
		}
	case "技能", "Agent":
		var packs []ir.PromptPack
		if d.selKind == "技能" {
			packs = d.bundle.Skills
		} else {
			packs = d.bundle.Agents
		}
		for _, x := range packs {
			if x.ID == d.selID {
				lines = append(lines, "名称: "+x.Name, "描述: "+x.Description)
				if len(x.Tools) > 0 {
					lines = append(lines, "工具: "+strings.Join(x.Tools, ", "))
				}
				lines = append(lines, "", x.Body)
			}
		}
	case "Hook":
		for _, x := range d.bundle.Hooks {
			if x.ID == d.selID {
				lines = append(lines, "事件: "+string(x.Event), "类型: "+x.Handler.Type, "命令: "+x.Handler.Command)
			}
		}
	case "设置":
		for _, x := range d.bundle.Settings {
			if x.ID == d.selID {
				lines = append(lines, "键: "+x.Key, "值: "+fmt.Sprintf("%v", x.Value))
			}
		}
	}
	var rows []layout.FlexChild
	for _, ln := range lines {
		ln := ln
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body2(th, ln)
			lbl.Color = cs.Text
			return layout.UniformInset(unit.Dp(2)).Layout(gtx, lbl.Layout)
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

// filterItems 按类型 chip + 搜索词过滤实体（F02）。
func (d *desktopApp) filterItems() []entityItem {
	var out []entityItem
	kindSel := d.kindFilter.Value
	q := strings.ToLower(strings.TrimSpace(d.searchEd.Text()))
	for _, it := range d.items {
		if kindSel != "" && kindSel != "全部" && it.kind != kindSel {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(it.id+" "+it.note+" "+it.kind), q) {
			continue
		}
		out = append(out, it)
	}
	return out
}
