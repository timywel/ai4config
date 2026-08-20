# 配置/能力管理功能体系调研报告

> 调研档案（research/ 只增不改）｜日期：2026-08-20｜方法：7 个标杆官方文档 webfetch + cfg4ai 现有规格与代码只读盘点
> 服务目标：回应"需要完整的各方面配置和能力的管理功能"——定义一流配置管理工具的能力全集，映射为 cfg4ai 桌面应用（`cmd/cfg4ai-desktop`，Gio）与本地 Web GUI（`internal/gui`）的功能 backlog。

## 0. 结论速览（TL;DR）

- cfg4ai 的**数据面已是一流的**：IR 八类实体、五层 scope、墓碑遮蔽、脱敏管线、快照、写入协议、sync 白名单——严谨度对标 chezmoi。**管理面几乎空白**：桌面应用仅 5 页（仪表盘计数 / 实体三字段列表 / 采集 / 迁移 / 快照），`internal/gui` 仅 8 个 API。用户批评成立。
- 一流配置管理工具的能力全集可归纳为 **10 维度 43 个功能点**（§2 总矩阵）；cfg4ai 缺失 31 个，其中 8 个属"没有就不配叫管理工具"的 P0（§3）。
- 多数 P0/P1 **不需要动 IR v0.3 规格**：标签/收藏/泛化禁用可走 `profiles/<p>/annotations.yaml` 侧车（profiles/ 本就在 sync 白名单内）；审计走 `logs/audit.jsonl`（logs/ 本就不入库）；编辑/历史/冲突/备份全部复用现有引擎与存储能力。需走规格变更流程（PROJECT-PLAN §7）的仅 3 项（§6）。

## 1. 标杆功能速览

### 1.1 chezmoi（dotfiles 管理标杆）

- **三态模型**：source state（仓库）/ target state（期望）/ destination state（磁盘实况）；`status`/`diff` 预览、`apply` 幂等落盘——"先看后写"是管理工具的基本礼仪。
- **管理全生命周期命令面**：`add`（纳管）/ `re-add`（回流刷新）/ `edit`（编源）/ `edit --apply` / `forget`（脱管留盘）/ `unmanage` / `remove` / `destroy`（源+目标双删）/ `purge` / `managed` / `unmanaged` / `ignored`（**覆盖率三视图**）/ `cat` / `verify` / `archive`（导出 tar）/ `import`（导入归档）/ `merge` / `merge-all` / `doctor` / `state` / `upgrade`。
- **模板体系**处理机器差异（`.chezmoi.<format>.tmpl` + 50+ 模板函数 + init 交互 prompt 函数）；`.chezmoiexternal` 引用外部归档；`.chezmoiscripts` 的 `run_onchange_` 脚本做变更驱动自动化；`.chezmoiignore` / `.chezmoiremove` 声明排除与删除。
- **secret 不绑定单一后端**：20+ 密码管理器模板函数（1Password/Bitwarden/Vault/keyring…）+ age/gpg 全文件加密。
- 对 cfg4ai 启示：collect≈re-add、export≈apply 已同构；缺 **edit（管理面编辑）、managed/unmanaged 覆盖率视图、archive/import 可携带备份、destroy 级联语义**（墓碑+prune 已有底子）。

### 1.2 VS Code Profiles + Settings Sync（profile 管理标杆）

- **Profiles editor 单点管理**：新建（空 / 复制现有 / 官方模板六套：Python、Data Science、Doc Writer、Node、Angular、Java——扩展+设置+片段组合包）、图标、**预览（新窗口试跑后再创建）**、**内容子集勾选**（settings/keybindings/MCP servers/snippets/tasks/extensions 六类可裁剪，未选类回落默认 profile）、文件夹/工作区关联（打开目录自动切 profile）、Apply Setting/Extension to all Profiles（跨 profile 广播）、`.code-profile` 导出 / GitHub gist 分享（vscode.dev 链接打开**先预览再导入**）、Temporary Profile（一次性沙盒）、CLI `--profile`。
- **Settings Sync**：同步内容分类勾选；首次接入 **Merge / Replace Local / Merge Manually** 三选；冲突 **Accept Local / Accept Remote / Show Conflicts**（diff 编辑器逐项解决）；**本地+远程双备份视图**（Show Synced Data；本地备份按类型带时间戳、30 天，远程每资源留 20 版）；**Synced Machines 设备视图**（命名、远程踢下线）；machine 范围键不同步 + `ignoredSettings`/`ignoredExtensions` 排除清单；导出时机器相关键自动剥离。
- 对 cfg4ai 启示：内容子集勾选、冲突三向解决、备份版本浏览、设备管理，全是 GUI 可直接对标的交互。

### 1.3 JetBrains Settings Sync

- 分类打包同步（UI / Code / Tools / System 四大类勾选）+ 跨 IDE 产品共享；Push Settings to Account / Get Settings from Account 显式二选一。
- **插件状态同步规则极精细**：双端已装→同步启停；单端已装且启用→给对端安装；单端已装但禁用→不装；本端卸载→对端**禁用而非卸载**（可回滚的删除传播）。
- 手动 Export Settings → ZIP；Import 时**选择性勾选组件**。
- 对 cfg4ai 启示：启停语义与删除传播要"软"（禁用优先于删除）——cfg4ai 墓碑机制同思想，缺 GUI 呈现与条目级启停。

