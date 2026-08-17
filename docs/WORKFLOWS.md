# cfg4ai 业务流程规范

> 版本：v1.0（2026-08-17）｜ 作用：把 CLI 命令背后的业务过程定义为**步骤级流程**，作为实现与测试的共同基准。
> 每个流程：触发 → 步骤 → 分支/异常 → 出口。流程与规格冲突时以规格为准并回写本文档。

## 图例

`[步骤]` 执行体 ｜ `<数据>` 数据产物 ｜ `◇判断` 分支点 ｜ `⚠` 异常/保护路径 ｜ 括注规格章节为权威依据。

## W1 采集流程（`cfg4ai collect`）

> 依据：CLI-SPEC §2、IR-SCHEMA §2.3（墓碑）、ARCHITECTURE §9（脱敏）

```
触发：用户执行 collect（可选 --tool/--scope/--path 过滤）
[1] 取仓库写锁 .lock（⚠ 锁占用→报错退出，码 1；stale 锁→按 doctor 指引清理）
[2] 对目标适配器逐个 Detect(ctx)（只读）→ <Location 列表>
[3] ◇ 路径命中注册项？→ 是：比对 first_commit 指纹复核（D10）
      ⚠ 不匹配→Warning"疑似路径复用劫持"，建议 relink；--yes 下中止该项目采集
[4] 逐 Location：◇ 源目录存在且可读？
      ⚠ 不存在/不可读→中止该 Location 并 Warning（绝不标记墓碑——红队 T-01 防线）
[5] Import(ctx, loc) → <Bundle>；行级敏感扫描：
      ◇ 结构化字段命中→默认抽取为 secretref（用户可否决）｜自由文本命中→仅 Warning 不改写
[6] 与既有 profile 做 reconcile（同 origin.tool+path+id 整体更新）：
      目录健在但条目消失→标记 tombstone（reconcile 仅限本次 Location 边界）
      ⚠ 导出物占位符/空值回采→永不覆盖已有 secretref，冲突 Warning
[7] 展示 diff 摘要（新增/更新/墓碑/脱敏数）→ ◇ 用户确认（--yes 跳过常规确认）
[8] 打快照 → 写入 profile（atomicfile 通道）→ 更新 origin hash 索引
出口：Warnings 非空 → 退出码 5；否则 0。任何中断 → 快照可回滚（W5）。
```

## W2 导出流程（`cfg4ai export`）

> 依据：ARCHITECTURE §5（管线/写入协议）、CLI-SPEC §5、IR-SCHEMA §4

```
触发：export --to <tool> [--project P] [--ai] [--dry-run] [--force]
[1] 取锁 → Load 全局+项目 profile → Merge（五层序 merge-by-id/concat；项目墓碑遮蔽全局同 id）
[2] ◇ 有效配置为空 且 目标已有文件？→ ⚠ 空集保护：拒绝；--force 也需显式警告交互（--yes 不豁免）
[3] Map：IR → 目标模型（规则引擎）；按能力矩阵逐项判定降级（两级规则，ADAPTERS §5）→ 记 Warnings
[4] ◇ --ai？→ Assist（引擎层）：先 consent 检查
      ⚠ AI 配置段变更后首次 --ai → 强制重新 consent（D12）
      AI 结果必须 dry-run 预览+人工确认；--yes 不豁免；无人值守须 --ai-approve（记决策日志）
[5] Render → 目标格式字节流
[6] Verify 两级：格式校验（必做，失败即中止）；
      round-trip 自检：Export dry 写→重新 Import→语义 diff（忽略异构 x- 与白名单降级）→ 差异入 Warnings
[7] 外来内容检查（exports/ 清单）：◇ 文件不在清单=外来｜hash 变=被外部改 → 确认四选项：
      overwrite / skip / view-diff / backup-overwrite；--force=全部 backup-overwrite
      （比对前字节级规范化：CRLF→LF、去 BOM，防确认疲劳）
[8] 目标进程运行检测（Detect.Running）→ 提示"需重启/Reload 生效"
[9] ◇ --dry-run？→ 是：输出文件清单+diff，不落盘，退出
      否：自动快照目标区域 → 写入协议落盘（temp 同目录→fsync→rename→父目录 sync；
      ⚠ Windows 占用→指数退避重试→终失败报路径；⚠ 批量中途失败→逆序清理+快照恢复）
[10] 更新 exports/<tool>/<scope>/manifest.yaml（路径+hash+时间）
出口：逐条输出 Warnings（降级/跳过/secretref 占位）；非空 → 退出码 5。
```

