# cfg4ai-desktop UI/UX 设计规格 v1.0

> 适用：`cmd/cfg4ai-desktop`（Go + Gio 即时模式 GUI，无 CGO）。
> 目标：替换当前 `material.NewTheme()` 默认蓝白主题 + 纯文字按钮导航 + RadioButton 列表的"low爆了"界面。
> 一手调研来源（2026-08 抓取）：
> - VS Code UX Guidelines（容器架构 / Command Palette / Quick Pick / Walkthroughs）：code.visualstudio.com/api/ux-guidelines
> - GitHub Primer design tokens（三级 token、中性色阶、按钮全态值、字号、阴影）：primer.style/product/primitives
> - Raycast 扩展 UI/UX 规范（空态防闪烁、Action Panel 图标一致性、命名规范）：developers.raycast.com
> - Gio 官方文档（unit.Dp/Sp、Theme 机制、坐标系）：gioui.org/doc/architecture
> - Linear / Warp / Arc / Notion / Obsidian 公开界面取色与社区测量值

## 0. 调研结论：顶级开发者工具抄什么

| 工具 | 核心设计语言 | 落地到 cfg4ai 的规则 |
|------|--------------|----------------------|
| **Linear** | 深色优先（app 底 `#08090A`），靛蓝 accent `#5E6AD2`，Inter 13px 正文，边框仅 8% alpha，圆角 6–8 | 深色为默认主题；accent 采用靛蓝系；边框只用低 alpha，不用实灰 |
| **Raycast** | 命令面板是唯一中枢；列表项 = 图标+标题+副标题+快捷键；**空态必须先 loading 后空白**（防闪烁）；placeholder 必填 | ⌘K 命令面板作为全局中枢；空态渲染规则照抄（§5） |
| **Warp** | 深色底 + 卡片化 block、圆角 ~10、输入/输出分区 | 仪表盘统计卡、快照列表全部卡片化 |
| **VS Code** | Activity Bar + Primary Sidebar + Editor + Panel + Status Bar 五容器分层；Walkthrough 清单式首次引导；Quick Pick 模糊选择 | 侧栏+内容区+状态栏三容器骨架；首次引导用 checklist 页（§5.2） |
| **GitHub Primer** | 三级 token（base→functional→component）；深浅色阶倒置复用；按钮 rest/hover/active/disabled 全态色值公开 | 全套 token 命名法 + 浅色色板直接采用 Primer 公开值（§1.2/§1.3） |
| **GitHub Desktop** | 左侧仓库列表 + 右侧主区，主区顶部工具条 | 内容区 Header 工具条（标题+主操作+⌘K 提示） |
| **Arc** | 侧栏即应用；深色底贯穿标题栏的无边框感；圆角 8–12 | 窗口整体深色底贯穿，无亮色标题栏割裂 |
| **Notion / Obsidian** | 行内操作 hover 才显现；空态 = 图标+一句话+主按钮 | 列表行内操作默认隐藏，hover 淡入（§3.2） |

### 设计原则（5 条，全部可检查）

1. **深色优先**：默认 Dark；Light 为同级完整实现，非反色凑合。
2. **颜色只走 token**：代码中禁止出现字面量 hex，全部引用 `theme.Palette` 字段（对应 Primer functional 级）。
3. **图标+文字**：任何导航/按钮/列表项必须有图标（Lucide 风格 1.5dp 描边线性图标），杜绝纯文字按钮。
4. **每个状态切换都有动画**：hover/选中/进出/加载均有 80–300ms 过渡，时长缓动统一走 token（§4）。
5. **键盘可达**：⌘K（Ctrl+K）命令面板 + ↑↓←→ 导航 + Enter 执行 + Esc 关闭，核心流程零鼠标可用。

---

## 1. 设计系统

### 1.1 Token 分层（Primer 三级制）

```
base（色阶原值，禁直接用） → functional（bg/surface、fg/primary…，组件用它） → component（button-primary-bg-hover…，特殊态才定义）
```

代码落地：`internal/desktopui/theme` 包内 `Palette` struct 即 functional 层；组件只读 `Palette`，不读 base。

### 1.2 深色色板（Dark，默认主题）

base 中性色阶（自 Linear `#08090A` / GitHub Dark `#0d1117` 体系调合）：

| token | hex | 用途 |
|-------|-----|------|
| gray-0 | `#0B0C0E` | app 背景 |
| gray-1 | `#101114` | 侧栏背景 |
| gray-2 | `#16181D` | 卡片 surface |
| gray-3 | `#1C1F26` | 浮层 elevated（toast/模态/命令面板/popover） |
| gray-4 | `#23262E` | 按下/强化 hover 底 |
| gray-5 | `#2B2F38` | 强边框、分隔块 |
| gray-6 | `#3A3F4B` | emphasis 边框、禁用文字 |
| gray-7 | `#6B7280` | 三级文字（placeholder/禁用） |
| gray-8 | `#9BA1AA` | 二级文字 |
| gray-9 | `#C9CDD3` | 弱一级文字 |
| gray-10 | `#F2F3F5` | 一级文字 |

functional token（**组件只准引用这一层**）：

