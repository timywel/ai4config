// Command cfg4ai-desktop 是 cfg4ai 的原生桌面应用（Gio 即时模式 GUI，纯 Go 无 CGO）。
// 双击即用：全部功能（采集/迁移/快照/浏览）在窗口内点按钮完成，无需命令行。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
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
	pageSecret
	pageDrift
	pageHistory
	pageActivity
	pageDiscover
	pageGraph
	pageSync
)

var pageNames = []string{"仪表盘", "实体", "采集", "迁移", "快照", "密钥", "一致性", "历史", "活动", "发现", "关系", "同步"}

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

	ts         *desktopui.ThemeStore // 主题（A/B/D 三主题，D 默认）
	themeGroup *desktopui.ChipGroup  // 主题切换
	icons      desktopui.IconSet     // 图标集

	bundle      *ir.Bundle                // 完整实体数据（详情用）
	selKind     string                    // 选中实体类型
	selID       string                    // 选中实体 id
	detailBtns  map[int]*widget.Clickable // 实体行点击
	closeDetail *widget.Clickable         // 关闭详情

	newBtn      *widget.Clickable    // 新建入口
	newTemplate *desktopui.ChipGroup // 新建模板选择
	newType     *desktopui.ChipGroup // 新建类型
	newID       *widget.Editor       // 新建 id
	newCreate   *widget.Clickable    // 确认新建
	deleteBtn   *widget.Clickable    // 删除（墓碑）
	restoreBtn  *widget.Clickable    // 恢复（去墓碑）
	showRecycle bool                 // 回收站视图
	recycleBtn  *widget.Clickable    // 回收站切换

	multiSel     map[string]bool      // 多选集合（F11）
	checkboxes   map[int]*widget.Bool // 行复选框
	batchEnable  *widget.Clickable    // 批量启用
	batchDisable *widget.Clickable    // 批量禁用
	batchDelete  *widget.Clickable    // 批量删除

	syncRemote *widget.Editor    // 同步远端地址
	presetName *widget.Editor    // 团队预设名（F23）
	presetPush *widget.Clickable // 下发预设

	paletteOpen   bool              // 命令面板开（F17）
	paletteEditor *widget.Editor    // 面板搜索框
	paletteSel    int               // 选中项
	paletteList   *widget.List      // 结果列表
	syncPush      *widget.Clickable // 推送
	syncPull      *widget.Clickable // 拉取
	syncInit      *widget.Clickable // 初始化远端
	paletteBtn    *widget.Clickable // 命令面板唤出

	annotations *profile.Annotations // 治理元数据（labels/favorite，F18）
	favBtn      *widget.Clickable    // 收藏切换

	schedEnum  *widget.Enum      // 定时计划间隔（F19）
	schedBtn   *widget.Clickable // 保存定时计划
	schedTimer *time.Ticker      // 后台定时器

	permExec      int            // 能力旗标：可执行命令数（F20）
	permNet       int            // 能力旗标：网络访问数
	permEnv       int            // 能力旗标：环境变量数
	permRisk      int            // 明文风险数（F21）
	actionStats   map[string]int // 使用统计：按操作类型计数（F22）
	driftCache    []driftItem    // 漂移缓存（渲染不重复 Detect，防每帧 exec）
	coverageCache []coverageItem // 覆盖率缓存
	showNew       bool           // 新建表单显隐

	searchEd   *widget.Editor       // 搜索框（F02）
	kindFilter *desktopui.ChipGroup // 类型过滤 chip（F02）
	filtered   []entityItem         // 过滤后的实体

	editing    bool              // 编辑态
	editBody   *widget.Editor    // 正文编辑
	editBtn    *widget.Clickable // 进入编辑
	editSave   *widget.Clickable // 保存
	editCancel *widget.Clickable // 取消
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
	d.ts = desktopui.NewThemeStore(loadThemeMode()) // 持久化偏好，默认 D
	d.themeGroup = desktopui.NewChipGroup(desktopui.ModeNames[d.ts.Mode])
	d.icons = desktopui.MustIcons()
	d.detailBtns = map[int]*widget.Clickable{}
	d.closeDetail = new(widget.Clickable)
	d.newBtn = new(widget.Clickable)
	d.newType = desktopui.NewChipGroup("指令")
	d.newTemplate = desktopui.NewChipGroup("空白")
	d.newID = &widget.Editor{}
	d.newCreate = new(widget.Clickable)
	d.deleteBtn = new(widget.Clickable)
	d.restoreBtn = new(widget.Clickable)
	d.recycleBtn = new(widget.Clickable)
	d.syncRemote = &widget.Editor{}
	d.presetName = &widget.Editor{SingleLine: true}
	d.presetPush = new(widget.Clickable)
	d.syncPush = new(widget.Clickable)
	d.syncPull = new(widget.Clickable)
	d.syncInit = new(widget.Clickable)
	d.paletteBtn = new(widget.Clickable)
	d.favBtn = new(widget.Clickable)
	d.schedEnum = new(widget.Enum)
	d.schedEnum.Value = "off"
	d.schedBtn = new(widget.Clickable)
	d.paletteEditor = &widget.Editor{SingleLine: true}
	d.paletteList = &widget.List{List: layout.List{Axis: layout.Vertical}}
	d.multiSel = map[string]bool{}
	d.checkboxes = map[int]*widget.Bool{}
	d.batchEnable = new(widget.Clickable)
	d.batchDisable = new(widget.Clickable)
	d.batchDelete = new(widget.Clickable)
	d.searchEd = &widget.Editor{}
	d.kindFilter = desktopui.NewChipGroup("全部")
	d.editBody = &widget.Editor{}
	d.editBtn = new(widget.Clickable)
	d.editSave = new(widget.Clickable)
	d.editCancel = new(widget.Clickable)
	d.migrateDry = new(widget.Bool)
	d.migrateFrom.Value = "claude-code"
	d.migrateTo.Value = "codex"

	root, err := paths.DataHome()
	if err == nil {
		d.repo, _ = store.Open(root)
	}
	d.reload()
	return d
}

