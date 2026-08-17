# CLI 命令规范

> 版本：v0.3（吸收 R1 调研 + R2 红队审计；决策依据 RESEARCH-SUMMARY D1–D17）｜ 二进制名：`cfg4ai` ｜ 框架：cobra

## 0. 全局约定

```text
cfg4ai <command> [subcommand] [flags]
```

全局 flag：

| Flag | 说明 |
|------|------|
| `--home <dir>` | 覆盖 SSOT 仓库位置（等价 `CFG4AI_HOME`） |
| `--format text\|json` | 输出格式 |
| `--yes` | 跳过常规交互确认。**不豁免**：AI 转换确认（§5）、sync preflight 阻断（§8）、空集导出保护（§5） |
| `--verbose` / `--quiet` | 日志级别 |
| `--no-ai` | 本次禁用 AI 辅助 |
| `--secrets-backend keyring\|file\|none` | 覆盖 secret 后端降级链 |

退出码：`0` 成功；`1` 通用错误；`2` 用法错误；`3` 校验失败；`4` 用户中止；`5` 部分成功（**Warnings 非空即 5**）。CI 守门：collect/export 的 `0/5` 可放行（5 留痕）；**sync 不放行 5**（preflight 命中必须人工处置）。

`--type` 取值全集（v0.3）：`instruction|mcp|skill|agent|command|workflow|hook|setting`。

## 1. `cfg4ai scan`

```text
cfg4ai scan [--path <dir>...]
```

只读探测：全局/项目位置 + 目标进程运行检测。输出工具、位置、scope（五层）、条目数估算、采集状态。

## 2. `cfg4ai collect`

```text
cfg4ai collect [--tool <id>...] [--scope global|project|local|remote|managed|all] [--path <dir>]
```

- 行为：Import → 敏感扫描（分级处置）→ diff 摘要 → 确认 → 写入+快照。
- **防误判中止（D8）**：源目录不存在/不可读（盘掉线、未挂载、权限不足）→ **中止该 Location 采集并报 Warning，绝不标记墓碑**；墓碑仅当目录存在且条目消失时标记。
- **指纹复核（D10）**：`--path` 命中注册项时仍比对 first_commit，不匹配 → 警告并建议 `relink`。
- **reconcile 边界（D10）**：仅作用于本次实际采集的 `(origin.tool, origin.path)` 集合。
- **占位符回采保护（D8）**：导出物中的 secretref 占位符/空值再采集时永不覆盖已有 secretref，冲突记 Warning。
- **运行时状态不采集**：OAuth 会话、trust 标记、auto-memories、内建默认值（差分采集）。
- 增量凭 `origin.raw_hash`；幂等。

## 3. `cfg4ai list` / `show` / `diff`

```text
cfg4ai list [--profile global|<pid>] [--type ...] [--scope ...]
cfg4ai show <entity-id> [--profile ...]
cfg4ai diff [--a global --b <pid>]
cfg4ai diff --tool codex
```

## 4. `cfg4ai link` / `relink` / `projects` / `unlink`

```text
cfg4ai link <path> [--profile <pid>]
cfg4ai relink [path]
cfg4ai projects
cfg4ai unlink <pid>
```

- 指纹规范化（去协议/去 `.git`/host 小写/scp 转标准）+ 二次判别（first_commit+确认），否则新建 pid 记 `same_remote_as`。
- relink 后触发 `exports/` 清单 rebase（路径重写，见 §8）。

## 5. `cfg4ai export`

```text
cfg4ai export --to <tool-id> [--project <path>] [--dry-run]
             [--only instruction,mcp] [--include-foreign]
             [--ai] [--ai-approve] [--force]
```