| token | 值 | 用途 |
|-------|----|------|
| `bg/app` | `#0B0C0E` | 窗口底色（贯穿标题栏） |
| `bg/sidebar` | `#101114` | 侧栏 |
| `bg/surface` | `#16181D` | 卡片、内容区面板 |
| `bg/elevated` | `#1C1F26` | toast / 模态 / 命令面板 / 下拉 |
| `bg/inset` | `#08090A` | 输入框、代码块（比 app 更深一档） |
| `bg/hover` | `#FFFFFF` @ 6%（`#FFFFFF0F`） | 通用 hover 底 |
| `bg/active` | `#FFFFFF` @ 10%（`#FFFFFF1A`） | 按下底 |
| `bg/selected` | `#5E6AD2` @ 14%（`#5E6AD224`） | 选中底（导航/列表/命令面板） |
| `border/subtle` | `#FFFFFF` @ 8%（`#FFFFFF14`） | 卡片描边、分隔线 |
| `border/default` | `#FFFFFF` @ 12%（`#FFFFFF1F`） | 输入框、按钮描边 |
| `border/emphasis` | `#FFFFFF` @ 20%（`#FFFFFF33`） | hover 后描边 |
| `fg/primary` | `#F2F3F5` | 标题、正文 |
| `fg/secondary` | `#9BA1AA` | 副标题、说明 |
| `fg/tertiary` | `#6B7280` | placeholder、禁用、时间戳 |
| `fg/on-accent` | `#FFFFFF` | accent 底上的文字 |
| `accent/default` | `#5E6AD2` | 主 accent（Linear 同源靛蓝） |
| `accent/hover` | `#6C76E0` | accent hover |
| `accent/active` | `#7C84E8` | accent 按下 |
| `accent/fg` | `#9CA3F5` | 深底下 accent 色文字/图标（提亮保 4.5:1 对比度） |
| `accent/subtle` | `#5E6AD2` @ 14% | accent 浅底（badge、选中） |
| `focus/ring` | `#5E6AD2` | 焦点环，2dp 实线 + 2dp 外扩 |

语义色（深色，采用 GitHub Dark 公开值，对比度已验证）：

| 角色 | fg/emphasis | subtle 底 | border |
|------|-------------|-----------|--------|
| success | `#3FB950` | `#3FB950` @ 15% | `#3FB950` @ 30% |
| warning | `#D29922` | `#D29922` @ 15% | `#D29922` @ 30% |
| danger | `#F85149` | `#F85149` @ 15% | `#F85149` @ 30% |
| info | `#4493F8` | `#4493F8` @ 15% | `#4493F8` @ 30% |
### 1.3 浅色色板（Light，完整同级实现；功能性色值直接采用 Primer Light 公开值）

base 中性色阶：

| token | hex | 用途 |
|-------|-----|------|
| gray-0 | `#FFFFFF` | 卡片 surface / elevated |
| gray-1 | `#F6F7F9` | app 背景（Primer muted 微调） |
| gray-2 | `#F0F2F5` | 侧栏背景 |
| gray-3 | `#EFF1F4` | hover 底（solid 版） |
| gray-4 | `#E6E9EE` | active 底 |
| gray-5 | `#DEE2E8` | 强分隔 |
| gray-6 | `#D1D9E0` | 默认边框（Primer `borderColor-default`） |
| gray-7 | `#818B98` | 三级文字（Primer disabled） |
| gray-8 | `#59636E` | 二级文字（Primer `fgColor-muted`） |
| gray-9 | `#394048` | 弱一级文字 |
| gray-10 | `#1F2328` | 一级文字（Primer `fgColor-default`） |

functional token：

| token | 值 | 用途 |
|-------|----|------|
| `bg/app` | `#F6F7F9` | 窗口底色 |
| `bg/sidebar` | `#F0F2F5` | 侧栏 |
| `bg/surface` | `#FFFFFF` | 卡片、内容面板 |
| `bg/elevated` | `#FFFFFF` | 浮层（靠阴影分层） |
| `bg/inset` | `#F6F7F9` | 输入框、代码块 |
| `bg/hover` | `#1F2328` @ 5%（`#1F23280D`） | 通用 hover |
| `bg/active` | `#1F2328` @ 10%（`#1F23281A`） | 按下 |
| `bg/selected` | `#5E6AD2` @ 12%（`#5E6AD21F`） | 选中底 |
| `border/subtle` | `#1F2328` @ 8%（`#1F232814`） | 卡片描边、分隔线 |
| `border/default` | `#D1D9E0` | 输入框、按钮描边（Primer 原值） |
| `border/emphasis` | `#818B98` | hover 描边（Primer 原值） |
| `fg/primary` | `#1F2328` | Primer 原值 |
| `fg/secondary` | `#59636E` | Primer 原值 |
| `fg/tertiary` | `#818B98` | Primer 原值 |
| `fg/on-accent` | `#FFFFFF` | — |
| `accent/default` | `#4F56C4` | 浅底下加深一档，白底对比度 ≥ 4.5:1 |
| `accent/hover` | `#4349AC` | — |
| `accent/active` | `#383E94` | — |
| `accent/fg` | `#4F56C4` | 浅底直接用 default 作文字/图标色 |
| `accent/subtle` | `#5E6AD2` @ 12% | badge / 选中浅底 |
| `focus/ring` | `#4F56C4` | 2dp + 2dp 外扩 |

语义色（浅色，全部 Primer Light 公开值）：

| 角色 | fg/emphasis | subtle 底 | border |
|------|-------------|-----------|--------|
| success | `#1A7F37` | `#DAFBE1` | `#4AC26B` @ 40% |
| warning | `#9A6700` | `#FFF8C5` | `#D4A72C` @ 40% |
| danger | `#D1242F` | `#FFEBE9` | `#FF8182` @ 40% |
| info | `#0969DA` | `#DDF4FF` | `#54AEFF` @ 40% |

### 1.4 字体

字族（Gio 落地：embed TTF + `font.Typeface` 注册；Noto Sans SC 需做子集化控制体积）：

