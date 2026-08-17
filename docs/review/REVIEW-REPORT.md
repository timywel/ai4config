# 专家团审计报告 — cfg4ai 设计文档 v0.1-draft

> 审计时间：2026-08-16 ｜ 审计对象：`docs/ARCHITECTURE.md`、`docs/IR-SCHEMA.md`、`docs/CLI-SPEC.md`、`docs/ADAPTERS.md`
> 专家团构成：① 系统架构评审 ② 数据模型/schema 评审 ③ AI 工具生态事实核查（含官方文档实检）④ 跨平台与安全审计 ⑤ Go vs Rust 选型分析（单独成册，见 [GO-VS-RUST.md](./GO-VS-RUST.md)）
> 说明：多位专家独立发现的同类问题已合并，标注全部来源。

## 0. 总体结论

设计方向获一致认可：Hub-and-Spoke + IR 是此类问题的标准解，`x-<tool>` 透传、`origin` 追踪、能力矩阵降级、"规则优先 AI 兜底"均属正确锚点；文档完整度在草案阶段属上乘。事实核查**未发现硬性错误**（各工具配置地图通过率 86%–95%）。

但存在 **8 项 BLOCKER**——集中在三条主线上，动工前必须闭环：

1. **合并语义主线**：merge-by-id 粒度、采集再合并、删除传播三处核心语义空白（涉及 IR 的核心承诺"可合并"）。
2. **round-trip 主线**：Verify 环节无接口支撑、settings id 撞车、`@path` import 未建模、零丢失承诺边界未声明。
3. **安全落地主线**：keyring 无降级链（headless/麒麟必断）、blobs/快照/sync 存在明文 secret 泄漏通道、原子写入的平台失败模式未定义。

另：P0 路线图存在闭环矛盾（双层 merge 依赖 P1 的 link）；MAJOR 级问题 34 项（合并后 22 个主题）。

---

## 1. BLOCKER 清单（动工前必须解决）

