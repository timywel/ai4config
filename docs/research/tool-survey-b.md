# AI 编码工具生态调研（扩展 B 组）：IR 压力测试

> 日期：2026-08-16 ｜ 调研人：扩展 B 组（自动化调研）
> 基线：IR-SCHEMA v0.2（`docs/IR-SCHEMA.md`，实体：Instruction / McpServer / Skill / Agent / Command / Workflow / Setting；scope：global | project）
> 对象：OpenCode、Amp、Goose、Zed、JetBrains（AI Assistant 2026.2 + Junie CLI）、Trae（TraeCode）
> 方法：官方文档实检（webfetch + 浏览器渲染 SPA 页面）；结论均标注来源 URL；无法核实处标【存疑】。
> 主流五工具对照组：Claude Code / Codex / Copilot / Zhanlu / Gemini（见 `docs/ADAPTERS.md` §3）。

---

## 卡片 1：OpenCode

**一句话定位**：开源终端 AI 编码 agent（TUI 为主，另有桌面应用/IDE 扩展/Web/Server 多形态），以单文件 JSONC 配置 + Markdown 目录约定覆盖 agents/commands/skills/rules 全概念。

### 1.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 主配置 | `~/.config/opencode/opencode.json`（或 `.jsonc`） | `<proj>/opencode.json`（启动时自 cwd 向上至 Git 根搜索） | JSON/JSONC，`$schema: https://opencode.ai/config.json` | `model`、`small_model`、`provider`、`agent`、`command`、`mcp`、`permission`、`tools`、`instructions`、`formatter`、`lsp`、`snapshot`、`autoupdate`、`share`、`compaction`、`watcher`、`plugin`、`disabled_providers`/`enabled_providers`、`default_agent`、`subagent_depth`、`shell`、`server`、`attachment.image`、`experimental.policies` |
| TUI 配置 | `~/.config/opencode/tui.json` | `<proj>/tui.json` | JSONC（独立 schema `opencode.ai/tui.json`） | `theme`、`keybinds`、`scroll_speed`、`diff_style`、`cursor`、`mouse`、`attention`（legacy 的 `theme`/`keybinds`/`tui` 键自动迁移） |
| rules/instruction | `~/.config/opencode/AGENTS.md` | `<proj>/AGENTS.md`（cwd 向上逐级；无 AGENTS.md 时回退 `CLAUDE.md` / `~/.claude/CLAUDE.md`） | Markdown | 纯文本；`opencode.json` 的 `instructions` 数组可追加文件/glob/**远程 URL** |
| agents | `~/.config/opencode/agents/*.md` | `.opencode/agents/*.md` | Markdown + YAML frontmatter | `description`（必填）、`mode: primary|subagent|all`、`model`、`prompt`（支持 `{file:}`）、`temperature`、`top_p`、`steps`、`disable`、`hidden`、`color`、`permission`（per-agent 覆盖）；其余键透传给 provider |
| commands | `~/.config/opencode/commands/*.md` | `.opencode/commands/*.md` | Markdown + frontmatter | `description`、`agent`、`model`、`subtask`；正文 `template` 支持 `$ARGUMENTS`、`$1..$n`、`` !`shell` `` 注入、`@file` 引用 |
| skills | `~/.config/opencode/skills/<name>/SKILL.md`；兼容 `~/.claude/skills/`、`~/.agents/skills/` | `.opencode/skills/`、`.claude/skills/`、`.agents/skills/`（项目侧自 cwd 向上至 git worktree 逐级扫描） | SKILL.md + frontmatter | `name`（须匹配目录名，正则 `^[a-z0-9]+(-[a-z0-9]+)*$`）、`description`（≤1024 字符）、`license`、`compatibility`、`metadata` |
| mcp | `opencode.json` 的 `mcp` 段 | 同左（项目配置并入） | JSON | `mcp.<name>.type: local|remote`；local: `command[]`、`cwd`、`environment`、`enabled`、`timeout`；remote: `url`、`headers`、`oauth`（对象 `{clientId,clientSecret,scope}` 或 `false`）、`timeout` |
| permissions | `opencode.json` 的 `permission` 段 | 同左 | JSON | `permission.<tool>: allow|ask|deny` 或 glob 映射（如 `bash: {"*": "ask", "git status *": "allow"}`，**后匹配优先**）；`task` 键控制可调用的 subagent |
| 组织下发 / 托管层 | `.well-known/opencode`（远程组织默认）；`/Library/Application Support/opencode/`、`/etc/opencode/`、`%ProgramData%\opencode`（managed）；macOS MDM `ai.opencode.managed` plist | — | JSON / plist | 与 opencode.json 同构；优先级见下 |

**优先级链**（后者覆盖前者，配置**合并而非替换**）：remote（`.well-known/opencode`）→ global → `OPENCODE_CONFIG` 自定义文件 → project → `.opencode` 目录 → `OPENCODE_CONFIG_CONTENT` 内联 → managed 文件 → macOS MDM（最高，用户不可覆盖）。变量替换：`{env:VAR}`、`{file:path}`（相对配置文件目录或 `~` 绝对）。MCP OAuth token 存 `~/.local/share/opencode/mcp-auth.json`。

来源：https://opencode.ai/docs/config/ 、https://opencode.ai/docs/agents/ 、https://opencode.ai/docs/commands/ 、https://opencode.ai/docs/rules/ 、https://opencode.ai/docs/mcp-servers/ 、https://opencode.ai/docs/skills/

### 1.2 能力矩阵

| 能力 | 支持情况 |
|------|---------|
| instructions | ✅ AGENTS.md（项目+全局）+ CLAUDE.md 兼容回退 + `instructions` 数组（glob/远程 URL） |
| mcp | ✅ local/remote 双形态、OAuth（自动 DCR + 预注册凭据）、`enabled` 开关、per-agent 启停 |
| skills 类 | ✅ SKILL.md 六处加载位（含 Claude/agents 兼容位），permission.skill glob 控制 |
| rules | ✅（即 AGENTS.md 体系） |
| workflows | ⚠️ 无独立工作流实体；commands 的 `subtask`/`agent` 组合 + plugins（JS/TS）近似 |
| 独特机制 | primary/subagent 双层 agent 模型、内置 agent（build/plan/general/explore/scout/compaction/title/summary）、远程组织配置与 MDM 托管、tui.json 与主配置分离、`{file:}`/`{env:}` 变量、plugins（`.opencode/plugins/`、npm 包） |

### 1.3 独特概念清单（主流五工具没有或不全有）

1. **远程组织配置源**：`.well-known/opencode` HTTPS 端点下发组织默认配置（含 MCP 服务器默认禁用、用户按需 `enabled: true` 开启）。
2. **八层配置优先级**：含 `OPENCODE_CONFIG_CONTENT` 内联运行时覆盖、macOS MDM `.mobileconfig` 托管（企业管控粒度超过 Claude Code 的 managed 层）。
3. **agent 的 `mode`（primary/subagent/all）+ `hidden` + `color`**：agent 有"是否可被 Tab 切换/被 @ 提及/仅编程调用"的可见性维度。
4. **commands 模板占位符**：`$ARGUMENTS`/`$1..$n` 位置参数、`` !`cmd` `` shell 输出注入、`@file` 内容内联、`subtask` 强制 subagent 化。
5. **内置系统 agent**：compaction/title/summary 等 hidden system agent（IR 的 Agent 实体面向用户 agent，系统 agent 是另一物种）。
6. **`tools` 与 `permission` 双轨**（`tools` deprecated 转向 `permission`），permission 支持对 bash 命令、task（subagent 调用）、skill 的 glob 规则且**末条匹配优先**。
7. **tui.json 独立 schema**：UI 配置与 agent 行为配置物理分离。

### 1.4 IR 压力测试（对 IR-SCHEMA v0.2）

| # | 概念 | IR 能否表达 | 判定 |
|---|------|------------|------|
| 1 | `opencode.json` 顶层全部键 | ✅ Setting（`setting.opencode.<key>`，value 任意嵌套） | 通过 |
| 2 | agents markdown（mode/hidden/color/steps） | ⚠️ PromptPack 正文+frontmatter 可装；`mode`/`hidden`/`color` 无标准字段，须 `x-opencode` 透传 | 通过（x- 兜底） |
| 3 | commands（`$ARGUMENTS`、shell 注入、`@file`） | ✅ Command 正文原样保留；`@file` 可入 `imports` | 通过 |
| 4 | skills 六处加载位 | ✅ origin.path 记录；`applies_to` 标注 | 通过 |
| 5 | permission glob（bash/task/skill，末条优先） | ⚠️ Setting 可整体装载，但跨工具翻译无目标概念（Claude permissions allow/deny 规则近似可译） | 通过（降级翻译） |
| 6 | **远程组织配置 `.well-known/opencode`** | ❌ scope 只有 global\|project，无 remote/managed 层；且来源是 URL 而非文件路径 | **击穿 B-1** |
| 7 | **macOS MDM plist 托管层** | ❌ 同上；且格式为 plist 非 JSON | **击穿 B-1（合并）** |
| 8 | `instructions` 数组含**远程 URL** | ⚠️ Instruction.imports.path 可记录 URL，但"运行时拉取远程指令"的语义与 blob 快照冲突（内容会漂移） | 弱击穿 B-2 |
| 9 | MCP `oauth` 对象/`oauth:false`、`cwd` | ❌ McpServer 无 `oauth`/`cwd` 标准字段，仅 x- 透传；跨工具导出丢失 | 弱击穿 B-3 |
| 10 | primary/subagent + `default_agent` + `subagent_depth` | ⚠️ Setting 可装后者；前者无标准字段 | 通过（x- 兜底） |
| 11 | tui.json 独立文件 | ✅ Setting 按 origin.path 回写 | 通过 |
| 12 | 内置系统 agent（compaction/title/summary） | ⚠️ 非用户配置，不应采集；采集规则需排除 | 通过（适配器职责） |

### 1.5 真实样本

1. 官方 config schema 与示例（含 provider/mcp/permission）：https://opencode.ai/docs/config/ （schema 地址 `https://opencode.ai/config.json`）
2. 官方 agents 示例 `~/.config/opencode/agents/security-auditor.md`（frontmatter `mode: subagent` + `permission: edit: deny`）：https://opencode.ai/docs/agents/
3. 官方 skill 示例 `.opencode/skills/git-release/SKILL.md`（含 `license`/`compatibility`/`metadata`）：https://opencode.ai/docs/skills/
4. 源码仓库（文档页脚/编辑链接指向）：https://github.com/anomalyco/opencode

### 1.6 时效状态

- 文档最后更新 2026-08-14，高度活跃。
- 仓库现归属 `anomalyco/opencode`（原 SST 团队项目，组织已迁移；`brew install anomalyco/tap/opencode` 为新 tap）。旧 `sst/opencode` 路径仍被大量二手资料引用，适配器 Detect 时以新组织为准。
- 配置格式处于快速演进中：`tools` 键已 deprecated（转向 `permission`）、`maxSteps` → `steps`、`theme`/`keybinds`/`tui` 键迁移至 tui.json——golden-file 测试须跟上漂移。

---

## 卡片 2：Amp (ampcode)

**一句话定位**：Sourcegraph 系的"前沿 agent"，CLI + Web 双形态，以 threads（云端可分享会话）为一等资产，配置极简（单一 settings.json + AGENTS.md），靠 skills/plugins/checks 扩展。

### 2.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 主配置 | `~/.config/amp/settings.json`（或 `.jsonc`；Windows `%USERPROFILE%\.config\amp\settings.json`） | `.amp/settings.json`（自 cwd 向上搜至 repo 根；workspace 覆盖 user，唯 `amp.keymap` 反向） | JSON/JSONC | 全部键带 `amp.` 前缀：`amp.mcpServers`、`amp.mcpPermissions`、`amp.tools.disable`、`amp.skills.path`、`amp.skills.disableClaudeCodeSkills`、`amp.keymap`、`amp.showCosts`、`amp.notifications.enabled`、`amp.git.commit.*`、`amp.defaultVisibility`、`amp.thread.autoArchiveOnQuit`、`amp.remoteThreadCreation.enabled`、`amp.updates.mode`、`amp.fuzzy.alwaysIncludePaths`；legacy：`amp.permissions`、`amp.guardedFiles.allowlist`、`amp.dangerouslyAllowAll` |
| 托管层 | `/Library/Application Support/ampcode/managed-settings.json`、`/etc/ampcode/managed-settings.json`、`%ProgramData%\ampcode\managed-settings.json` | — | JSON | 同 settings.json + `amp.admin.compatibilityDate` |
| instructions | `$HOME/.config/amp/AGENTS.md`、`$HOME/.config/AGENTS.md`；系统级 `/etc/ampcode/AGENTS.md`、`/Library/Application Support/ampcode/AGENTS.md`、`%ProgramData%\ampcode\AGENTS.md` | `AGENTS.md`（cwd 及父目录至 `$HOME`；子树文件懒加载）；无 AGENTS.md 时回退 `AGENT.md`/`CLAUDE.md` | Markdown | 正文 `@path` 引用（支持 glob、`@~/`）；**被引用文件 frontmatter `globs:` 实现按文件类型粒度生效** |
| skills | `~/.config/agents/skills/`、`~/.agents/skills/`、`~/.config/amp/skills/`、`~/.claude/skills/`、`~/.claude/plugins/cache/`、`amp.skills.path` 自定义目录、内置、个人 skills 仓库（git）、workspace skills 仓库（git） | `.agents/skills/`、`.claude/skills/`（项目及父目录） | 目录 + SKILL.md | frontmatter `name`/`description`；**skill 可携带兄弟 `mcp.json` 或 frontmatter `mcpServers`**（含 `includeTools` glob 过滤） |
| checks（代码评审规则） | `$HOME/.config/amp/checks/`、`$HOME/.config/agents/checks/` | `.agents/checks/`（含子目录作用域，就近覆盖同名） | Markdown + frontmatter | `name`（必填）、`description`、`severity-default: low|medium|high|critical`、`tools`（子代理可用工具白名单） |
| plugins | `~/.config/amp/plugins/`（或 `$XDG_CONFIG_HOME/amp/plugins/`）；personal/workspace 插件 git 仓库 | `.amp/plugins/` | TS/JS（单文件或目录 `index.ts`） | `amp.on(event)`、`amp.registerTool/Skill/Command`、`amp.ai.ask`、`amp.createAgent`/`registerAgentMode`（须配 `// @amp-agent-mode` 注释） |
| mcp | `amp.mcpServers`（settings.json）或 `amp mcp add` CLI | `.amp/settings.json` 同键（**workspace 级 MCP 需显式 `amp mcp approve`**） | JSON | local: `command`/`args`/`env`；remote: `url`/`headers`；公共：`includeTools`（glob）；`${VAR}` 环境变量插值；OAuth token 存 `~/.amp/oauth/` |
| threads | 云端（ampcode.com/threads/T-xxx），非本地文件 | 同左 | — | `amp.defaultVisibility` 按 repo origin 映射 `private|workspace|group` |

来源：https://ampcode.com/manual （Configuration、Agent Skills、Code Review、Plugins、MCP 各节）

### 2.2 能力矩阵

| 能力 | 支持情况 |
|------|---------|
| instructions | ✅ AGENTS.md 多层（项目/父目录/用户/系统）+ globs 粒度化 + 兼容 CLAUDE.md |
| mcp | ✅ settings/CLI/skill 内嵌三通道；workspace 信任审批；OAuth；`amp.mcpPermissions` 命令/URL 模式 allow/reject 规则列表 |
| skills 类 | ✅ 11 级来源优先级 + skill 内嵌 MCP + git 仓库分发（personal/workspace） |
| rules | ✅（AGENTS.md 体系 + checks 这一评审专用变体） |
| workflows | ⚠️ 无声明式工作流；plugins（事件驱动 TS 代码）+ schedules（代理自调度）近似 |
| 独特机制 | threads 云端资产化与跨线程引用（`@T-<id>`）、Oracle（第二意见模型工具）、Librarian（跨仓搜索子代理）、orbs/runners（远程执行）、modes（low/medium/high/ultra 能力预设）、subagents 自动并行 |

### 2.3 独特概念清单

1. **Threads 一等资产**：会话存云端、可分享/引用/搜索（`label:`、`file:`、`parent:` 查询语法）——配置工具视角下的"会话层"概念。
2. **Checks（`.agents/checks/*.md`）**：代码评审专用的 prompt 包，带 `severity-default` 与工具白名单，**按目录就近覆盖**。
3. **Skill 内嵌 MCP**：`mcp.json` 随 skill 加载才暴露工具（上下文经济学），同名服务器 CLI/config 优先。
4. **Plugins 为 TypeScript 代码**：事件钩子（`session.start`/`tool.call`/`tool.result`/`agent.start`/`agent.end`）可 allow/reject/modify/synthesize 工具调用，`agent.end` 返回 `continue` 可续轮。
5. **Workspace MCP 信任审批**（`.amp/settings.json` 里的服务器须 `amp mcp approve`）。
6. **11 级 skill 来源优先级**（含两个 git 仓库源与 Claude 兼容目录），`amp.skills.disableClaudeCodeSkills` 开关。
7. **Orbs/Runners/Schedules**：远程机器执行与代理自定闹钟唤醒（运行时设施，配置仅 `amp.remoteThreadCreation.enabled` 等少量键）。
8. **系统级 AGENTS.md**（`/etc/ampcode/`、`%ProgramData%\ampcode\`）——instruction 的第三层 scope。

### 2.4 IR 压力测试

| # | 概念 | IR 能否表达 | 判定 |
|---|------|------------|------|
| 1 | settings.json 全部 `amp.*` 键 | ✅ Setting（key 含点号，映射 `setting.amp.<key>`） | 通过 |
| 2 | AGENTS.md 层级与 `@` 引用、被引用文件 `globs:` | ✅ Instruction + `imports`；globs 可映射 `file_patterns`（语义略有差异：Amp 的 globs 在被引用文件上） | 通过 |
| 3 | skills 11 级来源 | ⚠️ origin.path 可记录；但"personal/workspace git 仓库"是**远程分发渠道**，本地路径只是缓存 | 弱击穿 B-2（合并） |
| 4 | **skill 内嵌 MCP（mcp.json/mcpServers + includeTools）** | ❌ Skill 与 McpServer 是独立实体，无"skill 携带并按需激活 MCP"的关联字段 | **击穿 B-4** |
| 5 | **checks（severity-default/tools/目录作用域）** | ❌ PromptPack 的 `trigger.type` 枚举（slash-command\|mention\|manual\|hook）无"评审时自动激活"；`severity-default` 无标准字段 | 弱击穿 B-5 |
| 6 | **plugins（TS 代码 + 事件钩子）** | ❌ 无"可执行插件"实体；assets 可存文件但事件语义全丢 | **击穿 B-6** |
| 7 | threads/orbs/schedules | ❌ 非文件配置，超 IR 采集范围 | 范围外（需显式声明） |
| 8 | `amp.mcpPermissions` 规则列表 | ✅ Setting 不透明 value | 通过 |
| 9 | workspace MCP 审批状态 | ❌ 运行时状态（`amp mcp approve` 记录位置未在文档公开） | 范围外 |
| 10 | 系统级 `/etc/ampcode/AGENTS.md` | ❌ scope 无 system/managed 层 | **击穿 B-1（合并）** |
| 11 | MCP `includeTools`、`${VAR}` 插值 | ⚠️ `includeTools` 无标准字段；`${VAR}` 按既有"同工具保留、跨工具搬运+Warning"规则 | 弱击穿 B-3（合并） |

### 2.5 真实样本

1. 官方 settings 示例（`amp.mcpServers` 三个服务器 + `${SRC_ACCESS_TOKEN}` 插值）：https://ampcode.com/manual#configuration
2. 官方 check 示例 `.agents/checks/perf.md`（`severity-default: medium`、`tools: [Grep, Read]`）：https://ampcode.com/manual#code-review （手册内 "Code Review" 一节）
3. 官方 skill 内嵌 MCP 示例（`chrome-devtools` 的 `includeTools: ["navigate_*", ...]`）：https://ampcode.com/manual （"MCP servers in skills" 一节）
4. Neovim 插件仓库：https://github.com/ampcode/amp.nvim

### 2.6 时效状态

- 高度活跃（manual 含 GPT-5.6、Claude Fable 5 等 2026 模型路由）。
- 理念声明"No backcompat, no legacy features"——配置面漂移风险高（legacy permissions 已被 plugin 机制取代，`smart`/`deep`/`rush` modes 已废弃并映射到新 modes）。适配器版本护栏（MinVersion/MaxVersion）必须严格。
- 企业能力（managed-settings、MCP registry allowlist、Entitlements）在单独 appendix 维护。

---

## 卡片 3：Goose (block → AAIF)

**一句话定位**：开源通用 AI agent（Desktop + CLI + API，Rust 实现），MCP 深度集成最早最彻底，独有 recipes（参数化 YAML 工作流）与 .goosehints 双体系；2026-04 起归属 Linux 基金会 Agentic AI Foundation（AAIF）。

### 3.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 主配置 | `~/.config/goose/config.yaml`（macOS/Linux）；`%APPDATA%\Block\goose\config\config.yaml`（Windows） | —（无项目级 config.yaml） | YAML | `active_provider`、`providers.<name>.{enabled,model,configured}`、`GOOSE_TEMPERATURE`、`GOOSE_MAX_TOKENS`、`GOOSE_MODE: auto|approve|chat|smart_approve`、`GOOSE_MAX_TURNS`、`GOOSE_PLANNER_PROVIDER/MODEL`、`GOOSE_TOOLSHIM*`、`GOOSE_CLI_*`、`GOOSE_ALLOWLIST`、`GOOSE_RECIPE_GITHUB_REPO`、`GOOSE_AUTO_COMPACT_THRESHOLD`、`SECURITY_PROMPT_*`、`GOOSE_TELEMETRY_ENABLED`、`GOOSE_SEARCH_PATHS`、`otel_exporter_otlp_*`、`extensions` |
| 权限/秘密 | `~/.config/goose/permission.yaml`；`secrets.yaml`（keyring 不可用时明文降级）；`permissions/tool_permissions.json`（运行时自动管理） | — | YAML/JSON | permission.yaml 由 `goose configure` 写入 |
| extensions（MCP 等） | config.yaml `extensions.<name>` | recipe 内 `extensions` 数组 | YAML | `type: builtin|platform|stdio|streamable_http|frontend|inline_python`（`sse` 仅兼容保留）；stdio: `cmd`/`args`/`envs`/`env_keys`（**声明式 env 需求清单**）；http: `uri`/`headers`；公共：`enabled`、`timeout`（秒）、`bundled`、`available_tools`（工具白名单）、`display_name` |
| hints/instruction | `~/.config/goose/.goosehints` | `<proj>/.goosehints` 及任意子目录（嵌套加载，git repo 内自 cwd 至 repo 根 + 访问子目录时追加） | 纯文本 | 自然语言 + `@file` 内联引用；`CONTEXT_FILE_NAMES` 环境变量可改文件名（默认 `["AGENTS.md", ".goosehints"]`） |
| skills | 官方文档设独立章节（Agent Skills 规范） | 同左 | SKILL.md | 加载路径本次未逐页实检【存疑：以 using-skills 页为准】 |
| recipes（工作流） | 当前目录、`GOOSE_RECIPE_PATH` 目录列表、`GOOSE_RECIPE_GITHUB_REPO` 指定 GitHub 仓；Desktop 另有 recipe library 包装格式 | 同左 | YAML/JSON | `version`、`title`、`description`（必填）、`instructions`/`prompt`（至少其一）、`parameters[]`（`key`/`input_type: string|number|boolean|date|file|select`/`requirement: required|optional|user_prompt`/`default`/`options`）、`activities[]`（Desktop 气泡）、`extensions[]`、`sub_recipes[]`（`name`/`path`/`values`/`sequential_when_repeated`）、`settings.{goose_provider,goose_model,temperature,max_turns}`、`response.json_schema`、`retry.{max_retries,checks[].type:shell,command,on_failure,timeout_seconds}`；Jinja 模板 `{{ param }}`、`{% extends %}` 继承、内置 `{{ recipe_dir }}`；Desktop 包装：`{name, recipe:{...}, isGlobal, lastModified, isArchived}` |
| slash 命令 | config.yaml `slash_commands: [{command, recipe_path}]` | — | YAML | 命令名 → recipe 文件指针 |
| 提示模板 | `~/.config/goose/prompts/`（覆盖内置模板） | — | 文本 | — |
| 环境变量 | 全量 `GOOSE_*`（优先级高于 config.yaml）；`GOOSE_PATH_ROOT` 可整体迁移数据根 | — | — | 含 `GOOSE_MOIM_MESSAGE_TEXT/FILE`（**每轮注入 working memory**）、`AGENT=goose`、`AGENT_SESSION_ID`（注入子进程） |

来源：https://goose-docs.ai/docs/guides/config-files 、https://goose-docs.ai/docs/guides/environment-variables 、https://goose-docs.ai/docs/guides/context-engineering/using-goosehints 、https://goose-docs.ai/docs/guides/recipes/recipe-reference

### 3.2 能力矩阵

| 能力 | 支持情况 |
|------|---------|
| instructions | ✅ .goosehints（全局+嵌套）+ CONTEXT_FILE_NAMES 自定义文件名 + MOIM 每轮注入 |
| mcp | ✅ stdio/streamable_http + OAuth + allowlist（URL 白名单）+ 70+ 官方文档化扩展 |
| skills 类 | ✅ Agent Skills 规范（using-skills）+ prompt 模板覆盖 |
| rules | ✅（.goosehints 体系） |
| workflows | ✅✅ **recipes 是六工具中最强的工作流实体**（参数化、子 recipe 编排、结构化输出、重试校验） |
| 独特机制 | builtin/platform/frontend/inline_python 非 MCP 扩展、toolshim（弱模型工具调用解释层）、adversary mode（对抗审查）、计划模式独立 planner 模型、ACP server 双向身份 |

### 3.3 独特概念清单

1. **Recipes**：参数化（Jinja `{{ }}` + 类型/requirement 校验）、`sub_recipes` 并行/顺序编排、`response.json_schema` 结构化输出、`retry`（shell 检查 + on_failure 清理命令）、模板继承、GitHub 仓分发、Desktop library 元数据包装——远超"命名的 prompt 包"。
2. **非 MCP 扩展类型**：`builtin`（内置）、`platform`（代理进程内，如 `summon` 子代理委派）、`frontend`（前端提供工具）、`inline_python`（uvx 执行内联 Python）——扩展光谱超出 MCP client 范畴。
3. **`.goosehints` 嵌套懒加载**：访问子目录时才加载该目录 hints；`CONTEXT_FILE_NAMES` 把"指令文件名"本身变成可配置项。
4. **`env_keys` 声明式秘密需求**：recipe/extension 声明需要的 env 变量名，运行时交互式索取并存系统 keyring。
5. **`GOOSE_MOIM_MESSAGE_*` working memory**：每轮强制注入的持久文本（介于 instruction 与 system prompt 之间）。
6. **`GOOSE_MODE`（auto/approve/chat/smart_approve）+ toolshim**：工具执行策略与弱模型适配层。
7. **Desktop 包装格式**：同一 recipe 存在 CLI 格式与 Desktop library 格式（`{name, recipe, isGlobal, isArchived}`）两种序列化。

### 3.4 IR 压力测试

| # | 概念 | IR 能否表达 | 判定 |
|---|------|------------|------|
| 1 | config.yaml 全键（GOOSE_* 大写键） | ✅ Setting | 通过 |
| 2 | extensions stdio/streamable_http | ✅ McpServer（`cmd`→`command`、`envs`→`env`、`uri`→`url`；`timeout` 秒→`timeout.startup_ms` 需单位换算） | 通过（适配器换算） |
| 3 | **extensions builtin/platform/frontend/inline_python** | ❌ `transport` 枚举（stdio\|sse\|http）不含；这些根本不是 MCP client 配置 | **击穿 B-7** |
| 4 | **recipes 参数化/重试/结构化输出/子编排** | ⚠️ Workflow 实体只有 description/trigger 薄字段；全部进 `x-goose` 可保真但 IR 层无感知，跨工具导出必降级 | 弱击穿 B-8 |
| 5 | `env_keys` 声明式 secret 需求 | ⚠️ 无标准字段（与 IR 的 secretref 抽取方向相反：这是"占位待填"而非"已填抽取"） | 弱击穿 B-3（合并） |
| 6 | .goosehints 嵌套 | ✅ Instruction `subtree` | 通过 |
| 7 | CONTEXT_FILE_NAMES / MOIM 注入 | ✅ Setting；MOIM 文本亦可建模为 always-on Instruction + x-goose | 通过 |
| 8 | slash_commands → recipe_path 指针 | ⚠️ Command 正文可写指针，但"引用外部 recipe 文件"需 `imports` 机制配合 | 通过（组合表达） |
| 9 | Desktop 包装格式 | ⚠️ 同一实体两种序列化，适配器须识别 `recipe` 包装层 | 通过（适配器职责） |
| 10 | secrets.yaml 明文降级 | ✅ 与 IR secret 后端降级链同构（keyring→file）；采集时强制脱敏 | 通过 |

### 3.5 真实样本

1. 官方 config.yaml 完整示例（provider + extensions + GOOSE_* 混合）：https://goose-docs.ai/docs/guides/config-files
2. 官方 recipe 完整示例（parameters/extensions/settings/retry/response 全字段）：https://goose-docs.ai/docs/guides/recipes/recipe-reference
3. 官方 .goosehints 全局/嵌套示例（`@coding-standards.md` 引用）：https://goose-docs.ai/docs/guides/context-engineering/using-goosehints
4. 社区 recipe 仓（文档中 `GOOSE_RECIPE_GITHUB_REPO: "aaif-goose/goose-recipes"` 示例指向）与 Recipe Cookbook：https://goose-docs.ai/recipes ；主仓 https://github.com/aaif-goose/goose

### 3.6 时效状态

- 活跃；**重大变更：2026-04-07 官宣迁入 Agentic AI Foundation（AAIF，Linux 基金会）**，GitHub 由 `block/goose` 迁至 `aaif-goose/goose`，文档站由 `block.github.io/goose` 迁至 `goose-docs.ai`（旧站 302）。来源：https://goose-docs.ai/blog/2026/04/07/goose-moves-to-aaif
- 配置面注意：`sse` 扩展类型进入兼容保留；recipe 校验规则以源码 `validate_recipe.rs` 为准（文档明说）；Windows 配置根在 `%APPDATA%\Block\goose\`（品牌目录仍叫 Block）。
- Goose 有"Goose2 / ACP"演进线（ACP server、被 Zed/JetBrains 接入），config 热更新语义在迁移中（文档注明 Desktop 设置保存不重启但会话不切换 provider 实例）。

---

## 卡片 4：Zed（编辑器 AI）

**一句话定位**：开源高性能编辑器，AI 能力以 settings.json 单文件驱动（`agent.*` + `context_servers` + `language_models.*`），独有 agent profiles（模型+工具集预设）与三通道 agent 路径（Zed Agent / External Agents via ACP / Terminal Threads）。

### 4.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 主配置 | `~/.config/zed/settings.json`（Windows `%APPDATA%\Zed\settings.json`） | `<proj>/.zed/settings.json`【存疑：项目级路径为 Zed 惯例，本次实检页面未直接给出，以官方 configuring-zed 页为准】 | JSONC | AI 相关：`agent.*`（见下）、`context_servers`、`language_models.*`、`edit_predictions.*`、`features.edit_prediction_provider`、`disable_ai`、顶层 `profiles`（**编辑器 settings profiles**） |
| agent 行为 | settings.json `agent` 段 | 同左（项目覆盖） | JSON | `default_model.{provider,model}`、`profiles.<name>.{name,tools{},enable_all_context_servers,context_servers.<mcp>.tools{},default_model}`（内置 Write/Ask/Minimal）、`tool_permissions.default: confirm|allow|deny`（v0.224+；per-tool 规则 MCP 键格式 `mcp:<server>:<tool>`）、`auto_compact.{enabled,threshold}`、`inline_assistant_model`、`commit_message_model`、`thread_summary_model`、`compaction_model`、`subagent_model`、`commit_message_instructions`、`inline_alternatives`、`model_parameters[]`、`favorite_models`、`single_file_review`、`notify_when_agent_waiting`、`play_sound_when_agent_done` |
| instructions | `~/.config/zed/AGENTS.md`（Windows `%APPDATA%\Zed\AGENTS.md`） | 项目根首个匹配：`.rules`、`.cursorrules`、`.windsurfrules`、`.clinerules`、`.github/copilot-instructions.md`、`AGENT.md`、`AGENTS.md`、`CLAUDE.md`、`GEMINI.md` | Markdown | 纯文本（project 覆盖 personal） |
| skills | `~/.agents/skills/<name>/` | `<worktree>/.agents/skills/<name>/`（仅 trusted worktree 加载；仅平铺一层，不支持嵌套） | SKILL.md 目录 | `name`、`description`（≤1024B，目录编录总预算 50KB）、`disable-model-invocation`；资源 `scripts/`、`references/`、`assets/` |
| mcp（context servers） | settings.json `context_servers` 段；或 Zed 扩展（mcp-server-*） | 同左（项目 settings 亦可配） | JSON | `<name>.command/args/env`（本地）或 `url/headers`（远程）；无 Authorization 头时自动走 MCP OAuth |
| 编辑器 profiles | settings.json 顶层 `profiles.<name>.{base, settings{...}}` | — | JSON | 任意 settings 子集的命名预设（`settings profile selector: toggle` 切换） |
| External Agents | settings.json `agent_servers`（ACP 集成，Claude/Codex/OpenCode/Gemini 等） | — | JSON | ACP agent 注册 |

来源：https://zed.dev/docs/ai/agent-settings.md 、https://zed.dev/docs/ai/agent-profiles.md 、https://zed.dev/docs/ai/instructions.md 、https://zed.dev/docs/ai/skills.md 、https://zed.dev/docs/ai/mcp.md 、https://zed.dev/docs/reference/all-settings.md

### 4.2 能力矩阵

| 能力 | 支持情况 |
|------|---------|
| instructions | ✅ 个人 AGENTS.md + 9 文件名候选的项目指令（兼容一众友商文件名） |
| mcp | ✅ context_servers（本地/远程/OAuth）+ 扩展市场一键装 + `notifications/tools/list_changed` 热重载 |
| skills 类 | ✅ SKILL.md（`disable-model-invocation`、zed://skill 链接分享、project 覆盖 global） |
| rules | ⚠️ **Rules 已被官方废弃**：可复用 rules → Skills，always-on rules → Instructions；`.rules` 仅作兼容指令文件 |
| workflows | ⚠️ 无声明式工作流实体 |
| 独特机制 | **agent.profiles**（工具集+默认模型预设）、三 agent 路径（Zed/External ACP/Terminal）、编辑器 settings `profiles`、worktree trust、edit prediction 独立配置（provider + disabled_globs）、per-feature 模型分工（commit 消息/摘要/compaction/subagent 各用不同模型） |

### 4.3 独特概念清单

1. **agent.profiles**：命名预设 = 内置工具开关表 + MCP 服务器/工具开关 + 默认模型；与 tool_permissions（allow/deny/confirm）正交分层。
2. **三通道 agent 路径**：Zed Agent / External Agents（ACP 转发 Zed 的 MCP 配置）/ Terminal Threads（CLI 自读自有配置）——同一编辑器内多套配置边界并存（文档明示 Configuration Boundaries）。
3. **编辑器级 `profiles`**（顶层键）：整套 settings 的命名快照（演示/写作场景切换）。
4. **per-feature 模型设置**：`commit_message_model`、`thread_summary_model`、`compaction_model`、`subagent_model`、`inline_assistant_model` 各司其职；`agent.commit_message_instructions` 是场景化指令。
5. **worktree trust 门控 project skills/instructions**（安全边界）。
6. **zed://skill?data= 自包含分享链接**（base64url 内嵌 SKILL.md）。
7. **edit prediction 独立配置面**（`edit_predictions.provider`、`disabled_globs` 默认排除 `**/.env*` 等敏感文件）。

### 4.4 IR 压力测试

| # | 概念 | IR 能否表达 | 判定 |
|---|------|------------|------|
| 1 | `agent.*`/`language_models.*` 设置 | ✅ Setting（嵌套 value） | 通过 |
| 2 | **agent.profiles（工具集×模型预设）** | ⚠️ Setting 可整体装载，但它是被 UI/会话引用的一等功能实体；无跨工具对应概念（最近似：OpenCode agent mode、Amp modes） | 弱击穿 B-9 |
| 3 | context_servers | ✅ McpServer（command/args/env 或 url/headers 直接同构） | 通过 |
| 4 | AGENTS.md + 9 候选文件名 | ✅ Instruction（origin.path 记录实际文件名） | 通过 |
| 5 | skills（`disable-model-invocation`） | ✅ Skill + x-zed 透传该标志 | 通过 |
| 6 | **`.rules` 废弃迁移** | ⚠️ 采集端须把 `.rules` 识别为 Instruction 并记录 deprecation Warning；IR 无"已废弃"标记位 | 通过（适配器职责） |
| 7 | per-feature 模型/commit 指令 | ✅ Setting | 通过 |
| 8 | **worktree trust 状态** | ❌ 运行时信任标记，非配置 | 范围外 |
| 9 | External Agents（ACP 注册） | ⚠️ Setting 可装 `agent_servers`；但 ACP agent 自身的配置在对方工具里（边界问题） | 通过（仅采集 Zed 侧） |
| 10 | edit_predictions 配置 | ✅ Setting | 通过 |

### 4.5 真实样本

1. 官方 profile 示例（Dagger container-use：关全部内置工具、只开该 MCP 的 10 个工具）：https://zed.dev/docs/ai/mcp.md （引用自 https://container-use.com/agent-integrations#zed）
2. 官方 context_servers 配置示例（local/remote/oauth 三形态）：https://zed.dev/docs/ai/mcp.md
3. 官方 settings.json 全量示例（`// ~/.config/zed/settings.json` 注释开头）：https://zed.dev/docs/reference/all-settings.md
4. 生态：skills.sh 社区注册表（文档直接推荐 `find-skills`、`frontend-design`、`pdf`）：https://skills.sh ；主仓 https://github.com/zed-industries/zed