| 角色 | 字族栈 | 用途 |
|------|--------|------|
| UI | `Inter`, `PingFang SC`, `Microsoft YaHei UI`, `Noto Sans CJK SC` | 全部界面文字 |
| Mono | `JetBrains Mono`, `Cascadia Code`, `Consolas`, `Noto Sans Mono CJK SC` | 路径、快照 ID、命令、键帽 |

层级（size sp / line-height / weight；对齐 Primer 字号 xs12 sm14 md16 lg20 xl32 与 Linear/VS Code 的 13px 正文密度）：

| 层级 | size/行高 | 字重 | 颜色 | 用途 |
|------|-----------|------|------|------|
| Display | 28sp / 36 | 600 | fg/primary | 仪表盘大数字 |
| Title-L | 20sp / 28 | 600 | fg/primary | 页面标题（Header 内） |
| Title-M | 16sp / 24 | 600 | fg/primary | 卡片标题、区块标题 |
| Title-S | 14sp / 20 | 600 | fg/primary | 模态标题、列表分组标题 |
| Body | 13sp / 20 | 400 | fg/primary | 正文、按钮、输入、列表项 |
| Body-Strong | 13sp / 20 | 600 | fg/primary | 列表项主标题、强调 |
| Caption | 12sp / 16 | 400 | fg/secondary | 辅助说明、副标题 |
| Micro | 11sp / 14 | 500 | fg/tertiary | badge、分组小标题（可加 +4% 字距） |
| Mono | 12.5sp / 18 | 400 | fg/secondary | 路径、ID、代码 |

### 1.5 间距（4dp 基数）

| token | dp | 典型用途 |
|-------|----|----------|
| space-0.5 | 2 | icon 与文字的微调、键帽内距 |
| space-1 | 4 | 图标↔文字最小间隙、列表行间距 |
| space-1.5 | 6 | 紧凑控件内距、按钮 icon 间隙 |
| space-2 | 8 | 按钮水平内距、输入框内距、表单项间隙 |
| space-3 | 12 | 卡片小内距、列表项水平内距 |
| space-4 | 16 | 卡片标准内距、区块小间隔 |
| space-5 | 20 | 统计卡内距 |
| space-6 | 24 | 页面内容 padding、区块间隔 |
| space-8 | 32 | 大区块间隔 |
| space-12 | 48 | 页面级留白、空态上下留白 |
| space-16 | 64 | 空态容器垂直留白 |

规则：组件内距只用 6/8/12/16；页面布局只用 16/24/32；禁止出现 5、7、9、13 等非刻度值。

### 1.6 圆角 / 描边 / 阴影

圆角：

| token | dp | 用途 |
|-------|----|------|
| r-1 | 3 | badge、checkbox、tag、键帽 |
| r-2 | 6 | 按钮、输入框、列表项、tooltip（Linear 同档） |
| r-3 | 8 | 卡片、导航选中底、toast（Warp block 同档） |
| r-4 | 12 | 模态、命令面板、popover（Arc 同档） |
| r-full | 999 | pill、状态点 |

描边：默认 1dp `border/subtle`；交互控件 1dp `border/default`；focus 环 2dp `focus/ring` + 2dp 外扩（外扩色 = `bg/app`，模拟 outline-offset）。

阴影（Gio 无 box-shadow 原语：用 `clip.RRect` 多层偏移+径向渐变叠画近似，或预渲染 9-patch PNG 贴图；深色模式下阴影不可见，一律改用"背景提亮一档 + border/subtle"分层）：

| 层级 | 浅色参数（offset-y / blur / color） | 深色参数 |
|------|--------------------------------------|----------|
| E0 | none | none |
| E1 卡片静止 | `0 1px 1px rgba(31,35,40,.04), 0 1px 2px rgba(31,35,40,.03)`（Primer resting-small） | none，用 border/subtle |
| E2 卡片 hover / 下拉 | `0 1px 1px rgba(37,41,46,.10), 0 3px 6px rgba(37,41,46,.12)`（Primer resting-medium） | border/default |
| E3 toast / popover | `0 8px 24px rgba(0,0,0,.16)` + 1px border/subtle | `0 8px 24px rgba(0,0,0,.48)` + 1px border/subtle |
| E4 模态 / 命令面板 | `0 24px 48px rgba(0,0,0,.24)` + 1px border/subtle（Primer floating-medium 简化） | `0 24px 48px rgba(0,0,0,.56)` + 1px border/subtle |

### 1.7 图标系统

- 风格：Lucide / Octicons 系**线性图标**，1.5dp 描边，圆角端点；禁填充面性图标与 emoji。
- 尺寸档：14（按钮内）/ 16（导航、列表、输入）/ 20（卡片标题）/ 48（空态）。
- 颜色：默认 `fg/secondary`；hover 所在行变 `fg/primary`；选中态变 `accent/fg`；状态图标用语义色 fg。
- 导航映射：仪表盘=`layout-dashboard`、实体=`boxes`、采集=`download-cloud`、迁移=`arrow-left-right`、快照=`camera`、设置=`settings-2`。
- Gio 落地：SVG path 数据内嵌 `internal/desktopui/icon/icons.go`（约 40 个图标，每图标一个 `[]byte` path 指令），`clip.Path` + `paint.Fill` 描边渲染；接口 `icon.Draw(gtx, name, size, color)`。
---

## 2. 布局与导航骨架

