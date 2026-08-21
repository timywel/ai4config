# cfg4ai 任务看板（P0 / M1）

> 版本：v1.0（2026-08-17）｜ 维护规则：任务完成即勾选并注明实际工时与偏差；新增/变更任务先登记再动手（PROJECT-PLAN §7 变更流程）。
> 状态标记：`[ ]` 未开始 ｜ `[~]` 进行中 ｜ `[x]` 完成 ｜ 预估单位：人日

## 图例：DoD 引用

单个任务出口标准 = PROJECT-PLAN §5 门禁五条 + 本表"验收"列 + TEST-PLAN §5 责任矩阵。

---

## T0 工程基建（预估 0.5d）

- [x] **T0.1** git init + 首次提交（.gitattributes 已就位）｜ 验收：仓库可提交，CRLF 防护生效（2026-08-17；修正：误纳入的 .venv/.claude/.github 已移出索引）
- [x] **T0.2** Makefile/任务脚本：`build`/`test`/`lint`/`fmt`（封装 CGO_ENABLED=0 与便携 Go PATH）｜ 验收：四条命令可用（2026-08-17；Makefile + build.ps1 双平台）
- [x] **T0.3** e2e 脚手架（标准 testing+os/exec；testscript 仓库已下线，见 TEST-PLAN §3 变更记录）+ 首组用例跑通｜ 依赖：T0.2 ｜ 验收：`go test ./cmd/...` 执行用例（2026-08-17）

## T1 core/ir（预估 1.5d）

- [x] **T1.1** YAML 序列化/反序列化（yaml.v3 Node 层；frontmatter 拆分；x- 展开/收拢）｜ 验收：round-trip 单测（注释保留用例必含）（2026-08-17；codec.go + codec_test.go）
- [x] **T1.2** 12 条校验规则实现（IR-SCHEMA §5 逐条）｜ 依赖：T1.1 ｜ 验收：每条正反例单测（2026-08-17；validate.go + validate_test.go；发现并回写规格矛盾：secretref 字符集放行大写）
- [x] **T1.3** id 解析器（首个点号分隔 type；setting 三段式）｜ 验收：点号 name、大写、非法字符用例（2026-08-17；id.go + id_test.go）

## T2 core/profile（预估 2d）

- [x] **T2.1** profile 读写（manifest.yaml + 各实体文件布局）｜ 依赖：T1.1（2026-08-17；store.go，含 instructions/mcp/packs/hooks/settings 全类型，x- 扩展与正文保留，0600 原子写）
- [x] **T2.2** 五层合并物化：merge-by-id 浅字段级 + concat 两段式 + 墓碑遮蔽（IR-SCHEMA §2）｜ 依赖：T2.1 ｜ 验收：E0 全项（含遮蔽、防误判前提）（2026-08-17；merge.go，11 项合并单测全绿）
- [x] **T2.3** ir_version 链式迁移框架（占位 v1 恒等）｜ 验收：高于实现版本拒绝（2026-08-17；migrate.go + 版本超限/非法 policy 测试）

## T3 store（预估 2d）

- [x] **T3.1** 仓库初始化 + 目录 0700/文件 0600 校验修正 + 写锁 .lock（gofrs/flock）｜ 验收：并发竞写 e2e 剧本（2026-08-17；store.go，锁竞争单测通过）
- [x] **T3.2** 快照（manifest+blob 引用）与 restore 反向快照｜ 依赖：T3.1（2026-08-17；snapshot.go，创建/恢复/回收单测）
- [x] **T3.3** blob 内容寻址 + 脱敏管线（先扫描替换→落盘→零命中校验；双 hash）｜ 依赖：T4.1 ｜ 验收：红队 T-05 相关用例（2026-08-17；blob.go + secrets/sanitize.go）
- [x] **T3.4** 导出清单 exports/（读写+hash 规范化比对+rebase 接口）｜ 验收：外来内容三态判定单测（2026-08-17；exports.go，CRLF/BOM 规范化+rebase 单测）

## T4 secrets（预估 1.5d）

- [x] **T4.1** 三级后端降级链（99designs/keyring → age 文件 → none）+ secret_backend 记录｜ 验收：headless 模拟降级单测（2026-08-17；backend.go/file.go，age 加密往返+错误口令拒绝单测；CGO_ENABLED=0 纯 Go 验证）
- [x] **T4.2** 敏感扫描器（外置规则集+熵检测；结构化/自由文本分级处置；豁免清单）｜ 验收：规则命中/误报用例（2026-08-17；scan.go，gitleaks 规则集+熵检测+豁免）
- [x] **T4.3** 占位符回采保护（永不覆盖已有 secretref）｜ 验收：红队 T-03 用例（2026-08-17；protect.go）