var version = "1.0.0-dev" // 由 ldflags 注入
func main() {
	d := newDesktopApp()
	go func() {
		w := new(app.Window)
		w.Option(app.Title("cfg4ai 配置治理 v"+version), app.Size(unit.Dp(960), unit.Dp(680)))
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
		d.driftCache = nil
		d.coverageCache = nil
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
	d.annotations, _ = profile.LoadAnnotations(d.repo.Path(store.DirProfiles, "global"))
	d.stats.tools = len(adapters.List())

	// 权限审计聚合（F20）：MCP/Hook 能力旗标
	d.permExec, d.permNet, d.permEnv = 0, 0, 0
	if d.bundle != nil {
		for _, m := range d.bundle.MCPServers {
			if m.Command != "" {
				d.permExec++
			}
			if m.URL != "" {
				d.permNet++
			}
			if len(m.Env) > 0 {
				d.permEnv++
			}
		}
		for _, h := range d.bundle.Hooks {
			if h.Handler.Command != "" {
				d.permExec++
			}
		}
	}
	// 使用统计聚合（F22）：审计日志按操作类型计数
	d.actionStats = map[string]int{}
	if entries, err := d.repo.ReadAudit(500); err == nil {
		for _, e := range entries {
			d.actionStats[e.Op]++
		}
	}
	d.stats.entities = len(d.items)
	if snaps, err := d.repo.ListSnapshots(); err == nil {
		d.stats.snapshots = len(snaps)
		d.snapList = nil
		for _, s := range snaps {
			d.snapList = append(d.snapList, snapItem{id: s.ID, note: s.Note, files: len(s.Files), restore: new(widget.Clickable)})
		}
	}
	// 重数据缓存：reload 时计算一次，渲染路径只读（防每帧 exec 弹窗）
	d.driftCache = d.loadDrift()
	d.coverageCache = d.loadCoverage()
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
			d.paintBackground(gtx, cs) // 主题背景（D=渐变光斑，A/B=纯色）

			// 命令面板键盘事件（Ctrl+K 唤出）

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

			// 命令面板唤出
			if d.paletteBtn.Clicked(gtx) {
				d.paletteOpen = true
				d.paletteEditor.SetText("")
				d.paletteSel = 0
				// 主题切换检测
				if label := d.themeGroup.Value; label != "" && label != desktopui.ModeNames[d.ts.Mode] {
					d.applyThemeByLabel(label)
				}
			}
			// 收藏切换
			if d.favBtn.Clicked(gtx) && d.selID != "" {
				d.doToggleFavorite()
			}
			// 定时计划保存
			if d.schedBtn.Clicked(gtx) {
				d.doSaveSchedule()
			}
			if d.closeDetail.Clicked(gtx) {
				d.selID = ""
				d.editing = false
			}
			// 编辑
			if d.editBtn.Clicked(gtx) && d.selID != "" && !d.editing {
				d.editing = true
				d.editBody.SetText(d.currentBody())
			}
			if d.editSave.Clicked(gtx) && d.editing && !d.loading {
				d.loading = true
				go d.doSaveEdit()
			}
			if d.editCancel.Clicked(gtx) && d.editing {
				d.editing = false
			}
			// 新建
			if d.newBtn.Clicked(gtx) {
				d.showNew = !d.showNew
			}
			if d.newCreate.Clicked(gtx) && d.showNew && !d.loading {
				d.loading = true
				go d.doNew()
			}
			// 删除（墓碑）/恢复
			if d.deleteBtn.Clicked(gtx) && d.selID != "" && !d.loading {
				d.loading = true
				go d.doDelete()
			}
			if d.restoreBtn.Clicked(gtx) && d.selID != "" && !d.loading {
				d.loading = true
				go d.doRestore()
			}
			// 回收站切换
			if d.recycleBtn.Clicked(gtx) {
				d.showRecycle = !d.showRecycle
			}
			// 同步
			if d.syncInit.Clicked(gtx) && !d.loading {
				d.loading = true
				go d.doSyncInit()
			}
			if d.syncPush.Clicked(gtx) && !d.loading {
				d.loading = true
				go d.doSyncPush()
			}
			if d.syncPull.Clicked(gtx) && !d.loading {
				d.loading = true
				go d.doSyncPull()
			}
			// 团队预设下发
			if d.presetPush.Clicked(gtx) && !d.loading {
				d.loading = true
				go d.doPresetPush()
			}
			// 批量操作
			if d.batchEnable.Clicked(gtx) && !d.loading {
				d.loading = true
				go d.doBatchToggle(false)
			}
			if d.batchDisable.Clicked(gtx) && !d.loading {
				d.loading = true
				go d.doBatchToggle(true)
			}
			if d.batchDelete.Clicked(gtx) && !d.loading {
				d.loading = true
				go d.doBatchDelete()
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
	navIcons := []*widget.Icon{
		d.icons.Dashboard,     // 仪表盘
		d.icons.List,          // 实体
		d.icons.Download,      // 采集
		d.icons.Sync,          // 迁移（SwapHoriz）
		d.icons.Snapshot,      // 快照
		d.icons.Key,           // 密钥
		d.icons.CompareArrows, // 一致性
		d.icons.History,       // 历史
		d.icons.Timeline,      // 活动
		d.icons.Explore,       // 发现
		d.icons.Hub,           // 关系
		d.icons.Renew,         // 同步
	}
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
		return d.navItem(gtx, th, cs, d.icons.Refresh, "刷新", false, d.refreshBtn)
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return d.navItem(gtx, th, cs, d.icons.Search, "命令面板", d.paletteOpen, d.paletteBtn)
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout))
	// 主题切换（A/B/D）
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		lbl := material.Caption(th, "主题")
		lbl.Color = cs.TextSecondary
		return lbl.Layout(gtx)
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return d.themeGroup.Layout(gtx, th, cs, []string{desktopui.ModeNames[desktopui.ModeDarkPro], desktopui.ModeNames[desktopui.ModeLightClean], desktopui.ModeNames[desktopui.ModeGlass]})
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
	case pageSecret:
		children = append(children, d.secretPage(th, titleColor)...)
	case pageDrift:
		children = append(children, d.driftPage(th, titleColor)...)
	case pageHistory:
		children = append(children, d.historyPage(th, titleColor)...)
	case pageActivity:
		children = append(children, d.activityPage(th, titleColor)...)
	case pageDiscover:
		children = append(children, d.discoverPage(th, titleColor)...)
	case pageGraph:
		children = append(children, d.graphPage(th, titleColor)...)
	case pageSync:
		children = append(children, d.syncPage(th, titleColor)...)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// dashboardPage 仪表盘。
// dashboardPage 健康看板（F14）：健康分大卡 + 统计卡 + 分类计数卡。
func (d *desktopApp) dashboardPage(th *material.Theme) []layout.FlexChild {
	cs := d.ts.Colors
	tombs, secretN, driftN, valErrs := 0, 0, 0, 0
	if d.bundle != nil {
		for _, x := range d.bundle.Instructions {
			if x.Tombstone {
				tombs++
			}
		}
		for _, x := range d.bundle.MCPServers {
			if x.Tombstone {
				tombs++
			}
		}
		for _, x := range d.bundle.Skills {
			if x.Tombstone {
				tombs++
			}
		}
		secretN = len(d.scanSecretRefs())
		for _, it := range d.driftCache {
			if it.status != "一致" {
				driftN++
			}
		}
		issues := ir.Validate(d.bundle, ir.ValidateOptions{CurrentIRVersion: profile.CurrentIRVersion})
		for _, is := range issues {
			if is.Level == ir.SeverityError {
				valErrs++
			}
		}
	}
	score := 100 - tombs*2 - driftN*3 - valErrs*5
	if score < 0 {
		score = 0
	}
	scoreColor := cs.Success
	if score < 60 {
		scoreColor = cs.Danger
	} else if score < 85 {
		scoreColor = cs.Accent
	}
	return []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H5(th, "健康看板").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.H3(th, fmt.Sprintf("%d", score))
						l.Color = scoreColor
						l.Alignment = 1
						return l.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Body2(th, "配置健康分（0-100）")
						l.Alignment = 1
						l.Color = cs.TextSecondary
						return l.Layout(gtx)
					}),
				)
			})
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(statCard(th, fmt.Sprintf("%d", d.stats.tools), "接入工具")),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(statCard(th, fmt.Sprintf("%d", d.stats.entities), "采集实体")),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(statCard(th, fmt.Sprintf("%d", d.stats.snapshots), "快照数")),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(healthCard(th, cs, fmt.Sprintf("%d", tombs), "墓碑条目", cs.Accent)),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(healthCard(th, cs, fmt.Sprintf("%d", secretN), "secretref", cs.Accent)),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(healthCard(th, cs, fmt.Sprintf("%d", driftN), "漂移项", cs.Danger)),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(healthCard(th, cs, fmt.Sprintf("%d", valErrs), "校验失败", cs.Danger)),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if d.repo != nil {
				l := material.Body2(th, "仓库："+d.repo.Root)
				l.Color = cs.TextSecondary
				return l.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Body1(th, "权限审计（F20）").Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Body2(th, fmt.Sprintf("可执行命令 %d · 网络访问 %d · 环境变量 %d", d.permExec, d.permNet, d.permEnv)).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Caption(th, fmt.Sprintf("密钥引用 %d · 校验失败 %d（明文风险详见密钥页/一致性页）", secretN, valErrs))
						l.Color = cs.TextSecondary
						return l.Layout(gtx)
					}),
				)
			})
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Body1(th, "使用统计（F22）").Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Body2(th, d.actionStatsText())
						l.Color = cs.TextSecondary
						return l.Layout(gtx)
					}),
				)
			})
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
	// 标题行：计数 + 新建 + 回收站切换
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return material.H6(th, fmt.Sprintf("实体（%d/%d）", len(filtered), len(d.items))).Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(th, d.newBtn, "新建")
				btn.Background = cs.Accent
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceS)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				label := "回收站"
				if d.showRecycle {
					label = "返回"
				}
				return material.Button(th, d.recycleBtn, label).Layout(gtx)
			}),
		)
	}))
	// 新建表单（showNew 时显示）
	if d.showNew {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Body2(th, "新建条目（选类型 + 模板 + id）：").Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return d.newType.Layout(gtx, th, cs, []string{"指令", "技能", "Agent"})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceS)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Caption(th, "从模板：").Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return d.newTemplate.Layout(gtx, th, cs, templateOptions())
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceS)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Editor(th, d.newID, "id 名称（如 coding-style）").Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceS)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, d.newCreate, "创建")
						btn.Background = cs.Accent
						return btn.Layout(gtx)
					}),
				)
			})
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceS)}.Layout))
	}
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
									if d.checkboxes[i] == nil {
										d.checkboxes[i] = &widget.Bool{}
									}
									cb := d.checkboxes[i]
									cb.Value = d.multiSel[it.id]
									if cb.Update(gtx) {
										if cb.Value {
											d.multiSel[it.id] = true
										} else {
											delete(d.multiSel, it.id)
										}
									}
									return material.CheckBox(th, cb, "").Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceS)}.Layout),
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
	// 批量动作条（有选中时显示）
	if len(d.multiSel) > 0 {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(th, fmt.Sprintf("已选 %d 项", len(d.multiSel)))
						lbl.Color = cs.Text
						return lbl.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceM)}.Layout),
					layout.Rigid(material.Button(th, d.batchEnable, "启用").Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceS)}.Layout),
					layout.Rigid(material.Button(th, d.batchDisable, "禁用").Layout),
					layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceS)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, d.batchDelete, "删除")
						btn.Background = cs.Danger
						return btn.Layout(gtx)
					}),
				)
			})
		}))
	}
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
					if d.editing {
						return layout.Dimensions{}
					}
					btn := material.IconButton(th, d.editBtn, d.icons.Edit, "编辑")
					return btn.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					star := d.icons.StarBorder
					if d.annotations != nil && slices.Contains(d.annotations.Favorite, d.selID) {
						star = d.icons.Star
					}
					btn := material.IconButton(th, d.favBtn, star, "收藏")
					return btn.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.IconButton(th, d.closeDetail, d.icons.Close, "关闭")
					return btn.Layout(gtx)
				}),
			)
		}))
		rows = append(rows, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout))
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if d.editing {
				return d.editArea(gtx, th, cs)
			}
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
	children = append(children,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(th, "定时快照（F19）").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.RadioButton(th, d.schedEnum, "off", "关闭").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.RadioButton(th, d.schedEnum, "hourly", "每小时").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.RadioButton(th, d.schedEnum, "daily", "每天").Layout(gtx)
		}),
		layout.Rigid(material.Button(th, d.schedBtn, "保存定时计划").Layout),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
	)
	for _, s := range d.snapList {
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
	d.repo.Audit("collect", "user", "global", fmt.Sprintf("新增 %d 条", totalNew), "ok", nil, 0)
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
	d.repo.Audit("export", "user", "global", fmt.Sprintf("迁移到 %s，%d 文件", to, len(res.Written)), "ok", nil, len(res.Warnings))
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
	d.repo.Audit("snapshot", "user", "global", "创建快照 "+id, "ok", nil, 0)
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
	d.repo.Audit("restore", "user", "global", "恢复快照 "+id, "ok", nil, 0)
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

// editArea 编辑态：正文编辑 + 保存/取消（F03）。
func (d *desktopApp) editArea(gtx layout.Context, th *material.Theme, cs desktopui.Colors) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body2(th, "编辑正文（"+d.selID+"）")
			lbl.Color = cs.TextSecondary
			return lbl.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceS)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return material.Editor(th, d.editBody, "正文内容...").Layout(gtx)
			})
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(th, d.editSave, "保存")
					btn.Background = cs.Accent
					return btn.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceM)}.Layout),
				layout.Rigid(material.Button(th, d.editCancel, "取消").Layout),
			)
		}),
	)
}