### 1.4 1Password / Bitwarden（敏感配置管理维度）

- **组织维度**：folders（嵌套）/ favorites / tags / vaults / collections；search + filter；custom fields；attachments。
- **健康看板**（Watchtower / Vault Health Reports）：弱密码、复用、泄露 breach、未开 2FA、http 不安全站点、**expiring items（到期提醒，API 凭据 3 个月阈值）**、重复项一键清理、**Developer secrets on disk（.env/SSH 私钥等明文落盘扫描）**；分类计数横幅 + 全应用 alert banner。
- **历史与审计**：password history（近 5）、item 版本历史可恢复、组织 Event Logs（谁何时查看/修改/分享）、manage devices（设备清单+踢出）、emergency access、**master password re-prompt（看敏感值二次确认）**。
- **导入导出**：encrypted export（口令加密）、20+ 来源导入器、CLI/SDK/API 全套。
- 对 cfg4ai 启示：secretref 三级后端架构正确；缺 **secret 管理界面（清单/年龄/dangling/re-prompt 查看）、磁盘明文风险看板（scanner 已有，只差聚合视图）、统一操作审计**。

### 1.5 Docker Desktop / Lens（资源管理界面组织标杆）

- Docker Desktop：按资源类型分视图（Containers/Images/Volumes）；列表**行内状态徽标 + 生命周期动作**（start/stop/restart/delete）+ 详情抽屉（logs/exec/inspect）。**MCP Catalog & Toolkit 与 cfg4ai 高度同构**：server 目录浏览、一键启用/禁用、凭据托管、客户端自动接线。
- Lens：Catalog（全部资源按类型分组+收藏）/ Hotbar（工作区快捷切换）/ 资源树 + 实时 metrics + port-forward 管理。
- 对 cfg4ai 启示：**资源类型分组 + 行内状态 + 详情抽屉 + 行内动作**是 Gio 实体页的直接版式参考；MCP server 的"启用/禁用 + 凭据 + 连通性"三件套是 MCP 管理卡片样板。

### 1.6 Raycast（扩展/命令管理标杆）

- **Root Search** 检索一切（命令/扩展/snippets/quicklinks），频度排序，alias/hotkey 二级寻址；Action Panel 承载条目全部动作（uninstall/configure/view source）。
- **Store**：浏览/搜索/分类，详情页（commands/screenshots/author），回车即装；Installed 过滤；后台自动更新 + 手动 Check for Extension Updates；**本地开发扩展不自动更新（来源分级）**。
- **Import/Export 教科书**：`.rayconfig` 单文件打包 11 类数据、**passphrase ≥8 位加密**、跨平台导入、**导入时分类勾选**、**冲突自动跳过**（quicklink 同链接跳过、snippet 同 keyword 跳过，其余合并不覆盖）；**Scheduled Exports**（日/周/月 + 保留最近 1/5/10 份 + 指向云同步盘目录）。
- 对 cfg4ai 启示：分类勾选导入、加密备份包、计划导出、命令面板，全部可照搬。

### 1.7 Homebrew / npm（包管理隐喻标杆）

- **能力生命周期**：install / update / uninstall + **enable/disable**（brew services）；状态三件套：list / **outdated**（可更新清单）/ **info**（含被禁用/弃用原因，`brew log` 给处分记录）。
- **版本纪律**：pin/unpin 钉版本；lockfile（npm）；cleanup 自动回收旧版本（30 天）；`auto_updates` 自我更新实体跳过升级（**更新来源分级**）。
- **审计与解释**：npm audit / audit fix、brew doctor、**npm explain（谁依赖它 = 影响面分析）**、npm prune（移除未声明依赖）。
- 对 cfg4ai 启示：条目启停、pin（不随采集回流更新）、影响面分析（谁引用了这个 mcp server）可直接借用包管理语汇。
## 2. 功能体系总矩阵（维度 × 功能点 × 标杆做法 × cfg4ai 现状差距）

> 现状判定依据：CLI-SPEC/ARCHITECTURE/IR-SCHEMA 规格 + `cmd/cfg4ai-desktop`（Gio 五页）与 `internal/gui`（8 API）代码实证。"CLI 有 / GUI 无"记为**半缺失**——桌面应用用户够不着。

### 2.1 内容的增删改查（CRUD）

| 功能点 | 标杆做法 | cfg4ai 现状差距 |
|--------|---------|----------------|
| 条目创建（空白/复制） | Bitwarden New Item；VS Code New Profile（空/复制/模板三源） | **缺失**：只能采集回流；全局→项目复制无 |
| 内容编辑 | chezmoi `edit`/`edit --apply`（编源即管源）；Bitwarden Edit | **缺失**：SSOT 只能手工改文件，GUI 无编辑入口 |
| 删除分级 | chezmoi forget/unmanage/remove/destroy 四级语义 | 半缺失：墓碑+prune 机制在（IR-SCHEMA §2.3），GUI 无删除入口、无回收站视图 |
| 启用/禁用 | JetBrains 插件启停（禁用优先于卸载）；brew services start/stop | 半缺失：仅 McpServer 有 `disabled`；instruction/skill/hook/setting 无禁用语义 |
| 组织分类（文件夹/标签/收藏） | Bitwarden folders（嵌套）/favorites；1Password tags/vaults | **缺失**：仅类型+五层 scope 两个硬维度 |
| 跨层复制/移动/广播 | VS Code 复制 profile、Apply Setting to all Profiles | **缺失** |
| 重命名（id 级联） | Bitwarden 重命名无感（内部 id 引用） | **缺失**：id=文件名（校验规则 7），改名涉及引用级联（skill.mcp_servers、secretref 路径），无工具支持 |