窗口：默认 **1120×720dp**，最小 **880×560dp**（`app.Size(unit.Dp(1120), unit.Dp(720))` + `app.MinSize`）。
三容器骨架（VS Code 分层思想，砍掉 Panel）：**侧栏 232dp + 内容区 + 状态栏 28dp**。

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│ bg/app 贯穿全窗口（含标题栏区域），无亮色割裂                                        │
│┌──────────────232dp────────────┬────────────────────────内容区────────────────────┐│
││ ▣ cfg4ai            [⏾主题]   │ 页面标题 Title-L        [主操作按钮] [⌘K kbd]     ││  ← Header 52dp，px24
││                               │ 副标题 Caption（fg/secondary）                    ││
││ ─────── border/subtle ─────── │────────────── border/subtle 1dp ─────────────────││
││                               │                                                  ││
││ 主分组 MICRO 小标题 "工作台"    │   ┌─内容 padding 24dp，max-w 960dp 居中────────┐ ││
││ ┌──────────────────────────┐  │   │                                              │ ││
││ │▎⊞ 仪表盘              1 │  │   │   （页面内容：卡片 / 列表 / 表单）              │ ││  ← 选中项：bg/selected
││ └──────────────────────────┘  │   │                                              │ ││    + 左侧 2dp accent 条
││   ⊟ 实体                  2  │  │   │                                              │ ││
││   ⬇ 采集                  3  │  │   │                                              │ ││
││   ⇄ 迁移                  4  │  │   │                                              │ ││
││   ◉ 快照                  5  │  │   │                                              │ ││
││                               │   │                                              │ ││
││ ─────── border/subtle ─────── │   │                                              │ ││
││ MICRO 小标题 "系统"            │   │                                              │ ││
││   ⚙ 设置                      │   │                                              │ ││
││   ?  关于                     │   │                                              │ ││
││                               │   └──────────────────────────────────────────────┘ ││
││ [spacer]                      │                                                  ││
││ ┌──────────────────────────┐  │                                                  ││
││ │● 仓库已就绪   v0.3.0      │  │                                                  ││  ← 侧栏底部状态卡
││ └──────────────────────────┘  │                                                  ││
│├──────────────────────────────┴──────────────────────────────────────────────────┤│
││ ● global ｜ F:\…\cfg4ai          上次采集 2 分钟前 ｜ 实体 128 ｜ 快照 6   ⏾深色  ││  ← 状态栏 28dp
│└──────────────────────────────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────────────────────┘
```

### 2.1 侧栏（Sidebar）

- 宽 **232dp**；`bg/sidebar`；右侧 1dp `border/subtle` 分隔；可折叠为 **56dp** 图标栏（VS Code Activity Bar 式），折叠动画 200ms。
- 内边距：左右 12dp，上下 12dp。
- 顶部品牌区：高 48dp = logo 20×20 accent + "cfg4ai" Title-S + 右侧主题切换 icon-btn 28×28。
- 分组小标题：Micro 11sp `fg/tertiary`，px 12，上 16 下 4。
- 导航项（替换现有纯文字 material.Button）：
  - 高 **32dp**，r-2 6，px `8+10`，icon 16×16 + 10dp 间隙 + Body 13sp。
  - rest：文字 `fg/secondary`，图标 `fg/secondary`，透明底。
  - hover：`bg/hover`，文字/图标 `fg/primary`，80ms 过渡。
  - selected：`bg/selected` + 文字/图标 `accent/fg` + 字重 500 + **左侧 2dp accent 指示条**（指示条跟随选中项滑动 200ms，见 §4-1）。
  - 快捷键徽标：右侧 kbd 显示 `1`–`5`（Ctrl+1..5 切页）。
- 底部状态卡：r-3 8，`bg/surface`，border/subtle，p 10/12：状态点 8×8 `success` + "仓库已就绪" Caption + 版本号 `fg/tertiary`。

### 2.2 内容区（Content）

- `bg/app`；内容 padding **24dp**；正文列 **max-width 960dp** 居中（超宽屏不拉伸）。
- Header 高 **52dp**：左 = Title-L 页面标题 + 其下 Caption 副标题（fg/secondary）；右 = 页面主操作按钮（Primary md）+ ⌘K 键帽提示（Ghost sm："⌘K 命令"）。
- Header 与内容之间无卡片感，仅靠留白分层；内容区内卡片间距 16dp。

### 2.3 状态栏（Status Bar）

- 高 **28dp**，`bg/sidebar`，顶部 1dp `border/subtle`，px 16，Caption 12sp `fg/secondary`。
- 左（工作区级）：状态点 + profile 名 + 仓库根路径（Mono 12.5，超长中间省略）。
- 右（会话级）：上次操作摘要（如"上次采集 2 分钟前"）｜实体数 ｜快照数 ｜主题切换入口。
- 执行后台任务时：左侧出现 12dp spinner + 任务名，完成时状态点绿色脉冲一次（§4-15）。

### 2.4 命令面板（⌘K，全局浮层）

```
┌──────────────────────────────────────────────────────────────────┐
│                  backdrop #000 @ 40%（深色 56%）                  │
│         ┌────────────── 640dp，r-4 12，E4，bg/elevated ──────┐   │
│         │ ⌕  搜索命令、工具、实体…                       Esc │   │ ← 输入行 h48，fs15
│         │──────────────── border/subtle ─────────────────────│   │
│         │ 命令                                    MICRO 分组  │   │
│         │  ⇄ 迁移向导…        迁移              Ctrl+M       │   │
│         │  ⬇ 采集全部工具     采集              Ctrl+E       │   │
│         │  ◉ 创建快照…        快照              Ctrl+S       │   │
│         │  ⊞ 前往仪表盘       导航              Ctrl+1       │   │
│         │ 工具                                                │   │
│         │  ⬇ 采集 claude-code  工具                          │   │
│         │  ⬇ 采集 codex        工具                          │   │
│         └────────────────────────────────────────────────────┘   │
│            top 96dp ｜ max-h 480dp ｜ ↑↓ 环绕选择 ｜ Enter 执行   │
└──────────────────────────────────────────────────────────────────┘
```
---

## 3. 组件规格

### 3.1 按钮 Button

变体 × 状态（所有按钮 r-2 **6dp**，字重 500，禁用态 = `bg/hover` 底 + `fg/tertiary` 字）：

| 变体 | rest | hover | active | 用途 |
|------|------|-------|--------|------|
| Primary | bg `accent/default`，fg `#FFFFFF` | bg `accent/hover` | bg `accent/active` + scale 0.98 | 每页最多 1 个主操作 |
| Secondary | bg `bg/surface`，1dp `border/default`，fg `fg/primary` | bg `bg/hover`，border/emphasis | bg `bg/active` | 常规操作 |
| Ghost | 透明，fg `fg/secondary` | bg `bg/hover`，fg `fg/primary` | bg `bg/active` | 工具条、行内操作 |
| Danger | bg `danger`，fg `#FFFFFF` | danger 提亮 8% | danger 加深 8% + scale 0.98 | 不可逆操作 |
| Ghost-Danger | 透明，fg `danger` | bg danger subtle | bg danger subtle 加深 | 行内危险操作 |