// currentBody 取选中实体的当前正文。
func (d *desktopApp) currentBody() string {
	if d.bundle == nil {
		return ""
	}
	switch d.selKind {
	case "指令":
		for _, x := range d.bundle.Instructions {
			if x.ID == d.selID {
				return x.Body
			}
		}
	case "技能":
		for _, x := range d.bundle.Skills {
			if x.ID == d.selID {
				return x.Body
			}
		}
	case "Agent":
		for _, x := range d.bundle.Agents {
			if x.ID == d.selID {
				return x.Body
			}
		}
	}
	return ""
}

// doSaveEdit 保存编辑（写回 profile + 校验 + 审计）。
func (d *desktopApp) doSaveEdit() {
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
	newBody := d.editBody.Text()
	if d.bundle == nil {
		d.setMsg("无数据", true)
		return
	}
	// 写回对应实体的 Body
	updated := false
	switch d.selKind {
	case "指令":
		for i := range d.bundle.Instructions {
			if d.bundle.Instructions[i].ID == d.selID {
				d.bundle.Instructions[i].Body = newBody
				updated = true
			}
		}
	case "技能":
		for i := range d.bundle.Skills {
			if d.bundle.Skills[i].ID == d.selID {
				d.bundle.Skills[i].Body = newBody
				updated = true
			}
		}
	case "Agent":
		for i := range d.bundle.Agents {
			if d.bundle.Agents[i].ID == d.selID {
				d.bundle.Agents[i].Body = newBody
				updated = true
			}
		}
	}
	if !updated {
		d.setMsg("未找到实体", true)
		return
	}
	// 校验（IR-SCHEMA 12 条）
	issues := ir.Validate(d.bundle, ir.ValidateOptions{CurrentIRVersion: profile.CurrentIRVersion})
	for _, is := range issues {
		if is.Level == ir.SeverityError {
			d.setMsg("校验失败: "+is.Message, true)
			return
		}
	}
	// 写回 profile（global）
	m := &profile.Manifest{IRVersion: profile.CurrentIRVersion, Profile: profile.Meta{Name: "global", Kind: "global"}}
	if err := profile.Save(d.repo.Path(store.DirProfiles, "global"), d.bundle, m); err != nil {
		d.setMsg("写回失败: "+err.Error(), true)
		return
	}
	d.editing = false
	d.setMsg("已保存 "+d.selID, false)
	d.reload()
}

// ---- OPT-B2：新建/删除（墓碑）/恢复 操作 ----

