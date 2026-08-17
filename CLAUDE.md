# CLAUDE.md — aicfg 开发参与指南

> Auto-loaded on every session. 优先级低于 AGENTS.md，两者冲突时以 AGENTS.md 为准。
>
> 项目权威设计文档在 `docs/`（ARCHITECTURE / CLI-SPEC / IR-SCHEMA / ADAPTERS + research/ + review/）。本文件是**开发协作规范**，不重复设计内容；设计问题一律以 `docs/` 为准。

---

## 用户核心规则（绝对禁止违反）

**未经用户明确批准，禁止修改任何代码文件（.go/.mod 等）和非计划类文档。**

- 仅允许直接改写：**计划类文档**（文件名含 PLAN、PROPOSAL 等明确标注为"计划"）
- 代码修改：必须获得用户逐条确认
- 所有文档修改（除计划类外）：必须获得用户逐条确认
- 违反此规则视为严重越权行为

**例外：计划文档已批准豁免**

- 用户引用某个计划文档路径（如 `按照 /path/to/plan.md 执行`）时，**该文档视为已经用户批准**，无需再次确认，直接执行
- 计划文档 = 已批准授权，立即动手，不询问、不等待

---

## 用户工作习惯

### 1. 计划驱动执行模式

用户通过**引用具体计划文档路径**来指派工作，AI 必须：

- 按文档中每个 task 逐一完成，不得跳过
- 实时追踪 task 状态：`not-started` → `in-progress` → `completed`（或 `partial`）
- 无法完成的 task 必须记录原因并更新回文档

### 2. 滚动迭代计划模式

用户习惯用 `update plan-N` → `update plan task-N.json` 的组合进行迭代推进：

- 完成当轮计划后，**必须**将遗留内容、后续优化点提炼为 `update plan-(N+1).md` + `update plan task-(N+1).json`
- 保存至对应目录，供用户审批后下一轮继续

### 3. 测试标准（固定组合，不可省略）

每次功能修改/升级完成后必须依次执行：

1. **单元测试** — `go test ./<改动的包>/...`，验证改动点
2. **回归测试** — 全量 `go test ./...`，确保既有功能未受影响
3. **golden-file 双向测试** — 涉及 IR/适配器序列化时必跑（Import/Export round-trip）
4. **对抗用例回归** — `docs/research/adversarial-cases.md`（38 条，按阶段取子集）
5. **testscript 场景测试** — 涉及 CLI 命令时：命令交互、flag、退出码（`0/1/2/3/4/5`）符合 `docs/CLI-SPEC.md` §0
6. 发现问题 → **循环修复+重新测试**，直至全部通过

涉及发布工程时追加：

7. **跨平台构建验证** — `CGO_ENABLED=0` 六 target 构建（`docs/ARCHITECTURE.md` §8），Unix target 断言 static（`ldd` 检查）

### 4. 文档规范

- **文档语言中文优先**（`docs/` 现有实践）
- 计划类文档统一归档至 `plan/{待完成|进行中|已完成}/`
- 正式文档入 `docs/`，按主题建子目录（现有：`research/` 调研档案与对抗用例、`review/` 评审与红队）
- 过期/与当前架构不符的文档 → 移出主目录，归档处理
- 设计文档头部标注版本与决策依据（现有实践：`| 版本：v0.3（…决策依据 RESEARCH-SUMMARY D1–D17）`）

### 5. 执行质量要求

用户对**敷衍执行零容忍**，以下行为是明确禁止的：

- 仅靠文件名/表面信息进行分类或处理，不深入读取内容
- 跳过个别 task 或部分完成就汇报"完成"
- 遗漏任何代码、配置、文档的改动点（必须全量覆盖）
- 遇到问题不回退、不记录直接跳过

#### 5.1 集成完整性检查（4 件套）

> **铁律**：aicfg 的两类核心扩展点（**新适配器**、**新 CLI 命令**）必须**同时完成 4 件事**，缺一不算"完成"。