### 4.6 时效状态

- 高度活跃。**重大变更：Rules 机制 2026 年被 Skills + Instructions 取代**（官方 instructions 页明示"Migrating from Rules"），旧文档/二手资料中的 `.rules` 说法大面积过时；`agent.always_allow_tool_actions` 布尔于 v0.224.0 起被 `agent.tool_permissions.default` 取代。
- Zed 文档提供 llms.txt 与 .md 镜像（对采集器友好）。

---

## 卡片 5：JetBrains（AI Assistant + Junie CLI）

**一句话定位**：IDE 内嵌 AI（AI Assistant 插件，配 IDE 内部存储与 UI 驱动配置）+ 2026 年转型为独立终端 agent 的 Junie CLI（`~/.junie/` 全家桶：config.json/mcp/skills/agents/commands/hooks/trust）。

### 5.1 配置地图（AI Assistant 侧）

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 项目规则 | —（UI：Settings \| Tools \| AI Assistant \| Rules） | `<proj>/.aiassistant/rules/*.md` | Markdown + 编辑器内 Rule type 选择 | Rule type 五态：`Always` / `Manually`（`@rule:`、`#rule:` 引用）/ `By model decision`（须配 Instruction 描述）/ `By file patterns`（须配 Patterns 如 `*.kt`、`src/**`）/ `Off` |
| agent 指令 | — | `AGENTS.md` / `CLAUDE.md`（按 agent 各自约定） | Markdown | 纯文本 |
| mcp | UI 配置（全局级，"Server level: global"） | UI 配置（项目级） | JSON 片段 | `{"mcpServers": {"<name>": {"command","args"} }}` 或 `{"url": "..."}`；支持 STDIO/Streamable HTTP/SSE（legacy）；**可 Import from Claude（claude_desktop_config）** |
| skills | IDE 内部存储：`%LOCALAPPDATA%\JetBrains\<product><version>\aia\agents\.agents\skills`（macOS `~/Library/Caches/JetBrains/...`、Linux `~/.cache/JetBrains/...`）；外部 Git registry（GitHub 仓 URL） | `<proj>/.agents/skills`；亦可直接安装到 `<proj>/.codex/skills`、`<proj>/.claude/skills`、`~/.codex/skills`、`~/.claude/skills`（跨工具代管） | SKILL.md 目录 | Agent Skills 规范 |
| IDE 作为 MCP server | Settings \| Tools \| MCP Server（2025.2+ 内置插件） | — | — | 对外暴露 IDE 工具（analyze_calls/build_project/debugger xdebug_* 等）；可自动改写 Claude Code/Codex/VS Code 等客户端的 MCP 配置文件 |
| prompt library | UI 管理 | — | — | （未展开实检） |