func (d *desktopApp) doNew() {
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
	kind := d.newType.Value
	idText := strings.TrimSpace(d.newID.Text())
	if idText == "" {
		d.setMsg("请输入 id", true)
		return
	}
	typePrefix := map[string]string{"指令": "instruction.", "技能": "skill.", "Agent": "agent."}
	prefix := typePrefix[kind]
	if prefix == "" {
		prefix = "instruction."
	}
	// 模板正文（选模板时用模板内容）
	bodyText := "（新建，请编辑正文）\n"
	if tpl := templateByTitle(d.newTemplate.Value); tpl != nil {
		bodyText = tpl.Body
		if tpl.Kind != "" && tpl.Kind != kind {
			kind = tpl.Kind
			typePrefix := map[string]string{"指令": "instruction.", "技能": "skill.", "Agent": "agent."}
			if p2 := typePrefix[kind]; p2 != "" {
				prefix = p2
			}
		}
	}
	fullID := prefix + idText
	if _, _, err := ir.ParseID(fullID); err != nil {
		d.setMsg("id 非法: "+err.Error(), true)
		return
	}
	if d.bundle == nil {
		d.bundle = &ir.Bundle{IRVersion: profile.CurrentIRVersion, Scope: ir.ScopeGlobal}
	}
	if _, ok := d.bundle.Lookup(fullID); ok {
		d.setMsg("id 已存在: "+fullID, true)
		return
	}
	switch prefix {
	case "instruction.":
		d.bundle.Instructions = append(d.bundle.Instructions, ir.Instruction{Header: ir.Header{ID: fullID, IRVersion: profile.CurrentIRVersion}, Activation: ir.ActivationAlways, Body: bodyText})
	case "skill.":
		d.bundle.Skills = append(d.bundle.Skills, ir.PromptPack{Header: ir.Header{ID: fullID, IRVersion: profile.CurrentIRVersion}, Kind: ir.KindSkill, Name: idText, Body: bodyText})
	case "agent.":
		d.bundle.Agents = append(d.bundle.Agents, ir.PromptPack{Header: ir.Header{ID: fullID, IRVersion: profile.CurrentIRVersion}, Kind: ir.KindAgent, Name: idText, Body: bodyText})
	}
	m := &profile.Manifest{IRVersion: profile.CurrentIRVersion, Profile: profile.Meta{Name: "global", Kind: "global"}}
	if err := profile.Save(d.repo.Path(store.DirProfiles, "global"), d.bundle, m); err != nil {
		d.setMsg("保存失败: "+err.Error(), true)
		return
	}
	d.newID.SetText("")
	d.showNew = false
	d.setMsg("已创建 "+fullID, false)
	d.reload()
}

func (d *desktopApp) doDelete() {
	defer func() {
		d.loading = false
		if d.win != nil {
			d.win.Invalidate()
		}
	}()
	if d.repo == nil || d.bundle == nil || d.selID == "" {
		return
	}
	markTombstoneB2(d.bundle, d.selID)
	m := &profile.Manifest{IRVersion: profile.CurrentIRVersion, Profile: profile.Meta{Name: "global", Kind: "global"}}
	if err := profile.Save(d.repo.Path(store.DirProfiles, "global"), d.bundle, m); err != nil {
		d.setMsg("删除失败: "+err.Error(), true)
		return
	}
	d.setMsg("已删除（回收站可恢复）: "+d.selID, false)
	d.selID = ""
	d.reload()
}

func (d *desktopApp) doRestore() {
	defer func() {
		d.loading = false
		if d.win != nil {
			d.win.Invalidate()
		}
	}()
	if d.repo == nil || d.bundle == nil || d.selID == "" {
		return
	}
	clearTombstoneB2(d.bundle, d.selID)
	m := &profile.Manifest{IRVersion: profile.CurrentIRVersion, Profile: profile.Meta{Name: "global", Kind: "global"}}
	if err := profile.Save(d.repo.Path(store.DirProfiles, "global"), d.bundle, m); err != nil {
		d.setMsg("恢复失败: "+err.Error(), true)
		return
	}
	d.setMsg("已恢复: "+d.selID, false)
	d.reload()
}

func markTombstoneB2(b *ir.Bundle, id string) {
	for i := range b.Instructions {
		if b.Instructions[i].ID == id {
			b.Instructions[i].Tombstone = true
		}
	}
	for i := range b.Skills {
		if b.Skills[i].ID == id {
			b.Skills[i].Tombstone = true
		}
	}
	for i := range b.Agents {
		if b.Agents[i].ID == id {
			b.Agents[i].Tombstone = true
		}
	}
	for i := range b.MCPServers {
		if b.MCPServers[i].ID == id {
			b.MCPServers[i].Tombstone = true
		}
	}
	for i := range b.Hooks {
		if b.Hooks[i].ID == id {
			b.Hooks[i].Tombstone = true
		}
	}
	for i := range b.Settings {
		if b.Settings[i].ID == id {
			b.Settings[i].Tombstone = true
		}
	}
}

func clearTombstoneB2(b *ir.Bundle, id string) {
	for i := range b.Instructions {
		if b.Instructions[i].ID == id {
			b.Instructions[i].Tombstone = false
		}
	}
	for i := range b.Skills {
		if b.Skills[i].ID == id {
			b.Skills[i].Tombstone = false
		}
	}
	for i := range b.Agents {
		if b.Agents[i].ID == id {
			b.Agents[i].Tombstone = false
		}
	}
	for i := range b.MCPServers {
		if b.MCPServers[i].ID == id {
			b.MCPServers[i].Tombstone = false
		}
	}
	for i := range b.Hooks {
		if b.Hooks[i].ID == id {
			b.Hooks[i].Tombstone = false
		}
	}
	for i := range b.Settings {
		if b.Settings[i].ID == id {
			b.Settings[i].Tombstone = false
		}
	}
}

// ---- OPT-B4：密钥页（F08 secret 管理界面） ----

// secretItem 一个 secretref 条目。
type secretItem struct {
	ref     string
	entity  string
	backend string
}

// scanSecretRefs 扫 bundle 找全部 secretref（MCP env/headers + settings 值）。
func (d *desktopApp) scanSecretRefs() []secretItem {
	var out []secretItem
	if d.bundle == nil {
		return out
	}
	scan := func(entity, field, v string) {
		if secrets.IsSecretRef(v) {
			out = append(out, secretItem{ref: v, entity: entity, backend: backendNameFor(d)})
		}
	}
	for _, m := range d.bundle.MCPServers {
		for k, v := range m.Env {
			scan(m.ID, "env."+k, v)
		}
		for k, v := range m.Headers {
			scan(m.ID, "headers."+k, v)
		}
	}
	for _, s := range d.bundle.Settings {
		if v, ok := s.Value.(string); ok {
			scan(s.ID, s.Key, v)
		}
	}
	return out
}

// backendNameFor 当前 secret 后端名（徽标显示）。
func backendNameFor(d *desktopApp) string {
	if d.repo == nil {
		return "none"
	}
	b, err := secrets.ResolveBackend("", d.repo.Root, nil)
	if err != nil {
		return "none"
	}
	return string(b.Type())
}

// secretPage 密钥页：secretref 清单（F08）。
func (d *desktopApp) secretPage(th *material.Theme, titleColor color.NRGBA) []layout.FlexChild {
	cs := d.ts.Colors
	items := d.scanSecretRefs()
	var children []layout.FlexChild
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return material.H6(th, "密钥（secretref 清单）").Layout(gtx)
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body2(th, "当前后端："+backendNameFor(d)+"；secret 明文永不回显，设值经后端写入")
		lbl.Color = cs.TextSecondary
		return layout.UniformInset(unit.Dp(desktopui.SpaceS)).Layout(gtx, lbl.Layout)
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout))
	if len(items) == 0 {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body1(th, "无 secretref（采集时结构化字段的敏感值会自动抽取为占位符）")
			lbl.Color = cs.TextSecondary
			return lbl.Layout(gtx)
		}))
	}
	for _, it := range items {
		it := it
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return d.icons.Key.Layout(gtx, cs.Accent)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceM)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body2(th, it.ref)
								lbl.Color = cs.Text
								lbl.Font.Weight = font.Medium
								return lbl.Layout(gtx)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Caption(th, "所属："+it.entity+" ｜ 后端："+it.backend)
						lbl.Color = cs.TextSecondary
						return lbl.Layout(gtx)
					}),
				)
			})
		}))
	}
	return children
}

// ---- OPT-C1：一致性页（漂移检测与冲突处置，F06） ----

type driftItem struct {
	tool   string
	name   string
	status string // 一致 / 仅SSOT / 仅磁盘 / 双方改
	detail string
}