**新适配器（如 `internal/adapters/<tool>/`）**：

| # | 必须 | 说明 | 验证方式 |
| --- | --- | --- | --- |
| 1 | **注册** | 包内 `init()` 调 `adapters.Register(...)`，且 `internal/adapters/all/all.go` 加 blank import | `grep -r "<tool>" internal/adapters/all/` |
| 2 | **能力矩阵** | `Meta().Capabilities` 覆盖全部 EntityKind（instruction/mcp/skill/agent/command/workflow/hook/setting），标注 SupportLevel + Note | 单测断言矩阵完整 |
| 3 | **golden-file 双向测试** | Import/Export round-trip，显式覆盖 YAML 注释/键序保留；幂等（第二次 Export 全 no-op） | `go test ./internal/adapters/<tool>/` |
| 4 | **文档登记** | `docs/ADAPTERS.md` 工具配置地图 + 目标语义差异表（D16：合并/极性/覆盖差异由适配器消化） | diff 检查 ADAPTERS.md |

**新 CLI 命令（如 `aicfg collect`）**：

| # | 必须 | 说明 | 验证方式 |
| --- | --- | --- | --- |
| 1 | **cobra 注册** | 命令挂到 root，flag 定义完整 | `go run ./cmd/aicfg --help` |
| 2 | **CLI-SPEC 对照** | flag 名/退出码/确认链（`--yes` 不豁免项）符合 `docs/CLI-SPEC.md` | 逐条对照 |
| 3 | **testscript 测试** | 命令交互场景（含错误路径与退出码） | `go test ./cmd/aicfg/` |
| 4 | **文档登记** | CLI-SPEC 命令-阶段对照表（§11）更新 | diff 检查 CLI-SPEC.md |

**反模式（明确禁止）**：

- ❌ 写 "feat: <tool> adapter" 但只写了 Import 没写 Export（或反之）
- ❌ 适配器写完没注册进 `all/`（编译期不会报错，运行时查无此器）
- ❌ 写完立刻交付，**没跑过 golden-file round-trip**
- ❌ commit 描述与实际变更范围脱节（commit 标题误导）
- ❌ 单测全绿就以为完成（单测只覆盖包内逻辑，不验证注册链路与管线集成）

**Why**（第一性推导）：

- 适配器写了 ≠ 接入系统——`adapters → all → cmd` 靠 blank import 链接，漏掉任何一个环节，运行时该工具对 `collect`/`export` 不可见，且**编译期无任何报错**。
- 单测全绿会**掩盖**集成完整性问题——单测只测包内逻辑，不测 Register 链路与能力矩阵是否被引擎正确消费。
- aicfg 的正确性靠证伪机制逼近（`docs/ARCHITECTURE.md` §2）：round-trip diff + golden-file + 对抗用例库。**跳过 round-trip 等于放弃正确性保证**。

### 6. 减法偏好

用户倾向保持系统精简（"做减法"）：

- 删除功能需即时验证，防止引发连锁问题
- 遇到严重影响系统稳定性的删除，必须回退并记录

### 7. 文件整理原则

- 临时文件、会话文件 → `temp/`
- 计划文件 → `plan/{待完成|进行中|已完成}/`
- 架构/规范/调研文档 → `docs/` 对应子文件夹
- 过期/与当前架构不符的文档 → 移出主目录，归档处理

### 8. Spec 骨架与 Scope Discipline

> 策略：**只追加，不覆盖**任何现有规则。

#### 8.1 六段式 Spec 骨架（plan/待完成/PLAN-*.md 强制）

新建的 `plan/待完成/PLAN-*.md` **必须**包含以下骨架。小任务可裁剪 section 数量，但 `ASSUMPTIONS` 和 `Success Criteria` 不可省略。