| # | 主题 | 来源 | 问题 | 处置建议 |
|---|------|------|------|---------|
| B1 | **merge-by-id 粒度未定义** | 架构①、schema② | 三处文档只说"同 id 项目覆盖全局"，未定义整条替换 vs 字段级合并。后果截然不同（项目层只想改全局 MCP server 的一个 env 时怎么办） | IR-SCHEMA §1.2 新增合并语义专节：定为**浅字段级 merge**（标量/object 覆盖、数组整体替换、未出现字段继承），补全局 8 字段+项目 2 字段的合并对照示例 |
| B2 | **采集再合并与删除传播缺失** | 架构① | `content_hash` 只能检测变更，检测不到源端删除——SSOT 永久残留孤儿条目，违背 SSOT 承诺；增量 collect 的再合并规则（origin、多来源 x- 字段如何更新）未定义 | 定义"采集合并"规则（同 origin.tool+scope 整体 reconcile）+ **墓碑机制**（消失的条目标记 `tombstone: true`，导出视为不存在，`cfg4ai prune` 物理清除） |
| B3 | **Verify 环节无接口支撑** | 架构① | 管线第 6 步"round-trip 自检"需要导出后重新 Import 对比，但 `Export` 只返回 `error`，拿不到写入位置。P0 验收"round-trip 无丢失"依赖此环节 | `Export` 改返回 `([]WrittenFile, error)`（路径+hash）；引擎用返回值再 Import 得 Bundle' 做语义 diff（忽略异构 x- 字段与白名单降级项），差异入 Warnings |
| B4 | **P0 闭环自相矛盾** | 架构① | P0 验收含"双层 merge"，但 `link`/registry 排在 P1——项目 profile 无法关联磁盘目录，双层 merge 在 P0 既无法使用也无法验收 | 将最小 `link`（仅路径→pid 绑定，不做指纹/relink）提前到 P0；成本极低且让 P0 就能验证双层继承核心假设 |
| B5 | **Settings id 撞车** | schema② | 示例 `id: settings.model` + `tool_scope: [zhanlu]`——Claude 的 model 与 Zhanlu 的 model 同 id，直接违反唯一性校验 | id 规范改为三段式 `settings.<tool>.<key>`；校验规则同步；明确跨工具翻译映射表属导出器职责、不产生新 id |
| B6 | **`@path` import 语法未建模** | schema② | CLAUDE.md 支持 `@path/to/file` 导入（可嵌套 5 层），当前 schema 无字段承载：不采集则内容丢失，内联展开则破坏用户文件结构。round-trip 硬反例 | Instruction 新增 `imports` 字段（path+blob 快照+resolved 标记）与 `roundtrip_policy: preserve|inline`（默认 preserve：正文保留 @path 原样、引用入 blobs、导出还原目录结构）；校验补 imports 循环检测 |
| B7 | **keyring 无降级链** | 平台④ | go-keyring 的 Linux 后端依赖 session D-Bus + Secret Service + `login` collection——麒麟服务器版/headless/CI 必然不可用，而 CLI 恰恰支持 `--yes`（CI 场景）。IR 规则 4"强制转 secretref 或拒绝"会把 collect 卡死 | §9 定义三级降级链：系统 keyring → 加密文件（age/sops，口令或环境变量注入，CI 唯一出路）→ `--secrets-backend=none`（占位符导出、人工填）。IR 记录每个 secretref 实际后端；`doctor` 输出后端状态。库选型评估改 `99designs/keyring`（自带 file 后端自动探测） |
| B8 | **明文 secret 泄漏通道** | 平台④ | blobs"保真原始字节"与"脱敏"在同一字段上互相矛盾：`~/.claude.json`、`.mcp.json` 的 env 明文是常态，保真入 blob → 经 `sync push`（全目录 git、无白名单）推远端 = 明文具化出域。快照含"目标工具配置区"同理 | 写死方案：blob 存脱敏后内容 + 双 hash（`raw_hash` 增量比对、`stored_hash` 指向 blob），落盘管线强制"先扫描替换、后落盘、零命中校验"；`sync` 改**白名单制**（仅 profiles/、registry.yaml、config.yaml 入库，snapshots/blobs/logs/cache 强制 gitignore）；ADAPTERS 为每个工具补"已知含明文文件"清单 |
| B9 | **原子写入平台失败模式未定义** | 平台④ | 仅一句"temp+rename"缺四个硬约束：temp 必须与目标**同卷同目录**（Linux EXDEV / Windows 跨卷失败）；Windows 目标被占用（杀软/索引/IDE 持锁）即失败，无重试退避；缺 fsync 语义（写完未 sync 即 rename，掉电窗口不原子）；批量写入非原子与 §5.3"不留半成品"矛盾 | 新增"写入协议"小节：temp 建在同目录、写后 Sync+rename+父目录 Sync、Windows sharing violation 指数退避重试、批量事务边界（全部 temp 就位再逐一 rename，失败逆序清理+快照补偿）；§5.3 改为"单文件原子，批量以快照补偿"。封装内部 atomicfile 包，禁止适配器手写 |

---

## 2. MAJOR 清单（P0 内解决，按主题合并）

### 引擎与适配器职责
| # | 来源 | 问题 | 建议 |
|---|------|------|------|
| M1 | ① | **AI 语义转换职责矛盾**：管线暗示引擎主导，`ExportOpts.AIAssist` 又暗示适配器主导 | Assist 收到引擎层（持有能力矩阵+IR）；适配器保持纯粹（Import/Render/Write）；从 ExportOpts 删除 AIAssist。依赖链 `cmd → migrate → {adapters, aiassist, store}` 无环 |
| M2 | ① | **`--yes` 与"AI 转换必须确认"冲突**（CI 场景 `--yes --ai` 绕过确认） | `--yes` 不豁免 AI 确认；无人值守需显式 `--ai-approve` 并记决策日志 |
| M3 | ① | **"非本工具生成内容"无识别机制**（交互确认无法落地） | 持久化导出清单 `exports/<tool>/<scope>/manifest.yaml`（路径+hash+时间）：不在清单=外来（确认）；hash 变=被外部改（确认）；一致=直接覆盖 |
| M4 | ①+② | **能力降级规则两处不一致**（IR-SCHEMA 说降为 prompt 文件，ADAPTERS 说并入 instruction 附录） | 统一为两级规则："目标有最近概念→映射该形态；无→instruction 附录"；IR-SCHEMA 改为引用 ADAPTERS §5 |

