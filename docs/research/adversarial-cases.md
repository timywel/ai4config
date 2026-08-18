# 对抗用例库 — cfg4ai 设计文档 v0.2

> 构造时间：2026-08-16 ｜ 配套报告：[review/REDTEAM.md](../review/REDTEAM.md)（FMEA 编号 F*.* 与威胁编号 T-* 相互引用）
> 目的：把"v0.2 会在哪里失败"落成可复现的场景。每条给出精确输入状态（文件内容/目录结构），按 v0.2 文档条文推演预期行为，再指出文档未定义/会出错的点。
>
> **覆盖状态图例**：【未定义】= 四份文档对该点无任何条文；【部分覆盖】= 语义有定义但防护参数/边界行为空缺（已覆盖部分注明章节）；【已覆盖】= v0.2 已明确定义，列为回归锚点防未来退化。
> **验证手段图例**：单测（core/ir、atomicfile、store 层）/ e2e（testscript+golden-file，CLI 全链路）/ 手工（依赖真实 OneDrive、SSH、移动盘等环境）/ 文档澄清（先补设计再谈实现）。
>
> **统计**：独立用例 35 条 —— 【未定义】26、【部分覆盖】8、【已覆盖】1；组合故障 3 条（G 类，由已列未定义点级联而成，不重复计数）。

---

## A. IR 表达力对抗

| 用例名 | 输入状态 | 操作 | 按 v0.2 推演的预期行为 | 文档未定义/会出错的点 | 验证手段 |
|--------|---------|------|----------------------|----------------------|---------|
| AC-A1 import-cycle-direct | 项目根 `CLAUDE.md` 正文含 `@./docs/a.md`；`docs/a.md` 正文含 `@../CLAUDE.md` | `collect` → `export --to codex`（目标 instruction 为单文件） | 规则 9：imports 引用图 DFS 检环，记 Warning 不阻断（IR-SCHEMA §5）；`roundtrip_policy: preserve` 默认保留 @path 原样 | 【部分覆盖】环**检测**已覆盖；但 `roundtrip_policy: inline` 时展开环引用必然死循环/爆栈——inline 展开的环防御（深度上限？环点截断标记？）未定义。preserve 是默认值，但用户可配 inline | e2e（两种 policy 各跑一次） |
| AC-A2 import-cycle-via-symlink | `docs/x.md` 是指向 `docs/y.md` 的 symlink；`CLAUDE.md` 含 `@./docs/x.md`；`y.md` 含 `@./docs/x.md` | `collect` | DFS 按引用路径字符串建图：`x.md → y.md`（穿透后）vs 引用串 `x.md`——若按字符串判环，路径 `x.md→x.md` 成环；若按真实路径判环，`y.md→y.md` 成环。两条实现路径结果不同 | 【未定义】环检测的节点标识（引用字符串 vs EvalSymlinks 真实路径）未定义；与 §8"采集 lstat 不跟随"叠加后，imports 解析到底跟不跟链接，无条文 | 单测（建图函数）+ 文档澄清 |
| AC-A3 skill-name-case-dot | `~/.claude/skills/MyHelper.v2/SKILL.md`（目录名含大写与点号） | `collect` | 规则 1：name 须匹配 `[a-z0-9][a-z0-9-]*`；规则 7：id 末段必须等于所在目录名。`MyHelper.v2` 不合法 → 校验失败 | 【未定义】采集侧 id 派生规则缺失：是拒绝导入（Warning 跳过该 skill）还是规范化（`myhelper-v2`）？若规范化，与既有 `skill.myhelper-v2`（来自另一工具的目录 `myhelper-v2/`）撞 id 时谁赢？若拒绝，规则 7"必须等于目录名"与规范化必然冲突——两条规则打架 | 单测 + 文档澄清 |
| AC-A4 mcp-env-single-key | 全局 profile：`mcp.filesystem` 含 `env: {ROOT:/data, DEBUG:"1"}`、`args: ["-y","srv","/data"]`；项目 profile 同 id 条目仅含 `env: {DEBUG:"0"}`、`args: ["--verbose"]` | `export --to claude-code --project <p>` | §2.1 浅字段级：object 整体覆盖 → 有效 `env={DEBUG:"0"}`（`ROOT` 蒸发）；数组整体替换 → 有效 `args=["--verbose"]`（`-y srv /data` 全丢） | 【部分覆盖】合并语义本身已定义（§2.1【已覆盖】），但导出物 MCP server 必然启动失败，管线全程无一处 Warning 提示"继承键/数组元素被项目层丢弃"。用户视角是静默损坏 | 单测（合并引擎 Warning 断言） |
| AC-A5 setting-key-with-dot | `%APPDATA%\Code\User\settings.json` 含 `"editor.fontSize": 14`、`"workbench.colorTheme": "Dark+"`、`"[python]": {"editor.tabSize": 4}` | `collect --tool copilot` | 规则 1：Setting id 必须 `setting.<tool>.<key>` 三段式。key=`editor.fontSize` → `setting.vscode.editor.fontSize` 四段 → 校验失败；key=`[python]` 含方括号 → 同样违规 | 【未定义】VS Code 生态 key 普遍含点号/方括号，三段式与真实世界不兼容：转义规则（`setting.vscode."editor.fontSize"`？）、或末段合并解释（`tool=vscode, key=editor.fontSize` 按"第二段起全归 key"）均无条文。B5 修复引入三段式时未考虑 key 自身字符集 | 单测 + 文档澄清 |
| AC-A6 instruction-32k | 全局 8 条 instruction（各 3KB）+ 项目 6 条（各 4KB），concat 后约 48KB | `export --to codex --project <p>` | ADAPTERS §3.2 载明 Codex `project_doc_max_bytes` 默认 32KiB；合并→渲染产出 48KB 单文件写入成功，CLI 报告成功 | 【未定义】能力矩阵无容量维度（SupportLevel 只有 None/Partial/Full）；Verify 两级校验均不含"目标体积上限"检查；超限产物被 Codex 静默截断，用户无感知（联动 FMEA F3.2） | e2e（构造 33KiB 边界用例） |