```markdown
# PLAN-XXX-<主题>-YYYYMMDD-HHMMSS

## ASSUMPTIONS（强制）

写 spec 之前先列出"我假定如下"，等用户确认再继续。

- 假定 1：xxx
- 假定 2：xxx
  → 立刻纠偏或确认后继续。

## 1. Objective（做什么 + 为什么）

- 用户是谁、解决什么痛点
- 成功长什么样（用户视角）

## 2. Commands（可执行命令，不是工具名）

- Build: xxx
- Test: xxx
- Lint: xxx
- Dev: xxx

## 3. Project Structure（改/新增哪些路径）

- 路径 → 用途
- 标注"新增"/"修改"

## 4. Code Style（一段真实代码示例，胜过三段描述）

- 命名 / 格式 / 风格（用项目里已有的风格，不要发明）

## 5. Testing Strategy（测什么、怎么测、覆盖多少）

- 测试框架 / 位置 / 覆盖目标

## 6. Boundaries（三段式）

- **Always**: xxx
- **Ask first**: xxx（涉及 schema 变更、加依赖、改 CI 等）
- **Never**: xxx（提交密钥、删测试、动 vendor 目录等）

## Success Criteria（可验证的完成条件）

- 条件 1：xxx（具体可测）
- 条件 2：xxx

## Open Questions（待用户决策的悬而未决项）

- Q1: xxx
- Q2: xxx
```

**与滚动迭代模式的关系**：本骨架是单个 PLAN 的内部结构，外层 `update plan-N` → `update plan task-N.json` 的迭代节奏**不变**。

#### 8.2 Rule 0.5：Scope Discipline（任务越界禁止）

**只动任务范围内的事，范围外的事看见但不要碰。**

明确禁止的越界行为：

- ❌ "顺手"重构任务范围外的代码
- ❌ 删除或"清理"看不懂的注释
- ❌ 改自己没在改的文件的 import
- ❌ 加 spec 里没有的"看起来有用"的 feature
- ❌ 把"读懂这段代码"当成"改写这段代码"的通行证
- ❌ 改语法风格只为了"现代化"

发现任务外问题时，**记下来但不修**：

```markdown
## Noticed but not touching

- <file-path>: 有个未使用的 import（与本任务无关）
- <file-path>: 错误提示文案可优化（独立任务）
  → 是否需要我创建后续任务？
```

#### 8.3 Rule 0.6：增量节奏（~100 行 / 任务）

**单个 task 的代码改动量目标 ~100 行，可接受 ~300 行，超过 ~1000 行必须拆分。**

一个 task 算"一个"：

- 自包含的单一修改
- 包含相关测试
- 提交后系统仍可工作
- 是"feature 的一部分"而不是"整个 feature"

**超过 1000 行的拆分策略**：

| 策略          | 用法                         | 适用         |
| ------------- | ---------------------------- | ------------ |
| Stack         | 提交一个、再基于它开始下一个 | 串行依赖     |
| By file group | 不同 reviewer 分批           | 跨模块横切   |
| Horizontal    | 先抽公共层/桩，再做消费者    | 分层架构     |
| Vertical      | 拆成更小的端到端切片         | feature 增量 |

**架构级任务豁免**：涉及多文件联动的架构级重构（IR 模型迁移、store 写入协议、迁移引擎管线等跨包联动）**不受 ~1000 行上限约束**，但仍需在 PLAN 的 Boundaries 段说明豁免理由，并拆成多个 commit 提交（每个 commit 单一职责、可独立回滚）。

**反模式**：一个 commit 同时做"加新适配器 + 重构老适配器 + 改 build 配置"。

---

## 输出路径严格分流（根目录零散文件强制约束）

> **aicfg 运行时产物 ≠ 项目内开发工具产物 ≠ 正式源码/计划/文档**，三者必须各归各位，根目录严禁出现散落产物。

### 规则表