### 5.2 配置地图（Junie CLI 侧）

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 主配置 | `~/.junie/config.json`（Windows `%USERPROFILE%\.junie\config.json`）；另有 `~/.junie/settings.json`（UI 设置，优先级高于用户 config.json） | `<proj>/.junie/config.json`（**须项目被信任才加载**） | JSON | `model`、`provider`、`brave`、`flags`、`mcp-locations`/`mcp-default-locations`、`skill-locations`、`command-locations`、`agent-locations`、`model-locations`、`auto-update`、`guidelines-location`、`time-limit`、`byok`、`proxies`、`hooks`；`--config-location` 可叠加任意文件 |
| guidelines | `~/.junie/AGENTS.md` | 发现顺序：`.junie/AGENTS.md` → `AGENTS.md` → `.junie/guidelines.md` 或 `.junie/guidelines/`（legacy，仍支持）；首开项目时检测他工具记忆文件并建议导入 `.junie/AGENTS.md` | Markdown | 纯文本；项目覆盖全局，相同内容自动去重 |
| mcp | `~/.junie/mcp/mcp.json` | `<proj>/.junie/mcp/mcp.json` | JSON | `mcpServers` 结构（与 IDE 版 Junie 相同）；`/mcp` 安装助手；OAuth；`--mcp-location` 叠加 |
| skills | `~/.junie/skills/<name>/` | `<proj>/.junie/skills/<name>/`（项目优先）；检测 `.cursor/skills/`、`.claude/skills/`、`.codex/skills/` 并建议导入 | SKILL.md 目录 | Agent Skills 规范 |
| subagents | `~/.junie/agents/`、`~/.agents/` | `<proj>/.junie/agents/`、`<proj>/.agents/`；检测 `.cursor/agents/` 等并建议导入 | Markdown + frontmatter | `name`、`description` 等；仅能自动委派，不可手动调用；模型策略 `SameModelOnly|Auto` 存 `~/.junie/settings.json` |
| commands | `~/.junie/commands/*.md` | `<proj>/.junie/commands/*.md` | Markdown + frontmatter | `description`；正文 `$argumentName` 命名参数（`/explain file=src/main.kt`） |
| hooks（EAP） | `~/.junie/config.json` 的 `hooks` 段（**项目 .junie/config.json 的 hooks 默认被忽略**，须显式 `--config-location` 才执行） | 同左 | JSON | 事件：`SessionStart`（matcher: source）、`UserPromptSubmit`、`PreToolUse`（matcher: 工具名）、`Stop`、`StopFailure`（matcher: 9 种错误如 rate_limit）、`SessionEnd`（matcher: reason）、`PermissionRequest`（matcher: 工具名）；hook: `{type: command, command, timeout, blockOnError}` |
| 允许清单 | `~/.junie/allowlist.json` | — | JSON | Action Allowlist（命令模式白名单，"Always allow" 累积） |
| 信任标记 | `~/.junie/trust`（标记文件，完整性密钥存系统 keychain/凭证管理器/Secret Service） | — | 标记文件 | exact-project 与 parent-directory 两级信任作用域 |
| 全局状态/凭据 | `~/.junie/`（secure_credentials.json 等） | — | JSON | BYOK key 等 |