### 2.2 搜索与发现

| 功能点 | 标杆做法 | cfg4ai 现状差距 |
|--------|---------|----------------|
| 全文搜索 | Raycast Root Search（频度排序）；Bitwarden vault search | **缺失**：GUI 无搜索框，正文/description 不可检索 |
| 结构化过滤 facet | Bitwarden filter；VS Code 同步分类勾选 | 半缺失：CLI `list --type/--scope/--profile` 有，GUI 无 |
| 命令面板 | Raycast alias/hotkey；VS Code Command Palette | **缺失** |
| 未管理内容发现（覆盖率） | chezmoi `unmanaged`/`ignored`/`status` | 半缺失：`scan` 有雏形，无"磁盘有/SSOT 无"覆盖率视图与一键纳管 |
| 排序/最近使用 | 1Password recently used；Raycast 频度排序 | **缺失** |

### 2.3 版本与历史

| 功能点 | 标杆做法 | cfg4ai 现状差距 |
|--------|---------|----------------|
| 版本时间线浏览 | VS Code Show Synced Data 备份视图；Bitwarden password history | 半缺失：快照 list/create/restore 有，无内容预览、无两版本对比 |
| 任意版本 diff | chezmoi `diff` | 半缺失：CLI `diff` 有；**快照间 diff 无** |
| 条目级历史与单条恢复 | 1Password item 版本历史逐条恢复 | **缺失**：快照整库粒度，无法按 id 回溯/单条回滚 |
| 操作审计日志 | Bitwarden Event Logs；1Password activity log | 半缺失：仅 AI 决策日志（ARCHITECTURE §5.2），统一审计无 |
| 撤销/重做 | VS Code 备份恢复 | 半缺失：restore 有反向快照；无操作级 undo |
### 2.4 冲突与一致性

| 功能点 | 标杆做法 | cfg4ai 现状差距 |
|--------|---------|----------------|
| 漂移检测（SSOT vs 磁盘） | chezmoi `status`/`diff`（期望态 vs 实况对照） | 半缺失：CLI `diff --tool` 有（T19.2），GUI 无 |
| 冲突可视化解决 | VS Code Merge/Replace/Merge Manually + diff 编辑器逐项 Accept Local/Remote | 半缺失：外来内容四选项 CLI 有，GUI 无逐项 diff 解决器 |
| 多来源竞争可视（"谁在赢"） | VS Code Settings 编辑器显示来源层 | **缺失**：五层 merge 完善，merged 视图胜出层/被遮蔽层不可视 |
| 健康诊断 | chezmoi `doctor`；brew doctor | 半缺失：`doctor` 全量（CLI-SPEC §9），GUI 无结构化呈现与修复引导 |
| 同步冲突引导 | VS Code 冲突三选；JetBrains Push/Get 显式选择 | 半缺失：pull 冲突走标准 git（有意不封装），GUI 无冲突态提示与引导 |

### 2.5 关系与依赖

| 功能点 | 标杆做法 | cfg4ai 现状差距 |
|--------|---------|----------------|
| 依赖关系图 | Lens 资源树关联；npm `ls --all` | **缺失**：`skill.mcp_servers`、`instruction.imports` 引用已在 IR，无可视化 |
| 覆盖矩阵（条目×工具） | VS Code Profiles 内容子集勾选浏览 | **缺失**：`applies_to` 已有，无矩阵视图 |
| 孤儿/断链检测 | brew doctor（broken linkage）；npm prune | 半缺失：规则 9 管 imports 无环；mcp 引用悬空、secretref dangling（doctor 有）未进 GUI |
| 影响面分析（谁依赖它） | npm `explain`/why | **缺失**：改一个 mcp server 影响哪些 skill，无查询 |

### 2.6 批量与自动化

| 功能点 | 标杆做法 | cfg4ai 现状差距 |
|--------|---------|----------------|
| 批量启停/删除/迁移 | brew `upgrade` 全量；JetBrains 插件批量状态 | **缺失**：GUI 无多选；CLI 仅 migrate 整体级 |
| 规则/策略驱动 | chezmoi `.chezmoiignore`/`.chezmoiremove`/`run_onchange_`；VS Code `ignoredSettings/ignoredExtensions` | 半缺失：sync 白名单内建；用户可配的采集排除/导出过滤规则界面无 |
| 定时自动化 | Raycast Scheduled Exports（日/周/月+保留 1/5/10+指向云盘）；brew 30 天自动 cleanup | 半缺失：快照 retention 有；定时快照/采集无（watch 在开放问题清单） |
| 批量字段编辑（广播） | VS Code Apply Setting to all Profiles | **缺失** |
### 2.7 备份与恢复

