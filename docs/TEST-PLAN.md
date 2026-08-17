# cfg4ai 测试计划

> 版本：v1.0（2026-08-17）｜ 上游：ARCHITECTURE §10/§12、IR-SCHEMA §5、review/REDTEAM.md、research/adversarial-cases.md
> 原则（原理四）：正确性无法正向证明，测试体系的目的是让每类已知错误**必被某一层发现**。

## 1. 测试金字塔与职责

```
        ┌─────────────────┐
        │ E4 平台冒烟矩阵   │  麒麟 V10（headless/UKUI）、六 target 构建断言
        ├─────────────────┤
        │ E3 对抗用例回归   │  38 条 adversarial-cases（红队防线不过夜）
        ├─────────────────┤
        │ E2 CLI e2e       │  testscript 剧本：全命令正常流+异常流
        ├─────────────────┤
        │ E1 适配器 golden │  双向 golden-file（含注释/键序保留用例）
        ├─────────────────┤
        │ E0 单元测试       │  core 包：合并语义/校验/指纹/写入协议/脱敏
        └─────────────────┘
```

| 层 | 工具 | 覆盖对象 | 失败含义 |
|----|------|---------|---------|
| E0 | `go test`（标准 testing + table-driven） | core/ir、core/profile、core/registry、core/secrets、atomicfile、platform/paths | 逻辑错误 |
| E1 | golden-file（`testdata/<tool>/`） | 每适配器 Import/Export 双向 | 格式漂移/注释丢失 |
| E2 | **标准 testing + os/exec**（编译产物黑盒驱动，用例表驱动） | CLI 命令全路径（W1–W6 流程） | 流程破坏 |
| E3 | 对抗用例库（38 条 → 可执行化） | 红队 Top10 + FMEA 高危项 | 已知失败模式复发 |
| E4 | CI matrix + 手动冒烟 | 六 target 静态断言、麒麟实测 | 分发事故 |

## 2. E0 单元测试范围（按包）

| 包 | 必测点 | 规格依据 |
|----|--------|---------|
| core/ir | 12 条校验规则逐条正反例；id 解析（首个点号分隔、点号 name）；Activation/HookEvent 词表 | IR-SCHEMA §5 |
| core/profile | merge-by-id 浅字段级（数组整体替换/object 覆盖/未写字段继承）；concat 两段式（层级>priority>path 字典序）；墓碑遮蔽（项目遮蔽全局）；五层序 | IR-SCHEMA §2 |
| core/registry | git remote 规范化 4 规则；二次判别（first_commit 不一致→新建 pid）；same_remote_as | ARCHITECTURE §4 |
| core/secrets | 三级后端降级链；敏感扫描分级（结构化抽取/自由文本仅警告）；secretref 解析与 dangling | ARCHITECTURE §9、IR-SCHEMA §5 |
| atomicfile | 同目录 temp；写入内容一致；symlink 穿透（目标存在/父目录链接/新建三场景）；perm 应用（Unix）；失败清理无残留 temp | ARCHITECTURE §5.3 |
| platform/paths | 三分支目录解析；APPDATA 缺失回退；ExpandRaw/CollapseRaw 往返 | ARCHITECTURE §8 |

## 3. E2 CLI e2e（标准 testing + os/exec）

> 技术选型变更（2026-08-17）：原选型 `rogpeppe/go-testscript` 仓库已从 GitHub 消失（各代理一致 404，fork 亦不可用），降级为**零新依赖方案**：标准 testing + os/exec 驱动编译产物。txtar 剧本格式待生态明朗后再评估引入。

- 位置：`cmd/cfg4ai/e2e_test.go`；二进制在 TestMain 中编译一次至临时目录，用例表驱动（命令参数/期望 stdout 正则/期望退出码/期望文件状态）。
- 每个 WORKFLOWS 流程至少 1 正常流 + 每 `⚠` 保护 1 异常流。
- P0 必备用例：`scan`、`collect`（含目录不存在中止）、`export`（含空集保护、外来内容四选项、dry-run 不落盘）、`link/relink`、`snapshot/restore`、`doctor`、`config`、退出码 0/5/2/3 断言。
- 双进程并发写锁竞争用例（W1[1]）。