来源：https://www.jetbrains.com/help/ai-assistant/configure-project-rules.html 、https://www.jetbrains.com/help/ai-assistant/mcp.html 、https://www.jetbrains.com/help/ai-assistant/agent-skills.html 、https://www.jetbrains.com/help/ai-assistant/configure-agent-behavior.html 、https://junie.jetbrains.com/docs/junie-cli-configuration.html 、https://junie.jetbrains.com/docs/guidelines-and-memory.html 、https://junie.jetbrains.com/docs/junie-cli-mcp-configuration.html 、https://junie.jetbrains.com/docs/agent-skills.html 、https://junie.jetbrains.com/docs/junie-cli-subagents.html 、https://junie.jetbrains.com/docs/custom-slash-commands.html 、https://junie.jetbrains.com/docs/junie-cli-hooks.html 、https://junie.jetbrains.com/docs/junie-cli.html

### 5.3 能力矩阵

| 能力 | AI Assistant | Junie CLI |
|------|-------------|-----------|
| instructions | ✅ AGENTS.md/CLAUDE.md（按 agent）+ `.aiassistant/rules`（五态规则） | ✅ AGENTS.md 三处发现 + legacy guidelines + 全局 `~/.junie/AGENTS.md` |
| mcp | ✅ UI 配置全局/项目级 + Claude 导入 + **IDE 自身当 server** | ✅ `.junie/mcp/mcp.json` 双级 + 安装助手 |
| skills 类 | ✅ IDE 缓存目录 + registry + **代管他工具目录** | ✅ 双级目录 + 他工具导入 |
| rules | ✅ 五态（Always/Manual/Model decision/File patterns/Off） | ✅（guidelines 体系） |
| workflows | ⚠️ 无 | ⚠️ hooks（EAP）为事件自动化；无声明式工作流 |
| 独特机制 | IDE 内嵌存储、Rule type 五态、IDE 即 MCP server、跨工具代管安装 | 项目信任标记（keychain 完整性校验）、allowlist.json、brave mode 三档、7 事件 hooks、legacy guidelines 兼容 |