// loadDrift 检测各工具 SSOT vs 磁盘漂移（MCP 对比）。
func (d *desktopApp) loadDrift() []driftItem {
	var out []driftItem
	if d.repo == nil {
		return out
	}
	sb, err := profile.Load(d.repo.Path(store.DirProfiles, "global"), ir.ScopeGlobal)
	if err != nil {
		return out
	}
	for _, a := range adapters.List() {
		tool := string(a.Meta().ID)
		// SSOT 中该工具的 MCP
		ssot := map[string]ir.MCPServer{}
		for _, m := range sb.Bundle.MCPServers {
			if m.Origin != nil && m.Origin.Tool == tool {
				ssot[m.Name] = m
			}
		}
		// 磁盘现状
		disk := map[string]ir.MCPServer{}
		locs, _ := a.Detect(context.Background())
		for _, loc := range locs {
			b, err := a.Import(context.Background(), loc)
			if err != nil {
				continue
			}
			for _, m := range b.MCPServers {
				disk[m.Name] = m
			}
		}
		// 对比
		for name, s := range ssot {
			dsk, ok := disk[name]
			if !ok {
				out = append(out, driftItem{tool, name, "仅SSOT", "磁盘无此 server"})
				continue
			}
			if s.Command != dsk.Command || s.URL != dsk.URL {
				out = append(out, driftItem{tool, name, "双方改", "SSOT 与磁盘字段不一致"})
			} else {
				out = append(out, driftItem{tool, name, "一致", ""})
			}
		}
		for name := range disk {
			if _, ok := ssot[name]; !ok {
				out = append(out, driftItem{tool, name, "仅磁盘", "SSOT 未采集"})
			}
		}
	}
	return out
}

// driftPage 一致性页（F06）。
func (d *desktopApp) driftPage(th *material.Theme, titleColor color.NRGBA) []layout.FlexChild {
	cs := d.ts.Colors
	items := d.driftCache
	var children []layout.FlexChild
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return material.H6(th, "一致性（SSOT vs 磁盘）").Layout(gtx)
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body2(th, "检测各工具配置与 SSOT 的漂移；漂移项可处置")
		lbl.Color = cs.TextSecondary
		return layout.UniformInset(unit.Dp(desktopui.SpaceS)).Layout(gtx, lbl.Layout)
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout))

	driftCount := 0
	for _, it := range items {
		if it.status != "一致" {
			driftCount++
		}
	}
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body2(th, fmt.Sprintf("共 %d 项配置，%d 项漂移", len(items), driftCount))
		lbl.Color = cs.Text
		lbl.Font.Weight = font.Medium
		return lbl.Layout(gtx)
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceS)}.Layout))

	for _, it := range items {
		it := it
		if it.status == "一致" {
			continue // 只显示漂移项
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						var badgeColor color.NRGBA
						switch it.status {
						case "仅SSOT":
							badgeColor = cs.Accent
						case "仅磁盘":
							badgeColor = cs.Success
						default:
							badgeColor = cs.Danger
						}
						return desktopui.Badge(gtx, cs, it.status, badgeColor, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Caption(th, it.status)
							lbl.Color = cs.Surface
							return lbl.Layout(gtx)
						})
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceM)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body2(th, it.tool+" / "+it.name)
								lbl.Color = cs.Text
								lbl.Font.Weight = font.Medium
								return lbl.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Caption(th, it.detail)
								lbl.Color = cs.TextSecondary
								return lbl.Layout(gtx)
							}),
						)
					}),
				)
			})
		}))
	}
	return children
}

// ---- OPT-C2：历史页（F07 历史时间线） ----

// historyPage 历史时间线：快照节点 + 选中实体的版本链。
func (d *desktopApp) historyPage(th *material.Theme, titleColor color.NRGBA) []layout.FlexChild {
	cs := d.ts.Colors
	var children []layout.FlexChild
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return material.H6(th, "历史时间线").Layout(gtx)
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body2(th, "快照为整库时间点；恢复会反向快照现状")
		lbl.Color = cs.TextSecondary
		return layout.UniformInset(unit.Dp(desktopui.SpaceS)).Layout(gtx, lbl.Layout)
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout))

	if d.repo == nil {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(th, "仓库未就绪").Layout(gtx)
		}))
		return children
	}
	snaps, err := d.repo.ListSnapshots()
	if err != nil || len(snaps) == 0 {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body1(th, "无快照——到「快照」页创建")
			lbl.Color = cs.TextSecondary
			return lbl.Layout(gtx)
		}))
		return children
	}
	// 时间轴（倒序：最新在前）
	for i := len(snaps) - 1; i >= 0; i-- {
		s := snaps[i]
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return d.icons.Snapshot.Layout(gtx, cs.Accent)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceM)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body2(th, s.ID+"  "+s.Note)
								lbl.Color = cs.Text
								lbl.Font.Weight = font.Medium
								return lbl.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Caption(th, s.CreatedAt.Format("2006-01-02 15:04:05")+fmt.Sprintf(" ｜ %d 文件", len(s.Files)))
								lbl.Color = cs.TextSecondary
								return lbl.Layout(gtx)
							}),
						)
					}),
				)
			})
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceS)}.Layout))
	}
	return children
}

// healthCard 健康分类计数卡。
func healthCard(th *material.Theme, cs desktopui.Colors, num, label string, accent color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.H4(th, num)
					l.Color = accent
					l.Alignment = 1
					return l.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Caption(th, label)
					l.Color = cs.TextSecondary
					l.Alignment = 1
					return l.Layout(gtx)
				}),
			)
		})
	}
}

// ---- OPT-C4：活动页（F15 审计日志时间线） ----

// activityPage 活动时间线（logs/audit.jsonl，倒序最新在前）。
func (d *desktopApp) activityPage(th *material.Theme, titleColor color.NRGBA) []layout.FlexChild {
	cs := d.ts.Colors
	var children []layout.FlexChild
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return material.H6(th, "活动时间线").Layout(gtx)
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body2(th, "操作审计（collect/export/edit/delete/restore/sync/ai 决策）")
		lbl.Color = cs.TextSecondary
		return layout.UniformInset(unit.Dp(desktopui.SpaceS)).Layout(gtx, lbl.Layout)
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout))
	if d.repo == nil {
		return children
	}
	entries, err := d.repo.ReadAudit(50)
	if err != nil || len(entries) == 0 {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body1(th, "暂无操作记录")
			lbl.Color = cs.TextSecondary
			return lbl.Layout(gtx)
		}))
		return children
	}
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return desktopui.Badge(gtx, cs, e.Op, cs.Accent, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Caption(th, e.Op)
							lbl.Color = cs.Surface
							return lbl.Layout(gtx)
						})
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceM)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body2(th, e.Detail)
								lbl.Color = cs.Text
								return lbl.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Caption(th, e.Ts.Format("01-02 15:04:05")+" ｜ "+e.Actor+" ｜ "+e.Result)
								lbl.Color = cs.TextSecondary
								return lbl.Layout(gtx)
							}),
						)
					}),
				)
			})
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceS)}.Layout))
	}
	return children
}

// ---- OPT-D1：发现页（覆盖率视图，F09） ----

// coverageItem 一个工具的覆盖率。
type coverageItem struct {
	tool      string
	disk      int // 磁盘条目数
	managed   int // 已纳管条目数
	unmanaged []string
}