## T5 platform/paths（预估 0.5d，可与 T1–T4 并行）

- [x] **T5.1** ExpandRaw/CollapseRaw 实现（~/、%APPDATA%、$XDG 变量双向）｜ 验收：三平台单测 + 往返一致（2026-08-17；paths_expand.go，全平台变量识别 + 最长前缀折叠）

## T6 atomicfile 完整化（预估 0.5d，可并行）

- [x] **T6.1** Windows 共享冲突指数退避重试（SHARING_VIOLATION/ACCESS_DENIED 分类）｜ 验收：占用注入单测（2026-08-17；x/sys/windows 错误分类 + 指数退避 + 占用路径报错）

## T7 adapters/claudecode（预估 2.5d）

- [x] **T7.1** Detect（全局+项目+managed 只读+进程检测）｜ 验收：样本环境探测单测（2026-08-17；detect.go/process.go）
- [x] **T7.2** Import：CLAUDE.md（含 @import/imports）、settings.json、.mcp.json、agents/commands/skills、hooks（31 事件）、rules、~/.claude.json 局部 patch 读取｜ 依赖：T1–T4 ｜ 验收：golden-file Import 全绿（2026-08-17；import.go/parse.go，全局/项目/local 三组用例）
- [x] **T7.3** Export：物化布局（ADAPTERS §3.1 导出布局列）、settings 数组拼接语义消化、局部 patch 写回｜ 依赖：T6、T9 ｜ 验收：golden-file Export + round-trip 自检无差异（白名单内）（2026-08-17；render.go/export.go，round-trip 字段级一致 + dry-run 不落盘）

## T8 adapters/codex（预估 2d）

- [x] **T8.1** Detect + Import：config.toml（全键含 mcp_servers 极性取反）、AGENTS.md 逐目录/subtree、AGENTS.override.md、hooks、profiles 映射｜ 验收：golden-file Import 全绿（2026-08-17；detect.go/import.go/parse.go，极性取反+timeout 换算+trusted-gate）
- [x] **T8.2** Export：TOML 整块重写+快照兜底策略、机器级键项目级跳过 Warning｜ 验收：golden-file 双向 + trusted-gate 用例（2026-08-17；render.go/export.go，极性/换算/机器级键/round-trip 5 组用例）

## T9 core/migrate 引擎（预估 2d）

- [x] **T9.1** 管线编排：Load→Merge→Map→Render→Verify→Write（W2 步骤 1–10）｜ 依赖：T2、T3（2026-08-17；engine.go）
- [x] **T9.2** Verify 两级 + 空集保护 + 外来内容四选项交互｜ 验收：红队 T-01 链路 e2e（2026-08-17；verify.go/foreign.go，空集保护+三态确认 Hooks）
- [x] **T9.3** 降级引擎（能力矩阵驱动两级规则 + Warnings 汇总）｜ 验收：降级用例（2026-08-17；degrade.go，workflow→command→instruction 附录链）
- 附：**架构修正**——WrittenFile 增 Content，写盘职责上收引擎（适配器纯渲染），修正外来内容检查时序缺陷

## T10 CLI 接入（预估 1.5d）

- [x] **T10.1** cobra 命令树：scan/collect/export/migrate/link/list/show/snapshot/restore/doctor/config（flag 与退出码按 CLI-SPEC §0–§11）｜ 依赖：T7–T9（2026-08-17；cmd/cfg4ai/cmd/ 全命令；link 最小实现并入 collect 项目 profile）
- [x] **T10.2** 输出格式化（text/json）+ Warnings→退出码 5 语义｜ 验收：e2e 剧本全命令（2026-08-17；退出码 0/1/2/3/4/5 + warnExit；真实二进制 scan/doctor 验证通过）

## T11 对抗回归（预估 1d）

- [x] **T11.1** P0 子集可执行化：T-01/T-03/T-05/T-06/T-07 + IR 表达力类（id 点号、@import 环、JSONC 密度）+ 文件系统类（symlink 农场、只读、长路径）｜ 验收：全部转 TestAdversarial_* 且绿（2026-08-17；11 个对抗测试全绿，覆盖 A1/A3/A4/A5/B1/B4/E3/F1/F5；**修复真实 bug：JSON 解析遇 BOM 报错**）
- [x] **T11.2** "文档澄清"类 12 条逐条核对记录（2026-08-17；adversarial-cases.md 附二，含状态表与新增变更候选）

## T12 P0 验收（预估 0.5d）