### 5.4 独特概念清单

1. **IDE 内嵌版本化缓存存储**：AI Assistant 的 skills 装在 `%LOCALAPPDATA%\JetBrains\<product><version>\aia\agents\.agents\skills`——随 IDE 版本漂移的 Caches 目录，不是稳定配置路径。
2. **Rule type 五态**：尤其 `By model decision`（模型自决）与 `Off`（休眠不删）。
3. **IDE 作为 MCP server**：把 IDE 能力（构建、检查、调试器）反向暴露给外部 agent，配置方向与"IDE 消费 MCP"相反。
4. **跨工具代管安装**：AI Assistant 可把 skill 直接装进 `~/.claude/skills`、`~/.codex/skills`——一个工具写另一个工具的配置目录。
5. **Junie 项目信任体系**：信任标记 + keychain 完整性密钥 + parent 继承；untrusted 项目用临时隔离目录且不加载任何项目配置。
6. **Junie hooks 的安全默认**：项目仓库内的 hooks 默认不执行（防投毒），须显式 `--config-location`。
7. **Brave mode 三档 / Action Allowlist 累积式白名单**。
8. **配置优先级四层**：CLI flags > `~/.junie/settings.json` > 项目 config.json（须信任）> 用户 config.json。

### 5.5 IR 压力测试

