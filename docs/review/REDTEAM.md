# 红队审计报告 — cfg4ai 设计文档 v0.2

> 审计时间：2026-08-16 ｜ 审计对象：ARCHITECTURE / IR-SCHEMA / CLI-SPEC / ADAPTERS（均 v0.2）
> 审计方法：FMEA（失败模式与影响分析）+ 对抗用例构造（见 [research/adversarial-cases.md](../research/adversarial-cases.md)）
> 与前序评审的关系：REVIEW-REPORT.md 回答"设计缺了什么"并已闭环；本报告回答"闭环之后，系统仍会在什么具体场景下丢用户数据、写坏用户配置、做出错误关联"。凡 v0.2 已覆盖的点，标注【已覆盖】并指出章节，不重复计数。

---

## 0. 摘要

- FMEA 覆盖 12 个组件，识别失败模式 **43 条**：残余风险 **高 13 / 中 20 / 低 10**。
- 对抗用例库 **38 条**（独立用例 35 + 组合故障 3）：独立用例中指向**完全未定义行为 26 条**、**部分覆盖（定义了语义但防护/参数空缺）8 条**、**已覆盖回归锚点 1 条**；组合故障 3 条由已列未定义点级联而成，不重复计数。
- 组合故障剧本 4 个（§2），均为多组件级联，单个组件的防护措施在级联中互相失效。

### 最关键的三个设计缺口

| # | 缺口 | 一句话 |
|---|------|--------|
| GAP-1 | **删除语义黑洞** | 墓碑判定不区分"盘掉线/读取失败"与"真删除"；项目层墓碑不遮蔽全局同 id；"合并后无任何条目时目标文件如何处置"全文未定义——三者叠加构成一条从误判到清空用户配置的完整级联（F2.1/F2.2/F2.4 + CF-2）。 |
| GAP-2 | **导出信任链跨机断裂** | `exports/` 清单不在 sync 白名单也不在排除清单（归属未定义）；hash 对比无字节级规范化；外来内容确认的选项集与 `--force` 行为未定义。换机/多 scope/目标工具自重写三类场景下，识别机制从"保护"反转为"确认疲劳后的批量覆盖"（F5.1–F5.4）。 |
| GAP-3 | **项目身份可被路径复用劫持** | 路径命中注册项时是否校验指纹未定义；多 clone 合并后 `origin.path` 以相对形态存储导致 `(tool, path, id)` 定位互踩。rename+原地重建、多机 registry 回流两个剧本均可实现 profile 张冠李戴（F6.1/F6.2 + CF-3）。 |

---

## 1. FMEA 报告

> 格式：失败模式 ｜ 触发条件（具体场景）｜ 用户可感知后果 ｜ 现有设计检测/缓解 ｜ 残余风险 ｜ 改进建议

### 1.1 IR 合并引擎（merge-by-id 浅字段级 / concat 两段式）

| # | 失败模式 | 触发条件 | 用户可感知后果 | 现有设计检测/缓解 | 残余风险 | 改进建议 |
|---|---------|---------|---------------|------------------|---------|---------|
| F1.1 | **数组整体替换陷阱** | 全局 `mcp.filesystem` 有 `args: ["-y","server-filesystem","/data"]`；项目层只想追加 verbose，写 `args: ["--verbose"]`。导出后全局 args 整体消失 | MCP server 在项目中启动失败（缺 `-y` 与路径参数），用户无从得知是合并语义所致 | 语义本身已定义（IR-SCHEMA §2.1【已覆盖】），但**无任何 Warning** 提示"项目层数组替换全局数组且元素减少" | 高 | 合并时输出数组替换 diff Warning（元素减少/首元素变更必报）；或 `merge_policy` 支持数组 `append` 策略 |
| F1.2 | **object 字段整体覆盖的键蒸发** | 全局 `env: {ROOT:/data, DEBUG:"1"}`，项目层写 `env: {DEBUG:"0"}`——§2.1 示例的反面：用户直觉是深合并，实际 `ROOT` 蒸发 | server 因缺 `ROOT` 环境变量行为异常 | 同 F1.1：语义已定义，示例甚至演示了键减少，但无"被覆盖键清单"提示 | 中 | 合并报告列出"本次覆盖丢弃的全局键"；文档明示"object 不递归" |
| F1.3 | **concat 边界标记被手工破坏后条目塌缩** | 导出物化插入 `<!-- cfg4ai:begin <id> -->` 边界注释（IR-SCHEMA §3.1）；用户手工编辑 CLAUDE.md 时删掉/调换了标记。下次 collect"无标记的旧文件按单条目整体导入" | 原 5 条目塌缩为 1 条目，新 id（按文件派生）与旧 id 脱节：旧条目被标墓碑，合并关系、per-条目 priority、`applies_to` 全部丢失 | 有标记↔无标记两条路径定义，但**标记部分损坏/错位**（5 个 begin 剩 3 个）的行为未定义 | 中 | 采集时校验标记配对完整性，损坏时 Warning 并按残留标记启发式拆分而非整体塌缩 |
| F1.4 | **双层 merge_policy 冲突未仲裁** | 全局 manifest 设 `instructions: concat`，项目 manifest 设 `instructions: project-only`。两层都有 manifest（IR-SCHEMA §2.2），合并时用哪层策略全文未定义 | 导出结果随实现选择漂移；同一 SSOT 在不同版本行为不同 | 未定义 | 中 | 明确规定"仅项目层 merge_policy 生效，全局层该键忽略并 Warning"（或反向），写入校验规则 |
| F1.5 | **Setting key 含点号击穿三段式 id** | VS Code settings 大量 key 含点号：`editor.fontSize`、`[python].editor.tabSize`。按规则 1 `setting.<tool>.<key>` 三段式，`setting.vscode.editor.fontSize` 变四段，校验自相矛盾 | VS Code settings 无法建模，采集报错或 id 被静默截断/转义（实现自由发挥） | 未定义（B5 修复引入三段式但未考虑 key 本身含点） | 高 | key 段允许转义（`setting.vscode."editor.fontSize"`）或改分隔规则；校验规则 1 与规则同步修订 |