### 合并与物化语义
| # | 来源 | 问题 | 建议 |
|---|------|------|------|
| M5 | ② | **concat 的 priority 与层级顺序冲突**（全局 priority=500 vs 项目 100 谁在前？项目默认值未给） | 两段式排序：先 scope 分层（global 块在前）、层内 priority 升序、同值按 origin.path 字典序；默认 global=100/project=200 |
| M6 | ② | **跨工具采集的同一性与 applies_to 默认值**：默认 `[all]` 会把 Codex 指令写进 CLAUDE.md 污染 round-trip | applies_to 缺省 = `[origin.tool]`（保守闭环）；migrate 流程提供批量改 applies_to 的交互；collect 按 (origin.tool, origin.path) 定位更新，不跨工具自动并条目 |
| M7 | ②+① | **多文件回写路由与部分拥有权**：4 个 settings 来源混采后如何分流回写？新增条目写哪？`~/.claude.json` 含运行时状态只能局部 patch | 三条规则：(a) 有 origin.path 的写回原文件，新条目按适配器声明的 `default_write_target`；(b) 非专用文件只许局部 patch 禁止整体重写；(c) `export --project` 声明 `materialize: inherited-skip（默认）| inherited-inline` |
| M8 | ② | **导出物化布局无规则**（多源文件采集→concat 后写单文件还是拆回多文件） | ADAPTERS 每工具表增加"导出布局"列；"物化布局由目标适配器唯一决定，origin 多文件信息不参与导出布局" |
| M9 | ② | **Settings 合并粒度与嵌套值**：entries 是列表却说"标量 override"；permissions/hooks 嵌套结构表示未定义 | 改"merge-by-id（条目级覆盖）"；key=顶层键、value=任意 YAML 嵌套值；permissions/hooks 等声明为不透明 value 不参与跨工具翻译 |
| M10 | ② | **McpServer 字段缺口**：缺 timeout（startup_ms/tool_sec）、env_file（VS Code envFile）、原始 name 保留（id 规范化后原键名丢失无法还原） | §3.2 增补 `name`（导出键名唯一来源）、`timeout{}`、`env_file` |
| M11 | ② | **Copilot applyTo 无承载**：`*.instructions.md` 的 `applyTo: "**/*.py"`（文件 glob 作用域）与 applies_to（工具维度）正交 | Instruction 新增 `file_patterns: []` 字段（对应 copilot applyTo / cursor globs） |
| M12 | ② | **子目录层级与 settings.local 无处安放**：Codex AGENTS.md 逐级覆盖、Claude settings.local.json（不入 git 第三层） | Instruction 增 `subtree` 可选字段；Settings 增 `local: true` 标记（sync 排除）；或至少在"已知限制"声明取舍 |

### 迁移保真与指纹
| # | 来源 | 问题 | 建议 |
|---|------|------|------|
| M13 | ①+② | **零丢失承诺边界未声明**：JSONC/TOML 注释、键序经 YAML 中转必然丢失；P0 验收无法判定 | §1.2 分级承诺：**字段级零丢失**（强承诺）/ **文档层不保证**（显式免责）；可选兜底：高密度注释源文件原文入 blobs（`origin.raw_blob`），导出 `render_mode: overlay` 时以原文为底做字段级 patch |
| M14 | ① | **git_remote 指纹两漏洞**：URL 未规范化（`git@` vs `https://` vs 尾部 `.git` 必然失配）；同 remote 多机多目录 clone 会被强行并为一项目 | 规范化函数（去协议/去 .git/host 小写/scp 转标准）；命中后二次判别（first_commit 一致+用户确认），否则新建 pid 并记 `same_remote_as` |
| M15 | ① | **多跳迁移 x- 生命周期未定义**（A→B→C：B 编辑后导回 A 时 x-A 与新值冲突规则） | 规则：x-<tool> 绑定"最近一次源自该 tool 的 import"；异构再采集不触碰他工具 x-；导回 A 时标准字段以新值为准、x-A 仅补 A 特有字段 |