| # | 概念 | IR 能否表达 | 判定 |
|---|------|------------|------|
| 1 | `.aiassistant/rules`（Always/File patterns） | ✅ Instruction（`file_patterns` 对应 Patterns） | 通过 |
| 2 | **Rule type: By model decision / Off** | ❌ `trigger.type` 枚举（slash-command\|mention\|manual\|hook）无 `model-decision`；`Off` 需 disabled 标志位 | **击穿 B-5（与 Amp checks 合并）** |
| 3 | AI Assistant MCP（UI 配置） | ✅ McpServer（mcpServers 片段同构）；但存储位置在 IDE 配置（options XML 或内部库），**非文档化文件路径**【存疑：具体落盘文件未公开】 | 通过（Detect 难度高） |
| 4 | **IDE 缓存目录 skills** | ⚠️ 路径含 `<product><version>`，版本升级即漂移；Detect 须枚举 JetBrains 产品×版本矩阵 | 弱击穿 B-10 |
| 5 | **跨工具代管（写 ~/.claude、~/.codex）** | ⚠️ 语义上属于目标工具的配置（cfg4ai 采集时归 claude-code/codex 适配器）；Junie 侧无残留文件可采 | 通过（归属目标工具） |
| 6 | Junie config.json 全键 | ✅ Setting | 通过 |
| 7 | Junie guidelines 三处发现 + legacy | ✅ Instruction | 通过 |
| 8 | Junie mcp/skills/agents/commands | ✅ McpServer / Skill / Agent / Command | 通过 |
| 9 | **Junie hooks（7 事件、matcher、timeout、blockOnError）** | ⚠️ 可装 Setting（`setting.junie.hooks` 不透明 value），但事件名与 Claude Code/Trae 高度趋同却无法跨工具翻译；项目级 hooks 的"默认不执行"安全语义丢失 | 弱击穿 B-11 |
| 10 | **信任标记/allowlist/secure_credentials** | ❌ keychain + 运行时状态 | 范围外（allowlist.json 可作 Setting 采集） |
| 11 | **配置优先级四层（flags > settings.json > project config > user config）** | ❌ IR 合并语义只有 global→project 两层 + merge_policy；Junie 存在"用户 settings.json 压过项目 config.json"的倒挂优先级 | 弱击穿 B-12 |

### 5.6 真实样本

1. 官方 guidelines 目录仓（Java/Spring Boot/Nuxt/Django/Gin，含 with-explanations 变体）：https://github.com/JetBrains/junie-guidelines （419 stars，README 明示 `.junie/AGENTS.md` 用法）
2. 官方 hooks 配置示例（SessionStart `aws sso login`、PreToolUse matcher `Bash`、SessionEnd matcher `prompt_input_exit|logout`）：https://junie.jetbrains.com/docs/junie-cli-hooks.html
3. 官方 subagent 示例 `.junie/agents/changelog.md`：https://junie.jetbrains.com/docs/junie-cli-subagents.html
4. 官方项目规则示例（General Code Review Guidelines md 直链）：https://resources.jetbrains.com/help/img/idea/2026.2/project_code_review_guidelines.md （来源页：https://www.jetbrains.com/help/ai-assistant/configure-project-rules.html ）

### 5.7 时效状态

- AI Assistant 文档版本 2026.2（2026-08 更新），活跃。
- **重大变更：Junie 由 IDE 插件转型为 Junie CLI 独立产品**（`junie.jetbrains.com/docs/` 新站；`www.jetbrains.com/help/junie/` 旧路径 301 至新站）。Junie CLI 的 hooks/subagents 模型策略等处于 EAP。
- 注意同名混淆：`AGENTS.md` 规范（agents.md）被 Junie/AI Assistant 同时采用；legacy `.junie/guidelines.md` 仍兼容但属过渡形态。

---

## 卡片 6：Trae（TraeCode / TraeWork）

**一句话定位**：字节跳动 AI IDE（VS Code 系 fork，IDE + SOLO 双模式，新产品线 TraeWork 客户端），概念覆盖最全（规则/技能/命令/记忆/钩子/MCP），项目配置集中在 `.trae/` 目录，全局在 `~/.trae/`。