## W3 迁移流程（`cfg4ai migrate`）

> 依据：CLI-SPEC §6。= W1（--tool from）+ W2（--to，--include-foreign）

```
[1] W1 全流程采集源工具（含 consent/扫描/墓碑规则）
[2] ◇ 交互确认"将源工具条目 applies_to 纳入目标"（批量改写指引）
[3] W2 全流程导出到目标（--include-foreign 纳入异构条目）
出口：两段 Warnings 合并输出；任一阶段中止则整体中止，快照保留。
```

## W4 同步流程（`cfg4ai sync`）

> 依据：CLI-SPEC §8、ARCHITECTURE §9（白名单/preflight）

```
push：
[1] 取锁 → preflight 全仓敏感扫描（含自由文本）
      ⚠ 命中→阻断或显式确认；CI 不得放行退出码 5（红队 T-05）
[2] 白名单过滤（profiles/、registry.yaml、config.yaml、exports/）→ git add/commit/push
pull：
[1] 取锁 → git pull → ◇ 冲突？→ 提示走标准 git 流程（本工具不封装合并）
[2] ◇ AI 配置段变化？→ 标记"下次 --ai 强制重新 consent"（D12）
[3] ◇ exports/registry 路径与指纹失配（换机/重定位）？→ 引导 rebase（路径重写+hash 重建）
[4] blob 悬空检查（imports[].blob/raw_blob 查无此物）→ 清单入 doctor；
      导出遇悬空降级：imports 保留正文路径+Warning / raw_blob 回退全量重渲染+Warning
```

## W5 恢复流程（`cfg4ai snapshot/restore`）

> 依据：CLI-SPEC §7

```
snapshot create：取锁 → SSOT 全量只读副本（manifest+blob 引用，天然去重）→ retention 计数
restore <id>：
[1] 取锁 → 对现状打反向快照（可回滚此次 restore 本身）
[2] ◇ 目标 IDE 运行中？→ 警告（热重载覆写风险）
[3] --dry-run 预览 → 确认 → 写入协议落盘
snapshot prune/gc：按 retention（默认近 20 份+按天去重）回收；blob 标记-清除
prune：物理清除墓碑条目 + 级联清理 keyring 孤儿
```

## W6 项目关联流程（`link` / `relink`）

> 依据：ARCHITECTURE §4、CLI-SPEC §4

```
link <path>：
[1] 计算指纹：git remote 规范化（去协议/去.git/host 小写/scp 转标准）> root_name+first_commit > root_name
[2] ◇ 命中注册项？→ 二次判别：first_commit 一致 + 用户确认 → 合并绑定；
      否则新建 pid 并记 same_remote_as（防多 clone 错并）
[3] 写入 registry.yaml（持锁）→ 后续 collect/export 即含项目 profile
relink [path]：同上流程反向查找（凭指纹认亲）→ paths 追加新路径 → 触发 exports 清单 rebase
```

## 流程-测试映射

每个流程的 `⚠` 保护路径与 `◇` 分支都有对应测试义务：W1[4]/W2[2]/W4[1] 等红队防线必须出现在对抗用例回归中（TEST-PLAN §4）；正常路径由 e2e testscript 覆盖（TEST-PLAN §3）。