### 1.2 墓碑机制

| # | 失败模式 | 触发条件 | 用户可感知后果 | 现有设计检测/缓解 | 残余风险 | 改进建议 |
|---|---------|---------|---------------|------------------|---------|---------|
| F2.1 | **"源端消失"不区分 ENOENT 与读取失败 → 全量墓碑** | `~/.claude` 在移动硬盘/网络盘上，盘符掉落或挂载失败；或 WSL 路径临时不可达。collect 时 Import 拿不到条目，与"用户删光了配置"无法区分 | 该 `(tool, path)` 来源全部条目标记墓碑；下一次 export 时墓碑视为不存在——目标配置文件被渲染成空壳（见 F2.4），用户配置"集体蒸发" | **未定义**。IR-SCHEMA §2.3 只说"本次源端已消失的条目标记 tombstone" | 高 | ① 采集区分"源目录不存在（盘掉线）"与"源文件不存在（真删除）"：前者直接中止并报警，不做任何墓碑；② 单次 collect 墓碑比例超阈值（如 >30% 或 >5 条）需显式确认 |
| F2.2 | **项目层墓碑不遮蔽全局同 id → "已删除"的配置复活** | 全局有 `mcp.filesystem`，用户在某项目里不需要它，删了项目层 `.mcp.json` 的对应段（或它从未在项目层存在过——用户需要的是"屏蔽"）。项目层墓碑（若产生）在合并时"视为不存在"→ 全局条目浮现进有效配置 | 用户在项目里明确删掉的 MCP server，导出后又出现在项目配置里 | 未定义。墓碑语义只定义了"导出视为不存在"，未定义其在**跨层合并**中的遮蔽效力 | 高 | 定义项目层墓碑对全局同 id 的遮蔽语义（墓碑参与合并并吞噬胜出）；或显式提供 `disabled: true` / `inherit: false` 机制并在文档写明"想屏蔽请用 X" |
| F2.3 | **墓碑重建竞态** | 用户删 server A → collect（A 墓碑）→ 用户重建同名 A（新内容）→ collect。(tool,path,id) 命中墓碑条目整体更新——tombstone 标志是否随整体更新清除，未明文规定 | 若实现未清除标志：重建的 server 导出时仍"视为不存在"，用户以为导出成功实际缺失 | 未明文（"整体更新"可推断含清除，但未写死） | 低 | 明文规定"collect 命中墓碑条目时整体替换并清除 tombstone 标志"，加校验 |
| F2.4 | **无条目时导出物处置未定义** | 级联于 F2.1/F2.2：合并后某目标文件对应条目为空集。导出是写空文件（`{"mcpServers":{}}`）？删除文件？还是跳过？三者在 F2.1 场景下后果截然不同（跳过=保命，写空=清配置，删除=清配置+丢手工注释） | 轻则配置文件变空壳，重则整文件消失 | 未定义。CLI-SPEC §5 只定义了"将创建/修改/跳过的文件清单"，未定义空集语义 | 高 | 明确规定"空集 → 跳过且不触碰既有文件"；删除文件必须是独立显式操作 |

### 1.3 迁移管线（Map/Assist/Render/Verify/Write）

| # | 失败模式 | 触发条件 | 用户可感知后果 | 现有设计检测/缓解 | 残余风险 | 改进建议 |
|---|---------|---------|---------------|------------------|---------|---------|
| F3.1 | **AI 输出无二次校验：secretref 被改写破坏** | Assist 阶段 AI 润色/翻译 instruction 或 prompt 正文，正文中含 `secretref://cfg4ai/global/mcp.fs/env.TOKEN`（用户写在 MCP 用法说明里）。AI 将其"修正"为 `secretref://cfg4ai/global/mcp.fs/env/TOKEN`（改分隔符）或直接删除。管线 Assist→Render 之间无任何完整性校验 | 导出文件含无效引用字符串；运行期 secret 解析失败且无任何 Warning 指向根因 | 出域脱敏管输入（§5.2），但**AI 返回侧无扫描、无 secretref 格式守恒校验**。规则 6 只管"后端查询失败"，不管"引用字符串被改坏" | 中 | Assist 输出后强制：① secretref 出现次数/取值守恒比对；② 输出侧敏感扫描（防 AI 编造伪 token）；③ 破坏即回退该条目的 AI 结果并 Warning |
| F3.2 | **降级产物无长度护栏 → 目标工具静默截断** | 20 个 skill 降级为 Copilot/Codex instruction 附录（ADAPTERS §5），concat 后超 Codex `project_doc_max_bytes`（32KiB，ADAPTERS §3.2 已载明上限）；或超 Copilot instructions 实践上限 | 导出"成功"，目标工具截断加载，排在后面的指令静默失效 | 能力矩阵只有 SupportLevel 三档，无容量维度；Verify 格式校验是否含大小检查未定义 | 中 | 能力矩阵增加容量维度；导出前总长度预检，超限 Warning + 按 priority 截断建议 |
| F3.3 | **`migrate --dry-run` 下 collect 副作用未定义** | `migrate = collect && export --include-foreign`（CLI-SPEC §6）。`--dry-run` 时 collect 是否仍写 SSOT、标墓碑、打快照？"中间产物正常落 SSOT"一句未排除 dry-run | 用户只想预览迁移效果，SSOT 已被改写、墓碑已标记 | 未定义 | 中 | 明确 dry-run 下 collect 走内存态不落盘；或拆分为两个显式步骤的提示 |
| F3.4 | **round-trip 忽略项 + 退出码 5 CI 放行 → 降级静默化** | Verify 忽略"异构 x- 与白名单降级项"，差异入 Warnings；Warnings 非空退出码 5；CLI-SPEC §0 建议 CI 将 0 与 5 都视为通过 | 逐次 export 累积的降级差异在 CI 中隐形，直到用户发现目标工具行为不对 | 设计上已有 Warnings 通道，但"5 视为通过"的建议**亲手关掉了警报器** | 中 | CI 建议改为"5 需人工复核新增的 Warning 条目"；提供 `--warnings-as-error` |

