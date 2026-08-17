# cfg4ai 项目计划

> 版本：v1.0（2026-08-17）｜ 性质：**活文档**——任何范围/里程碑变更必须先改本文档再动手
> 上游约束：[ARCHITECTURE.md](./ARCHITECTURE.md) §12 路线图；任务看板：[TASKS.md](./TASKS.md)；流程：[WORKFLOWS.md](./WORKFLOWS.md)；测试：[TEST-PLAN.md](./TEST-PLAN.md)

## 1. 项目目标与边界

一套 Go CLI：采集 20 个已调研 AI 编码工具的配置入 SSOT，经 IR 互转并交付。功能目标 G1–G5 见 ARCHITECTURE §1.2。边界（非目标）见 §1.3 与 §16——**超出边界的需求一律走变更流程（§7），不允许顺手实现**。

## 2. 文档体系与维护规则（标准化根基）

| 文档 | 职责 | 变更时机 |
|------|------|---------|
| ARCHITECTURE / IR-SCHEMA / CLI-SPEC / ADAPTERS | **规格**（What） | 仅经变更流程（§7）修改；代码与规格冲突时以规格为准并回写规格 |
| PROJECT-PLAN（本文档） | 计划（When/Who/How much） | 里程碑评审时更新 |
| WORKFLOWS | 业务流程（How，步骤级） | 流程变更随规格变更联动 |
| TEST-PLAN | 测试策略与门禁 | 新增测试层级/门禁调整时更新 |
| TASKS | 任务看板（Do/DoD 勾选） | 每日更新；任务完成即勾 |
| AGENTS.md | 工程约定速查 | 结构/命令变化时更新 |
| research/、review/ | 档案（只增不改；修正以新文档追加） | — |

**规则：文档先行**。任何代码改动若涉及规格行为，必须先改规格、再改代码、同步更新 TASKS。

## 3. 里程碑

| 里程碑 | 内容 | 出口标准（DoD） | 目标 |
|--------|------|----------------|------|
| M0 设计冻结 | 四份规格 v0.3 + 骨架编译通过 + 本计划 | ✅ 已达成（2026-08-17） | — |
| M1 P0 最小闭环 | claudecode/codex 适配器 + collect/export/link + store/写入协议/脱敏管线 | ARCHITECTURE §12 P0 验收全项（互转、字段级 round-trip、双层继承 e2e、对抗用例 P0 子集绿） | TBD |
| M2 P1 生态扩展 | copilot/zhanlu/gemini + claude-desktop + grokbuild 适配器、relink、secrets 三级后端 | §12 P1 验收（七适配器 golden-file 全绿、两条 e2e 链、麒麟冒烟） | TBD |
| M3 P2 智能化 | aiassist consent 链、TUI、sync（白名单+preflight）、扩展适配器 | §12 P2 验收 | TBD |
| M4 P3 平台化 | 团队共享、GUI、外置进程插件 | §12 P3 验收 | TBD |

## 4. P0 阶段计划（M1 的展开）

### 4.1 阶段入口（DoR）
- 规格 v0.3 冻结；骨架 `go build`/`go vet`/`gofmt` 全绿（已达成）
- Go 1.25.7 环境就绪（便携安装 `C:\Users\Wel\dev\go`）

### 4.2 阶段划分（依赖序即施工序）

```
T0 工程基建 → T1 core/ir → T2 core/profile → T3 store → T4 secrets →
T5 paths / T6 atomicfile（可与 T1-T4 并行）→
T7 claudecode → T8 codex（依赖 T1-T6）→ T9 migrate 引擎 →
T10 CLI（cobra）→ T11 对抗回归 → T12 P0 验收
```

任务明细（内容/依赖/DoD/预估）全部在 [TASKS.md](./TASKS.md) 维护；本文档只跟踪阶段级状态。

### 4.3 阶段出口（DoD）
1. ARCHITECTURE §12 P0 验收四项全过；
2. TEST-PLAN §6 P0 门禁全绿（含对抗用例 P0 子集）；
3. 规格与实现零已知偏差（评审确认）；
4. TASKS 中 P0 任务全部勾选。

## 5. 质量门禁（每个 task 的 DoD 模板）

单个 task 完成 = 同时满足：
1. 代码实现 + 规格同步（规格受影响时已先改规格）；
2. 测试按 TEST-PLAN §5 责任矩阵补齐（单测/golden-file/e2e 按需）；
3. `gofmt -l` 为空、`go vet` 通过、`go test ./...` 通过；
4. 无 CGO 依赖引入；
5. TASKS.md 勾选并注明实际工时/偏差原因。

## 6. 风险管理

| 风险 | 等级 | 应对 | 联动文档 |
|------|------|------|---------|
| 上游工具格式漂移 | 高 | 版本护栏 + golden-file 雷达 + 时效跟踪清单 | ARCHITECTURE §12/§13 |
| 删除误判级联（红队 T-01） | 高 | 防误判中止/遮蔽/空集保护已实现于规格；T1/T2/T9 必须逐条落地并有对抗用例守护 | REDTEAM.md |
| secret 泄漏通道 | 高 | 脱敏管线（T3/T4）+ sync preflight；泄漏类用例优先回归 | ARCHITECTURE §9 |
| 单人维护吞吐 | 中 | WBS 任务粒度 ≤2 天/个；超出的拆分 | TASKS.md |
| Go 环境仅便携安装 | 低 | PATH 说明入 AGENTS.md；发布用 CI 构建不依赖本机环境 | AGENTS.md |

## 7. 变更管理流程（规格变更的唯一通道）

1. **提出**：在任何文档或代码中发现规格问题 → 记录到 TASKS.md"变更候选"区；
2. **评估**：影响面分析（涉及哪些规格章节/哪些适配器/是否破坏 round-trip 承诺）；
3. **决策**：采纳则**先改规格文档**（升版本号或注记修订），拒绝则说明理由；
4. **实施**：代码跟随规格；TASKS 登记；
5. **验证**：受影响的测试层级更新并回归。

**禁止**：代码先于规格变更落地（"先实现后补文档"）。

## 8. 当前状态速览

- [x] 设计 v0.3 + 两轮评审 + 红队 + 20 工具调研
- [x] 命名定案 cfg4ai（GitHub 零占用验证）
- [x] P0 骨架编译通过（评审修复 8 项后）
- [ ] M1 P0 实现（见 TASKS.md）