- [x] **T12.1** 里程碑评审（2026-08-18；review/P0-ACCEPTANCE.md：四项验收全过、门禁全绿、规格偏差 3 项记录在案）
- [x] **T12.2** 版本标记 v0.1.0（.goreleaser.yaml 六 target 静态构建配置就位）

---

## 变更候选区（未决策，勿动手）

| # | 提案 | 来源 | 状态 |
|---|------|------|------|
| C1 | watch 模式（文件变更自动采集） | ARCHITECTURE §13 开放问题 | 待评估 |
| C2 | profile 远程订阅（团队下发） | ARCHITECTURE §13 开放问题 | 待评估 |
| C3 | 权限/审批模型标准化（Roo groups 等） | RESEARCH-SUMMARY D17 | P3 候选 |
| C4 | 合并覆盖致继承键丢失时的 Warning（防静默损坏） | 对抗用例 AC-A4 | P1 候选 |
| C5 | GBK/非 UTF-8 编码探测与转换策略 | 对抗用例 AC-B3 | 待评估 |
| C6 | JSONC 解析器引入评估（VS Code settings 注释容忍） | 对抗用例 AC-B1 | P1 候选 |

## 缺陷区

| # | 现象 | 影响流程 | 状态 |
|---|------|---------|------|
| — | （暂无） | | |

---

# P1 阶段任务（M2，ARCHITECTURE §12 P1）

## T13 core/registry（项目关联与指纹，预估 2d）

- [x] **T13.1** registry.yaml 读写 + 项目注册表结构（projects: id/paths/fingerprint/profile/same_remote_as）｜ 验收：读写往返单测（2026-08-18）
- [x] **T13.2** 指纹计算：git remote 规范化（去协议/去 .git/host 小写/scp 转标准）+ root_name + first_commit｜ 验收：4 种 URL 形态归一用例（2026-08-18，8 形态全绿）
- [x] **T13.3** link/relink + 二次判别（first_commit 一致+确认才合并，否则新建 pid 记 same_remote_as）+ collect 路径命中指纹复核（D10）｜ 验收：AC-E1/AC-E2 对抗用例（2026-08-18，二次判别+路径劫持复核全绿）

## T14 adapters/copilot（预估 2d）