### 1.4 写入协议（temp+rename / Windows 重试 / symlink 穿透）

| # | 失败模式 | 触发条件 | 用户可感知后果 | 现有设计检测/缓解 | 残余风险 | 改进建议 |
|---|---------|---------|---------------|------------------|---------|---------|
| F4.1 | **快照补偿自身失败的路径未定义** | 批量 rename 第 5/10 个失败（权限），逆序清理+恢复快照；恢复过程中掉电/磁盘满/杀软锁文件 | 目标目录处于"部分新+部分旧+部分丢失"三态混合，无任何机制可回到一致态 | 单文件原子+快照补偿已定义（§5.3【已覆盖】至补偿启动为止）；**补偿失败后的状态报告与人工指引未定义** | 中 | 补偿失败时输出"现场冻结"报告（哪些文件是新、哪些是旧、快照 id）；doctor 增加半完成事务检测 |
| F4.2 | **只读文件误报"文件被占用"** | `.mcp.json` 被 `chmod 444` / Windows 设只读属性（dotfiles 管理工具常见做法）。Windows 下 rename 覆盖只读目标返回 ACCESS_DENIED——与杀软持锁同码，进入指数退避重试 N 次后报"文件被占用" | 用户按提示去关 IDE/杀软，重试仍失败，排查方向被误导 | 重试机制已定义（§5.3），但**错误分类未定义** | 低 | 重试前检测目标只读属性/权限位，只读场景直接报"文件只读：attrib -R / chmod +w" |
| F4.3 | **temp 残留清理责任未定义** | 进程在 temp 就位后、rename 前被杀（断电/任务管理器强杀）。`.<name>.tmp-<pid>-<rand>` 残留在目标目录 | Claude/VS Code 目录里累积垃圾文件；某工具若按目录扫描（如 skills 目录）可能把 temp 当有效内容加载 | 未定义 | 低 | 启动时/doctor 扫描已知目标目录的 `*.tmp-*` 残留并清理；pid 成分可用于判活 |
| F4.4 | **目录级 symlink 农场的采集规则自相矛盾** | `~/.claude` 整体是指向 `~/dotfiles/claude` 的 symlink（dotfiles 农场标准形态）。采集规则"lstat 不跟随，target 在已声明采集根内才解析"（§8/ADAPTERS §2.9）：target 在 `~/dotfiles` 不在 `~/.claude` 内 → 按字面不解析 → **采不到任何内容**；写入侧 EvalSymlinks 穿透又会写进 dotfiles 仓库 | 采集空 vs 写入穿透的不对称：用户 dotfiles 仓库被写入（git 工作区变脏），而采集端可能根本没读到这些文件 | 文件级链接已定义；**目录级链接（配置根本身是链接）未定义** | 中 | 明确定义"配置根目录级 symlink 一律穿透读取并记录真实路径，导出清单同时记录链接路径与真实路径" |
| F4.5 | **Windows 无父目录 Sync 等价物** | §5.3 fsync 链标注"（Unix）"；Windows 掉电后目录项（rename 记录）持久性依赖 NTFS 日志，粒度不同 | 极端掉电窗口下 Windows 侧目录项回滚概率高于 Unix | 半覆盖（Unix 链完整） | 低 | 文档明示 Windows 的等价保证级别（MoveFileEx+NTFS  journaling），或在 Windows 补 `FlushFileBuffers(目录句柄)` |

### 1.5 导出清单（外来内容识别）