### 6.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 规则（Rule） | `~/.trae/user_rules`（Windows `%userprofile%/.trae/user_rules`） | `<proj>/.trae/rules/**/*.md`（**最多 3 层嵌套**；任意子目录亦可放 `.trae/rules/` 按目录生效） | Markdown + frontmatter | `alwaysApply`（始终生效）/ `globs`（指定文件生效，逗号分隔通配符）/ `description`（智能生效场景描述）/ `scene: git_message`（**提交信息生成规则**）/ 无前述字段 = 手动触发生效（`#Rule` 引用）；兼容导入 `AGENTS.md`、`CLAUDE.md`、`CLAUDE.local.md`（设置开关） |
| 技能（Skill） | `~/.trae/skills`（Windows `%userprofile%/.trae/skills`） | `<proj>/.trae/skills/<name>/`；兼容 `<proj>/.agents/skills/`（须开启"启用 .agents 技能目录"开关；重名时 `.trae/skills` 优先） | SKILL.md 目录 | `name`、`description` + 正文（使用场景/指令/示例）；禁用列表落盘 `<proj>/.trae/skill-config.json`（仅记录被禁用的项目技能） |
| 命令（Command） | `~/.trae/commands` | `<proj>/.trae/commands/**/*.md`（最多 3 层嵌套） | Markdown | 名称/描述/指令（`---` 分隔）；SOLO Agent 内置 `/plan`、`/spec` |
| 记忆（Memory） | `~/.trae/memory/user_profile.md` | **全局目录下的项目分桶**：`~/.trae/memory/projects/{project_path}/project_memory.md` | Markdown | AI 自动维护（识别偏好/规则）+ 手动编辑；仅本地不可跨机 |
| 钩子（Hook） | `~/.trae-cn/hooks.json`（Windows `%userprofile%/.trae-cn/hooks.json`）【存疑：全局为 `.trae-cn` 而项目为 `.trae`，官方文档原文如此，疑似国内/国际版目录差异】 | `<proj>/.trae/hooks.json`（多项目工作区默认写入第一个项目）；可导入 Claude Code 的 `~/.claude/settings.json` / `.claude/settings[.local].json` hooks | JSON | `hooks.<EventName>[].{matcher, hooks[]}`；事件：`SessionStart`/`UserPromptSubmit`/`PreToolUse`/`PostToolUse`（matcher 正则匹配工具名）/`Stop`（可阻断要求续作）/`Notification`（异步）；stdin JSON、stdout（JSON 流程控制或纯文本附加上下文）、退出码 2 语义化；`TRAE_ENV_FILE` 环境变量注入文件；沙箱运行/本地自动运行两档 |
| mcp | UI 配置（设置 > MCP；可粘贴"原始配置 JSON"到全局 mcp.json）【存疑：全局 mcp.json 落盘路径文档未给出】 | `<proj>/.trae/mcp.json`（**默认关闭**，须开启"启用项目级 MCP"） | JSON | `mcpServers.<name>.{command,args,env}`（stdio）或 `{url,headers}`（HTTP/SSE/Streamable HTTP）；超时经 env/headers 注入 `START_MCP_TIMEOUT_MS`/`RUN_MCP_TIMEOUT_MS`；变量仅支持 `${workspaceFolder}`；命令中**不能含空格** |
| SOLO / Builder | SOLO 模式内置 `/plan`、`/spec` 命令；Builder 配置属 TraeWork 侧（本次未展开） | — | — | — |

来源：https://docs.trae.ai/ide/rules 、https://docs.trae.ai/ide/skills 、https://docs.trae.ai/ide/slash-commands 、https://docs.trae.ai/ide/memories 、https://docs.trae.ai/ide/hook-configuration-reference 、https://docs.trae.ai/ide/model-context-protocol 、https://docs.trae.ai/ide/add-mcp-servers （均为 SPA 页面，经浏览器渲染实检；附 `?_lang=zh` 参数可稳定复现）

### 6.2 能力矩阵

| 能力 | 支持情况 |
|------|---------|
| instructions | ✅ user_rules（全局）+ `.trae/rules/`（项目、3 层嵌套、子目录级）+ 兼容 AGENTS.md/CLAUDE.md/CLAUDE.local.md |
| mcp | ✅ 市场 + 手动 JSON；项目级默认关闭（安全默认）；超时用 env/headers 伪字段 |
| skills 类 | ✅ SKILL.md + 内置技能（TRAE-generate-mini-app/TRAE-debugger/TRAE-code-review）+ .agents/skills 兼容 + skill-config.json 禁用列表 |
| rules | ✅ 四种生效方式（始终/指定文件/智能/手动）+ scene: git_message |
| workflows | ⚠️ 无独立工作流实体；hooks 事件自动化 + 命令嵌套分类近似 |
| 独特机制 | **记忆（memory）双文件体系**、hooks 兼容 Claude Code 导入、`TRAE_ENV_FILE` 会话级环境注入、规则/命令 3 层嵌套、子目录 `.trae/rules` 模块级规则 |

### 6.3 独特概念清单

1. **记忆（Memory）实体**：`user_profile.md`（全局）+ `projects/{project_path}/project_memory.md`（项目分桶存全局目录）——AI 自动写入的持久偏好，介于"配置"与"状态"之间。
2. **`scene: git_message`**：规则的作用场景维度（仅生成提交内容时生效），与 alwaysApply/globs 正交且可叠加多文件。
3. **规则/命令 3 层嵌套 + 子目录 `.trae/rules/`**：目录即分类，模块级隔离（提及/读取该目录文件时才生效）。
4. **规则"智能生效"**：按 `description` 由模型自决加载（与 JetBrains `By model decision` 同构）。
5. **hooks 与 Claude Code 互操作**：直接导入 `~/.claude/settings.json` 的 hooks 段；事件模型与 Claude Code 对齐（SessionStart/UserPromptSubmit/PreToolUse/PostToolUse/Stop/Notification）。
6. **`TRAE_ENV_FILE`**：SessionStart hook 向文件写环境变量、供后续 hook 与 RunCommand 使用——会话内环境传递机制。
7. **项目级 MCP 默认关闭**：`.trae/mcp.json` 存在不生效，须显式开启（安全默认）。
8. **超时伪字段**：`START_MCP_TIMEOUT_MS`/`RUN_MCP_TIMEOUT_MS` 借 env/headers 通道传递。

### 6.4 IR 压力测试

| # | 概念 | IR 能否表达 | 判定 |
|---|------|------------|------|
| 1 | user_rules + `.trae/rules/`（含 globs/alwaysApply） | ✅ Instruction（`file_patterns` 对应 globs） | 通过 |
| 2 | **智能生效（description 模型自决）** | ❌ trigger 枚举无 `model-decision`（与 JetBrains 合并） | **击穿 B-5（合并）** |
| 3 | **`scene: git_message`** | ⚠️ Instruction 无 scene 维度；Zed 以 Setting（`agent.commit_message_instructions`）表达，两工具形态不一 | 弱击穿 B-13 |
| 4 | **记忆（memory/projects/{path}）** | ❌ 项目语义、全局存储：IR scope 按路径归属会误判为 global；且"AI 自维护"语义无表达 | **击穿 B-14** |
| 5 | skills + skill-config.json 禁用列表 | ✅ Skill + Setting | 通过 |
| 6 | commands（3 层嵌套） | ✅ Command（subtree/目录形态） | 通过 |
| 7 | hooks.json（6 事件 + 沙箱档位 + TRAE_ENV_FILE） | ⚠️ Setting 可装；事件语义与 Claude/Junie 趋同但无法翻译 | 弱击穿 B-11（合并） |
| 8 | `.trae/mcp.json`（mcpServers） | ✅ McpServer；"项目级默认关闭"开关是 Setting | 通过 |
| 9 | **超时伪字段 env/headers 注入** | ⚠️ McpServer 有 `timeout` 标准字段，但 Trae 语义是"以 env 传超时"，双向转换有损（导出须还原为 env 形式） | 弱击穿 B-3（合并） |
| 10 | `${workspaceFolder}` 变量 | ⚠️ 与 VS Code 插值同族，按既有"同工具保留、跨工具搬运+Warning"规则 | 通过 |
| 11 | `.trae-cn`（全局 hooks）与 `.trae` 双目录 | ⚠️ origin.path 如实记录即可；但同一工具两个全局根是 Detect 新形态 | 通过（适配器职责） |

### 6.5 真实样本

1. 官方规则示例（提交信息规则 `---\nscene: git_message\n---` + `git-commit-message.md` 自动生成）：https://docs.trae.ai/ide/rules
2. 官方 MCP 配置示例（`mcpServers` stdio/HTTP 双形态 + `${workspaceFolder}` + 超时伪字段）：https://docs.trae.ai/ide/add-mcp-servers
3. 官方 hooks 示例（SessionStart 注入上下文、PreToolUse 拦截高危命令、Stop 跑验收测试）：https://docs.trae.ai/ide/hook-configuration-reference
4. 官方规则嵌套结构示例（`.trae/rules/` frontend/backend/devops 三层）：https://docs.trae.ai/ide/rules