尺寸：

| 尺寸 | 高 | 水平 padding | 字号 | icon | icon↔字距 |
|------|----|--------------|------|------|-----------|
| sm | 24dp | 10dp | 12sp | 14dp | 4dp |
| md | 32dp | 14dp | 13sp | 14dp | 6dp |
| lg | 36dp | 16dp | 14sp | 16dp | 6dp |

- focus 可见：2dp `focus/ring` + 2dp 外扩（仅键盘 Tab 显示，鼠标点击不显示）。
- 加载态：icon 位替换为 14dp spinner（2dp 描边，1s/圈），文字改"采集中…"，整钮禁用。
- 所有按钮必须带 icon（无语义图标时用 `chevron-right`/`plus` 兜底），杜绝纯文字按钮。

### 3.2 列表项 ListItem（实体 / 快照行）

```
┌────────────────────────────────────────────────────────────────┐
│▎[icon16] 主标题 Body-Strong      [类型 badge]  [行内操作 hover]│
│▎         副标题 Caption（mono 路径 / 描述）                    │
└────────────────────────────────────────────────────────────────┘
```

- 单行 h **40dp** / 双行 h **56dp**；px 12；r-2 6；行间间隙 2dp。
- hover：`bg/hover` 80ms；selected：`bg/selected` + 左侧 2dp accent 条。
- 行内操作（icon-btn 28×28：复制/打开目录/恢复）**默认 opacity 0**，hover 该行时 100ms fade-in（Notion/Obsidian 模式）；键盘 focus 行同样显现。
- 类型 badge：Micro 11sp，px 6 py 2，r-1 3，语义分组配色——指令=info subtle、MCP=accent subtle、技能=success subtle、Agent=warning subtle、Hook=danger subtle、设置=neutral。
- 快照行追加：Mono 12.5 的 ID 前 8 位 + Caption 相对时间（"2 小时前"）+ 文件数 badge。

### 3.3 卡片 Card

- rest：`bg/surface` + 1dp `border/subtle` + r-3 **8dp** + padding **16dp** + E1。
- 可交互卡 hover：border 升 `border/default` + E1→E2 + translateY **-1dp**，200ms ease-out。
- 统计卡（仪表盘，Warp block 风）：padding **20dp**；右上角 icon 16 `fg/tertiary`；大数字 Display 28 `fg/primary`（count-up 400ms）；label Caption `fg/secondary`；三卡等高、间距 16dp、flex 均分。
- 结构卡（区块容器）：标题行 icon 20 + Title-M，下 12dp 后 1dp `border/subtle` 分隔，再下 16dp 内容。
- 危险卡：border 换 `danger` @ 40%，标题 icon 用 danger 色。
### 3.4 输入 Input / Select / Checkbox

Input（替换现有 material.Editor 裸框）：

- h **36dp**（多行 min 72dp 自适应）；`bg/inset`；1dp `border/default`；r-2 6；px 12 py 8；Body 13sp。
- placeholder：`fg/tertiary`，**必填**（Raycast 规范：不允许无 placeholder 的输入框）。
- hover：`border/emphasis` 80ms。
- focus：border `accent/default` + 2dp `focus/ring` @ 25% alpha 外发光，120ms fade。
- error：border `danger` + 下方 4dp 处 Caption `danger` 错误文案 + 左侧 16dp `alert-circle` 图标。
- disabled：`bg/hover` + `fg/tertiary`。
- 结构：可选 16dp 前缀图标（`fg/tertiary`，左 10dp）；有内容时右侧显现 16dp 清除钮（hover fade 100ms）。

Select（替换现有 RadioButton 竖排的工具/范围选择）：

- 触发器样式同 Input，右侧 16dp `chevron-down`；展开 popover：`bg/elevated` + border/subtle + r-3 8 + E3；选项 h 32dp，hover `bg/hover`，选中项右侧 `check` 16 `accent/fg`；popover 进出 scale 0.98→1 + fade 120ms。
- 选项 > 8 个时（本应用 13 个工具）popover 顶部内嵌过滤输入框（VS Code Quick Pick 模式）。