| # | 失败模式 | 触发条件 | 用户可感知后果 | 现有设计检测/缓解 | 残余风险 | 改进建议 |
|---|---------|---------|---------------|------------------|---------|---------|
| F5.1 | **`exports/` 的 sync 归属未定义 → 换机信任链断裂** | sync 白名单仅 `profiles/`、`registry.yaml`、`config.yaml`（§9），排除清单含 snapshots/blobs/logs/cache/secrets.age/.lock——**`exports/` 两边都不沾**。换机后 manifest 缺失（若实现选择不同步）或含旧机绝对路径（若同步了，路径键失配） | 新机器上首次 export 把所有目标文件判为"外来内容"；用户使用 `--force`（或逐个点覆盖确认）→ **目标工具里多年手工维护的配置被 SSOT 整体覆盖** | 识别机制本身已定义（§5.3），但其状态的跨机生命周期未定义 | 高 | ① 明确 exports/ 入白名单（hash 仍然有效，路径规范化存储）；② 或提供 `cfg4ai export --adopt`（以现状重建 manifest）；③ 换机首次 export 强制全量 diff 展示 |
| F5.2 | **hash 对比无字节级规范化 → 确认疲劳 → 习惯性覆盖** | VS Code 重写 `.vscode/mcp.json`（格式化+键序重排）；Claude 向 `~/.claude.json` 写运行时字段；Prettier/EditorConfig 插件把 LF 改 CRLF。字节变、语义未变 → 每次 export 都弹"被外部修改"确认 | 用户被反复打扰后形成"闭眼点 Y"肌肉记忆；某次目标工具真写了有价值的新字段（如新版本自动迁移的键），被一并覆盖丢失 | 局部 patch 原则保护了非专用文件（§3.4b【部分覆盖】）；**专用文件（.mcp.json 整写）与 hash 规范化未定义** | 高 | hash 前做格式规范化（JSON 键排序+换行统一）；确认交互从"覆盖？"改为语义 diff 展示+三选（覆盖/跳过/以现状重建基线） |
| F5.3 | **同一物理文件被多 scope manifest 重复声明** | `~/.claude.json` 同时承载：全局 settings（局部 patch）、全局 user-scope MCP、**local-scope MCP**（按项目路径隔离 `"projects": {...}`，ADAPTERS §3.1）。local-scope 条目在 scope 归类上属 project 还是 global 未定义；若 project export 也写此文件，`exports/claude-code/global/` 与 `.../project/` 两份 manifest 都声称拥有它 | 交替执行 global/project export 时，两份 manifest 的 hash 互相过期，每轮都弹"被外部修改" | 未定义（scope 与物理文件多对一映射未建模） | 中 | manifest 以物理文件为主键唯一归属，hash 按 JSON 路径分段记录（只比对自己 patch 的区段） |
| F5.4 | **外来内容确认的选项集与 `--force` 语义未写死** | "不在清单=外来内容（确认）"——确认对话框给用户什么选项？覆盖/跳过/合并/备份？`--force` 跳过确认后是"覆盖全部外来"还是"跳过全部外来"？全文只能从"一致=直接覆盖"反推 | 实现若选错默认（force=覆盖），F5.1 场景直接变数据丢失事故 | 未定义 | 高 | CLI-SPEC 写死：确认选项={覆盖, 跳过, 全部跳过}；`--force`=对"一致"与"外来"均覆盖但**逐文件列出清单**；新增 `--force-keep-foreign` 或交互默认=跳过外来 |

### 1.6 registry 指纹（规范化+二次判别）

| # | 失败模式 | 触发条件 | 用户可感知后果 | 现有设计检测/缓解 | 残余风险 | 改进建议 |
|---|---------|---------|---------------|------------------|---------|---------|
| F6.1 | **路径直接命中时免指纹校验 → rename 重建张冠李戴** | `F:\projects\foo`（项目 A，已注册）被 rename 为 `foo-old`，原地新建同名目录 `F:\projects\foo`（项目 B，不同 git remote 或无 git）。B 里跑 collect：paths 列表命中 A 的注册项——**是否校验指纹，全文未定义**。若不校验：B 的内容采进 A 的 profile；B 的 `.claude/` 里没有 A 的条目 → (tool, path) 同源消失 → **A 的条目被批量墓碑** | A 的 profile 被 B 的内容替换+墓碑化；export 到 A（现在的 foo-old）时拿到 B 的配置 | 二次判别只定义在 link/relink 流程（§4/CLI §4）；**collect 自动路径命中的指纹复核未定义** | 高 | 路径命中注册项后强制校验指纹（git remote/first_commit/标记文件），不符即转入 relink 交互而非复用 pid |
| F6.2 | **多 clone 合并后 origin.path 相对形态互踩** | 同一 remote 的三个 clone（`~/work/foo`、`~/exp/foo`、`D:\foo`），first_commit 一致+用户确认合并为一个 pid（设计路径）。此后 A clone 采集 `mcp.filesystem`（origin.path=".mcp.json"）；B clone 无此 server，collect 时按 `(origin.tool, origin.path=".mcp.json", id)` 判定"同源已消失"→ **A 的条目被 B 标墓碑**。下一轮 A collect 又复活。配置在两个 clone 间随 collect 顺序振荡 | 项目级配置时有时无，导出结果取决于"最后谁 collect" | same_remote_as 防止了"错并"，但**合并后的多实例互踩未定义** | 高 | 条目定位键引入实例维度（如 `origin.host_path` 绝对形态或 registry path id）；或多 clone 合并时强制 Warning 声明此风险并建议独立 pid |
| F6.3 | **ssh config host 别名击穿规范化** | `~/.ssh/config` 定义 `Host gh → github.com`，remote 为 `git@gh:user/repo.git`。规范化只处理协议/.git/host 小写/scp 风格（§4），不解析 ssh config 别名 → 指纹 `gh/user/repo` ≠ `github.com/user/repo`，同一仓库被判为不同项目 | 换机（别名配置不同）后 relink 失败，产生孤儿 profile | 未定义 | 低 | 规范化时解析 ssh config host 别名（或至少 doctor 提示）；提供手动指纹修正命令 |
| F6.4 | **registry.yaml 损坏的启动行为与恢复路径未定义** | 用户手工编辑 registry.yaml 产生 YAML 语法错误 / sync 冲突解决后残留 `<<<<<<<` 标记。此后任何命令启动即解析失败——是拒绝全部命令？只读命令可用？有无 `.bak` 自动备份？doctor 能否修复？ | 全部命令不可用；用户手工修复 yaml 的能力参差不齐 | 快照含 registry 副本（§7）可推断能 restore，但**解析失败时能否跑到 restore 命令**本身成疑 | 中 | 启动解析失败时自动回退到最近快照的 registry 副本（只读模式运行+强 Warning）；写 registry 前自动 `.bak` |