| 功能点 | 标杆做法 | cfg4ai 现状差距 |
|--------|---------|----------------|
| 可携带加密备份包 | Raycast `.rayconfig`（passphrase ≥8 加密、11 类打包、跨平台）；Bitwarden encrypted export；chezmoi `archive` | **缺失**：快照在库内不可携带；sync 依赖 git 远端，无离线载体 |
| 选择性导入 | Raycast 分类勾选 + 冲突自动跳过（同链接/同 keyword 跳过，其余合并不覆盖）；JetBrains Import 勾选组件 | **缺失**：无导入概念（restore 整库覆盖） |
| 跨设备同步管理 | VS Code 分类勾选 + Synced Machines 设备视图；Raycast Cloud Sync | 半缺失：sync push/pull/preflight/rebase CLI 全有，GUI 无；设备视图无 |
| 灾难恢复引导 | chezmoi `init --apply` 空机一条命令 | 半缺失：doctor 换机 rebase 引导 CLI 有，GUI 向导无 |

### 2.8 模板与预设

| 功能点 | 标杆做法 | cfg4ai 现状差距 |
|--------|---------|----------------|
| 条目模板库 | VS Code Profile Templates 六套（创建时可浏览内容再确认） | **缺失**：新建 skill/instruction 无模板起点 |
| 一键预设/能力包 | Raycast Store（回车即装）；Docker MCP Catalog 一键启用 | **缺失**：常用 MCP server 配方（filesystem/github/postgres…）无预置 |
| 团队规范下发 | VS Code gist 分享+URL 导入预览；Raycast Teams；Bitwarden collections | 半缺失：remote scope 采集与 T28 共享就位，GUI 无下发管理台 |
| 变量参数化 | chezmoi 模板函数（`promptStringOnce` 等 init 交互） | **缺失** |

### 2.9 安全与权限

| 功能点 | 标杆做法 | cfg4ai 现状差距 |
|--------|---------|----------------|
| secret 清单与健康视图 | 1Password Watchtower（weak/reused/breach/expiring）；Bitwarden Vault Health Reports | **缺失**：secretref 三级后端/dangling 检测有机制，无清单/年龄/后端状态界面 |
| 敏感值查看二次确认 | Bitwarden master password re-prompt | **缺失** |
| AI 自动执行权限审计 | Docker MCP Toolkit 按 server 授权+凭据隔离；Raycast AI Extensions 工具白名单页 | **缺失**：`mcp.trust`/`auto_approve`、hook 命令、settings permissions 数据都在 IR，无聚合审计视图 |
| 磁盘明文风险看板 | 1Password Developer Watchtower（.env/SSH 私钥落盘扫描） | 半缺失：敏感扫描器（gitleaks 规则+熵）与 sync preflight 有，无"哪些工具配置区仍有明文"看板 |
| 到期/轮换提醒 | 1Password expiring items（API 凭据 3 个月阈值） | **缺失** |

### 2.10 可观测性

| 功能点 | 标杆做法 | cfg4ai 现状差距 |
|--------|---------|----------------|
| 健康评分与趋势 | 1Password Watchtower 分类计数横幅；brew doctor | 半缺失：doctor 检查全量，无评分/分级呈现 |
| 失效检测（连通性） | Docker 容器状态徽标+日志；Lens 实时 metrics；brew outdated | **缺失**：版本护栏 Detect 有；MCP 命令存在性/HTTP 连通性探测无 |
| 使用统计 | Raycast 频度排序；brew analytics；1Password recently used | **缺失** |
| 仪表盘 | Docker Desktop 资源总览；Lens 工作区 | 半缺失：GUI 仪表盘仅 3 个计数（tools/entities/snapshots） |
## 3. cfg4ai 缺失功能优先级清单（按用户价值排序）

排序原则：① "管理"本义优先（看→改→删建→开关→一致性→后悔药→secret）；② 纯 GUI 包装现有引擎能力的排前（投入小见效快）；③ 需要新数据面能力的排后；④ 每级内部按"缺失时的痛感"排序。