| 产物类型 | 目标路径 | 备注 |
| --- | --- | --- |
| **aicfg 运行时产物**（profiles / snapshots / blobs / registry / exports / secrets.age / cache / logs / `.lock`） | `$AICFG_HOME/`（默认平台用户目录，由 `internal/platform/paths` 封装，见 `docs/ARCHITECTURE.md` §7） | **绝不入仓库**；sync 白名单内仅 `profiles/`、`registry.yaml`、`config.yaml`、`exports/` |
| **项目内开发工具产物**（测试输出 / 调试 dump / 调研产物 / 外部脚本产物 / 临时归档） | `<repo>/temp/{对应子目录}/` | **注意是 `temp/` 不是 `tmp/`** |
| **项目正式源码 / 计划 / 正式文档** | `cmd/`、`internal/`、`docs/`、`plan/{待完成,进行中,已完成}/` 等正式目录 | 走计划驱动模式，需用户批准 |

### 根目录严禁出现

- ❌ `profiles/`、`snapshots/`、`blobs/`、`exports/`、`secrets.age` 等 SSOT 目录/文件（应放 `$AICFG_HOME/`）
- ❌ `产物/`、`artifacts/`、`cache/`、`research/`（散落副本；正式调研档案归 `docs/research/`）
- ❌ 散落的 `check*.go` / `test-dev*.go` / `debug*.go` / `main.go`（ad-hoc 调试）
- ❌ 散落的 `*.png` / `*.log` / `*.out` / `*.dump`
- ❌ `tmp/`（统一改用 `temp/`）

### 自检时机（强制）

1. **写任何文件前**：先判断产物类型 → 写到正确路径
2. **每次 session 结束 / 跑完测试 / 完成调研后**：根目录 `ls` 自检，散落文件立即归位
3. **commit 前**：`git status --short` 看到根目录 `??` 文件一律处理（移到 `temp/` 或 `.gitignore` 屏蔽）

---

## 开源协议合规（AI 编程强制要求）

### 问题背景

AI 编程工具在生成代码时，通常会：

- 覆盖文件头部版权声明
- 不主动保留 `Copyright` 声明
- 生成的代码看似"全新"，但可能包含开源项目逻辑

**违反开源协议可能导致法律风险**，必须强制检查。

### 规则摘要

| 协议           | 商业使用 | 修改分发 | 专利授权 | 合规要点                               |
| -------------- | -------- | -------- | -------- | -------------------------------------- |
| **MIT**        | ✅       | ✅       | ❌       | 保留版权声明 + 分发带 LICENSE          |
| **Apache 2.0** | ✅       | ✅       | ✅       | 保留声明 + 附 LICENSE + NOTICE（如有） |
| **BSD-3**      | ✅       | ✅       | ❌       | 保留声明 + 禁止用作者名背书           |
| **GPL v3**     | ✅       | ✅       | ✅       | **必须开源**衍生作品                   |

### Copyright 头标准格式（Go）

**MIT（项目自身代码，如选择 MIT）：**

```go
// Copyright (c) 2026 <作者>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction...
```

**Apache 2.0（第三方库）：**

```go
// Copyright 2026 [Original Author]
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
```

### 项目依赖许可证清单

> 以 `go.mod` 锁定版本的各库 LICENSE 文件为准；下表为选型清单（`docs/ARCHITECTURE.md` §10）的典型协议，**引入前必须核验**。

| 库 | 典型协议 | 合规要点 |
| --- | --- | --- |
| spf13/cobra | Apache-2.0 | 保留声明 + 附 LICENSE |
| charmbracelet/bubbletea（P2） | MIT | 保留 Copyright |
| gopkg.in/yaml.v3 | MIT / Apache-2.0 | 保留声明 |
| pelletier/go-toml/v2 | MIT | 保留 Copyright |
| sergi/go-diff | MIT | 保留 Copyright |
| 99designs/keyring | MIT | 保留 Copyright |
| go-git/go-git | Apache-2.0 | 保留声明 + NOTICE（如有） |
| gofrs/flock | BSD-3-Clause | 保留声明 |
| filippo.io/age | BSD-3-Clause | 保留声明 |

**新增依赖时**：先在 PR/PLAN 中登记许可证，再 `go get`。GPL/AGPL/SSPL 系依赖**禁止引入**（与静态链接分发冲突）。