Checkbox / Toggle：16×16，r-1 3；选中底 `accent/default` + `#FFF` 对勾，对勾 draw-on 描边动画 200ms。

### 3.5 空态 EmptyState（Primer Blankslate 结构）

```
│                                                                │
│                    [icon 48×48 fg/tertiary]                    │
│                         ↕ 16dp                                 │
│                  标题 Title-S（fg/primary）                     │
│                         ↕ 4dp                                  │
│           描述 Body 13sp（fg/secondary，居中，max-w 320dp）      │
│                         ↕ 16dp                                 │
│           [Primary md 主操作]  [Ghost md 次操作]                 │
│                                                                │
```

- 容器：内容区水平居中，max-w **360dp**，垂直留白 **64dp**。
- 图标 48×48，`fg/tertiary` @ 60%（可选 accent subtle 圆形 80×80 底衬）。
- **防闪烁铁律**（Raycast 规范）：`isLoading` 为真时渲染骨架屏，**绝不先渲染空态**；仅当数据就绪且 `len == 0` 才显示空态。空态进入动画：icon scale 0.9→1 + 整体 fade 300ms，延迟 50ms。
### 3.6 Toast（替换现有顶部文字 msg 条）

- 位置：**右下角**，距右/下各 **16dp**；纵向堆叠，间距 8dp；最多同时 3 条，超出排队。
- 尺寸：min-w 280 / max-w 400dp；min-h 44dp；padding 12/14；r-3 8；`bg/elevated` + 1dp `border/subtle` + E3。
- 结构：`[状态 icon 16 语义色] [标题 Body-Strong] [描述 Caption 可选] … [操作 Ghost sm 可选] [× 关闭]`。
- 底部 **2dp 倒计时进度条**（语义色，宽度随剩余时间线性收缩，Linear 风格）。
- 时长：success 4s / info 5s / warning 8s / **error 常驻必须手动关**；hover 暂停倒计时。
- 动效：进入 translateY 8→0 + fade 200ms ease-out；退出 translateY 4 + fade 150ms ease-in。
- 现有消息映射：`采集完成，新增 N 条` → success toast（操作钮"查看实体"）；`迁移失败: …` → error toast（操作钮"复制详情"）。

### 3.7 模态 Modal

- backdrop：`#000` @ 40%（深色 56%）；非危险操作点击 backdrop 可关，危险操作不可。
- 容器：w **480dp**（确认框 400 / 表单 560）；r-4 **12dp**；`bg/elevated` + border/subtle + E4；padding **24dp**。
- 结构：Title-S 标题 → 4dp → Body `fg/secondary` 描述 → 24dp → 按钮行右对齐（间距 8dp：Ghost"取消" + Primary/Danger 确认）。
- 动效：进入 scale 0.96→1 + fade 150ms ease-out；退出 100ms；backdrop fade 150ms。
- 焦点：focus trap 锁在模态内；Esc 关；Enter 触发主按钮；打开时焦点落主按钮（危险操作落"取消"）。
- 危险确认（恢复快照=覆盖当前仓库）：Danger 主按钮文案写明后果（"覆盖并恢复"），**二次点击确认**——首次点击后 3s 内按钮变"再次点击确认"态，超时回退。

### 3.8 命令面板 CommandPalette（⌘K / Ctrl+K）

结构见 §2.4 线框。规格：

- 容器：w **640dp**，距顶 **96dp**，r-4 12，`bg/elevated` + border/subtle + E4，max-h **480dp**。
- 输入行：h **48dp**，左 14dp `search` 16dp `fg/tertiary`，字号 **15sp**，placeholder"搜索命令、工具、实体…"；下方 1dp `border/subtle`。
- 结果项 h **40dp**：icon 16 + 名称 Body 13 + 类型 badge Micro + 右侧 kbd 快捷键；分组标题 Micro `fg/tertiary`（px 12，上 12 下 4）。
- kbd 键帽：Mono 11sp，`bg/hover` + 1dp `border/subtle`，r-1 3，px 5 py 1，min-h 18dp。
- 交互：↑↓ 环绕移动（选中 `bg/selected`），Enter 执行，Esc 关闭；输入即模糊过滤（名称+类型）；无结果走空态规范。
- 内置命令：5 个页面跳转（Ctrl+1..5）、采集全部工具（Ctrl+E）、采集×13 个具体工具、迁移向导（Ctrl+M）、创建快照（Ctrl+S）、切换深浅主题、打开仓库目录、复制仓库路径。
- 动效：打开 scale 0.98→1 + fade 120ms；关闭 fade 80ms。
---

## 4. 微交互与动画清单

### 4.0 动画 token

时长：

| token | 值 | 用途 |
|-------|----|------|
| t-micro | 80ms | 颜色/hover 背景过渡 |
| t-fast | 120ms | fade、focus 环、popover |
| t-normal | 200ms | 位移、卡片浮起、导航指示条、页面切换 |
| t-slow | 300ms | 模态、空态进入、主题切换 |

缓动（Gio 落地： cubic-bezier 四点插值函数）：

| token | 曲线 | 用途 |
|-------|------|------|
| ease-out | `cubic-bezier(0.16, 1, 0.3, 1)` | 一切"进入"（元素出现、浮层打开） |
| ease-standard | `cubic-bezier(0.4, 0, 0.2, 1)` | 状态切换（hover、选中、折叠） |
| ease-in | `cubic-bezier(0.7, 0, 0.84, 0)` | 一切"退出"（关闭、消失） |

### 4.1 动画清单（18 条，全部可逐条验收）