- 管线：merge（五层，墓碑遮蔽）→ Map →（`--ai`）→ Render → Verify（格式+round-trip）→ 写入协议落盘。
- **空集保护（D8）**：merged Bundle 条目数为 0 且目标已有文件 → 拒绝执行；`--force` 也需显式警告文案交互（`--yes` 不豁免此项）。
- **外来内容确认选项集（D9 写死）**：`overwrite / skip / view-diff / backup-overwrite`；`--force` = 全部 `backup-overwrite`。hash 对比前字节级规范化（CRLF→LF、去 BOM）。
- 物化分流 `materialize: inherited-skip（默认）| inherited-inline`；条目过滤按 `applies_to`；`--include-foreign` 纳入异构条目。
- **AI 确认链**：AI 结果必须确认；`--yes` 不豁免；`--ai-approve` 无人值守（记决策日志）；**AI 配置段变更后首次 `--ai` 强制重新 consent（D12）**。
- 目标 IDE 运行中 → 提示需重启/Reload。
- Warnings（降级/跳过/secretref 占位/ blob 悬空降级）逐条输出；非空退出码 5。

## 6. `cfg4ai migrate`

```text
cfg4ai migrate --from <tool-id> --to <tool-id> [--project <path>] [--ai] [--dry-run]
```

等价 `collect --tool <from>` + `export --to <to> --include-foreign`。

## 7. `cfg4ai snapshot` / `restore` / `gc` / `prune`

```text
cfg4ai snapshot list / create [--note "..."]
cfg4ai restore <snapshot-id> [--dry-run]     # 目标 IDE 运行中给警告
cfg4ai snapshot prune [--keep 20]
cfg4ai gc                                     # blob 标记-清除+快照回收
cfg4ai prune                                  # 墓碑物理清除+keyring 孤儿级联
```

## 8. `cfg4ai sync`（P2）

```text
cfg4ai sync init <git-remote-url>
cfg4ai sync push / pull / status
```

- **白名单**：`profiles/`、`registry.yaml`、`config.yaml`、`exports/`（D9）。`sync init` 自动写 `.gitignore` 基线（排除 snapshots/blobs/logs/cache/secrets.age/.lock）。
- **preflight（D11）**：push 前全仓敏感扫描（含自由文本），命中即阻断或显式确认；CI 不得放行 sync 的退出码 5。
- **换机/重定位 rebase（D9）**：pull 后或 relink 后，doctor 检测 exports/registry 路径与指纹失配 → 引导 rebase（路径重写+hash 重建）。
- **blob 悬空（D13）**：blobs 不入库；pull 后 doctor 报悬空清单；导出遇悬空按降级链处理（preserve 正文/全量重渲染+Warning）。
- pull 冲突走标准 git 流程；doctor 可检测冲突态。`local` scope 与 `per_machine` 条目不同步/需校正。

## 9. `cfg4ai doctor`

```text
cfg4ai doctor
```

检查项：SSOT 结构、IR 校验（12 条）、registry 存活与指纹复核、**exports/ 路径 rebase 检测**、secret 后端状态与 dangling 清单、keyring 孤儿、blob 悬空清单、stale lock、`CFG4AI_HOME` 云同步/权限校验、路径长度风险、适配器 Detect 与版本护栏、**AI 配置变更待重确认状态**、AI provider 连通性、git 冲突态。

## 10. `cfg4ai config`

```text
cfg4ai config set ai.provider openai-compatible
cfg4ai config set ai.base_url https://...    # 变更后下次 --ai 强制重新 consent（D12）
cfg4ai config set ai.enabled true
cfg4ai config set secrets.backend auto       # auto | keyring | file | none
cfg4ai config get <key> / list
```

api_key 强制交互录入存 secret 后端。

## 11. 命令-阶段对照

| 命令 | P0 | P1 | P2 |
|------|----|----|----|
| scan / collect / export / migrate | ✅（含防误判/空集保护/占位符回采） | | |
| link（最小实现） | ✅ | 指纹+relink+rebase | |
| list / show | ✅ | | |
| diff（独立命令） | | ✅ | |
| snapshot / restore | ✅ | prune 完善 | |
| doctor / config | ✅ 基础项 | ✅ 全量 | |
| gc / prune | | ✅ | |
| sync（白名单+preflight+rebase） / TUI | | | ✅ |