### 合规 Checklist

生成代码后**必须检查**：

- [ ] 新增文件包含正确的 Copyright 头
- [ ] 修改的文件保留原有 Copyright 声明
- [ ] 包含第三方库代码时保留其协议头
- [ ] 分发时附上 LICENSE 文件（goreleaser 打包配置内）

### 快速检查命令

```bash
# 检查是否遗漏版权声明
grep -rL "Copyright" cmd/ internal/ --include="*.go" || echo "全部含声明"
```

---

## 外部 Skill 集成使用规范

> **目的**：明确外部 AI 编码辅助 skill 的集成位置、触发场景、冲突降级规则。
> **作用域**：**仅限 AI 编码工具层**（用户级 `~/.claude/skills/` 等），与本项目源码**完全解耦**。
> **适用性说明**：以下清单已按 aicfg 项目（Go CLI，无前端）裁剪；不适用的场景已标注。

### 1. 集成位置（统一在用户级目录，不进仓库）

| Skill | 集成位置 | License | 类型 | 本项目适用性 |
| --- | --- | --- | --- | --- |
| **ponytail** | `~/.claude/skills/{ponytail,ponytail-review}/SKILL.md` | MIT | Prompt 指令集 | ✅ Go 代码生成 |
| **stop-slop** | `~/.claude/skills/stop-slop/SKILL.md` + `references/` | MIT | Prompt 指令集 | ✅ docs/plan 文档写作 |
| **full-output-enforcement** | `~/.claude/skills/full-output-enforcement/SKILL.md` | MIT | Prompt 指令集 | ✅ 长 spec/长文件完整输出 |
| **headroom** | MCP 配置（运行时：`pip install "headroom-ai[mcp]"`） | Apache-2.0 | MCP 运行时 | ✅ 上下文压缩基础设施 |
| ~~taste-skill / design-taste-frontend~~ | — | — | — | ❌ 本项目为 CLI，无前端场景（P2 TUI 阶段可重估） |

**禁止行为**：

- ❌ 禁止把以上 skill 复制或链接到本仓库内的 `cmd/`、`internal/`、`docs/` 等任何目录
- ❌ 禁止在 `go.mod` 中加入这些 skill 的依赖
- ❌ 禁止修改 skill 的原 `SKILL.md` 内容（违反 License 完整性，仅可整体搬运）

### 2. 触发场景规则

> **核心原则**：场景互斥 + 自动识别 + 手动覆盖

| 场景 | 主导 skill | 备注 |
| --- | --- | --- |
| **文档写作**（docs/ / plan/ / 说明类 .md） | **stop-slop** | 严格收窄到 `drafting, editing, or reviewing text`，不侵入 code/commit/对话 |
| **代码生成**（.go / 任何编程） | **ponytail full**（默认） | stdlib 优先与 Go 生态契合 |
| **代码 review**（diff 扫描过设计） | **ponytail-review** | 仅在显式调用 `/ponytail-review` 时触发 |
| **完整输出**（不截断 / 不省略） | **full-output-enforcement** | 长 spec / 长 golden-file 场景覆盖默认截断行为 |
| **上下文压缩**（长 context / 重复检索） | **headroom MCP** | **独立基础设施层**，不与上述互斥 |

### 3. 冲突降级矩阵（防止双规则矛盾）

| 同时启用 | 主导 | 让位 / 降级 |
| --- | --- | --- |
| headroom + 上述任一 | 上述任一 | headroom **关闭 output shaper**（`HEADROOM_OUTPUT_SHAPER=0`），避免与 stop-slop 的"少废话"重复 |
| headroom + 用户代码生成 | ponytail | headroom 只做**输入压缩**，不碰输出 |

### 4. headroom 安全开关（**强制**）

> headroom 是上述 skill 中**唯一需要运行时基础设施**的（Python + ONNX 模型 + MCP server），且**默认开启匿名遥测**。

