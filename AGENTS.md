# AGENTS.md — cfg4ai 项目指南

**cfg4ai**：AI 编码工具配置的采集、治理与迁移系统（Go CLI）。采集各 AI 工具（Claude Code、Codex、Copilot、Zhanlu、Gemini…）的指令/MCP/skills/hooks 等配置入统一 SSOT 仓库，经中间表示（IR）无损互转并交付到任意已接入工具。

## 权威文档（docs/）

改动设计前先读；规格与代码冲突时以规格为准并回写规格。

| 文档 | 内容 |
|------|------|
| `docs/ARCHITECTURE.md` | 总体架构（v0.3）：分层、迁移引擎、写入协议、存储、安全、路线图 |
| `docs/IR-SCHEMA.md` | IR 数据模型（v0.3）：八类实体、五层 scope、合并语义、12 条校验 |
| `docs/CLI-SPEC.md` | 命令规范（v0.3）：命令/flag/退出码 |
| `docs/ADAPTERS.md` | 适配器接口与 20 工具配置地图（v0.3） |
| `docs/PROJECT-PLAN.md` | **项目计划**：里程碑、DoR/DoD 门禁、变更管理流程（规格先行的唯一通道） |
| `docs/WORKFLOWS.md` | **业务流程**：W1–W6 步骤级流程（采集/导出/迁移/同步/恢复/关联） |
| `docs/TEST-PLAN.md` | **测试计划**：五层测试金字塔、责任矩阵、CI 门禁、麒麟冒烟矩阵 |
| `docs/TASKS.md` | **任务看板**：P0 任务分解（T0–T12，依赖/预估/验收）、变更候选与缺陷区 |
| `docs/research/`、`docs/review/` | 调研档案（20 工具）与评审/红队报告 |
| `docs/cfg4ai-design.html` | 单文件叙述版总览（适合通读） |
| `docs/DESKTOP-UI-DESIGN.md` | cfg4ai-desktop（Go+Gio）UI/UX 设计规格 v1.0：深浅双主题 token、布局骨架、组件规格、动画清单、首次引导 |

**开发铁律**：① 规格先行——代码改动涉及规格行为时必须先改规格再写码（PROJECT-PLAN §7）；② 完成任务按 PROJECT-PLAN §5 五条门禁自检后在 TASKS.md 勾选。

## 构建与测试

- 构建：`go build ./...`；**`CGO_ENABLED=0` 强制**（麒麟/静态分发纪律，CI 断言 ldd 为 static）
- 测试：`go test ./...`；适配器必须 golden-file 双向测试（含注释保留用例）；对抗用例回归见 `docs/research/adversarial-cases.md`
- 依赖纪律：禁止引入 CGO 依赖；新增外部依赖先记录评估

## 代码结构

```
cmd/cfg4ai/            CLI 入口（cobra 命令树，P0 接入）
internal/core/         ir / profile / registry / migrate / aiassist / secrets
internal/adapters/     Adapter 接口 + registry + all/（聚合）+ 每工具一个包
internal/platform/     paths/（跨平台路径，平台分支集中此处）
internal/atomicfile/   写入协议唯一实现（适配器禁止手写写文件）
internal/store/        SSOT 读写、快照、blob、锁、导出清单
```

## 关键约定

- 模块路径 `github.com/timywel/ai4config`（仓库地址 git@github.com:timywel/ai4config.git；项目/二进制名 cfg4ai）
- 目录 0700 / 文件 0600；换行保持源风格，仓库内置 `.gitattributes`（eol=lf）
- secret 永不入库：`secretref://` 占位 + 三级后端降级链
- AI 语义转换只在引擎层（core/migrate），适配器不感知 AI
- 正式名 cfg4ai（2026-08-17 定名）；二进制/命令名 `cfg4ai`，环境变量前缀 `CFG4AI_`