| ID | 功能 | 维度 | 用户价值（一句话） | 优先级 |
|----|------|------|------------------|--------|
| F01 | 实体详情浏览（内容级） | CRUD | 看得见才算管得了——现在实体页只有 kind/id/note 三字段 | **P0** |
| F02 | 结构化过滤 + 全文搜索 | 搜索发现 | 百级条目下定位配置的刚需 | **P0** |
| F03 | 条目编辑（表单化 frontmatter + 正文） | CRUD | 管理的第一动词；目前只能去文件系统改 | **P0** |
| F04 | 新建/删除/重命名（回收站） | CRUD | 不依赖任何 AI 工具即可从零搭配置；删除走墓碑可恢复 | **P0** |
| F05 | 启用/禁用 + applies_to 勾选 | CRUD/批量 | "能力开关"是配置管理的本义（包管理隐喻） | **P0** |
| F06 | 漂移检测与冲突处置视图 | 冲突一致 | 磁盘被工具改、SSOT 被手工改，没视图就是盲飞 | **P0** |
| F07 | 历史时间线 + 条目级 diff/恢复 | 版本历史 | 改坏了能回到"上一个好的"，且能只回滚一条 | **P0** |
| F08 | secret 管理界面 | 安全 | secretref 机制没有 UI 等于不存在 | **P0** |
| F09 | 未管理内容发现（覆盖率视图） | 发现 | "还有哪些配置没纳管"，一键采集 | P1 |
| F10 | 依赖关系图 + 断链检测 | 关系 | skill 引用的 mcp 还在不在，一眼看穿 | P1 |
| F11 | 批量操作（多选启停/删除/导出/标签） | 批量 | 治理性操作的效率门槛 | P1 |
| F12 | 加密备份包导出/选择性导入 | 备份 | 换机/分享的离线载体，不依赖 git 远端 | P1 |
| F13 | 模板库与预设包 | 模板 | 新条目从模板起步；团队规范落地抓手 | P1 |
| F14 | 健康看板 + MCP 连通性检测 | 可观测 | doctor 从文本升级为评分+处置入口 | P1 |
| F15 | 审计日志时间线 | 版本/安全 | 谁在何时改了什么（含 AI 决策回放） | P1 |
| F16 | sync GUI + 换机引导向导 | 备份/同步 | push/pull/preflight/rebase 全图形化 | P1 |
| F17 | 命令面板 | 搜索发现 | 高频操作的效率入口 | P2 |
| F18 | 标签/收藏 | CRUD 组织 | 自定义组织维度（条目量级上来后） | P2 |
| F19 | 定时快照/定时采集 | 自动化 | 无人值守保障 | P2 |
| F20 | AI 自动执行权限审计看板 | 安全 | trust/auto_approve/hook 命令聚合审视 | P2 |
| F21 | 磁盘明文风险看板 | 安全 | Watchtower developer-secrets 对标 | P2 |
| F22 | 使用统计/最近使用排序 | 可观测 | 排序与推荐依据 | P2 |
| F23 | 团队预设下发管理台 | 模板 | remote 层内容的策展与订阅管理 | P2 |
## 4. P0 功能详细设计要点

> 版式参照 Docker Desktop/Lens：左侧类型树 + 中间列表（行内状态徽标）+ 右侧详情抽屉。全部 P0 不新增外部依赖（守住 CGO_ENABLED=0）。

### F01 实体详情浏览（内容级）

- **交互**：实体页三栏。详情抽屉按类型渲染：instruction→Markdown 渲染 + frontmatter 表格；mcp→字段表（env 的 secretref 显示为 🔒 占位，不明文回显）；skill/agent→skill.yaml 表单化只读视图 + prompt.md 预览 + assets 清单；hook→event/matcher/handler 卡片；setting→key-value 树。头部徽标：scope、origin.tool、applies_to、墓碑态、最近采集时间。
- **数据模型**：展示层 DTO `EntityDetail{ Entity, SourcePath string（ExpandRaw 后真实路径）, RawSize int, WinnerScope Scope（merged 视图胜出层）, MaskedBy []Scope（被遮蔽层） }`——顺带解决 2.4"谁在赢"可视。
- **衔接**：`core/profile` 读 + merge 物化视图；`platform/paths` ExpandRaw；secretref 永不解析回显。零引擎改动。

### F02 结构化过滤 + 全文搜索

- **交互**：顶部搜索框（匹配 id/name/description/正文，结果高亮命中片段）；左侧 facet 面板：类型 / scope / origin.tool / activation / 墓碑态 / 禁用态；组合过滤即时生效；URL/状态可分享（桌面内跳转链接）。
- **数据模型**：条目量级 <10⁴，内存线性扫描 + 前缀索引足够；**不引入 bleve**（依赖纪律）。查询结构 `Filter{ Types, Scopes, Tools, Text, States }`。
- **衔接**：纯展示层；过滤状态同步给 F11 批量操作做选择集。

### F03 条目编辑（表单化 frontmatter + 正文）

- **交互**：详情页"编辑"进入编辑态：frontmatter 按类型生成表单（枚举下拉、数组 chips、secretref 字段专用"设值"按钮——写 keyring 不明文）；instruction/skill 正文给 Markdown 编辑区（等宽+预览切换）。保存 → 12 条校验 → 失败逐条定位行号；成功写回并记审计事件。**防覆盖语义**：保存前比对磁盘 profile 文件 mtime/hash，被外部改则提示"重新加载/强制覆盖"。
- **数据模型**：写回现有 profile 布局（instructions/<name>.md 等）；更新 `origin.stored_hash`；审计事件 `op=edit` 入 `logs/audit.jsonl`。**采集回流保护**：collect 增量凭 `origin.raw_hash`——源文件未变时 SSOT 手工编辑不被覆盖；源变了则按现有 reconcile 覆盖，但 diff 摘要中新增标注"⚠ 该条目含本地编辑（stored_hash≠上次采集值）"。
- **衔接**：`core/ir` validate（12 条）；`core/profile` 写（0600 原子写已有）；W1[7] diff 摘要加一个标注来源。

### F04 新建 / 删除 / 重命名（回收站）