## 4. E3 对抗用例回归

- 来源：`research/adversarial-cases.md`（38 条，已标注验证手段：单测/e2e/手工/文档澄清）。
- 管理规则：
  - 每条用例在代码库有唯一测试名映射（`TestAdversarial_<编号>`）；
  - "文档澄清"类 12 条不转代码，但需在 TASKS 对应 task 的 DoD 中核对；
  - **红队防线优先级**：T-01（墓碑误判）、T-03（占位符回采）、T-05（sync 泄漏）相关用例在 P0 必须先绿；
  - 新增对抗用例（评审/事故后）→ 先加用例再修复。
- P0 子集清单在 TASKS T11 维护。

## 5. 测试责任矩阵（task → 测试义务）

| 任务 | E0 | E1 | E2 | E3 |
|------|----|----|----|----|
| T1 core/ir | ✅ 校验/合并全项 | — | — | 相关用例 |
| T2 core/profile | ✅ 物化/遮蔽 | — | — | 墓碑类 |
| T3 store | ✅ 锁/快照/blob | — | ✅ 并发剧本 | T-01/T-10 |
| T4 secrets | ✅ 降级链/扫描 | — | ✅ 回采保护 | T-03/T-05 |
| T7/T8 适配器 | ✅ 映射规则 | ✅ 双向 golden | ✅ collect/export | 工具相关类 |
| T9 migrate | ✅ 管线单测 | — | ✅ 空集/外来内容 | T-01 链路 |
| T10 CLI | — | — | ✅ 全命令 | 退出码类 |

（完整映射随 TASKS.md 任务行维护；新增 task 时必须登记本矩阵。）

## 6. 门禁与 CI

### 6.1 本地提交前门禁
```
gofmt -l . 为空 ｜ go vet ./... ｜ CGO_ENABLED=0 go build ./... ｜ go test ./...
```

### 6.2 CI 流水线（GitHub Actions 规划）
1. **lint-test**（ubuntu）：gofmt/vet/test + `go test -race`；
2. **static-assert**：六 target 构建 + `ldd` 断言 static（麒麟红利保险丝）；
3. **golden**：适配器 golden-file 全量；
4. **adversarial**：对抗用例回归；
5. **coverage**：core 包行覆盖 ≥ 80%（P0 门禁；适配器包以 golden-file 为准不设行覆盖指标）；
6. **release-dry**：goreleaser snapshot 构建六 target。

### 6.3 麒麟冒烟矩阵（E4，M2 前手动执行）
| 环境 | 用例 |
|------|------|
| 麒麟 V10 服务器 amd64（headless） | collect/export 全流程 + keyring 降级到 file 后端 |
| 麒麟 V10 服务器 arm64（headless） | 同上 |
| 麒麟 UKUI 桌面 | keyring Secret Service 探测路径 |

## 7. 测试数据管理

- **golden 样本**：`internal/adapters/<tool>/testdata/`——`input/`（真实工具格式样例）↔ `ir/`（期望 IR YAML）↔ `output/`（期望导出文件树）；必须含**注释/键序保留用例**与**敏感字段抽取用例**。
- **真实样本库**：`testdata/realworld/`（社区真实配置，P0 期间持续扩充；入库存量须先脱敏审查——只收公开仓库样本且记录来源 URL）。
- **合成对抗数据**：随对抗用例各自内嵌（testdata 子目录）。

## 8. 验收映射

| 路线图验收项 | 验证手段 |
|-------------|---------|
| Claude ↔ Codex 指令/MCP/hooks 互转 | E1 golden + E2 `migrate` 剧本 |
| 字段级 round-trip 无丢失 | E1 round-trip 用例（导出→重导入→语义 diff 为空） |
| 双层继承 e2e | E2 link+export 剧本 + E0 合并单测 |
| 对抗用例 P0 子集 | E3 回归 |
| 麒麟 V10 冒烟（P1） | E4 矩阵 |

## 9. 缺陷管理

- 发现缺陷 → TASKS.md"缺陷"区登记（现象/复现/影响流程）→ 评估是否新增对抗用例 → 修复 → 勾选。
- 数据丢失/泄漏类缺陷 = 最高优先级，必须先补对抗用例再修复，修复后回归全量 E3。