### 安全与平台
| # | 来源 | 问题 | 建议 |
|---|------|------|------|
| M16 | ④ | **敏感扫描过粗且误报自残**：仅 3 类模式（漏 ghp_/glpat-/xoxb-/AIza/hf_/JWT 等）；instruction 正文里的教学示例 token 会被强制改写/阻断 | 规则库外置（参考 gitleaks 规则集+熵检测），支持自定义与豁免清单；处置分级：结构化字段命中→默认抽取（可否决），自由文本命中→仅 Warning 绝不自动改写 |
| M17 | ④ | **AI 数据出域无 consent**：出域的不止 secret——私有规范、内网 URL、git remote 内网域名；日志若记原文则 logs/ 成出域副本 | 首次使用显式 consent（per-profile `ai.enabled`）；脱敏范围扩至内网地址+可配置正则；AI 日志默认只记元数据，记原文需显式开关且强制 gitignore；企业场景提供端点 allowlist |
| M18 | ④ | **麒麟"等同 Linux"缺构建纪律**：纯 Go 静态编译是最大红利，但 CGO_ENABLED=0 未写成强制约束——任一 CGO 依赖回归即在 glibc 2.28 的麒麟上炸 | 发布构建强制 `CGO_ENABLED=0` + CI 断言（产物 ldd 为 static）；麒麟测试矩阵（V10 服务器 amd64/arm64 headless + UKUI 桌面）；架构矩阵可补 `linux/loong64`（龙芯，边际成本一行） |
| M19 | ④ | **SSOT 仓库权限无规定**（多用户 Linux 上 0755/0644 全员可读；CFG4AI_HOME 指向云同步目录无校验） | 目录 0700/文件 0600 每次写后校验；init/doctor 检测云同步/共享挂载强 Warning；区分 %APPDATA%（配置）与 %LOCALAPPDATA%（快照/blobs） |
| M20 | ④+① | **并发写无防护**：多进程/手工编辑/sync 并发直写 registry.yaml | `$CFG4AI_HOME/.lock` 跨平台 flock（gofrs/flock），写全程持锁、读快照读；stale lock 检测入 doctor |
| M21 | ④ | **CRLF 策略与 round-trip 冲突**（"按目标平台转换"会制造假漂移+git diff 噪音） | 默认"保持源文件换行风格（采集时探测），未知则 LF"，平台转换 opt-in；仓库内置 `.gitattributes` 钉死 eol=lf |
| M22 | ④ | **symlink 双向问题**：导出 temp+rename 会替换链接本身；采集跟随 symlink 可能越权读取（如指向 ~/.aws/credentials） | 写入前 EvalSymlinks 解析真实路径；采集默认 lstat 不跟随（链接本身作实体记录，target 在采集根内才解析）；export 回 Unix 还原链接、Windows 降级复制+Warning |

### 工程与路线图
| # | 来源 | 问题 | 建议 |
|---|------|------|------|
| M23 | ① | **Go plugin 不支持 Windows**，P3"插件化"按现表述必返工 | P3 直接锁定外置进程插件（hashicorp/go-plugin，gRPC over stdio），删除 Go plugin 选项；顺带允许非 Go 语言写适配器 |
| M24 | ① | **IDE 热重载未覆盖**（export 写 settings.json 时 IDE 运行中可能被内存态覆写） | Detect 增目标进程检测（best-effort）；export/restore 输出运行中提示 |
| M25 | ①+④ | **快照膨胀与 blob GC 缺失** | 快照改 manifest+blob 引用（天然去重）；retention 默认 N=20+按天去重；`cfg4ai gc`；blob 标记-清除；keyring 孤儿条目入 doctor 清单 |
| M26 | ② | **校验规则扩充**（现仅 5 条） | 补：secretref dangling（export 时 Warning+占位符、doctor 全量扫）；id 末段与目录/文件名一致；applies_to/origin.tool 取值校验；merge_policy 键值合法集；imports 无环；ir_version 高于实现版本拒绝 |

---

## 3. 事实核查摘要（专家③，含官方文档实检）

**结论：无 ❌ 级硬错误。** 键名（`mcpServers`/`servers`/`mcp_servers`）、目录层级、扩展名全部正确。需修订项：