- **交互**：新建向导：选类型→（F13 模板或空白）→id 自动规范化生成（校验规则 1/7 实时预检）。删除=标记 `tombstone: true` 进"回收站"过滤视图（Bitwarden 已删除项 30 天可恢复同款心智）；恢复=去墓碑；彻底删除=走现有 `prune`（级联清 keyring 孤儿已有）。重命名=新 id 实体 + 旧 id 墓碑 + **级联引用改写**：扫全 profile 改写 `skill.mcp_servers` 引用与 secretref 路径，弹窗列出受影响条目确认。
- **数据模型**：墓碑/级联清理复用 IR-SCHEMA §2.3/§3.6；回收站=墓碑过滤视图，零新存储。
- **衔接**：merge 遮蔽语义不变；删除传播到导出由现有管线自动生效（墓碑不导出）。
### F05 启用/禁用 + applies_to 勾选

- **交互**：列表行内开关（Docker 容器 start/stop 心智）。MCP 直接读写 `disabled` 字段（IR 已有）；instruction/skill/hook/setting 的禁用走侧车文件。详情页 applies_to 工具复选框组；新增"矩阵视图"页：行=条目、列=工具、格子=勾选（VS Code 内容子集勾选的矩阵化）。
- **数据模型**：新增 `profiles/<p>/annotations.yaml` 侧车：

```yaml
disabled: [skill.legacy-review, hook.old-guard]   # 泛化禁用清单
labels: { mcp.filesystem: [核心, 只读] }           # F18 复用
favorite: [instruction.coding-style]
pinned: [setting.codex.model]                      # 钉住：collect 回流不更新（brew pin 隐喻）
```

  - 侧车在 `profiles/` 内 → **天然落入 sync 白名单，零规格变更**；长期可升级实体头 `disabled` 字段（规格变更候选，§6）。
- **衔接**：引擎 merge 后、Map 前加一步 filter（读 annotations 剔除 disabled）；pinned 在 collect reconcile 时跳过该 id 更新；applies_to 过滤已有（export 条目过滤语义）。

### F06 漂移检测与冲突处置视图

- **交互**："一致性"页：每工具一张卡——SSOT vs 磁盘差异计数 + 最近比对时间；进入后按目标文件分组，每文件三态徽标（一致 / SSOT 新 / 磁盘新 / 双方改）；diff 视图（sergi/go-diff 已有依赖）；逐项动作：**采集回流**（collect 单项）/ **导出覆盖**（backup-overwrite）/ **忽略**（记 ignored，不再提示）。批量动作条复用外来内容四选项语义。
- **数据模型**：复用 `store/exports.go` 三态判定（外来/被改/在清单）+ `diff --tool` 核心；忽略清单进 annotations.yaml `ignored_drift: [tool:path]`。
- **衔接**：T19.2 diff 引擎、W2[7] 外来内容四选项（GUI 化为逐项复选）；不写盘时不持锁（读快照读）。

### F07 历史时间线 + 条目级 diff/恢复

- **交互**："历史"页：时间轴（快照节点：时间/note/条目数/触发操作）；选两节点 → 按 id 分组 diff（新增/删除/修改计数 + 逐条内联 diff）。条目详情页"历史"标签：跨快照回溯该 id 的版本链 → 任选版本"恢复此版本到当前 profile"（单条回滚，弹窗预览 diff 确认）。
- **数据模型**：`snapshots/<ts>/manifest` + blob 引用已有；条目历史=按 id 扫快照 manifest → blob 内容 → 序列化规范化后 diff（sergi/go-diff）。
- **衔接**：`store/snapshot.go` 已有整库 create/restore（含反向快照）；新增**条目级 restore**：从快照 blob 读单条目 → 校验 → atomicfile 写回 profile → 记审计。恢复前对当前 profile 打反向快照（复用现有语义）。

### F08 secret 管理界面

- **交互**："密钥"页：secretref 清单表（引用路径 → 所属实体/字段、后端徽标 keyring/file/none、年龄 = now − collected_at）；dangling 红显（doctor 清单复用）；行操作：**设值/轮换**（输入即写后端，永不回显明文）、**查看**（re-prompt 确认框后显示 10 秒自动遮蔽）、跳转所属实体。年龄 >90 天黄色、>180 天红色（1Password expiring 心智）。顶部横幅：当前后端与降级链状态。
- **数据模型**：扫全 profile 提取 `secretref://cfg4ai/<profile>/<entity>/<field>`；`core/secrets` backend Get/Set 已有；2KB 上限与字符集校验已有。
- **衔接**：IR-SCHEMA §3.6（级联清理/回采保护）不动；doctor dangling 输出复用；设值记审计（`op=secret-rotate`，不记值）。
## 5. P1 功能详细设计要点

### F09 未管理内容发现（覆盖率视图）

- **交互**："发现"页：对每适配器跑只读 Detect+Import（不落盘），按工具列出"磁盘条目 vs 已纳管条目"覆盖率进度条；未纳管条目列表（chezmoi `unmanaged` 心智）；行内"采集此项"按钮（定向 collect --tool --path）；已知不采集项（运行时锁文件、OAuth 会话等）灰显"有意排除"及原因（ADAPTERS 各工具排除清单已有文字记录，需结构化）。
- **数据模型**：`Coverage{ Tool, Location, DiskIDs [], ManagedIDs [], Excluded[]{ID, Reason} }`；Import 结果仅内存比对，不写 SSOT。
- **衔接**：适配器 Detect/Import 零改动；ADAPTERS.md 各工具的"不采集"清单需升级为机器可读（适配器 Meta 增 `Exclusions []`——适配器接口内聚变更，非 IR 变更）。