// loadCoverage 计算各工具覆盖率（磁盘 vs 已纳管）。
func (d *desktopApp) loadCoverage() []coverageItem {
	var out []coverageItem
	if d.repo == nil {
		return out
	}
	sb, err := profile.Load(d.repo.Path(store.DirProfiles, "global"), ir.ScopeGlobal)
	if err != nil {
		return out
	}
	for _, a := range adapters.List() {
		tool := string(a.Meta().ID)
		// 磁盘条目（重新 Import）
		disk := map[string]bool{}
		locs, _ := a.Detect(context.Background())
		for _, loc := range locs {
			b, err := a.Import(context.Background(), loc)
			if err != nil {
				continue
			}
			for _, m := range b.MCPServers {
				disk[m.ID] = true
			}
			for _, s := range b.Skills {
				disk[s.ID] = true
			}
			for _, i2 := range b.Instructions {
				disk[i2.ID] = true
			}
		}
		// 已纳管（origin.tool == tool）
		managed := map[string]bool{}
		for _, m := range sb.Bundle.MCPServers {
			if m.Origin != nil && m.Origin.Tool == tool {
				managed[m.ID] = true
			}
		}
		for _, s := range sb.Bundle.Skills {
			if s.Origin != nil && s.Origin.Tool == tool {
				managed[s.ID] = true
			}
		}
		for _, i2 := range sb.Bundle.Instructions {
			if i2.Origin != nil && i2.Origin.Tool == tool {
				managed[i2.ID] = true
			}
		}
		var unm []string
		for id := range disk {
			if !managed[id] {
				unm = append(unm, id)
			}
		}
		if len(disk) > 0 || len(managed) > 0 {
			out = append(out, coverageItem{tool: tool, disk: len(disk), managed: len(managed), unmanaged: unm})
		}
	}
	return out
}

// discoverPage 发现页：每工具覆盖率 + 未纳管条目。
func (d *desktopApp) discoverPage(th *material.Theme, titleColor color.NRGBA) []layout.FlexChild {
	cs := d.ts.Colors
	var children []layout.FlexChild
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return material.H6(th, "发现（磁盘 vs 已纳管覆盖率）").Layout(gtx)
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body2(th, "检测各工具磁盘配置中尚未纳入 SSOT 管理的条目")
		lbl.Color = cs.TextSecondary
		return layout.UniformInset(unit.Dp(desktopui.SpaceS)).Layout(gtx, lbl.Layout)
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout))

	items := d.coverageCache
	if len(items) == 0 {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body1(th, "未发现工具配置（先采集）")
			lbl.Color = cs.TextSecondary
			return lbl.Layout(gtx)
		}))
		return children
	}
	for _, it := range items {
		it := it
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			rate := 0
			if it.disk > 0 {
				rate = it.managed * 100 / it.disk
			}
			rateColor := cs.Success
			if rate < 50 {
				rateColor = cs.Danger
			} else if rate < 90 {
				rateColor = cs.Accent
			}
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body2(th, it.tool)
								lbl.Color = cs.Text
								lbl.Font.Weight = font.Medium
								return lbl.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body2(th, fmt.Sprintf("%d/%d 已纳管（%d%%）", it.managed, it.disk, rate))
								lbl.Color = rateColor
								return lbl.Layout(gtx)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if len(it.unmanaged) == 0 {
							return layout.Dimensions{}
						}
						lbl := material.Caption(th, "未纳管："+strings.Join(it.unmanaged, "、"))
						lbl.Color = cs.TextSecondary
						return layout.UniformInset(unit.Dp(desktopui.SpaceS)).Layout(gtx, lbl.Layout)
					}),
				)
			})
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceS)}.Layout))
	}
	return children
}

// ---- OPT-D2：关系页（依赖关系 + 断链检测，F10） ----

// graphEdge 一条依赖边。
type graphEdge struct {
	from   string // 引用方 id
	to     string // 被引用 id
	kind   string // references（skill→mcp）/ imports（instruction→文件）
	broken bool   // 断链（被引用 id 不存在）
}

// loadGraph 构建依赖边（skill/agent 的 mcp_servers 引用 + instruction 的 imports）。
func (d *desktopApp) loadGraph() []graphEdge {
	var out []graphEdge
	if d.bundle == nil {
		return out
	}
	// 已知的 mcp id 集合
	mcpIDs := map[string]bool{}
	for _, m := range d.bundle.MCPServers {
		mcpIDs[m.ID] = true
		mcpIDs["mcp."+m.Name] = true
	}
	// skill/agent 的 mcp_servers 引用
	check := func(fromID string, refs []string) {
		for _, r := range refs {
			broken := !mcpIDs[r] && !mcpIDs["mcp."+r]
			out = append(out, graphEdge{from: fromID, to: r, kind: "references", broken: broken})
		}
	}
	for _, s := range d.bundle.Skills {
		check(s.ID, s.MCPServers)
	}
	for _, a := range d.bundle.Agents {
		check(a.ID, a.MCPServers)
	}
	// instruction 的 imports（引用文件路径）
	for _, inst := range d.bundle.Instructions {
		for _, imp := range inst.Imports {
			out = append(out, graphEdge{from: inst.ID, to: imp.Path, kind: "imports", broken: !imp.Resolved})
		}
	}
	return out
}

// graphPage 关系页（F10）。
func (d *desktopApp) graphPage(th *material.Theme, titleColor color.NRGBA) []layout.FlexChild {
	cs := d.ts.Colors
	var children []layout.FlexChild
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return material.H6(th, "关系（依赖边 + 断链检测）").Layout(gtx)
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body2(th, "skill/agent 引用的 MCP、instruction 的 @import 链；断链红显")
		lbl.Color = cs.TextSecondary
		return layout.UniformInset(unit.Dp(desktopui.SpaceS)).Layout(gtx, lbl.Layout)
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout))

	edges := d.loadGraph()
	if len(edges) == 0 {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body1(th, "无依赖边（skill 未引用 MCP、instruction 无 @import）")
			lbl.Color = cs.TextSecondary
			return lbl.Layout(gtx)
		}))
		return children
	}
	broken := 0
	for _, e := range edges {
		if e.broken {
			broken++
		}
	}
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body2(th, fmt.Sprintf("共 %d 条依赖边，%d 条断链", len(edges), broken))
		lbl.Color = cs.Text
		lbl.Font.Weight = font.Medium
		return lbl.Layout(gtx)
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceS)}.Layout))
	for _, e := range edges {
		e := e
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			edgeColor := cs.TextSecondary
			if e.broken {
				edgeColor = cs.Danger
			}
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return desktopui.Badge(gtx, cs, e.kind, cs.Accent, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Caption(th, e.kind)
							lbl.Color = cs.Surface
							return lbl.Layout(gtx)
						})
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceM)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(th, e.from+"  →  "+e.to)
						lbl.Color = edgeColor
						if e.broken {
							lbl.Font.Weight = font.Bold
						}
						return lbl.Layout(gtx)
					}),
				)
			})
		}))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceS)}.Layout))
	}
	return children
}

// ---- OPT-D3：批量操作（F11） ----

// doBatchToggle 批量启用/禁用（annotations ToggleDisabled）。
func (d *desktopApp) doBatchToggle(disable bool) {
	defer func() {
		d.loading = false
		if d.win != nil {
			d.win.Invalidate()
		}
	}()
	if d.repo == nil || len(d.multiSel) == 0 {
		return
	}
	dir := d.repo.Path(store.DirProfiles, "global")
	ann, err := profile.LoadAnnotations(dir)
	if err != nil {
		ann = &profile.Annotations{}
	}
	for id := range d.multiSel {
		if disable {
			if !ann.IsDisabled(id) {
				ann.ToggleDisabled(id)
			}
		} else {
			if ann.IsDisabled(id) {
				ann.ToggleDisabled(id)
			}
		}
	}
	if err := profile.SaveAnnotations(dir, ann); err != nil {
		d.setMsg("批量操作失败: "+err.Error(), true)
		return
	}
	verb := "启用"
	if disable {
		verb = "禁用"
	}
	d.setMsg(fmt.Sprintf("已批量%s %d 条", verb, len(d.multiSel)), false)
	d.multiSel = map[string]bool{}
	d.checkboxes = map[int]*widget.Bool{}
	d.reload()
}