- [x] **T14.1** Detect + Import：.github/copilot-instructions.md、instructions/*.instructions.md（applyTo→file_patterns）、prompts、agents、mcp.json（servers+inputs）、settings.json
- [x] **T14.2** Export + golden-file 双向

## T15 adapters/zhanlu（预估 1.5d）

- [x] **T15.1** Detect + Import（zhanlu.json、AGENTS.md、.kilo/、~/.agents/skills；防御式探测）｜ 依赖本机实证校准
- [x] **T15.2** Export + golden-file 双向

## T16 adapters/gemini（预估 1.5d）

- [x] **T16.1** Detect + Import（settings.json ~240 键、GEMINI.md、.gemini/）+ Export
- [x] **T16.2** golden-file 双向 + Antigravity 时效跟踪

## T17 adapters/claude-desktop（预估 0.5d，轻量）

- [x] **T17.1** claude_desktop_config.json（mcpServers）采集/导出，与 claudecode 共享 MCP 适配代码

## T18 adapters/grokbuild（预估 1.5d）

- [x] **T18.1** Detect + Import（~/.grok/config.toml + 项目 .grok/、hooks 14 事件、skills）+ Export

## T19 secrets 完整化 + diff 命令（预估 1.5d）

- [x] **T19.1** collect 接入三级后端（keyring→file→none）+ secret_backend 记录 + 占位符回采保护接线
- [x] **T19.2** diff 独立命令（SSOT vs 磁盘现状 / profile 间）

## T20 P1 验收（预估 0.5d）

- [x] **T20.1** 七适配器 golden-file 全绿 + Claude→Copilot、Codex→Zhanlu e2e + relink 演示 + 麒麟 V10 冒烟
---

# P2 阶段任务（M3，ARCHITECTURE §12 P2）

## T21 core/aiassist（AI 辅助迁移，预估 2d）

- [x] **T21.1** Provider 接口 + OpenAI 兼容 HTTP 客户端（默认引导本地/私有端点）｜ 验收：mock provider 单测
- [x] **T21.2** consent 状态机（首次使用显式同意、AI 配置段变更强制重确认、决策日志）｜ 验收：变更重确认用例
- [x] **T21.3** 语义转换（skill 改写/冲突建议/语言适配）+ 脱敏（secret+内网地址）+ 引擎 Assist 步骤接入｜ 验收：脱敏用例 + export --ai e2e

## T22 sync（白名单同步，预估 2d）

- [x] **T22.1** sync init/push/pull/status（go-git 封装，白名单制 profiles/registry/config/exports）｜ 验收：bare 仓库双向同步用例
- [x] **T22.2** push 前置全仓敏感扫描（preflight，命中阻断）+ 换机 rebase 引导｜ 验收：红队 T-05 用例

## T23 TUI（预估 2d）

- [x] **T23.1** bubbletea 交互界面（collect/export 可视化确认 + diff 预览）

## T24 扩展适配器（预估 3d，调研卡片已备 research/tool-survey-a/b.md）

- [x] **T24.1** cursor（.cursor/rules/*.mdc、mcp.json）
- [x] **T24.2** windsurf→devin（.windsurf/ 与 .devin/ 双读）
- [x] **T24.3** aider（.aider.conf.yml、CONVENTIONS.md）
- [x] **T24.4** cline（.clinerules/、cline_mcp_settings.json）
- [x] **T24.5** roo（.roo/rules/、.roomodes）
- [x] **T24.6** opencode（opencode.json、AGENTS.md）

## T25 P2 验收（预估 0.5d）

- [x] **T25.1** AI 转换确认链完整演示 + sync preflight 阻断演示 + TUI 跑通 collect/export
---

# P3 阶段任务（M4，ARCHITECTURE §12 P3）

## T26 外置进程插件（适配器生态，预估 3d）

- [x] **T26.1** 插件协议定义（go-plugin net/rpc over stdio，Adapter 四方法 RPC 化）｜ 验收：协议接口单测
- [x] **T26.2** host 端：插件进程启动/调用/生命周期管理 + 注册到适配器注册表｜ 验收：示例插件进程互通
- [x] **T26.3** plugin 端 SDK：第三方实现 Adapter 的骨架包（任意语言可对等实现）+ 示例插件

## T27 GUI（桌面应用，预估 3d）

- [x] **T27.1** GUI 技术选型与骨架（Wails/Fyne 评估）+ 主界面（实体浏览 + collect/export 操作）

## T28 团队 profile 共享（预估 2d）

- [x] **T28.1** profile 远程订阅/下发（sync 白名单扩展 + 只读 remote 层采集）

## T29 P3 验收（预估 0.5d）

- [x] **T29.1** 第三方适配器插件接入示例跑通 + GUI 主界面可用 + 团队共享演示
---

# 优化线任务（OPT，依据 docs/OPTIMIZATION-PLAN.md）

## W-A 设计基座 + 最小可信（先行波次）

- [x] **OPT-A1** 修 goroutine 不重绘 bug（doXxx 完成后 w.Invalidate()）
- [x] **OPT-A2** 设计系统落地（internal/desktopui 包：双主题色板/字体/图标/卡片/Toast/Modal/组件库）
- [x] **OPT-A3** 导航改 component.NewNav 图标导航 + 工具选择改 chip 流式布局
- [x] **OPT-A4** F01 实体详情浏览（三栏+详情抽屉按类型渲染）
- [x] **OPT-A5** F02 过滤+全文搜索

## W-B 管理核心

- [x] **OPT-B1** F03 条目编辑（表单化+校验+防覆盖）
- [x] **OPT-B2** F04 新建/删除回收站/重命名级联
- [x] **OPT-B3** F05 启停+annotations.yaml 侧车+applies_to 矩阵
- [x] **OPT-B4** F08 secret 管理界面

## W-C 信任与治理

- [x] **OPT-C1** F06 漂移冲突处置视图
- [x] **OPT-C2** F07 历史时间线+条目级恢复
- [x] **OPT-C3** F14 健康看板+MCP 连通性
- [x] **OPT-C4** F15 审计日志时间线

## W-D / W-E（增强与效率，详见 OPTIMIZATION-PLAN §4）

- [x] **OPT-D1** F09 覆盖率视图（发现页）
- [x] **OPT-D2** F10 依赖关系图 + 断链检测
- [x] **OPT-D3** F11 批量操作
- [x] **OPT-D4** F12 加密备份包
- [x] **OPT-D5** F13 模板库
- [x] **OPT-D6** F16 sync GUI
- [ ] **OPT-E** F17-F23（命令面板/标签收藏/定时/审计看板等，按波次）

---

## 规格变更候选（走 PROJECT-PLAN §7 变更流程）

| # | 变更 | 涉及规格 |
|---|------|---------|
| S1 | 实体头增 disabled 字段 | IR-SCHEMA §1.1/§3 |
| S2 | annotations.yaml 侧车入规格 | IR-SCHEMA §1、CLI-SPEC |
| S3 | .cfg4aibak 备份包格式 | ARCHITECTURE §7、CLI-SPEC |
| S4 | doctor 结构化 HealthReport | CLI-SPEC §9 |