### 1.7 secret 三级降级链

| # | 失败模式 | 触发条件 | 用户可感知后果 | 现有设计检测/缓解 | 残余风险 | 改进建议 |
|---|---------|---------|---------------|------------------|---------|---------|
| F7.1 | **占位符导出物被回采 → secretref 断链 → 级联清理真值** | 机器 B（SSH/无 keyring，`none` 后端）export：`secretref` "保持占位符，导出物留空人工填"（§9）——若"留空"实现为空字符串或 `<TODO>` 文本，`.mcp.json` 出现 `"API_TOKEN": ""`。之后 B 上跑 collect：空串不命中敏感规则 → 作为普通值**覆盖 SSOT 中的 secretref**。B 上 prune：keyring 条目（A 机录入的真值）被判孤儿级联清理 | 换机一圈回来，所有 secret 引用链断裂且真值被清，需逐个人工重录 | 规则 6 定义了"查询失败→占位符导出"；**占位符的具体形态、回采时的识别规则均未定义** | 高 | ① 占位符统一为 `secretref://` 原文（导出物仍是合法引用，回采幂等）；② 或 collect 识别"导出清单内文件的空值字段"并保留 SSOT 原值；③ prune 孤儿清理前校验"引用是否曾被占位符覆盖" |
| F7.2 | **同 profile 后端漂移，部分解析失败** | 机器 A（keyring）录入 5 个 secret；机器 B（file 后端）又录入 3 个。profile sync 后两机各有部分 secret。B 上 export：keyring 后端的 5 个查询失败 → Warning+占位符（规则 6），file 后端的 3 个正常 | 同一导出物内 secret 半真半占位，目标工具部分功能异常 | `secret_backend` 逐实体记录+doctor 报告（§9【部分覆盖】）；**无后端间迁移/再平衡工具**，混态长期存在 | 中 | 提供 `cfg4ai secrets migrate --to keyring|file`；export 时占位符数量>0 时提升 Warning 级别 |
| F7.3 | **>2KB secret 被拒绝后的数据流向未定义** | 用户把 PEM 私钥（~2.5KB）放进 MCP env。抽取时超限拒绝（IR-SCHEMA §3.5）——拒绝后明文是留在实体里？丢弃？留在实体则脱敏管线"零命中校验才提交"（§9）必然卡住：命中了却未处理 → collect 整体失败？还是 Warning 放行明文入库（=泄漏通道）？ | 实现二选一都是坑：collect 死锁，或明文静默入库+sync | 未定义 | 中 | 明文规定拒绝后的处置：字段保留原文+`secret_pending: true` 标记+sync 排除该字段；或强制用户改用 `command` 侧引外部文件 |

### 1.8 脱敏入库管线（双 hash）

| # | 失败模式 | 触发条件 | 用户可感知后果 | 现有设计检测/缓解 | 残余风险 | 改进建议 |
|---|---------|---------|---------------|------------------|---------|---------|
| F8.1 | **自由文本真 secret 仅 Warning → sync 明文凭据出域** | 用户在 CLAUDE.md 正文贴了**真实** API key（"记住用这个 key 调用内部服务"，非教学示例）。自由文本命中仅 Warning 绝不自动改写（规则 4【已覆盖】防误改写）；但 Warning 被忽略（退出码 5 在 CI 算通过）→ 明文入 profiles/ → `sync push` 推上 git 远端 | 真实凭据泄漏到远端仓库（可能是公共 GitHub） | Warning 环节已覆盖；**sync push 前的二次扫描阻断未定义**——白名单管的是目录，不管内容 | 高 | sync push 前对入库文本跑推送侧扫描，高置信命中阻断并要求显式豁免；doctor 定期全量扫描已入库 profiles |
| F8.2 | **结构化字段抽取"可否决"后明文入库** | 用户嫌 keyring 麻烦，在抽取确认时逐项否决。否决后明文存 SSOT（profiles/ 在白名单内）→ sync 上远端。否决决定是否记忆（下次 collect 同字段还弹吗）、否决时的风险告知强度，均未定义 | 同 F8.1，且是用户主动选择路径 | 未定义 | 中 | 否决时二次确认+"此值将明文同步到 git 远端"明示；否决记忆 per-field 并可在 doctor 列出"明文持有清单" |
| F8.3 | **熵检测误报 → 确认疲劳 → 全局关扫描** | instruction 正文贴证书 PEM、长 base64 blob、commit hash 列表 → 每次 collect 弹抽取确认。用户烦后关闭熵检测/加宽豁免 → 真 secret 漏过 | 防护体系被自身误报率瓦解 | 规则库外置+豁免清单已定义（§9）；**无误报率护栏（如"仅结构化字段启用熵检测，自由文本只用规则匹配"）** | 中 | 自由文本默认关闭熵检测（只跑规则）；豁免粒度细到字段级 |

### 1.9 sync 白名单