// doBatchDelete 批量删除（墓碑）。
func (d *desktopApp) doBatchDelete() {
	defer func() {
		d.loading = false
		if d.win != nil {
			d.win.Invalidate()
		}
	}()
	if d.repo == nil || d.bundle == nil || len(d.multiSel) == 0 {
		return
	}
	for id := range d.multiSel {
		markTombstoneB2(d.bundle, id)
	}
	m := &profile.Manifest{IRVersion: profile.CurrentIRVersion, Profile: profile.Meta{Name: "global", Kind: "global"}}
	if err := profile.Save(d.repo.Path(store.DirProfiles, "global"), d.bundle, m); err != nil {
		d.setMsg("批量删除失败: "+err.Error(), true)
		return
	}
	d.setMsg(fmt.Sprintf("已批量删除 %d 条（回收站可恢复）", len(d.multiSel)), false)
	d.multiSel = map[string]bool{}
	d.checkboxes = map[int]*widget.Bool{}
	d.reload()
}

// ---- OPT-D5：模板库（F13） ----

// builtinTemplate 内置模板。
type builtinTemplate struct {
	ID    string
	Kind  string // 指令/技能/Agent
	Title string
	Body  string
}

// builtinTemplates 常用模板（OPTIMIZATION-PLAN F13）。
var builtinTemplates = []builtinTemplate{
	{ID: "tpl-code-review", Kind: "技能", Title: "代码评审 skill", Body: "# 代码评审\n\n对变更做本地评审：\n1. 正确性\n2. 边界与错误处理\n3. 性能\n4. 安全\n\n输出按严重度分级的问题清单。\n"},
	{ID: "tpl-commit", Kind: "指令", Title: "提交规范指令", Body: "# 提交规范\n\n- 提交信息用中文，格式：type(scope): 主题\n- type：feat/fix/docs/refactor/test/chore\n- 提交前必须跑通测试\n"},
	{ID: "tpl-safe-redline", Kind: "指令", Title: "安全红线指令", Body: "# 安全红线\n\n- 绝不把 secret/token/密钥写入代码或日志\n- 危险操作（删除/覆盖/推送）必须二次确认\n- 不执行来源不明的脚本\n"},
	{ID: "tpl-mcp-fs", Kind: "技能", Title: "MCP filesystem 配方", Body: "# filesystem MCP 配方\n\ncommand: npx -y @modelcontextprotocol/server-filesystem <目录>\n注意目录权限最小化。\n"},
}

// templateOptions 新建表单的模板 chip 选项。
func templateOptions() []string {
	out := []string{"空白"}
	for _, t := range builtinTemplates {
		out = append(out, t.Title)
	}
	return out
}

// templateByTitle 按标题找模板。
func templateByTitle(title string) *builtinTemplate {
	for i := range builtinTemplates {
		if builtinTemplates[i].Title == title {
			return &builtinTemplates[i]
		}
	}
	return nil
}

// ---- OPT-D6：同步页（F16 sync GUI） ----

// syncPage 同步页：远端配置 + push/pull + 状态。
func (d *desktopApp) syncPage(th *material.Theme, titleColor color.NRGBA) []layout.FlexChild {
	cs := d.ts.Colors
	var children []layout.FlexChild
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return material.H6(th, "同步（跨机）").Layout(gtx)
	}))
	// 状态卡
	if d.repo != nil {
		status, _ := d.repo.SyncStatus()
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body2(th, "状态："+status)
				lbl.Color = cs.Text
				return lbl.Layout(gtx)
			})
		}))
	}
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout))
	// 远端配置
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return material.Editor(th, d.syncRemote, "远端地址（git remote URL）").Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceS)}.Layout),
			layout.Rigid(material.Button(th, d.syncInit, "初始化").Layout),
		)
	}))
	children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout))
	// push/pull
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(th, d.syncPush, "推送")
				btn.Background = cs.Accent
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceS)}.Layout),
			layout.Rigid(material.Button(th, d.syncPull, "拉取").Layout),
		)
	}))
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout))
		// 团队下发台（F23）
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Body1(th, "团队预设下发（F23）").Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return material.Editor(th, d.presetName, "预设名（profile 名，下发到团队远端）").Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceS)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								btn := material.Button(th, d.presetPush, "下发")
								btn.Background = cs.Success
								return btn.Layout(gtx)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Caption(th, "下发 = 提交当前 profile 并 push 到团队远端；成员侧 pull 即得预设")
						l.Color = cs.TextSecondary
						return l.Layout(gtx)
					}),
				)
			})
		}))
		lbl := material.Caption(th, "push 前全仓敏感扫描（preflight），命中即阻断；白名单制：仅 profiles/registry/config/exports 入库")
		lbl.Color = cs.TextSecondary
		return layout.UniformInset(unit.Dp(desktopui.SpaceS)).Layout(gtx, lbl.Layout)
	}))
	return children
}

// doSyncInit 初始化远端。
func (d *desktopApp) doSyncInit() {
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
	url := strings.TrimSpace(d.syncRemote.Text())
	if url == "" {
		d.setMsg("请输入远端地址", true)
		return
	}
	if err := d.repo.SyncInit(url); err != nil {
		d.setMsg("初始化失败: "+err.Error(), true)
		return
	}
	d.setMsg("已初始化远端: "+url, false)
}

// doSyncPush 推送（preflight 扫描）。
func (d *desktopApp) doSyncPush() {
	defer func() {
		d.loading = false
		if d.win != nil {
			d.win.Invalidate()
		}
	}()
	if d.repo == nil {
		return
	}
	err := d.repo.SyncPush(secrets.DefaultScanner(), func(matches []secrets.ScanMatch) bool {
		d.setMsg(fmt.Sprintf("preflight 命中 %d 处疑似敏感内容，已阻断", len(matches)), true)
		return false
	})
	if err != nil {
		d.setMsg("推送失败: "+err.Error(), true)
		return
	}
	d.setMsg("已推送", false)
}

// doSyncPull 拉取。
func (d *desktopApp) doSyncPull() {
	defer func() {
		d.loading = false
		if d.win != nil {
			d.win.Invalidate()
		}
	}()
	if d.repo == nil {
		return
	}
	conflict, err := d.repo.SyncPull()
	if err != nil {
		if conflict {
			d.setMsg("pull 冲突：请按标准 git 流程处理", true)
		} else {
			d.setMsg("拉取失败: "+err.Error(), true)
		}
		return
	}
	d.setMsg("已拉取", false)
	d.reload()
}

// ---- OPT-E1：命令面板（Ctrl+K，F17，简化版：条件渲染替换内容区） ----

type paletteItem struct {
	label   string
	group   string
	pageIdx int
	act     func()
	selID   string
}

func (d *desktopApp) paletteBuild() []paletteItem {
	var items []paletteItem
	for i, name := range pageNames {
		i := i
		items = append(items, paletteItem{label: "跳转到「" + name + "」", group: "页面", pageIdx: i, act: func() { d.page = i }})
	}
	items = append(items,
		paletteItem{label: "新建条目", group: "动作", pageIdx: pageEntities, act: func() { d.page = pageEntities; d.showNew = true }},
		paletteItem{label: "采集全部工具", group: "动作", pageIdx: pageCollect, act: func() { d.page = pageCollect; d.collectTool.Value = ""; go d.doCollect() }},
		paletteItem{label: "创建快照", group: "动作", pageIdx: pageSnapshot, act: func() { d.page = pageSnapshot; go d.doSnapshotCreate() }},
		paletteItem{label: "刷新数据", group: "动作", act: func() { d.reload() }},
	)
	for _, it := range d.items {
		it := it
		items = append(items, paletteItem{label: it.id, group: "实体", pageIdx: pageEntities, selID: it.id, act: func() {
			d.page = pageEntities
			d.selKind = it.kind
			d.selID = it.id
		}})
	}
	q := strings.ToLower(strings.TrimSpace(d.paletteEditor.Text()))
	if q == "" {
		return items
	}
	var out []paletteItem
	for _, it := range items {
		if subsequenceMatch(strings.ToLower(it.label+" "+it.group), q) {
			out = append(out, it)
		}
	}
	return out
}