### 6.6 时效状态

- 活跃；**重大变更：2026-08 前后上线 TraeWork 客户端**（文档站首页即"重磅更新：TraeWork 客户端上线"公告），产品线分化为 TraeCode（IDE）/TraeWork（新客户端）/TraeCode Plugin/企业版四套文档树。来源：https://docs.trae.ai/
- 【存疑】访问异常记录：裸 URL `https://docs.trae.ai/ide/rules`（不带 `?_lang=zh`）在本次调研中被 302 至 `https://cursor.com/cn/docs/mcp`，疑似文档站跳转配置错误；带语言参数后恢复正常。适配器若做文档探测需注意。
- 注意目录不一致：规则/技能/命令/记忆用 `~/.trae/`，而全局 hooks 文档写作 `~/.trae-cn/hooks.json`（原文如此）。

---

## IR 击穿汇总

判定口径：**击穿** = IR 标准字段无法表达、x- 透传也会丢失关键语义或无法落盘；**弱击穿** = 可经 Setting/x- 兜底保真，但 IR 层无感知、跨工具迁移必降级；**范围外** = 非文件配置或运行时状态，建议 cfg4ai 显式声明不采集。

| 编号 | 击穿点 | 来源工具 | 严重度 | 说明 |
|------|--------|---------|--------|------|
| B-1 | **scope 层级不足**：无 remote/managed/system 层 | OpenCode（`.well-known/opencode`、MDM plist）、Amp（`/etc/ampcode`、`managed-settings.json`） | 高 | IR profile.kind 仅 global\|project；企业下发层既无法标注来源层级，来源还可能是 URL 而非文件 |
| B-2 | **远程内容源**：指令/skill 可来自 URL 或 git 仓库，本地只是缓存 | OpenCode（instructions 远程 URL）、Amp（personal/workspace skill git 仓）、Goose（GOOSE_RECIPE_GITHUB_REPO）、JetBrains（skill registry URL） | 中 | origin.path 假设本地路径；远程源内容会漂移，raw_hash 比对失效 |
| B-3 | **McpServer 字段缺口**：oauth 对象/false、cwd、includeTools/available_tools、env_keys、超时伪字段 | OpenCode、Amp、Goose、Trae | 中 | 仅 x- 透传；跨工具导出 MCP 配置时丢失 oauth 与工具过滤语义 |
| B-4 | **Skill↔MCP 绑定**：skill/recipe 内嵌 mcp.json，随包加载才暴露工具 | Amp（skill mcpServers）、Goose（recipe extensions） | 高 | Skill 与 McpServer 是独立实体，无关联字段；拆采后"按需加载"语义消失 |
| B-5 | **触发方式枚举不足**：model-decision（智能生效）、review-time（checks）、scene（git_message 场景） | JetBrains（By model decision/Off）、Trae（智能生效）、Amp（checks 评审激活） | 高 | PromptPack trigger.type 仅 slash-command\|mention\|manual\|hook |
| B-6 | **可执行插件实体缺失** | Amp（TS plugins：事件钩子+注册工具/命令）、OpenCode（plugins） | 中 | PromptPack assets 可存代码但事件语义全丢；本质是"配置即代码" |
| B-7 | **非 MCP 扩展类型**：builtin/platform/frontend/inline_python | Goose | 中 | McpServer.transport 枚举（stdio\|sse\|http）不含；根本不是 MCP client 配置 |
| B-8 | **Workflow 实体太薄**：recipe 的 parameters/response.json_schema/retry/sub_recipes/settings | Goose | 中 | 参数化工作流是声明式编排，超出"命名的 prompt 包" |
| B-9 | **Agent profile/mode 一等概念缺失** | Zed（agent.profiles）、OpenCode（mode: primary/subagent）、Amp（modes）、Junie（brave mode） | 中 | 四工具趋同出现"具名行为预设"，只能 Setting 不透明装载 |
| B-10 | **IDE 版本化缓存路径**（配置存在 Caches 目录、随版本漂移） | JetBrains AI Assistant | 低 | Detect 工程难点，非 schema 问题 |
| B-11 | **Hook 事件模型趋同但不可翻译**（SessionStart/UserPromptSubmit/PreToolUse/PostToolUse/Stop/Notification/SessionEnd/PermissionRequest） | Trae、Junie、Claude Code（对照组）、Goose（hooks 指南） | 中 | 目前只能作 Setting 不透明 value；事件名跨 4 工具高度一致，具备一等实体化条件 |
| B-12 | **优先级倒挂/多层**（用户 settings.json > 项目 config.json；8 层链） | Junie、OpenCode | 中 | IR 只有 global→project 两层 + merge_policy |
| B-13 | **场景化指令**（scene: git_message vs agent.commit_message_instructions） | Trae、Zed | 低 | 同一语义两种载体（规则文件 vs setting 键），IR 需归一策略 |
| B-14 | **项目语义、全局存储**（memory/projects/{path}/） | Trae 记忆 | 中 | scope 按路径归属误判；且"AI 自维护文件"是配置/状态混合体 |
| B-15（范围外合并） | 云端会话资产（Amp threads）、orbs/runners、schedules、信任/审批状态（Junie trust、Amp MCP approve、Zed worktree trust）、JetBrains UI 落盘位置 | Amp、Junie、Zed、JetBrains | — | 建议 cfg4ai 在 ADAPTERS 显式声明"不采集清单" |

## 对 cfg4ai 的启示

按投入产出排序：

1. **扩 scope/layer 模型（对应 B-1/B-12）**：`profile.kind` 增加 `managed`（或 origin 增加 `layer: user|org-remote|managed|mdm`），并在 manifest 记录优先级链顺序。OpenCode/Amp/Junie 三工具都出现第三层，这是企业场景刚需。
2. **McpServer 增字段（B-3）**：加 `oauth`（object|false）、`cwd`、`include_tools`/`available_tools`（工具白名单 glob）、`env_keys`（声明式 secret 需求，可与 secretref 联动：采集时发现 env_keys 引而未填→提示补录）。
3. **Hook 一等实体化（B-11）**：事件名在 Claude Code/Trae/Junie/Goose 已事实趋同（SessionStart、UserPromptSubmit、PreToolUse、PostToolUse、Stop、Notification 等），新增 `hook.<name>` 实体（event/matcher/command/timeout），跨工具翻译价值高。过渡期仍可由 Setting 承载。
4. **PromptPack trigger 扩展（B-5/B-13）**：`trigger.type` 增加 `model-decision`（配 description 字段）与 `scene`（如 git_message、code-review）；`disabled`/`Off` 状态需要标志位。
5. **Skill↔MCP 关联（B-4）**：Skill 实体增加可选 `bundled_mcp` 段（内嵌 server 定义），导出到无此概念的工具时降级为并列的 McpServer + Warning。
6. **Extension 类型扩展（B-7）**：McpServer.transport 枚举扩 `builtin|platform|frontend|inline_python`（或 Goose 适配器将其映射为非 MCP 的 Setting），避免硬塞 stdio 造成假阳性校验。
7. **Workflow 加厚（B-8）**：为 Goose recipes 增设标准字段 `parameters[]`/`response_schema`/`retry`/`sub_workflows[]`（均可选），使 recipe→command 降级导出时保留参数说明。
8. **profile/mode 抽象（B-9）**：暂缓一等实体化；先在 x- 透传，待第三个工具出现同构概念再抽象（当前 Zed profiles / OpenCode mode / Amp modes / Junie brave 语义差异仍大）。
9. **scope 归属判定规则（B-14）**：Detect 不能仅按路径前缀定 scope；Trae memory 这类"全局目录里的项目分桶"需适配器显式标注 `scope: project` + `origin.path` 全局路径的组合形态。
10. **不采集清单（B-15）**：在 ADAPTERS.md 为六工具补"范围外声明"：云端 threads、信任标记、审批状态、IDE 缓存内 UI 状态。

### 调研过程备注（时效与核实）

- Trae 文档站为字节 ByteDoc SPA，全部内容经浏览器渲染实检（2026-08-16）；`?_lang=zh` 参数可稳定复现。
- Junie 旧文档路径 `www.jetbrains.com/help/junie/` 已 301 至 `junie.jetbrains.com/docs/`。
- Goose 旧文档 `block.github.io/goose` 已 302 至 `goose-docs.ai`。
- 待复核项（【存疑】汇总）：Zed 项目级 settings 路径 `.zed/settings.json`；Trae 全局 hooks 目录 `.trae-cn` 与 `.trae` 的不一致；Goose skills 加载路径；JetBrains AI Assistant MCP 的落盘文件；OpenCode 组织迁移时间点。





