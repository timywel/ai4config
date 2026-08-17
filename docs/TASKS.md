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

- [ ] **T3.1** 仓库初始化 + 目录 0700/文件 0600 校验修正 + 写锁 .lock（gofrs/flock）｜ 验收：并发竞写 e2e 剧本
- [ ] **T3.2** 快照（manifest+blob 引用）与 restore 反向快照｜ 依赖：T3.1
- [ ] **T3.3** blob 内容寻址 + 脱敏管线（先扫描替换→落盘→零命中校验；双 hash）｜ 依赖：T4.1 ｜ 验收：红队 T-05 相关用例
- [ ] **T3.4** 导出清单 exports/（读写+hash 规范化比对+rebase 接口）｜ 验收：外来内容三态判定单测

## T4 secrets（预估 1.5d）

- [ ] **T4.1** 三级后端降级链（99designs/keyring → age 文件 → none）+ secret_backend 记录｜ 验收：headless 模拟降级单测
- [ ] **T4.2** 敏感扫描器（外置规则集+熵检测；结构化/自由文本分级处置；豁免清单）｜ 验收：规则命中/误报用例
- [ ] **T4.3** 占位符回采保护（永不覆盖已有 secretref）｜ 验收：红队 T-03 用例

## T5 platform/paths（预估 0.5d，可与 T1–T4 并行）

- [ ] **T5.1** ExpandRaw/CollapseRaw 实现（~/、%APPDATA%、$XDG 变量双向）｜ 验收：三平台单测 + 往返一致

## T6 atomicfile 完整化（预估 0.5d，可并行）

- [ ] **T6.1** Windows 共享冲突指数退避重试（SHARING_VIOLATION/ACCESS_DENIED 分类）｜ 验收：占用注入单测

## T7 adapters/claudecode（预估 2.5d）

- [ ] **T7.1** Detect（全局+项目+managed 只读+进程检测）｜ 验收：样本环境探测单测
- [ ] **T7.2** Import：CLAUDE.md（含 @import/imports）、settings.json、.mcp.json、agents/commands/skills、hooks（31 事件）、rules、~/.claude.json 局部 patch 读取｜ 依赖：T1–T4 ｜ 验收：golden-file Import 全绿
- [ ] **T7.3** Export：物化布局（ADAPTERS §3.1 导出布局列）、settings 数组拼接语义消化、局部 patch 写回｜ 依赖：T6、T9 ｜ 验收：golden-file Export + round-trip 自检无差异（白名单内）

## T8 adapters/codex（预估 2d）

- [ ] **T8.1** Detect + Import：config.toml（全键含 mcp_servers 极性取反）、AGENTS.md 逐目录/subtree、AGENTS.override.md、hooks、profiles 映射
- [ ] **T8.2** Export：TOML 整块重写+快照兜底策略、机器级键项目级跳过 Warning｜ 验收：golden-file 双向 + trusted-gate 用例

## T9 core/migrate 引擎（预估 2d）

- [ ] **T9.1** 管线编排：Load→Merge→Map→Render→Verify→Write（W2 步骤 1–10）｜ 依赖：T2、T3
- [ ] **T9.2** Verify 两级 + 空集保护 + 外来内容四选项交互｜ 验收：红队 T-01 链路 e2e
- [ ] **T9.3** 降级引擎（能力矩阵驱动两级规则 + Warnings 汇总）｜ 验收：降级用例

## T10 CLI 接入（预估 1.5d）

- [ ] **T10.1** cobra 命令树：scan/collect/export/migrate/link/list/show/snapshot/restore/doctor/config（flag 与退出码按 CLI-SPEC §0–§11）｜ 依赖：T7–T9
- [ ] **T10.2** 输出格式化（text/json）+ Warnings→退出码 5 语义｜ 验收：e2e 剧本全命令

## T11 对抗回归（预估 1d）

- [ ] **T11.1** P0 子集可执行化：T-01/T-03/T-05/T-06/T-07 + IR 表达力类（id 点号、@import 环、JSONC 密度）+ 文件系统类（symlink 农场、只读、长路径）｜ 验收：全部转 TestAdversarial_* 且绿
- [ ] **T11.2** "文档澄清"类 12 条逐条核对记录

## T12 P0 验收（预估 0.5d）

- [ ] **T12.1** 里程碑评审：ARCHITECTURE §12 P0 四项验收逐条演示 + TEST-PLAN §6.2 CI 门禁全绿 + 规格偏差清零
- [ ] **T12.2** 版本标记 v0.1.0（goreleaser 配置就位）

---

## 变更候选区（未决策，勿动手）

| # | 提案 | 来源 | 状态 |
|---|------|------|------|
| C1 | watch 模式（文件变更自动采集） | ARCHITECTURE §13 开放问题 | 待评估 |
| C2 | profile 远程订阅（团队下发） | ARCHITECTURE §13 开放问题 | 待评估 |
| C3 | 权限/审批模型标准化（Roo groups 等） | RESEARCH-SUMMARY D17 | P3 候选 |

## 缺陷区

| # | 现象 | 影响流程 | 状态 |
|---|------|---------|------|
| — | （暂无） | | |