| # | 场景 | 参数 |
|---|------|------|
| 1 | 导航选中指示条 | 2dp accent 竖条跟随选中项滑动，200ms ease-standard |
| 2 | 导航项 hover 背景 | 透明→`bg/hover`，80ms t-micro |
| 3 | 页面切换 | 新页 fade-in + translateX 8→0，200ms ease-out |
| 4 | 按钮 hover / active | 背景过渡 80ms；按下 scale 0.98 回弹 100ms |
| 5 | 卡片 hover 浮起 | translateY -1dp + E1→E2，200ms ease-out |
| 6 | 列表行内操作显现 | opacity 0→1，100ms（行 hover / focus 触发） |
| 7 | Toast 进 / 出 | translateY 8→0 + fade 200ms ease-out；退出 150ms ease-in |
| 8 | 模态进 / 出 | scale 0.96→1 + fade 150ms / 100ms；backdrop fade 150ms |
| 9 | 命令面板开 / 关 | scale 0.98→1 + fade 120ms / 80ms |
| 10 | spinner | 360° 旋转，1000ms linear infinite（描边 2dp，arc 270°） |
| 11 | 骨架屏 shimmer | 高亮条从左到右扫过占位块，1200ms infinite，色 = `bg/hover`→`bg/active` 渐变 |
| 12 | 主题切换 | 全量色 token 交叉插值 250ms（背景、文字、边框同步过渡，禁跳变） |
| 13 | 侧栏折叠 | 宽 232→56dp，200ms ease-standard，文字 fade 先出 |
| 14 | 输入框 focus 环 | 2dp ring fade-in 120ms |
| 15 | 状态栏任务完成脉冲 | 状态点 success 色放大 1→1.4→1，300ms |
| 16 | 成功对勾 draw-on | path 描边从 0 画到全长，300ms ease-out（快照创建成功、引导步骤完成） |
| 17 | 统计卡数字滚动 | count-up：旧值→新值插值，400ms ease-out，仅数值增大时 |
| 18 | tooltip | 悬停延迟 400ms 出现，fade 100ms，偏移 4dp，r-2 6，`bg/elevated` + Caption 12sp |

Gio 落地机制：动画值全部由 `gtx.Now` 驱动插值（记录 start 时间，每帧算 progress）；用 `op.InvalidateOp{}`（或 `InvalidateAfterCmd`）保持帧循环直到动画结束；封装 `anim.Tween{Duration, Ease, OnUpdate}`，组件持有 tween 状态。颜色插值在 NRGBA 空间逐通道线性插值即可（250ms 内无可见色带）。
---

## 5. 空状态与首次引导

### 5.1 各场景空态文案与结构（Blankslate 结构见 §3.5）

| 场景 | icon | 标题 | 描述 | 主操作 | 次操作 |
|------|------|------|------|--------|--------|
| 实体页无数据 | `boxes` | 还没有采集到配置 | 从本机已安装的 AI 工具中采集指令、MCP、技能等配置到统一仓库。 | 立即采集（跳采集页） | 了解采集范围 |
| 采集页未检测 | `search` | 未检测到已安装的 AI 工具 | cfg4ai 支持 13 种工具，安装后回到此页点击重新扫描。 | 重新扫描 | 查看支持列表 |
| 快照页无快照 | `camera` | 还没有快照 | 快照是仓库的完整时间点备份，恢复前也会自动创建反向快照。 | 创建第一个快照 | — |
| 命令面板无结果 | `search-x` | 没有匹配的命令 | 试试搜索工具名，如 "claude" 或 "codex"。 | — | — |
| 迁移页源为空 | `arrow-left-right` | 该工具暂无可迁移配置 | 先在「采集」页采集源工具配置，再执行迁移。 | 去采集 | 预览全部（dry-run） |
| 仓库未就绪（错误） | `alert-triangle` | 仓库打开失败 | {错误原因 Caption 单行}。检查目录权限后重试。 | 重试 | 复制错误详情 |
| 搜索/过滤无结果 | `search-x` | 没有匹配的实体 | 尝试更换关键字，或清除筛选条件。 | 清除筛选 | — |

加载中一律用骨架屏（3 条 h40 占位行 + shimmer，§4-11），**不得**先闪空态。

### 5.2 首次引导（VS Code Walkthrough 式清单页，非模态）

触发条件：首次启动且仓库为空（无实体且无快照）→ 仪表盘位置渲染欢迎页；设置中可随时重新打开。

```
┌────────────────────────────── 内容区（max-w 960）─────────────────────────────┐
│                                                                                │
│   ▣ cfg4ai                                                        [跳过引导 ×] │
│   Title-L 24sp：把分散的 AI 工具配置，收进一个仓库                 ↕ 8dp        │
│   Caption 14sp fg/secondary：采集 → 治理 → 迁移到任意工具，全程可快照回滚        │
│                                                                 ↕ 32dp        │
│   ┌─引导卡 bg/surface border/subtle r-3 8 p24────────────┐  进度 2/4 ●●○○      │
│   │ ✓ 1. 扫描本机工具                          已完成      │  ← check draw-on   │
│   │     Caption：已发现 claude-code、codex 等 5 个工具     │     300ms 动画     │
│   │ ────────────────────────────────────────────────────│                    │
│   │ ○ 2. 一键采集全部配置              [开始采集 Primary] │  ← 当前步 accent   │
│   │     Caption：自动脱敏，secret 不会进入仓库             │     左侧 2dp 条    │
│   │ ────────────────────────────────────────────────────│                    │
│   │ ○ 3. 浏览你的实体                                  │                    │
│   │ ○ 4. 创建第一个快照                                │                    │
│   └─────────────────────────────────────────────────────┘                    │
│   特性三连（三列 Caption + icon 16）：🔒 secretref 脱敏 ｜ ⇄ 双向无损互转 ｜ ◉ 随时回滚 │
└────────────────────────────────────────────────────────────────────────────────┘
```