| # | 失败模式 | 触发条件 | 用户可感知后果 | 现有设计检测/缓解 | 残余风险 | 改进建议 |
|---|---------|---------|---------------|------------------|---------|---------|
| F9.1 | **blobs 排除 → imports/raw_blob 换机悬空** | `imports[].blob`（@path 引用快照）、`origin.raw_blob`（高注释密度兜底）都指 blobs/，而 blobs/ 强制 gitignore（§9【已覆盖】的泄漏防护）。换机后 blob 不存在：`roundtrip_policy: inline` 导出需要 blob 内容→查无此物；overlay 导出需要原文底→查无此物 | inline 导出产物缺引用内容（可能静默产空段落）；overlay 静默退化为全量重渲染（注释全丢） | 未定义 blob 缺失时的降级行为 | 中 | 明文规定：blob 缺失时 inline→退化为保留 @path+Warning；overlay→退化重渲染+Warning；评估"imports 引用 blob 入白名单"（脱敏后内容理论上可同步，与泄漏防护不矛盾） |
| F9.2 | **go-git pull 工作区 checkout 不走 atomicfile** | sync pull 更新 profiles/ 与 registry.yaml 由 go-git 工作区写盘完成——§5.3"禁止适配器手写写文件逻辑"约束适配器，sync 是 store 层行为，是否走 atomicfile 未定义。pull 中途崩溃 → registry.yaml 半写 | 联动 F6.4：解析失败，启动即死 | 未定义 | 中 | 明文规定 sync 工作区写盘必须经 atomicfile；pull 前对将被更新的文件打快照 |
| F9.3 | **Roaming 双通道覆盖 git 通道** | Windows 域环境 %APPDATA% 漫游：CFG4AI_HOME 内容被域策略在多机复制（配置+registry.yaml），同时 git sync 也在同步 registry.yaml。两条通道时序不可控：Roaming 把旧 registry 复制回已 git pull 到新版的机器 | registry 版本振荡，paths/profile 绑定错乱 | %LOCALAPPDATA% 分流已定义（§7【部分覆盖】，blobs/快照在 LOCAL，但 registry/config 在 Roaming） | 低 | doctor 检测 Roaming 漫游策略并 Warning；文档建议域环境将 CFG4AI_HOME 指向 LOCAL 或非漫游路径 |

### 1.10 文件锁

| # | 失败模式 | 触发条件 | 用户可感知后果 | 现有设计检测/缓解 | 残余风险 | 改进建议 |
|---|---------|---------|---------------|------------------|---------|---------|
| F10.1 | **锁等待语义未定义** | 终端 1 `collect`（持锁 30s，大量文件扫描中）；终端 2 `export`。第二个实例：阻塞等待？等多久？立即报错？退出码几？CI 里两个 job 并行跑 cfg4ai 必然相撞 | CI 随机失败；交互用户面对"卡住"或莫名报错 | 互斥本身已定义（§7【已覆盖】）；等待/超时/报错语义未定义 | 中 | 写死：默认等待 10s 指数退避，超时退出码 1+"锁被 PID xxx 持有（命令: collect，已持有时长）"；`--lock-timeout` 可调 |
| F10.2 | **网络文件系统上 flock 弱化** | CFG4AI_HOME 在 NFS/SMB（doctor 的云同步/共享挂载检测强 Warning【已覆盖】）。flock 在部分 NFS 配置下不跨机生效 | 双机同时写 SSOT，git 层与文件层双重损坏风险 | doctor 检测已覆盖；但 Warning 之后**是否允许继续运行**未定义 | 低 | 检测到共享挂载时要求显式 `--i-know` 才允许写操作 |

### 1.11 快照 / GC

| # | 失败模式 | 触发条件 | 用户可感知后果 | 现有设计检测/缓解 | 残余风险 | 改进建议 |
|---|---------|---------|---------------|------------------|---------|---------|
| F11.1 | **快照"目标工具配置区"精确范围未定义** | export 触发快照含"目标工具配置区"（§7）。VS Code user data 目录含 extensions、缓存、workspaceStorage——整目录快照一次数 GB；反之若只快照 cfg4ai 写过的文件，restore 时非 cfg4ai 文件无保障（它们本未被触碰，倒也合理） | 实现选左：快照目录爆盘；选右：用户误以为 restore 能恢复 IDE 全部状态 | 未定义边界 | 中 | 明确定义：快照范围=本次 export 的 WrittenFile 集合 ∪ 导出清单历史文件；文档写明 restore 不保证 IDE 整体状态 |
| F11.2 | **反向快照挤占 retention 名额** | `restore` 先打反向快照（CLI §7【已覆盖】）。反向快照计入"最近 20 份"吗？一次 restore-prune 链可能把用户珍视的旧快照挤出保留窗口 | 关键历史快照静默消失 | 未定义 | 低 | 反向快照独立命名空间/独立 retention；或 restore 时提示"将产生快照 N，保留策略 X" |
| F11.3 | **prune/gc 前是否自动快照未定义** | `prune` 物理清除墓碑+级联清理 keyring（不可逆）；`gc` 清除 blob（不可逆）。两者前均无快照义务规定。用户 prune 后立即发现误删墓碑里的重要历史 | 不可逆损失 | 未定义（export 前快照已定义【已覆盖】，破坏性维护命令没有） | 中 | prune/gc 强制先打快照（同 restore 反向快照机制） |

### 1.12 AI consent