集成时**必须**配置以下 env：

```json
{
  "HEADROOM_TELEMETRY": "off",
  "HEADROOM_OUTPUT_SHAPER": "0"
}
```

- ✅ `HEADROOM_TELEMETRY=off` — 关闭匿名遥测（隐私优先）
- ✅ `HEADROOM_OUTPUT_SHAPER=0` — 关闭输出精简（避免与 stop-slop 弱冲突）

### 5. 手动触发 vs 自动触发

| 触发方式 | 命令 | 适用 |
| --- | --- | --- |
| **手动命令**（最可靠） | `/ponytail` `/ponytail-review` `/stop-slop` `/full-output-enforcement` | 明确要用某个 skill 时 |
| **语义自动加载** | 在消息中包含 skill description 里的关键词 | 日常使用，依靠 AI 工具语义判断 |
| **强制 prompt 注入** | 消息开头写 `按 ~/.claude/skills/<name>/SKILL.md 的规则做这个` | 当 description 语义匹配失败时强制加载 |

### 6. ponytail 强度档切换

| Level | 行为 | 适用 |
| --- | --- | --- |
| **lite** | 实现需求，但**点名**更懒的替代方案让用户选 | 试探性需求、不确定是否要 YAGNI |
| **full**（**默认**） | 阶梯强制 + stdlib 优先 + 最短 diff | 日常代码生成 |
| **ultra** | YAGNI 极端，删除 > 添加，挑战需求本身 | 紧急交付 / 用户明确说"最简实现" |

切换命令：`/ponytail lite|full|ultra` 或 `/ponytail stop`（恢复正常模式）。

### 7. 版权合规（与"开源协议合规"章节配合）

- ✅ MIT skill（ponytail / stop-slop / full-output-enforcement）— 商用 + 修改 + 分发均可，仅需保留 Copyright + LICENSE
- ✅ Apache-2.0 skill（headroom）— 同上 + 附 NOTICE（如有）
- ❌ 禁止删除/合并/重命名各 skill 的 LICENSE 文件
- ❌ 禁止在本仓库 `go.mod` 中声明这些 skill 为运行时依赖（它们是 AI 工具配置层，不是项目代码层）

---

## 问题报告强制追尾（第一性原理根因分析）

> **铁律**：凡是向用户报告"出现了问题"（bug、报错、回归、行为不符预期、计划偏差等），**问题消息的最后一条**必须追加"第一性原理根因分析"段，**禁止只交付现象 + 表面修复就结束**。

### 为什么需要这条规则（第一性推导）

1. **现象 ≠ 问题本身**：报错堆栈、卡住的位置、回归的测试用例只是 observation。Observation 解释了"发生了什么"，从不解释"为什么会发生"。
2. **治标不治本必然复发**：只修表面（patch 一行、关掉一个开关、回滚一个 commit）会留下隐藏路径，下次换个 trigger 又会引爆。
3. **AI 的默认偏差是"快速给答案"**：训练分布里"先说原因再说修复"远少于"先给修复再说原因"。**显式强制才能反转这个偏差**。
4. **第一性 = 还原到不可再分的事实**：剥掉"看似、可能、应该是"的猜测，回到"事实 A → 事实 B → 事实 C"的因果链，每一环必须可验证。

### 强制追尾模板

每次问题报告的**最后一段**（修复方案 / commit / 验证步骤之后）必须追加：

```markdown
---

## 🔍 第一性原理根因分析（强制追尾）

### 现象（Observation）
- 报错的原文 / 失败测试名 / 卡住的具体步骤
- 1-3 条事实，**不解释**、不归因

### 因果链（Causality，从第一环开始）
1. **第一环（不可再分的事实）**：xxx（命令输出、文件状态、版本号等可验证事实）
2. **第二环**：由第一环直接推出 → xxx
3. **第三环**：由第二环直接推出 → xxx
...

→ **根因（Root Cause）**：xxx

### 为什么没有更早发现

- 是哪条规则 / 哪个测试 / 哪个 checkpoint 缺失导致问题溜过去？
- 下次如何让"同形问题"在更早阶段暴露？

### 同类风险扫描

- 项目里还有哪些位置可能踩同样的坑？（路径 / 函数 / 模块）
- 是否需要补一条防御性规则或自动化检查？
```