规则：

- 步骤卡：每步 h 64dp；完成态 = 对勾 success 色 + 标题 `fg/tertiary` 划线；当前态 = 左侧 2dp accent 条 + 右侧 Primary sm 按钮；未来态 = `fg/tertiary`。
- 顶部进度：4 段圆点 8×8，完成段 `accent/default`，未完成 `bg/active`；或 2dp 进度条 accent。
- 第 2 步"开始采集"完成后：按钮变 spinner→对勾，toast success"采集完成，新增 N 条"，步骤自动打勾并高亮下一步（300ms 滚动到位）。
- 全部完成：引导卡 fade-out 300ms → 渲染正常仪表盘（统计卡 count-up 入场）。
- 点"跳过引导"：记录 `ui.onboarding_done=true`（写入仓库 `settings.json`，不打扰下次启动）。
---

## 6. Gio 落地映射与迁移步骤

### 6.1 包结构（新增 internal/desktopui/，不动 core 与 adapters）

```
internal/desktopui/
├── theme/
│   ├── tokens.go      # base 色阶 hex 常量（dark/light 各一组）
│   ├── palette.go     # Palette struct：functional token 字段 + Dark()/Light() 构造
│   ├── typography.go  # 9 级字号/行高/字重常量 + material.Theme 装配 Shaper/字体
│   ├── metrics.go     # 间距 12 档、圆角 5 档、描边、阴影 E1–E4 参数
│   └── motion.go      # 时长 4 档 + cubic-bezier 缓动函数
├── anim/
│   └── tween.go       # Tween{Start, Duration, Ease}，每帧 gtx.Now 求值，结束自动停 Invalidate
├── icon/
│   └── icons.go       # ~40 个 Lucide 风格 SVG path 数据 + Draw(gtx, name, size, color)
└── widget/
    ├── button.go      # 5 变体 × 3 尺寸 + 加载态
    ├── navitem.go     # 导航项 + 选中指示条动画
    ├── listitem.go    # 双行列表项 + hover 行内操作
    ├── card.go        # 卡片 / 统计卡（count-up）
    ├── input.go       # Input / Select（带过滤 popover）/ Checkbox
    ├── badge.go       # 类型 badge + kbd 键帽
    ├── toast.go       # Toast 栈管理器（右下、计时、hover 暂停）
    ├── modal.go       # 模态 + backdrop + focus trap
    ├── cmdpalette.go  # 命令面板（模糊过滤、↑↓ 环绕）
    ├── empty.go       # Blankslate 空态 + 骨架屏 Skeleton
    └── spinner.go     # spinner / 进度点
```

### 6.2 关键实现约束

- material.NewTheme() **只用于** Shaper/字体装配；颜色一律走 theme.Palette，禁用手写字面量 hex（lint 断言 color.NRGBA{ 只出现在 desktopui/theme 包内）。
- 主题切换：Palette 存于原子值，切换时每个色 token 跑 250ms 插值 tween（§4-12）。
- 阴影：widget/shadow.go 用 clip.RRect + 3–4 层偏移渐变矩形近似 E1–E4；深色模式跳过阴影只画 border。
- 图标：SVG path 指令手工转 clip.Path（viewBox 24，1.5dp 描边用 clip.Stroke）。
- 单位：全部 unit.Dp（布局）/ unit.Sp（文字），禁止裸 Px。

### 6.3 迁移步骤（每步独立可编译可运行，对应替换 main.go）

| 步骤 | 内容 | 验收 |
|------|------|------|
| M1 | theme 包：双主题 token + 字体装配（纯数据，配单测） | go test ./internal/desktopui/theme 通过 |
| M2 | 基础组件：Button/Badge/Card/ListItem/Input，替换现有 material 裸控件 | 五页面功能不变，视觉全换 |
| M3 | 骨架：侧栏（导航项+指示条+状态卡）+ Header + 状态栏 | 232/56dp 折叠可用，Ctrl+1..5 切页 |
| M4 | 浮层：Toast 栈 + Modal + 命令面板 ⌘K | 消息条消失，全部走 toast |
| M5 | 动效与引导：18 条动画 + 空态 + 首次引导页 | 首启空仓库出引导；完成进仪表盘 |

---

## 附：与现状（被批评版）的对照

| 现状问题 | 本方案对策 |
|----------|------------|
| 默认蓝白底（material.NewTheme 直出） | §1.2 深色优先自定义双主题 token |
| 纯文字按钮导航 | §2.1 图标+文字导航项 + 选中指示条 + 快捷键 |
| 无图标 | §1.7 Lucide 线性图标体系，按钮/导航/列表/badge 全覆盖 |
| 无动画 | §4 共 18 条动画，时长缓动 token 化 |
| radio 导航与竖排 RadioButton 选工具 | §3.4 Select + 过滤 popover（Quick Pick 模式） |
| 无暗色主题 | §1.2 深色为默认主题，§4-12 主题切换 250ms 过渡 |
| 顶部文字消息条 | §3.6 右下 Toast 栈（语义色 icon + 倒计时条 + 操作钮） |
| 实体行裸文本 [kind] id note | §3.2 双行 ListItem + 类型 badge + hover 行内操作 |