| # | 失败模式 | 触发条件 | 用户可感知后果 | 现有设计检测/缓解 | 残余风险 | 改进建议 |
|---|---------|---------|---------------|------------------|---------|---------|
| F12.1 | **ai.base_url 经 sync 分发，变更无需重新 consent → 端点投毒** | `config.yaml` 在白名单内（§9），`ai.base_url` 存其中。团队 git 远端被入侵（或恶意 PR），pull 后 base_url 指向攻击者端点。consent 是"首次使用显式同意"——端点变化不属于触发条件。下次 `--ai` 导出，merge 后 Bundle（含内网拓扑、私有规范）发往恶意端点 | 配置内容批量出域到攻击者服务器，全程无提示 | 企业 allowlist 已定义（§5.2【部分覆盖】，个人用户无此配置）；**端点变更触发重新 consent 未定义** | 高 | `ai.base_url`/`ai.provider` 变更即重置 consent 状态；每次 AI 调用前校验 consent 快照（端点 hash）与当前值一致 |
| F12.2 | **全局/项目 profile 的 ai.enabled 冲突未仲裁** | 全局 `ai.enabled: true`，项目 profile `ai.enabled: false`（或反之）。`export --project` 时用哪个？ | 用户在项目里明确关了 AI，导出仍触网（或反之困惑） | 未定义 | 低 | 项目层优先；文档写明 |
| F12.3 | **决策日志不入 sync → 换机审计链断** | `--ai-approve` 记录决策日志（§5.2），日志在 logs/，强制 gitignore（仅 log_payload 语境，但 logs/ 整体在排除清单） | 合规审计时无法跨机还原"谁在何时批准了什么 AI 转换" | 半覆盖 | 低 | 决策日志（元数据级，不含原文）纳入白名单单独文件；文档明示取舍 |

---

## 2. 组合故障剧本（级联分析）

> 单组件防护在级联中互相失效的场景。每个剧本给出精确时序。

### CF-1：export 断电 × VS Code 热重写 × sync pull

```
T0  机器 A：export --to copilot --project F:\app（快照 S1 完成 → 10 个 temp 就位 → rename 完成 6 个）
T1  断电 / 系统重启
T2  重启后用户打开 VS Code：检测到 .vscode/mcp.json 变更（T0 写入的半成品），
    VS Code 自动重载并按自身逻辑重写该文件（格式化+补默认键）→ 字节级 hash 变
T3  用户跑 sync pull：profiles 更新到队友推送的版本；exports/ 未同步（F5.1），manifest 仍是旧 hash
T4  用户再跑 export：
    - 半成品状态无检测（F4.3 temp 残留无人清理）
    - manifest hash ≠ VS Code 重写后的文件 → "被外部修改"确认（F5.2）
    - 用户确认疲劳点覆盖 → VS Code 写入的新键（新版本自动迁移产物）丢失
    - 新 profile（T3）与旧 manifest 错位 → 更多误判确认
```

**级联要点**：写入协议保住了单文件原子（T0 每个文件要么旧要么新），但批量非原子 + manifest 时序 + IDE 自写 + sync 不同步 manifest，四个"局部正确"叠加成用户数据丢失。

### CF-2：盘掉线 × 全量墓碑 × export 清空 × prune 清理

```
T0  用户配置在移动硬盘 E:\（~/.claude 是指向 E:\dotfiles\claude 的 symlink，F4.4）
T1  硬盘未挂载；用户跑 collect → 该 (tool, path) 来源采到 0 条目 → 全部标记墓碑（F2.1）
T2  用户未注意墓碑摘要（或 --yes 跳过），跑 export --to claude-code：
    墓碑视为不存在 → 合并后条目空集 → 目标文件处置未定义（F2.4）
    若实现为"写空/删除"→ ~/.claude 下配置被清空
T3  用户发现配置消失，慌乱中按某教程跑 cfg4ai prune 想"清理重来"
    → 墓碑物理清除 + keyring 孤儿级联清理（F7.1 同款灾难，且 prune 前无强制快照，F11.3）
T4  硬盘重新挂载：源文件其实都在，但 SSOT 引用已断、keyring 真值已清
    → collect 可重建条目，但 secret 全部需要人工重录
```

**级联要点**：每个环节都"按设计行事"，端到端是配置清空+secret 丢失。F2.1 一个补丁（区分目录不存在与文件不存在）即可斩断整条链。

### CF-3：rename 重建 × 多机 registry 回流

```
T0  机器 A：F:\proj\foo（项目甲）rename 为 foo-bak；原地 clone 了同名不同仓库 foo（项目乙）
T1  A 上 collect：路径命中甲的注册项，无指纹复核（F6.1）→ 乙的内容采进甲的 profile，
    甲的条目批量墓碑；registry paths 追加（未变，因为路径字符串相同）
T2  A 上用户未 sync push
T3  机器 B（旧 registry：paths=[F:\proj\foo] 绑定甲）：正常使用并 sync push
T4  A 上 sync pull：git 自动合并 registry.yaml（paths 相同无冲突）→ B 的版本覆盖/混杂 A 的本地修改
T5  此后 A 在 foo-bak（真甲）里跑 relink：指纹规范化后命中甲 profile，但 profile 内容已是乙的
    → export 到 foo-bak：乙的配置写入甲的项目目录，甲的原始条目已从墓碑到 prune
```

**级联要点**：路径作为隐式主键 + 指纹复核缺位 + sync 无版本向量，项目身份被同名目录劫持。

### CF-4：none 后端占位符 × 回采 × prune（详 F7.1）

时序见 F7.1。要点：降级链的每一级单独看都合理（none 后端是显式逃生舱），但占位符形态未标准化，使"导出→回采→prune"三步构成引用链绞杀。

---

## 3. Top10 威胁清单（按残余风险 × 数据后果排序）