### F10 依赖关系图 + 断链检测

- **交互**："关系"页全局图 + 条目详情"关系"标签局部图。节点=实体（类型着色），边三类：references（skill.mcp_servers→mcp id）、imports（instruction @import 链）、applies_to（→工具虚拟节点）。断链节点红显；点边跳转对端。影响面查询：选中 mcp → "被哪些 skill/agent 引用"列表（npm explain 心智）。
- **数据模型**：内存图 `Graph{ Nodes map[id]Entity; Edges []Edge{From, To, Kind} }`；断链=引用 id 在 merged 视图不存在（含被墓碑遮蔽）。
- **衔接**：校验规则 9（imports 无环）已有基础；断链检测结果接入 doctor 与 F14 健康看板。

### F11 批量操作

- **交互**：列表多选（Shift/Ctrl）→ 底部批量动作条：启用/禁用（F05 通道）、删除（墓碑，回收站可见）、加标签（F18）、改 applies_to、导出到工具（复用 export，多条目过滤参数）。动作前统一弹"影响预览"（条目数+类型分布+目标）。
- **数据模型**：批量=持锁单事务：全部校验通过才落盘；任一条失败整批回滚 + 逐条错误报告（复用写入协议的快照补偿，ARCHITECTURE §5.3）。
- **衔接**：退出码语义沿用（部分成功=5）；审计记批量事件（targets 列表）。

### F12 加密备份包导出 / 选择性导入

- **交互**：导出向导：范围（全库 / 按 profile / 按类型 / 按工具）→ 设口令（≥8 位，Raycast 同款纪律）→ 生成 `<name>.cfg4aibak`；导入向导：选文件 → 口令 → **内容清单勾选**（按类型/工具分组的树，Raycast 分类勾选）→ 冲突策略三选（跳过重复 / 覆盖 / 按 id 合并且保留本地较新）→ 执行预览 → 落盘。
- **数据模型**：备份包 = tar.gz{ `manifest.yaml`（备份格式版本、时间、来源机器、条目统计、内容 hash 清单）+ 白名单内容子集 }，整体 age passphrase 加密（filippo.io/age 已是依赖）。冲突判定：同 id 且同 stored_hash → 自动跳过（Raycast 规则）；同 id 不同 hash → 按所选策略。
- **衔接**：sync 白名单语义复用（哪些内容可出库）；导入走 collect 同款敏感扫描+校验管线；blob 引用悬空按 D13 降级链。

### F13 模板库与预设包

- **交互**：新建向导第二页"从模板"：内置模板卡（code-review skill、commit command、安全红线 instruction、常用 MCP 配方 filesystem/github/postgres/sqlite/playwright）+ 预览 + 变量填空表单（项目名/语言/路径）；预设包页："前端/后端/数据"组合一键生成一组条目进指定 profile（VS Code Profile Templates 心智，创建前可浏览内容清单）。
- **数据模型**：内置模板 embed 进二进制（零依赖）；用户模板存 `profiles/_templates/`（白名单内，可同步分享）；模板=普通 IR 实体 + `{{var}}` 占位 + `template.yaml`（变量声明：name/description/default/required——chezmoi prompt 函数的声明式版）。
- **衔接**：产物走 F04 新建管线（校验+审计）；预设包=一组模板+应用顺序，无新引擎语义。
### F14 健康看板 + MCP 连通性检测

- **交互**：仪表盘升级：健康分（0-100，doctor 检查项加权）+ 分类计数卡（校验失败 / secret dangling / 漂移项 / 墓碑数 / blob 悬空 / stale lock），点卡跳对应处置页。MCP 卡片"检测"按钮：stdio→命令存在性（lookPath）+ 启动握手（超时 5s，只拉起不调用业务方法）；http/sse→HEAD/GET 连通性（超时+状态码）。结果徽标：可达/命令缺失/超时/拒绝。
- **数据模型**：`HealthReport{ Score int; Categories []{Key, Count, Severity, JumpTarget}; MCPProbes []{ID, Status, Latency, Error} }`；探测结果缓存（5 分钟）不持久化敏感信息。
- **衔接**：doctor 检查项结构化输出（现在输出文本，需拆出机器可读结构——CLI 内部重构，非规格变更）；探测**只读纪律**：不进 export 管线、不触发目标工具写入。

### F15 审计日志时间线

- **交互**："活动"页：倒序时间线（操作徽标 collect/export/edit/delete/restore/sync/ai-consent/snapshot），每行：时间、操作、条目数、Warnings 数、结果；按操作/工具过滤；点击展开 targets 明细；AI 决策（consent、--ai-approve）特殊着色可回放。
- **数据模型**：`logs/audit.jsonl`（append-only；logs/ 本就不入库，无泄漏面）：`{ts, op, actor: user|ai, profile, targets: [ids], detail{}, warnings, result}`；单文件 >10MB 滚动。
- **衔接**：AI 决策日志（ARCHITECTURE §5.2）统一到此通道；各操作埋点点位：W1[8]/W2[10]/W5/F03/F04/F05/F11/F12。

### F16 sync GUI + 换机引导向导