## B. 格式对抗

| 用例名 | 输入状态 | 操作 | 按 v0.2 推演的预期行为 | 文档未定义/会出错的点 | 验证手段 |
|--------|---------|------|----------------------|----------------------|---------|
| AC-B1 jsonc-every-line-comment | `.vscode/settings.json`：JSONC，尾逗号，**每一行**都有行尾注释（团队规范说明），共 80 行 | `collect` → `export --to copilot`（回写同文件） | §1.3：注释/键序显式免责【已覆盖】；"注释密度高的源文件可入 blob 走 overlay patch 兜底" | 【部分覆盖】免责已覆盖；但"注释密度高"的**量化阈值未定义**——80/80 行注释算不算高？触发 overlay 的判定函数不存在，则兜底机制永不触发或随机触发，免责条款事实上无兜底 | golden-file + 文档澄清（阈值公式） |
| AC-B2 mixed-crlf-lf | `CLAUDE.md`：前 50 行 CRLF（Windows 编辑器写入），后 50 行 LF（WSL 里 `cat >>` 追加），混合比例 1:1 | `collect` → `export --to claude-code` | §8："保持源文件换行风格（采集时探测记录），未知则 LF" | 【未定义】混合文件的探测算法（首行？多数决？出现次数平局时？）未定义；1:1 比例下实现自由发挥，两次实现选择不同则 round-trip 不稳定 | 单测（探测函数 truth table） |
| AC-B3 gbk-claude-md | `CLAUDE.md` 为 GBK 编码（中文 Windows 老工具产出），含中文指令 200 行 | `collect` | IR-SCHEMA §1.1：文本统一 UTF-8/LF。GBK 字节流按 UTF-8 解码 → 乱码或 decode error | 【未定义】编码探测（chardet/BOM/声明）与转换策略未定义：乱码入库（污染 SSOT 且 sync 扩散）还是拒绝采集？导出回 GBK 环境时是写 UTF-8（原工具可能读不了）还是转回 GBK？ | 单测 + 文档澄清 |
| AC-B4 utf8-bom-settings | `~/.claude/settings.json` 带 UTF-8 BOM（`EF BB BF` 前缀，PowerShell `Out-File -Encoding utf8` 默认产物） | `collect` → `export --to claude-code` | Go `encoding/json` 对 BOM 直接报错；若实现 strip BOM 后解析成功，导出时是否加回 BOM 无规定 | 【未定义】BOM 的 strip/保留策略未定义；源有 BOM、导出无 BOM → `raw_hash` 永不匹配（每次 collect 都判"已变更"重新处理，增量机制失效） | 单测 |
| AC-B5 1mb-claude-md | `~/.claude/CLAUDE.md` 单文件 1MB（长期累积的企业规范） | `collect` → `cfg4ai diff` → `export --to codex --ai` | 采集入 instruction 实体（frontmatter+正文），SSOT 正常落盘 | 【未定义】无单文件大小护栏：① `sergi/go-diff` 对 1MB 文本 diff 的耗时/内存未评估；② `--ai` 语义转换必然超 token 上限，截断/分片策略未定义；③ 快照全量副本+按天去重，1MB×N 快照膨胀；④ 联动 AC-A6，Codex 上限必然爆 | e2e（性能基准+护栏断言） |
| AC-B6 toml-unknown-section | `~/.codex/config.toml` 含 `[mcp_servers.fs]`（cfg4ai 管理）+ 用户手工段 `[profiles.work]`、`[notice]` 及大段注释 | `export --to codex`（有 MCP 变更） | ADAPTERS §3.2：TOML 不保注释 → **整块重写**+快照兜底（§1.3 免责【已覆盖】注释部分） | 【未定义】"整块重写"是否保留 cfg4ai 未建模的 TOML 段（`[profiles.work]` 等非注释**内容**）？若按"解析→改 mcp 段→整体序列化"实现，未知段可保留；若按模板重渲染实现，未知段被抹——免责条款只免了注释，未免非注释段的丢失 | golden-file（含未知段用例） |