func subsequenceMatch(s, q string) bool {
	qi := 0
	for _, r := range s {
		if qi < len(q) && r == rune(q[qi]) {
			qi++
		}
	}
	return qi == len(q)
}

// paletteView 命令面板（paletteOpen 时替换内容区；简化版避免深层嵌套）。
func (d *desktopApp) paletteView(gtx layout.Context, th *material.Theme, cs desktopui.Colors) layout.Dimensions {
	items := d.paletteBuild()
	if d.paletteSel >= len(items) {
		// 回车执行选中项（Editor Submit）
		if d.paletteEditor.Submit {
			if d.paletteSel < len(items) {
				items[d.paletteSel].act()
			}
			d.paletteOpen = false
		}
		d.paletteSel = maxInt(0, len(items)-1)
	}
	return desktopui.Card(gtx, cs, nil, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return d.icons.Search.Layout(gtx, cs.TextSecondary) }),
					layout.Rigid(layout.Spacer{Width: unit.Dp(desktopui.SpaceS)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return material.Editor(th, d.paletteEditor, "输入命令或搜索…（↑↓ 选择，回车执行，Esc 关闭）").Layout(gtx)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(desktopui.SpaceM)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return d.paletteList.Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
					it := items[i]
					fg := cs.Text
					if i == d.paletteSel {
						fg = cs.Accent
					}
					return layout.UniformInset(unit.Dp(desktopui.SpaceS)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(th, "["+it.group+"] "+it.label)
						lbl.Color = fg
						return lbl.Layout(gtx)
					})
				})
			}),
		)
	})
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// doToggleFavorite 收藏切换（annotations.favorite，F18）。
func (d *desktopApp) doToggleFavorite() {
	if d.repo == nil || d.selID == "" {
		return
	}
	dir := d.repo.Path(store.DirProfiles, "global")
	ann, err := profile.LoadAnnotations(dir)
	if err != nil {
		ann = &profile.Annotations{}
	}
	if ann.Favorite == nil {
		ann.Favorite = []string{}
	}
	found := false
	out := ann.Favorite[:0]
	for _, f := range ann.Favorite {
		if f == d.selID {
			found = true
			continue
		}
		out = append(out, f)
	}
	if found {
		ann.Favorite = out
		d.setMsg("已取消收藏: "+d.selID, false)
	} else {
		ann.Favorite = append(ann.Favorite, d.selID)
		d.setMsg("已收藏: "+d.selID, false)
	}
	if err := profile.SaveAnnotations(dir, ann); err != nil {
		d.setMsg("收藏保存失败: "+err.Error(), true)
		return
	}
	d.annotations = ann
	d.repo.Audit("favorite", "user", "global", "切换收藏 "+d.selID, "ok", nil, 0)
}

// doSaveSchedule 定时快照计划（F19）：后台 ticker 按间隔创建快照。
func (d *desktopApp) doSaveSchedule() {
	if d.schedTimer != nil {
		d.schedTimer.Stop()
		d.schedTimer = nil
	}
	var interval time.Duration
	switch d.schedEnum.Value {
	case "hourly":
		interval = time.Hour
	case "daily":
		interval = 24 * time.Hour
	default:
		d.setMsg("定时计划已关闭", false)
		return
	}
	d.schedTimer = time.NewTicker(interval)
	go func(t *time.Ticker) {
		for range t.C {
			d.doSnapshotCreate()
		}
	}(d.schedTimer)
	d.setMsg("定时快照已开启: "+d.schedEnum.Value, false)
	d.repo.Audit("schedule", "user", "global", "定时快照 "+d.schedEnum.Value, "ok", nil, 0)
}

// actionStatsText 使用统计文本（F22）：top 操作类型计数。
func (d *desktopApp) actionStatsText() string {
	if len(d.actionStats) == 0 {
		return "暂无操作记录"
	}
	type kv struct {
		k string
		v int
	}
	var pairs []kv
	for k, v := range d.actionStats {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].v > pairs[j].v })
	var parts []string
	for i, p := range pairs {
		if i >= 6 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s×%d", p.k, p.v))
	}
	return "操作统计：" + strings.Join(parts, " · ")
}

// doPresetPush 团队预设下发（F23）：提交 profile 并 push 到团队远端。
func (d *desktopApp) doPresetPush() {
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
	name := strings.TrimSpace(d.presetName.Text())
	if name == "" {
		name = "global"
	}
	// 下发 = 当前 profile 已提交 + push 到团队远端（带审计标记）
	err := d.repo.SyncPush(secrets.DefaultScanner(), func(matches []secrets.ScanMatch) bool {
		d.setMsg(fmt.Sprintf("preflight 命中 %d 处疑似敏感内容，已阻断下发", len(matches)), true)
		return false
	})
	if err != nil {
		d.setMsg("下发失败: "+err.Error(), true)
		return
	}
	d.repo.Audit("preset-push", "user", name, "团队预设下发 "+name, "ok", nil, 0)
	d.setMsg("已下发团队预设: "+name, false)
}

// ---- 主题偏好持久化（desktop.json，本机 UI 设置不入 SSOT 仓库） ----

// loadThemeMode 读取主题偏好（默认 D 玻璃拟态）。
func loadThemeMode() string {
	root, err := paths.DataHome()
	if err != nil {
		return desktopui.ModeGlass
	}
	data, err := os.ReadFile(filepath.Join(root, "desktop.json"))
	if err != nil {
		return desktopui.ModeGlass
	}
	var cfg struct {
		Theme string `json:"theme"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return desktopui.ModeGlass
	}
	return cfg.Theme
}

// saveThemeMode 写入主题偏好。
func saveThemeMode(mode string) {
	root, err := paths.DataHome()
	if err != nil {
		return
	}
	data, _ := json.Marshal(map[string]string{"theme": mode})
	_ = os.WriteFile(filepath.Join(root, "desktop.json"), data, 0600)
}

// applyThemeByLabel 按 chip 显示名切换主题并持久化。
func (d *desktopApp) applyThemeByLabel(label string) {
	for mode, name := range desktopui.ModeNames {
		if name == label && mode != d.ts.Mode {
			d.ts.SetMode(mode)
			saveThemeMode(mode)
			if d.win != nil {
				d.win.Invalidate()
			}
			return
		}
	}
}

// paintBackground 主题背景（D=渐变+光斑；A/B=纯色）。
func (d *desktopApp) paintBackground(gtx layout.Context, cs desktopui.Colors) {
	if !cs.IsGlass {
		paint.Fill(gtx.Ops, cs.Bg)
		return
	}
	// 三段垂直渐变（0f0c29 → 302b63 → 24243e）
	sz := gtx.Constraints.Max
	paint.LinearGradientOp{
		Stop1:  f32.Pt(0, 0),
		Stop2:  f32.Pt(0, float32(sz.Y)),
		Color1: cs.GradientTop,
		Color2: cs.GradientBot,
	}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	// 光斑 A：右上紫
	glow(gtx, float32(sz.X)*0.85, float32(sz.Y)*0.05, float32(sz.X)*0.45, cs.GlowA)
	// 光斑 B：左下青
	glow(gtx, float32(sz.X)*0.1, float32(sz.Y)*0.9, float32(sz.X)*0.35, cs.GlowB)
}

// glow 画柔光圆斑（多圈同心圆近似柔边）。
func glow(gtx layout.Context, cx, cy, r float32, c color.NRGBA) {
	for i := 5; i >= 1; i-- {
		rr := r * float32(i) / 5
		alpha := uint8(int(c.A) * (6 - i) / 12)
		col := color.NRGBA{R: c.R, G: c.G, B: c.B, A: alpha}
		area := clip.Ellipse{
			Min: image.Pt(int(cx-rr), int(cy-rr)),
			Max: image.Pt(int(cx+rr), int(cy+rr)),
		}.Op(gtx.Ops)
		paint.FillShape(gtx.Ops, col, area)
	}
}