- **交互**："同步"页：远端地址配置、push/pull 按钮、状态卡（ahead/behind/冲突态/最近同步时间）；preflight 命中 → 敏感清单逐项展示 + 豁免确认（阻断语义不豁免，CLI-SPEC §8）；pull 后失配 → **换机引导向导**（检测 exports/registry 路径与指纹失配 → 展示映射预览 → 一键 rebase）；多机视图：registry paths 历史 + 每机同步状态（VS Code Synced Machines 简化版）。
- **数据模型**：`store/sync.go` 已有 init/push/pull/status + rebase 接口；GUI 仅编排。
- **衔接**：W4 流程不动；CI 不放行 sync 退出码 5 的纪律在 GUI 表现为"preflight 命中时禁用推送按钮直至处置"。
## 6. 落地路径与规格变更候选

### 6.1 实施顺序建议（与现行 P 阶段路线图正交，属"管理面"新线）

| 波次 | 内容 | 理由 |
|------|------|------|
| W-A | F01+F02+F05+F06 | 纯读+轻写，复用现有引擎最多，"看得见、改得了、信得过"最小闭环 |
| W-B | F03+F04+F07+F08 | 写入类 CRUD 与后悔药，依赖 F05 的 annotations 侧车约定 |
| W-C | F09+F11+F14+F15 | 治理面（覆盖/批量/健康/审计） |
| W-D | F10+F12+F13+F16 | 关系图、备份包、模板、同步界面 |
| W-E | P2 全部 | 效率与增强 |

### 6.2 需走规格变更流程（PROJECT-PLAN §7）的事项

| # | 变更 | 涉及规格 | 说明 |
|---|------|---------|------|
| S1 | 实体头增加 `disabled: bool`（泛化禁用语义） | IR-SCHEMA §1.1/§3 | 短期可用 annotations 侧车规避；长期应入 IR 使禁用参与 merge/导出语义 |
| S2 | annotations.yaml 入规格（labels/favorite/pinned/disabled 侧车） | IR-SCHEMA §1、CLI-SPEC | 即使 S1 采纳，labels/favorite/pinned 仍需侧车或实体头字段 |
| S3 | 备份包格式（.cfg4aibak manifest） | ARCHITECTURE §7、CLI-SPEC | 新命令 `backup export/import`；复用白名单与脱敏纪律 |
| S4 | doctor 输出结构化（HealthReport） | CLI-SPEC §9 | 内部重构为主，规格补一句输出契约 |

### 6.3 明确不建议做的（避免范围蔓延）

- **实时双向同步 / watch 模式**：ARCHITECTURE §1.3 已定非目标；F19 定时快照用系统计划任务即可，不内建守护进程。
- **配置继承超出现有五层**（profile 间继承，VS Code 官方也未支持，issue #188612）：现有五层 scope 已更强，不追。
- **secret 明文托管**：红线不动（ARCHITECTURE §1.3）。
- **GUI 内嵌 git 合并器**：sync 冲突走标准 git 流程是有意边界（CLI-SPEC §8），GUI 只做检测与引导。

### 6.4 与用户批评的对应关系

"完整的各方面配置和能力的管理功能"拆解后 = §2 的 43 个功能点；其中 P0 八项交付后，桌面应用即从"采集+导出壳"升级为可独立管理 SSOT 的工具（浏览/搜索/编辑/建删/开关/冲突/历史/密钥）；P1 八项交付后具备治理与迁移增强面（覆盖率/批量/依赖/备份包/模板/健康/审计/同步）；P2 为效率与合规增强。

---

## 附：调研来源

| 对象 | URL | 关键汲取 |
|------|-----|---------|
| chezmoi | https://www.chezmoi.io/ + /user-guide/command-overview/ | 命令全集（add/edit/status/diff/apply/forget/unmanaged/archive/import/doctor）、三态模型、模板体系、20+ secret 后端 |
| VS Code Profiles | https://code.visualstudio.com/docs/editor/profiles | Profiles editor、内容子集勾选、预览、模板六套、gist 分享、文件夹关联、Apply to all Profiles |
| VS Code Settings Sync | https://code.visualstudio.com/docs/editor/settings-sync | Merge/Replace/Manual 三选、冲突 diff 解决、本地+远程备份视图（20 版/30 天）、Synced Machines、ignored 清单 |
| JetBrains | https://www.jetbrains.com/help/idea/sharing-your-ide-settings.html | 分类同步、插件状态精细规则（卸载→对端禁用）、ZIP 选择性导入 |
| Bitwarden | https://bitwarden.com/help/（webvault/getting-started 及导航全集） | folders/favorites/filter、vault health reports、password history、event logs、encrypted export、CLI/SDK |
| 1Password | https://support.1password.com/watchtower/ | Watchtower 分类（breach/weak/reused/expiring/duplicates/developer secrets on disk）、alert banner |
| Raycast | https://manual.raycast.com/extensions + /import-export | Store 浏览安装、自动更新分级、.rayconfig 加密导出、分类勾选导入、冲突自动跳过、Scheduled Exports |
| Docker Desktop / Lens | https://docs.docker.com/desktop/ | 资源分视图、行内状态+生命周期动作、MCP Catalog/Toolkit（同构参照） |
| Homebrew / npm | https://docs.brew.sh/FAQ | pin/outdated/cleanup 30 天/auto_updates 跳过、info 给弃用原因、audit/explain 语汇 |