## C. 文件系统对抗

| 用例名 | 输入状态 | 操作 | 按 v0.2 推演的预期行为 | 文档未定义/会出错的点 | 验证手段 |
|--------|---------|------|----------------------|----------------------|---------|
| AC-C1 symlink-farm-root | `~/.claude` 本身是指向 `~/dotfiles/claude/` 的 symlink；其下 `CLAUDE.md`、`settings.json` 也是指向 dotfiles 各文件的独立链接 | `collect` → `export --to claude-code` | §8/ADAPTERS §2.9：采集 lstat 不跟随，"target 在已声明采集根内才解析"；写入 EvalSymlinks 穿透。采集根=`~/.claude`，链接 target=`~/dotfiles/...` 在根外 → 按字面**不解析** → 采到的是链接本身而非内容 | 【未定义】目录级链接与"配置根本身是链接"未建模：dotfiles 农场是目标用户群的标准形态，按字面规则该用户**什么都采不到**，而写入侧却穿透写进 dotfiles 仓库（git 工作区变脏）。采集/写入不对称。导出清单记录链接路径还是真实路径也未定 | e2e + 文档澄清 |
| AC-C2 readonly-mcp-json | `<proj>/.mcp.json` 设 Windows 只读属性（`attrib +R`，dotfiles 管理工具/chezmoi 常见做法） | `export --to claude-code --project <p>` | §5.3：rename 遇 ACCESS_DENIED 指数退避重试 N 次，仍失败报"文件被占用"并指明路径 | 【未定义】Windows `MoveFileEx` 覆盖只读目标返回 ACCESS_DENIED，与杀软持锁同码同路径 → 重试 N 次浪费数十秒后报**错误的原因**（"被占用"vs 实际"只读"）。Unix 侧 rename 只看目录写权限反而成功——同操作跨平台结果不一致 | e2e + 手工（Windows） |
| AC-C3 long-path-260 | 项目根 `C:\Users\Wel\source\repos\very-long-company-name\...\` 使 `<proj>\.claude\skills\<name>\SKILL.md` 全路径 268 字符 | `export --to claude-code --project <p>` | §8：`longPathAware` manifest + doctor 报告基准路径长度【已覆盖】 | 【部分覆盖】manifest 解决 cfg4ai 自身 API 调用；但 temp 文件名 `.<name>.tmp-<pid>-<rand>` 在原路径上**再追加 ~20 字符** → 已接近极限的路径因 temp 名越限失败；且目标工具自身无 manifest 时读不到该文件——导出"成功"但工具侧不可见 | 手工（Windows，边界长度矩阵） |
| AC-C4 onedrive-placeholder | 项目目录在 OneDrive"按需文件"模式目录下，`.claude/skills/` 下 30 个文件均为云占位符（未本地下载） | `cfg4ai scan` / `collect` | Detect 遍历目录、Import 读文件内容 → Windows 触发云端按需下载 | 【未定义】① scan（纯只读探测）是否会因 `lstat`/读首字节触发 30 个文件全量下载（用户流量+耗时）未定义；② 下载超时/失败时 collect 的降级（跳过该文件？整体失败？标墓碑？——联动 FMEA F2.1 灾难路径）未定义；③ 无 `FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS` 检测条文 | 手工（Windows+OneDrive） |
| AC-C5 case-insensitive-collision | 全局 profile 已含 `skills/Foo/`（来自 macOS 采集的目录 `Foo`）；另一来源采集到目录 `foo/` → 规范化后若小写化则撞 id，若不规范化则 SSOT 出现 `skills/Foo/` 与 `skills/foo/` 两目录 | `collect` → `sync push` → Windows 机 `sync pull` | 规则 1 的 name 字符集只允许小写 → 暗示派生 id 时小写化；两实体可能同 id 合并 | 【未定义】① id 派生的大小写规则与 AC-A3 同源未定义；② SSOT 物理目录名取 id 还是原始名未定——取原始名则 Linux 上两目录并存，sync 到 Windows/macOS 时 git checkout 直接冲突（同一路径）；取 id 则与规则 7"等于目录名"冲突 | e2e（Linux 建仓→Windows pull 矩阵） |
| AC-C6 target-dir-missing | 项目无 `.github/` 目录（从未用过 Copilot 项目级配置） | `export --to copilot --project <p>` | 导出布局要求写 `.github/instructions/*.instructions.md` → 需创建两级目录 | 【未定义】目录创建不在"单文件原子"承诺内：创建后rename 失败 → 残留空目录链（不在快照范围，restore 不会清理）；目录权限不足（如 `/opt/proj` 属 root）时的报错文案未定义 | 单测 |

## D. 并发/状态对抗

| 用例名 | 输入状态 | 操作 | 按 v0.2 推演的预期行为 | 文档未定义/会出错的点 | 验证手段 |
|--------|---------|------|----------------------|----------------------|---------|
| AC-D1 dual-collect-lock | 终端 1 执行 `collect`（大目录，扫描需 30s）；3s 后终端 2 执行 `export --to codex` | 两命令并发 | §7：写操作全程持 `.lock` 跨平台 flock 互斥【已覆盖】 | 【部分覆盖】互斥已覆盖；但终端 2 的行为未定义：阻塞等待？超时多久？立即失败（退出码几、报错是否指明持锁进程）？CI 并行 job 必踩 | e2e（两个进程并发，断言第二进程行为） |
| AC-D2 export-power-loss | `export` 流程：快照完成 → 10 个 temp 就位 → rename 完成 6 个时 `kill -9`（或断电） | 中断后重启，再跑 `export` | §5.3：批量非原子以快照补偿——但补偿发生在"失败时"，进程被杀意味着补偿逻辑根本没运行 | 【未定义】① 残留 4 个 `*.tmp-*` 与 6 个新文件：无启动时现场检测；② exports manifest 未更新（最后写？还是先写？时序未定义）→ 下次 export 把 6 个新文件误判"被外部修改"；③ manifest 自身写入是否走 atomicfile 未明文（"禁止适配器手写"未覆盖 store 层） | e2e（kill -9 注入点测试） |
| AC-D3 sync-pull-nonatomic | 机器 B `sync push`（更新 5 个 profile 文件+registry.yaml）；机器 A `sync pull` 拉到一半 `kill -9` | 中断后跑 `cfg4ai list` | CLI §8：sync 期间持仓库锁【已覆盖】；pull 冲突走标准 git 流程 | 【未定义】go-git 工作区 checkout 逐文件直写，不经 atomicfile（§5.3 禁令只约束适配器）→ registry.yaml 可处于半写状态 → 解析失败联动 AC-D4；pull 前是否对将更新文件打快照未定义 | e2e（pull 中断注入） |
| AC-D4 registry-yaml-corrupt | 手工编辑 `registry.yaml` 制造 YAML 语法错误（或 sync 冲突后残留 `<<<<<<< HEAD` 标记） | 跑任意命令（`list`/`collect`/`doctor`） | 启动加载 registry → 解析 panic/error | 【未定义】① 解析失败时是否**全部**命令不可用（包括能自救的 `restore`、`doctor`）？② 无 `.bak` 滚动备份条文；③ 快照里有 registry 副本（§7）但 restore 命令本身可能因启动加载失败而跑不起来——死锁 | e2e + 文档澄清（降级只读模式） |
| AC-D5 restore-external-modified | 快照 S1 后，外部编辑器手改 `~/.codex/config.toml` 加了一段配置 | `cfg4ai restore S1` | CLI §7：restore 原子化，**先对现状打反向快照**；目标 IDE 运行中警告 | 【已覆盖】反向快照机制定义完整，列为回归锚点。待验证：反向快照是否计入 retention 20 份名额（FMEA F11.2，restore 链会不会挤出旧快照——此子点仍属未定义） | e2e（回归锚点+retention 断言） |
| AC-D6 snapshot-under-cloud-lock | export 触发快照，目标工具配置区在 OneDrive 目录，快照复制途中 OneDrive 开始同步并持锁 | `export --to copilot` | 快照=SSOT 全量+目标工具配置区（§7） | 【未定义】① 快照读取被锁文件的降级（跳过该文件则快照不完整，restore 时静默缺文件？）未定义；② "目标工具配置区"精确边界未定义（FMEA F11.1：整个 VS Code User 目录 vs 仅 WrittenFile 集合） | 手工 |

## E. 语义对抗

| 用例名 | 输入状态 | 操作 | 按 v0.2 推演的预期行为 | 文档未定义/会出错的点 | 验证手段 |
|--------|---------|------|----------------------|----------------------|---------|
| AC-E1 same-remote-3-clones | `~/work/foo`、`~/exp/foo`、`D:\foo` 三个 clone，同一 remote（规范化后相同）、同一 first_commit；仅 work/foo 的 `.mcp.json` 有 server X | 三目录各跑一次 `link` → 二次判别确认合并为一个 pid → 然后在 exp/foo 跑 `collect` | §4：指纹命中+first_commit 一致+用户确认 → 合并（设计路径）；墓碑按 `(origin.tool, origin.path)` 判定，origin.path 存 raw 形态 `.mcp.json` | 【未定义】合并后互踩：exp/foo 的 `.mcp.json` 无 X → 同 `(tool, ".mcp.json")` 来源"X 已消失" → X 被标墓碑；下次 work/foo collect 又复活。配置随 collect 顺序振荡（FMEA F6.2）。二次判别防住了"错并"，没防"并后互踩" | e2e（三目录交替 collect 断言墓碑振荡） |
| AC-E2 rename-recreate-same-name | `F:\proj\foo`（项目甲，已注册 pid_A，git remote=github.com/a/foo）rename 为 `foo-bak`；原地新建 `F:\proj\foo` 为项目乙（git remote=github.com/b/bar 或无 .git） | 在新 `F:\proj\foo` 里 `collect` | registry paths 含 `F:\proj\foo` → 路径命中 pid_A；§4 的指纹计算/二次判别只写在 `link`/`relink` 命令语义里 | 【未定义】collect 路径命中时**是否强制复核指纹**无条文。不复核：乙的内容采进甲的 profile，且乙的 `.claude/` 缺甲的条目 → 甲的条目批量墓碑（FMEA F6.1+CF-3）。一条 collect 命令完成张冠李戴+原条目墓碑化 | e2e |
| AC-E3 cross-tool-import-loop | 项目根同时有 `CLAUDE.md`（含 `@./AGENTS.md`）与 `AGENTS.md`（含 `@./CLAUDE.md`） | `collect`（两工具各采一条 instruction） | 规则 9：imports 引用图无环（DFS）。但两条 instruction 分属不同 `(origin.tool, origin.path)` 条目——CLAUDE.md 的 imports=[AGENTS.md]，AGENTS.md 的 imports=[CLAUDE.md] | 【未定义】环检测的图论域未定义：单条目内部图（各自无环）还是全局引用图（有环）？若只建单条目图，跨条目环漏检；导出 inline 时两文件互相展开死循环（联动 AC-A1） | 单测 + e2e |
| AC-E4 x-field-version-drift | profile 含 `x-claude-code: {alwaysThinkingEnabled: true}`（v1.0 采集）；本机 Claude Code 升级到 v2.0（该字段改名 `thinking: {always: true}`，旧键废弃） | `export --to claude-code` | ADAPTERS §2.4：版本超 MaxVersion 继续执行但输出告警【已覆盖】；§1.1 规则：导出回该工具时 x- 仅补特有字段 → 旧键原样写回 | 【部分覆盖】版本告警已覆盖；但 x- 字段的**版本适配**未定义：写回已废弃键 → 轻则 Claude 启动警告未知字段，重则新版对同名字段赋予新语义（配置被解释错）。x- 无 ir_version 式的逐字段版本机制 | golden-file（模拟版本矩阵） |
| AC-E5 migrate-roundtrip-duplicate | 甲机只有 Claude Code：CLAUDE.md 采集为 `instruction.x`（`applies_to: [claude-code]` 缺省保守闭环） | `migrate --from claude-code --to codex`；之后在 Codex 侧正常使用；再跑 `collect --tool codex` | migrate 含 `--include-foreign`（CLI §5/§6【已覆盖】），条目导出到 `AGENTS.md`；collect 时该文件作为新来源采入 → 生成 `instruction.y`（origin.tool=codex，新 id） | 【未定义】同一语义内容在 SSOT 变两条独立条目（x: claude 系、y: codex 系），此后各自演化发散；§2.1"不跨工具自动合并条目"是有意设计【已覆盖】，但 migrate 场景产生的**同源重复**无去重/无链接机制，长期必发散 | e2e（migrate 往返断言条目数） |
| AC-E6 inherited-skip-no-hierarchy | 全局 profile 10 条 instruction；项目 profile 2 条；目标为无双层加载机制的假想工具 X（适配器只声明单文件） | `export --to X --project <p>`（默认 `materialize: inherited-skip`） | §3.4(c)：skip=仅写项目自有+覆盖条目，"依赖目标工具自身层级机制"提供全局部分 | 【未定义】目标无层级机制时 skip 语义破产：导出物只含 2 条，全局 8 条静默缺失——而 inherited-inline 才是对此类目标的正确选择。适配器能力矩阵无双层加载维度，引擎无法自动降级到 inline | 单测 + 文档澄清 |

## F. secret 对抗

| 用例名 | 输入状态 | 操作 | 按 v0.2 推演的预期行为 | 文档未定义/会出错的点 | 验证手段 |
|--------|---------|------|----------------------|----------------------|---------|
| AC-F1 apikey-in-instruction-body | `CLAUDE.md` 正文含真实可用 key（非教学示例）："调用内部服务统一使用 `sk-live-a1b2c3...`（40 字符）" | `collect` → `sync init <公共 GitHub 仓库>` → `sync push` | 规则 4：自由文本命中仅 Warning 绝不自动改写【已覆盖】；§9 白名单含 profiles/ → 该 instruction 文件入库 | 【部分覆盖】Warning 已覆盖；但退出码 5 在 CI 算通过（CLI §0）→ Warning 被无视 → 明文凭据随 push 上公共远端。sync push 前无内容侧二次扫描——白名单管目录不管内容（FMEA F8.1） | e2e（push 前扫描断言） |
| AC-F2 ssh-keyring-cron | Linux 服务器，SSH 会话（无 session D-Bus/Secret Service）；`secrets.backend: auto`；crontab 里配置了每夜 `cfg4ai collect --yes` | cron 触发 collect，遇到新 secret 需入库 | §9 降级链：keyring 不可用 → file（secrets.age，口令交互或环境变量注入）→ none | 【未定义】cron 环境无 TTY 且未设口令环境变量：file 后端的口令交互是挂起等输入（cron 僵尸进程）还是快速失败（退出码？）？无"非交互场景探测-降级"的行为矩阵；探测结果缓存策略（本次 keyring 不可用记住多久）也未定 | 手工（Linux cron 环境） |
| AC-F3 oversized-pem | `.mcp.json` 的 env 含 `MTLS_KEY`，值为 2.8KB PEM 私钥 | `collect` | IR-SCHEMA §3.5：单条 2KB 上限，超限拒绝并提示改用外部密钥管理【已覆盖"拒绝"动作】 | 【部分覆盖】拒绝后的数据流向未定义：明文留在实体 → 脱敏管线"零命中校验才提交"必然卡死（命中未处理）→ collect 整体失败？还是 Warning 放行明文入库（=泄漏通道，且 sync 白名单带走）？两个实现方向都是坑（FMEA F7.3） | 单测 + 文档澄清 |
| AC-F4 env-file-gitignored | 项目 `.env`（在 `.gitignore` 内，含 3 个真实 secret）；`.vscode/mcp.json` 的 server 配置含 `"envFile": "${workspaceFolder}/.env"` | `collect --tool copilot` → `export --to claude-code`（Claude `.mcp.json` 无 envFile 概念） | IR-SCHEMA §3.2：`env_file` 字段建模，相对路径按 origin.path 所在目录解析【已覆盖建模】 | 【未定义】① collect 是否读取/展开 env_file **内容**（读取则 .env secret 进管线——经脱敏抽取；不读则字段悬空）？② 导出到无 envFile 概念的目标时：内联展开（读 .env→secretref 化）还是降级 Warning 跳过？两条路径一条涉密一条功能缺失，无条文选择 | e2e + 文档澄清 |
| AC-F5 placeholder-recollect | 机器 A（keyring）正常；机器 B（SSH，`--secrets-backend=none`）从 git pull 到同一 profile | B 上 `export --to claude-code` → B 上 `collect --tool claude-code` → B 上 `cfg4ai prune` | §9：none 后端"secretref 保持占位符，导出物留空人工填"；规则 6：查询失败 Warning+占位符导出 | 【未定义】"留空"的具体形态未定义：空字符串 `""`/`"<TODO>"`/secretref 原文？若是空串 → B 的 collect 把空串当普通值采回，覆盖 SSOT 的 secretref → 引用断链；prune 时 A 机录入的 keyring 真值被判孤儿级联清理（FMEA F7.1+CF-4）。secret 全灭只需三步常规命令 | e2e（双机模拟） |

## G. 组合故障（级联剧本，对应 REDTEAM §2）

> 本类用例验证"单点防护在级联中互相失效"。每条给出精确时序脚本，期望系统的**整体**表现；不重复计入未定义行为统计（其组成点已在 A–F 类计）。

| 用例名 | 输入状态（时序脚本） | 操作 | 按 v0.2 推演的预期行为 | 文档未定义/会出错的点 | 验证手段 |
|--------|---------|------|----------------------|----------------------|---------|
| AC-G1 powerloss-vscode-sync（CF-1） | 时序：①`export --to copilot` rename 6/10 时断电；②重启后开 VS Code（自动重写 mcp.json：格式化+补新键）；③`sync pull`（队友推过新 profile，exports/ 未同步）；④再跑 `export` | 上述四步连跑 | 各环节独立看均"按设计"：单文件原子 ✓、IDE 提示 ✓、sync 白名单 ✓、manifest 识别 ✓ | 级联失效：temp 残留无清理（AC-D2）+ manifest hash 被 VS Code 重写打乱 → "被外部修改"误报（F5.2）+ 新 profile 与旧 manifest 错位 → 用户在连环确认中覆盖掉 VS Code 新写入的键。**期望行为（修复后）**：重启后首次运行报告"上次 export 中断现场"+ hash 规范化比对 + 语义 diff 确认 | e2e + 手工（断电用 kill 模拟） |
| AC-G2 disk-offline-tombstone-prune（CF-2） | `~/.claude` 是指向移动盘 `E:\dotfiles\claude` 的 symlink。时序：①E 盘未挂载跑 `collect` → 全墓碑；②未注意摘要跑 `export`；③发现配置消失，跑 `cfg4ai prune`；④挂载 E 盘 | 上述四步连跑 | ①墓碑机制 ✓ ②墓碑导出视为不存在 ✓ ③prune 物理清除+级联清理 keyring ✓ ④重新 collect 可重建条目 | 级联失效：①不区分"盘掉线"与"真删除"（F2.1）②空集导出物处置未定义（F2.4：写空/删除/跳过未定）③prune 前无强制快照（F11.3）→ keyring 真值不可逆清空。**期望行为**：源目录不存在时 collect 中止并报警、零墓碑；空集跳过不写；prune 前自动快照 | e2e（用临时盘符/挂载点模拟） |
| AC-G3 rename-sync-backflow（CF-3） | 机器 A：`F:\proj\foo`（甲，pid_A）rename 为 `foo-bak`，原地 clone 同名异仓 foo（乙）；A 上 collect（未 push）；机器 B（旧 registry）正常使用并 `sync push`；A 上 `sync pull`；A 在 `foo-bak` 跑 `relink` 并 export | 上述五步连跑 | ①路径命中 pid_A，指纹复核未定义（AC-E2）②git 自动合并 registry（paths 字符串相同，无冲突标记）③relink 凭指纹命中 pid_A | 级联失效：乙的内容已采进甲 profile+甲条目墓碑；B 的 push 与 A 的本地 registry 修改被 git 文本合并（无版本向量）；relink"成功"但 profile 内容已是乙的 → export 把乙的配置写进甲的目录。**期望行为**：collect 路径命中强制指纹复核；registry 合并冲突时 doctor 报警 | e2e（双机+git 远端模拟） |

---

## 附二：T11 核对记录（2026-08-17）

**可执行化完成（TestAdversarial_*，全绿）**：AC-A1（import 环 DFS）、AC-A3（id 规范化+放行）、AC-A4（浅合并行为）、AC-A5（setting 点号 key）、AC-B1（JSONC 边界记录）、AC-B4（BOM 剥离，**修复了 json 解析遇 BOM 报错的真实 bug**）、AC-E3（跨工具引用环）、AC-F1（自由文本仅 Warning 不改写）、AC-F5（占位符回采不覆盖，红队 T-03）。

**文档澄清类 12 条核对状态**：

| 用例 | 状态 | 说明 |
|------|------|------|
| AC-A2 symlink 环节点标识 | 部分实现 | 环检测按路径字符串建图（DetectImportCycle）；symlink 穿透判定留待采集层 |
| AC-A3 id 派生 | ✅ 已实现 | sanitizeIDName 规范化 + ParseID 放行点号/大写 |
| AC-A5 setting 三段式 | ✅ 已实现 | D2 放行点号；ParseSettingID 首点号分隔 |
| AC-B1 JSONC 注释阈值 | 记录 | JSONC 解析能力边界记录；密度阈值公式留待 P1 |
| AC-B3 GBK 编码探测 | 留待 | 编码探测/转换策略需设计（记录为变更候选） |
| AC-C1 symlink 农场 | ✅ 已实现 | 采集 lstat 不跟随 + 写入父目录穿透（T6） |
| AC-D4 registry 损坏恢复 | 留待 | registry 完整实现（含指纹/复核）在 P1；损坏降级只读模式待实现 |
| AC-E6 inherited-skip 无层级 | 留待 | 目标无层级机制时的 inline 自动降级待实现 |
| AC-F3 oversized PEM | 留待 | secretref 2KB 上限的拒绝/降级路径需在 sanitize 落地 |
| AC-F4 env_file 展开 | 留待 | env_file 已建模；内容读取/降级策略待实现 |
| AC-G1/G2/G3 组合故障 | 部分防护 | 防误判中止、空集保护、回采保护已实现并固化；断电现场检测、prune 前强制快照待实现 |

**新增变更候选**：① 合并覆盖导致继承键丢失时的 Warning（AC-A4 揭示的静默损坏风险）；② GBK/编码探测策略；③ JSONC 解析器引入评估。

---

## 附：验证手段落地建议

| 验证手段 | 建议落点 |
|---------|---------|
| 单测 | `internal/core/ir`（合并 Warning、id 派生、换行/BOM/编码探测、imports 建图）、`internal/atomicfile`（只读目标、temp 残留、中断注入）、`internal/core/registry`（指纹复核、损坏恢复） |
| e2e | `testscript` 剧本：本库每条 e2e 用例对应一个 `testdata/script/*.txtar`；中断注入用 `kill -9` 包装器；双机模拟用两个 `CFG4AI_HOME`+本地 git bare 仓库 |
| 手工 | Windows+OneDrive、SSH headless、移动盘掉线三类环境各建一页 checklist，发布前过一遍 |
| 文档澄清 | 标注"文档澄清"的 12 条（AC-A2/A3/A5、B1/B3、C1、D4、E6、F3/F4 及 G 类衍生）建议在动工前以 v0.3 修订闭环——这些不是实现选择问题，是语义真空 |