| 排名 | 威胁 | 涉及组件 | 触发链（一行） | 数据后果 | 残余风险 | 首选缓解 |
|-----|------|---------|---------------|---------|---------|---------|
| T-01 | 读取失败误判全量墓碑，级联清空目标配置 | 墓碑×导出×写入 | 盘掉线→collect 全墓碑→export 空集写空/删文件（F2.1+F2.4+CF-2） | 目标工具配置全灭 | **高** | 区分"源目录不存在"与"源文件删除"；空集跳过不写；墓碑比例阈值确认 |
| T-02 | 路径命中免指纹 → 项目张冠李戴 | registry×墓碑×sync | rename 原地重建→collect 采错 profile→批量墓碑→sync 扩散（F6.1+CF-3） | profile 错关联+原条目墓碑化 | **高** | 路径命中强制指纹复核，不符转 relink |
| T-03 | 占位符回采断链 → keyring 真值级联清理 | 降级链×脱敏×prune | none 后端导出空值→collect 覆盖 secretref→prune 清 keyring（F7.1+CF-4） | 全部 secret 永久丢失 | **高** | 占位符统一为 secretref 原文；prune 前强制快照 |
| T-04 | 换机 manifest 缺失 + --force → 覆盖手工配置 | 导出清单×sync | exports/ 归属未定义→新机全判外来→force 覆盖（F5.1+F5.4） | 目标工具多年手工配置丢失 | **高** | exports/ 入白名单或 --adopt 重建；写死 --force 语义与确认选项集 |
| T-05 | 自由文本真 secret 经 sync 明文上远端 | 脱敏×sync | 正文贴真 key→仅 Warning→CI 放行→push（F8.1） | 凭据泄漏（合规致命） | **高** | push 前推送侧扫描阻断+显式豁免 |
| T-06 | 多 clone 共享 profile 互踩墓碑 | registry×墓碑 | 三 clone 合并→origin.path 同形态→轮流墓碑（F6.2） | 项目配置振荡丢失 | **高** | 条目定位加实例维度；合并确认时风险明示 |
| T-07 | 项目层墓碑不遮蔽全局 → "已删除"配置复活 | 墓碑×合并 | 项目删 server→墓碑视为不存在→全局浮现（F2.2） | 违背用户删除意图，配置复活 | **中高** | 定义跨层遮蔽语义或提供 inherit:false |
| T-08 | hash 无规范化 → 确认疲劳 → 覆盖工具自写字段 | 导出清单×IDE | VS Code 自重写→hash 变→反复确认→闭眼覆盖（F5.2+CF-1） | 目标工具新键慢性丢失 | **中高** | hash 前规范化；确认交互改语义 diff |
| T-09 | ai.base_url 投毒无需重新 consent | AI consent×sync | config.yaml 入白名单→恶意 pull 改端点→下次 --ai 出域（F12.1） | 配置内容批量出域 | **中高** | 端点变更重置 consent |
| T-10 | blobs 不同步 → inline/overlay 导出静默缺内容 | sync×IR imports | 换机 blob 悬空→inline 缺引用段落（F9.1） | 迁移产物残缺且难察觉 | **中** | 定义 blob 缺失降级链；评估 imports blob 入白名单 |

候补（第 11–14 位）：F1.5（Setting key 含点号击穿 id，高危但属建模缺陷而非运行时数据丢失，动工期必修）、F3.1（AI 破坏 secretref）、F6.4/F9.2（registry 损坏无恢复路径）、F1.1（数组替换无 Warning）。

---

## 4. 已覆盖确认（审计中验证过、不凑数的点）

以下点在 v0.2 中已有明确定义，审计确认其设计方向正确，仅列章节备查：

| 点 | 覆盖位置 |
|----|---------|
| Codex enabled/disabled 极性取反 | IR-SCHEMA §3.2、ADAPTERS §3.2 |
| 合并语义精确定义（浅字段级/两段式/override） | IR-SCHEMA §2.1 |
| JSONC/TOML 注释、键序显式免责 + raw_blob overlay 兜底 | IR-SCHEMA §1.3 |
| 单文件原子写、temp 同卷同目录、fsync 链、Windows 重试 | ARCHITECTURE §5.3 |
| 写操作 flock 互斥、读快照读 | ARCHITECTURE §7 |
| sync 白名单制 + secrets.age/snapshots/blobs 强制 gitignore | ARCHITECTURE §9 |
| 指纹规范化规则 + 二次判别（first_commit+确认）防错并 | ARCHITECTURE §4、CLI-SPEC §4 |
| 敏感扫描分级处置（结构化抽取可否决/自由文本仅 Warning 不改写） | IR-SCHEMA 规则 4 |
| `--yes` 不豁免 AI 确认、`--ai-approve`+决策日志 | CLI-SPEC §0/§5 |
| restore 前反向快照、目标 IDE 运行中警告 | CLI-SPEC §7 |
| imports 引用图 DFS 无环校验 | IR-SCHEMA 规则 9 |
| secretref 2KB 上限、级联清理入 prune/doctor | IR-SCHEMA §3.5 |
| CFG4AI_HOME 云同步检测、0700/0600 权限、%LOCALAPPDATA% 分流 | ARCHITECTURE §7 |
| longPathAware manifest、doctor 路径长度报告 | ARCHITECTURE §8 |
| Codex 32KiB 上限事实载明（但未接防护，见 F3.2） | ADAPTERS §3.2 |

> 注：上述"已覆盖"指**语义定义**已覆盖；其中若干项的**运行时尚未接防护参数**（如 32KiB 只载明事实未做导出预检），此类在 FMEA 中以"部分覆盖"重新计入风险。