| 优先级 | 工具 | 修订点 |
|--------|------|--------|
| 高 | Copilot | "settings.json 相关键"已过时（VS Code 1.102 起 code/test generation instructions deprecated）；补文件机制：profile user data 的 `*.instructions.md`、Agent Host 的 `~/.copilot/instructions`；补 `.github/agents/*.agent.md`、Agent Host 的 `~/.copilot/mcp-config.json` 与 workspace `.mcp.json`（键名差异备注） |
| 高 | Gemini | `<proj>/.gemini/` 应从 instruction 行移至 settings 行；**时效警告：官方公告 Gemini CLI 将过渡为 Antigravity CLI（2026-06-18 起）**，影响 P1 排期与版本护栏 |
| 中 | Claude Code | 全局 `~/.claude/commands/` 在现行官方文档已无依据（官方声明 commands 并入 skills，同名 skill 优先）——主路径改 skills，commands 标 legacy；补录 `.claude/rules/*.md`（含 user 级）、`CLAUDE.local.md`、managed 层、MCP local scope（也存 ~/.claude.json 但按项目隔离） |
| 中 | Codex | 补 `AGENTS.override.md` 优先机制、`project_doc_max_bytes`（32KiB 上限）、项目级 config 的 trusted-gate 与不可覆盖键清单；`~/.codex/prompts/` 无法核实（疑似被 skills/plugins 取代），标注待确认 |
| 低 | Zhanlu | 本地实证通过：`F:\config-code\.zhanlu\agent-manager.json`、`C:\Users\Wel\.config\zhanlu\zhanlu.json`、`~/.agents/skills/`（25 个 skill）；`kilo.json`/`.kilo/` 无公开文档且本项目无实例，标注"约定待校准" |
| 低 | P2 候选 | Cursor/Windsurf/Aider 等按同标准逐条核对后再标 P2 |

## 4. Go 栈专项评审意见（专家①④，架构层面）

1. **接口必须加 `context.Context`**：Adapter 四方法与 aiassist 均无 ctx（keyring/AI 网络/大文件需要取消与超时），P0 后补会全量返工。
2. **`init()` 注册需补聚合包**：未被 import 的包 init 不执行——增加 `internal/adapters/all/all.go`（集中 blank import），或改显式注册（测试隔离性更好）。
3. **`pkg/irschema` 公开承诺过早**：P3 走外置进程插件后无需 import 本仓库类型，建议 P3 前移除或标注 unstable。
4. **keyring 库选型复议**：建议 `99designs/keyring`（file 后端降级）替代 `zalando/go-keyring`。⚠️ 两位专家对 go-keyring macOS 后端是否为 CGO 存在事实分歧（④称源码核实为 exec `/usr/bin/security` 纯 Go，⑤称为 cgo 调 Security framework）——拍板前用一次 `GOOS=darwin go build` 试编译即可裁定，十分钟成本。
5. **TOML 选型缺口**：Codex 是 P0，§10 选型表漏 TOML 库（go-toml v2 不保注释——与 M13 保真边界声明联动）。
6. **§10 勘误**："hexops/valgrind → 用 sergi/go-diff"为编辑残留，清理为 `sergi/go-diff`。
7. **Bundle 实现提示**：merge-by-id 是热路径，引擎内建 `map[id]entity` 索引避免 O(n²)（写入 core/ir 包注释即可）。

## 5. 处置计划建议

**文档修订（建议在动工前一次性完成）**：
- IR-SCHEMA.md：B1/B2/B5/B6 + M5/M6/M9/M10/M11/M12/M13/M26（合并语义专节、字段增补、校验规则扩至 11 条、保真边界声明）
- ARCHITECTURE.md：B3/B4/B7/B8/B9 + M1/M2/M3/M14/M15/M17-M22/M23-M25（接口签名、P0 调整、写入协议、降级链、白名单 sync、并发锁、风险表补 IDE 热重载）
- ADAPTERS.md：事实核查修订 6 项 + 导出布局列 + default_write_target + 已知含明文文件清单
- CLI-SPEC.md：M2（--ai-approve）+ `--include-foreign`/`--only` 与 applies_to 呼应 + gc/prune 命令 + 退出码 5 适用规则

**优先级**：8 项 BLOCKER 对应修订 → MAJOR → MINOR/NIT 可随修订顺带吸收（本报告完整 MINOR 23 项、NIT 14 项存档于各专家原始意见，修订时逐条对照）。

**选型决策**：见 [GO-VS-RUST.md](./GO-VS-RUST.md)——三场景分析均倾向 Go（个人/小团队场景下 YAML 保注释生态 + 麒麟静态分发 + 纯 Go git 是决定性因素），维持 §10 预选但需补上构建纪律与代价清单。