### 适用范围

| 触发场景 | 是否需要追尾 | 备注 |
| --- | --- | --- |
| 跑测试 / lint / build 失败 | ✅ 必须 | 哪怕是已知的 flaky test |
| 用户报告"这个东西不工作了" | ✅ 必须 | 第一性分析用户的复现路径 |
| 自我发现代码逻辑缺陷 / 越界违例（commit 误操作、根目录散落、reset --hard 等） | ✅ 必须 | **自我发现的违例更危险**，因为没有用户报告这道防线 |
| 计划执行偏差（task 标了 completed 实际只完成 80%） | ✅ 必须 | 复盘为什么虚报 |
| 单方面拍板的 ASSUMPTIONS 后来被证伪 | ✅ 必须 | 复盘为什么假定是错的 |
| 普通的进度汇报、commit 提交、文档成稿 | ❌ 不需要 | 没有"问题"就不需要追尾 |
| 纯提问 / 查询 / 阅读类任务 | ❌ 不需要 | 没有"问题"就不需要追尾 |

### 反模式（明确禁止）

- ❌ "问题原因可能是 xxx"（用"可能"糊弄） → 必须落到可验证事实
- ❌ 跳过"为什么没有更早发现"段 → 这段是防止下次复发的核心
- ❌ 把根因写成"我疏忽了 / 我没注意" → 这种归因不解决问题，必须还原到机制层
- ❌ 追尾段放在修复方案**之前** → 追尾是收口，必须放最后，逻辑顺序是"现象 → 修复 → 根因"
- ❌ 写完追尾不更新 CLAUDE.md → 如果追尾揭示了新的通用规则，必须立即沉淀

### 沉淀路径

追尾揭示的**通用规则**（适用于本项目未来所有 session）必须同步沉淀到以下位置之一：

| 沉淀类型 | 目标位置 |
| --- | --- |
| 项目级 invariant | `CLAUDE.md` / `AGENTS.md`（追加，不覆盖现有规则，需用户确认） |
| 一次性事件记录 | `temp/incidents/<日期>-<短主题>.md` |
| 计划类反思 | `plan/已完成/PLAN-*-ROOT-CAUSE-*.md` |

**Why**：根因分析的 value 在"防止复发"，不沉淀 = 白写。

### 与现有规则的关系

- **§1 计划驱动模式**：追尾段的"同类风险扫描"会产出新的 plan 任务，遵循 §2 滚动迭代
- **§5 执行质量要求**：追尾强化了"敷衍执行零容忍"——只给表面修复 = 敷衍
- **§6 减法偏好**：追尾段的"为什么没有更早发现"是减法偏好的具体抓手
- **§8.2 Scope Discipline**：同类风险扫描要警惕"顺手扩展任务范围"，仅记录、不擅自修

---

## 关联指针

| 本文件章节 | 指针文件 | 用途 |
| --- | --- | --- |
| §3 测试标准 | `docs/research/adversarial-cases.md` | 对抗用例库（38 条） |
| §5.1 集成完整性 | `docs/ADAPTERS.md` §2–§3 / `docs/CLI-SPEC.md` | 适配器规范与配置地图 / 命令规范 |
| 输出路径分流 | `docs/ARCHITECTURE.md` §7 | SSOT 存储布局与白名单 |
| 开源协议 | （无外迁，权威源 = 本文件） | 完整规范保留在本文件 |
| 外部 Skill 集成 | （无外迁，权威源 = 本文件） | 完整规范保留在本文件 |
| 问题追尾 | （无外迁，权威源 = 本文件） | 强制追尾模板 + 适用范围 + 反模式 |
