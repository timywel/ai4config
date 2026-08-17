# AI 编码工具生态调研（扩展 C 组）：遗漏核实与补卡

> 调研日期：2026-08-16 ｜ 调研人：生态调研扩展 C 组
> 基线：IR-SCHEMA v0.3（八类实体：Instruction / McpServer / Skill / Agent / Command / Workflow / Hook / Setting；五层 scope：managed>remote>local>project>global；activation：always|glob|manual|model-decision|scene）
> 对象：Grok Build（xAI）、Hermes Agent（Nous Research）、OpenClaw（OpenClaw Foundation）、Claude Desktop（Anthropic）
> 方法：官方文档实检（webfetch）；结论均标注来源 URL；无法核实处标【存疑】。
> 已调研 16 工具对照：基线五（claude-code / codex / copilot / zhanlu / gemini）+ A 组（Cursor / Windsurf→Devin / Aider / Cline / Roo Code）+ B 组（OpenCode / Amp / Goose / Zed / JetBrains / Trae），见 `tool-survey-a.md`、`tool-survey-b.md`。

**核实结论速览**：四个工具**全部真实存在且均有可采集的配置体系**。用户假设两处需修正：① Grok Build 不只是模型，是完整的 CLI/TUI 编码 agent，配置体系完善度超出预期；② OpenClaw 并非"复现 Claude Code 的开源项目"，而是多渠道 AI agent 网关（前身 Clawdbot），内置自研 agent runtime，可外挂 claude/codex/opencode CLI 作为后端。Claude Desktop 遗漏确认属实。

---

## 卡片 1：Grok Build（xAI）

**一句话定位**：xAI 官方编码 agent（TUI + headless + ACP 三形态），`~/.grok/config.toml` 单文件 TOML 配置 + 五层企业优先级链，概念覆盖（rules/skills/plugins/hooks/MCP/workflows/subagents/personas）与 Claude Code 全面对齐并零配置兼容读取 Claude 生态配置。

### 1.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 主配置 | `~/.grok/config.toml`（Windows `%USERPROFILE%\.grok\config.toml`） | `.grok/config.toml`（自 cwd 向上至 git root 逐级读取；同名项目 server **整体替换**用户级） | TOML | `[models] default`、`[model.<name>] {model, base_url, name, env_key, api_key}`、`[cli] auto_update`、`[ui] permission_mode`、`[permission] rules[]`、`[mcp_servers.*]`、`[skills] paths`、`[plugins] paths`、`[workflows] enabled`、`[sandbox] profile`、`[auth]` / `[auth.oidc]`、`[grok_com_config]`、`[subagents.personas]`、`[[marketplace.sources]]`、`[[version_overrides]]`、`[compat.claude]` / `[compat.cursor]`；支持 `$VAR` 展开 |
| 企业层（5 级优先级，低→高） | 1. `/etc/grok/managed_config.toml`；2. `~/.grok/managed_config.toml`；3. `~/.grok/config.toml`；4. `~/.grok/requirements.toml`；5. `/etc/grok/requirements.toml`（最高，fail-closed "pin"，用户/环境变量/远程设置均不可覆盖） | — | TOML | 与 config.toml 同构；`disable_bypass_permissions_mode` 仅 root-owned 源生效；另兼容读取 Claude `managed-settings.json` 子集（permission 规则、MCP allowlist、遥测/反馈开关、marketplace 限制），Grok 自有 requirements.toml 优先于它 |
| 规则（instructions） | `~/.grok/`（全局规则目录） | 每目录读取 `AGENTS.md`/`Agents.md`/`AGENT.md`/`CLAUDE.md`/`Claude.md`/`CLAUDE.local.md` + `.grok/rules/*.md`（兼容读 `.claude/rules/`、`.cursor/rules/`）；repo root→cwd 逐级，深层优先；`.gitignore` 忽略的文件跳过；嵌套 AGENTS.md 作用于子树 | Markdown | 纯文本；无 frontmatter；运行时可用 `--rules` 追加、`--system-prompt-override` 替换 |
| Skills | `~/.grok/skills/`；插件 `skills/`；`[skills] paths` 额外路径；兼容 `~/.agents/skills/`、`~/.claude/skills/` | `./.grok/skills/`（向上至 repo root）；`.claude/skills/` | 目录 + SKILL.md | frontmatter：`name`、`description`、`when-to-use`（别名 `when_to_use`）、`paths`（gitignore globs，命中文件前隐藏）、`allowed-tools`（不授权不限制，仅声明）、`argument-hint`、`user-invocable`（默认 true，仅字面 `true` 有效）、`disable-model-invocation`、`metadata`（string map）；`model`/`effort`/`license`/`compatibility` 接受但不生效；user-invocable skill 即 slash 命令（`/local:commit` 限定形） |
| Hooks | `~/.grok/hooks/*.json`（额外根 `~/.grok/hooks-paths`）；兼容读 Claude `.claude/settings.json` 与 Cursor `.cursor/hooks.json`（含 camelCase 事件名） | `.grok/hooks/*.json`（**须 `/hooks-trust` 或 `--trust` 授权**，存 `~/.grok/trusted_folders.toml`，覆盖项目 MCP/LSP） | JSON | `hooks.<Event>[].{matcher(正则,Claude 工具名自动映射), hooks:[{type: command|http, command|url, timeout(秒,默认5)}]}`；事件：`SessionStart/SessionEnd/UserPromptSubmit/PreToolUse(唯一阻塞)/PostToolUse/PostToolUseFailure/PermissionDenied/Stop/StopFailure/Notification/SubagentStart/SubagentStop/PreCompact/PostCompact`；stdin JSON + `GROK_HOOK_EVENT` 等 env；PreToolUse stdout `{"decision":"deny","reason"}` 或退出码 2 阻断，其余 fail-open |
| MCP | `~/.grok/config.toml` 的 `[mcp_servers.*]`；`grok mcp add` CLI；**兼容加载** `~/.claude.json`、`.cursor/mcp.json`、项目 `.mcp.json`（优先级低于 config.toml；`[compat.claude] mcps=false` 可关） | `.grok/config.toml`（`grok mcp add --scope project` 写入） | TOML | stdio：`command/args/env/startup_timeout_sec(默认30)/tool_timeout_sec(默认6000)`；remote：`url/headers`；`${VAR}` 与 `${VAR:-default}` 在五字段内展开；OAuth 浏览器流程，token 存 `~/.grok/mcp_credentials.json`；工具命名空间 `<server>__<tool>` |
| Plugins / Marketplaces | `~/.grok/plugins/`；`~/.grok/plugins/marketplaces/`；`[plugins] paths`；`--plugin-dir`；marketplace 源：`[[marketplace.sources]]` + `~/.grok/plugins/known_marketplaces.json` | `./.grok/plugins/` | 目录 | 插件可携带 skills/agents/hooks/MCP servers/LSP servers；插件 hooks 收 `GROK_PLUGIN_ROOT`/`GROK_PLUGIN_DATA` env |
| Subagents / Personas | `~/.grok/agents/`；`~/.grok/personas/*.toml`（或 `[subagents.personas]`） | `.grok/agents/`；`.grok/personas/*.toml` | TOML / 目录 | 内置类型 `general-purpose`/`explore`（只读）/`plan`（只读）；persona 为行为覆盖层（语气/焦点/契约） |
| Workflows | `~/.grok/workflows/<name>.rhai` | `.grok/workflows/<name>.rhai` | **Rhai 脚本** | `/create-workflow` 由 AI 起草（fan-out+verify+scope），`/workflow <name> [json-args]` 启动，pause/resume/stop/save；`[workflows] enabled=false` 或 `GROK_WORKFLOWS=0` 关闭；编排"有界子代理集" |
| 记忆 / 会话（运行时） | `~/.grok/sessions/`；xai-grok-shell 提供 `/memory`、`/dream`（记忆固化）、`/flush` | — | — | 跨会话记忆为 shell 提供能力，配置面未公开 |
| 认证 | `~/.grok/auth.json`；四种方式：Browser OIDC / Device code（RFC 8628）/ `auth_provider_command`（外部命令出 token）/ `XAI_API_KEY` | — | JSON | 每模型解析序：`model.api_key` > `model.env_key` > 会话 token > `XAI_API_KEY`；企业 OIDC（PKCE+refresh）；`[grok_com_config] disable_api_key_auth`（禁 API key 强制 SSO）、`force_login_team_uuid`（锁团队，列表可多个） |

来源：https://docs.x.ai/build/overview 、https://docs.x.ai/build/features/skills-plugins-marketplaces 、https://docs.x.ai/build/features/project-rules 、https://docs.x.ai/build/features/hooks 、https://docs.x.ai/build/features/permissions 、https://docs.x.ai/build/features/mcp-servers 、https://docs.x.ai/build/modes-and-commands 、https://docs.x.ai/build/cli/headless-scripting 、https://docs.x.ai/build/enterprise

**权限模型**：模式四态 `ask(默认)|auto(分类器)|always-approve|plan` + headless 增补 `dontAsk`（无显式 allow 即静默拒绝）/`acceptEdits`（自动批准编辑、shell 仍询问）；`[permission] rules = [{action: allow|deny, tool: bash|read|edit|..., pattern: "git *"}]`，deny 恒胜；`--allow/--deny` CLI 同构；legacy 键 `approval_mode`/`yolo` 兼容。来源：https://docs.x.ai/build/features/permissions 、https://docs.x.ai/build/enterprise

**Claude Code 兼容（官方零配置声明）**："Grok automatically reads Claude Code marketplaces, plugins, skills, MCPs, agents, hooks, and instruction files（`CLAUDE.md`、`Claude.md`、`CLAUDE.local.md`、`.claude/rules/`）alongside `.grok/`"——另有 `/import-claude` TUI 导入向导。来源：https://docs.x.ai/build/features/skills-plugins-marketplaces 、https://docs.x.ai/build/modes-and-commands

### 1.2 能力矩阵

| 能力 | 支持 | 说明 |
|------|------|------|
| instructions | ✅ | AGENTS.md 家族 + CLAUDE.md 家族 + `.grok/rules/` + 全局 `~/.grok/` + 嵌套子树作用域 |
| mcp | ✅ | stdio/http + OAuth + 双超时字段 + 跨工具兼容加载（claude/cursor/.mcp.json） |
| skills 类 | ✅ | SKILL.md + `paths` 条件激活 + 插件/marketplace 分发 |
| rules（文件作用域） | ✅（弱） | skill frontmatter `paths` globs；规则文件本体无 glob 维度 |
| workflows | ✅ | `.rhai` 脚本工作流（AI 起草、有界子代理编排、run 仪表盘） |
| custom modes | ⚠️ 内置 | 权限四态 + plan 模式（内置，非用户自定义实体）；personas 为行为覆盖层 |
| 独特机制 | **五层企业配置链（requirements pin）**、**Claude 生态全量兼容读取**、hooks 信任门控、ACP 一等支持、Rhai 工作流、subagent 内置类型、enterprise OIDC/ZDR | — |

### 1.3 独特概念清单（相对已调研 16 工具）

1. **五层企业配置链**：`/etc/grok/managed_config.toml` → `~/.grok/managed_config.toml` → `~/.grok/config.toml` → `~/.grok/requirements.toml` → `/etc/grok/requirements.toml`（pin 语义，fail-closed；`[[version_overrides]]` 版本条件补丁）——"managed" 与 "requirements" 双轨是新风。
2. **Claude 生态全量兼容读取**：marketplaces/plugins/skills/MCPs/agents/hooks/instruction files/`~/.claude.json`/`managed-settings.json` 子集——兼容广度为 20 工具中最大（Cline 仅读规则文件，Grok 读到 marketplace 与托管策略）。
3. **Hooks 信任门控**：项目 hooks/MCP/LSP 须 `/hooks-trust` 显式授权，决策落 `~/.grok/trusted_folders.toml`（Junie 信任体系同族，Grok 用 TOML 文件而非 keychain）。
4. **Rhai 脚本工作流**：`/create-workflow` AI 起草 → `.grok/workflows/*.rhai`——工作流载体是脚本语言而非 Markdown/YAML（20 工具中唯一）。
5. **权限模式五态**：`ask/auto(分类器)/always-approve/plan/dontAsk`（headless）+ `acceptEdits`——`dontAsk` 的"无 allow 即拒"为 CI 场景专门设计。
6. **`[grok_com_config] force_login_team_uuid`**：登录主体锁定团队 UUID（列表），token 每次签发/复用/静默刷新均校验——身份层 pin。
7. **Skill `paths`（gitignore globs）隐藏至命中**：与 Cline `paths:` 同语义但键名不同，行业继续收敛。
8. **`grok inspect`**：配置来源/指令/skills/插件/hooks/MCP 全量发现报告——采集器友好的官方 introspection。

### 1.4 IR 压力测试（IR-SCHEMA v0.3）

| 概念 | 能否表达 | 判定 |
|------|---------|------|
| config.toml 全部段（model/cli/ui/sandbox/auth/workflows…） | `Setting`（`setting.grok.<key>`，TOML 嵌套表→不透明 value）✅ | 标准字段 |
| 五层配置链（managed 双文件 + requirements 双文件） | 五层 scope 中的 `managed`（只读不物化）✅；但同一工具**四个 managed 来源路径**（/etc+user × managed+requirements）且 requirements 语义是"pin 最高优先级"而非普通 managed | 弱击穿：scope 一层装不下"managed 1/2/4/5 四级"，priority 排序需 x-grok 记录原始层级（**C-1**） |
| `[permission] rules`（allow/deny+pattern） | K9/D17 既定：Setting 不透明 value ✅ | 标准字段（x- 策略） |
| hooks（14 事件，Claude 超集） | `Hook` 实体 ✅；`SubagentStart/SubagentStop/PermissionDenied/PostToolUseFailure/StopFailure/PostCompact` 在标准事件交集之外→进 x-grok（v0.3 既定机制） | 标准字段 + x-（既定路径） |
| 项目 hooks 信任（trusted_folders.toml） | 运行时信任状态 | 范围外（不采集清单） |
| SKILL.md 全字段 | `Skill` ✅；`paths`→`file_patterns`（activation=glob）；`when-to-use`/`disable-model-invocation`/`allowed-tools` 无标准字段→x-grok | 标准字段 + x- |
| `[mcp_servers.*]`（stdio/http/OAuth/双超时） | `McpServer` ✅；`startup_timeout_sec`/`tool_timeout_sec` 与 `timeout.startup_ms/tool_sec` 单位换算 | 标准字段（适配器换算） |
| 兼容加载 `~/.claude.json`/`.cursor/mcp.json` | 归属 claude-code/cursor 适配器采集（B13 既定裁决） | 标准行为 |
| 项目 `.grok/config.toml` 逐级向上 + 同名 server **整体替换** | `merge_semantics.override: entry-replace` ✅（v0.3 词表已有） | 标准字段 |
| **workflows `.rhai` 脚本**（fan-out/verify/有界子代理编排，AI 起草） | Workflow 实体 `parameters/steps` 为声明式结构，装不下 Rhai 脚本语义；脚本本体可存 prompt.md/assets | 弱击穿（**C-2**：脚本即工作流，与 Goose recipe 同族但载体是代码） |
| personas（行为覆盖层） | Agent 变体，无标准字段 | x-grok |
| marketplace 源（`[[marketplace.sources]]` + known_marketplaces.json） | 远程分发渠道（B-2 同类） | 弱击穿（既定同类） |
| `auth_provider_command` / 企业 OIDC | Setting ✅（secret 走 secretref） | 标准字段 |
| `${VAR}`/`${VAR:-default}` 插值 | 同工具往返原样保留，跨工具 Warning（既定） | 标准行为 |

### 1.5 真实样本

1. 官方自定义模型配置示例（`[model.my-model]` + `[models] default`）：https://docs.x.ai/build/overview
2. 官方 MCP 配置示例（`[mcp_servers.filesystem]` stdio 全字段 + `[mcp_servers.linear]` remote headers `{{session_id}}`）：https://docs.x.ai/build/features/mcp-servers
3. 官方 hooks 示例（`PreToolUse` matcher `Bash` + command/timeout）：https://docs.x.ai/build/features/hooks
4. 官方权限规则示例（`[permission] rules` allow/deny/pattern 三元组）：https://docs.x.ai/build/features/permissions
5. 官方企业部署示例（`/etc/grok/requirements.toml`、`[grok_com_config] force_login_team_uuid`、`auth_provider_command` 契约）：https://docs.x.ai/build/enterprise

### 1.6 时效状态

- 高度活跃：xAI 官方文档站 docs.x.ai 2026 年持续更新（模型线 grok-4.6；安装器 `https://x.ai/cli/install.sh`；npm 包 `@xai-official/grok`）。
- 文档明示的活跃演进点：`sse` 未出现在 Grok MCP 传输中（stdio/http 双轨）；legacy `approval_mode`/`yolo` 键仍兼容；`[compat.*]` 开关表明兼容面仍在扩张。
- 另有 "Grok Bot"（云端 AI teammate）产品线，与 Grok Build 并列，本次未展开（非本地配置面）。
- 无弃维护迹象。

---

## 卡片 2：Hermes Agent（Nous Research）

**一句话定位**：Nous Research 出品的自学习开源 AI agent（CLI/TUI/Desktop/消息网关/ACP，GitHub 231k stars），`~/.hermes/` 目录全文件化配置（config.yaml + .env + SOUL.md + memories + skills + cron），独有"学习循环"（agent 自建/自改进 skills）与 7 种终端执行后端。**核实结论：用户列出的三种可能中，"Nous Research 的 Hermes 模型相关工具"命中——Hermes Agent 是 Hermes 模型团队官方出品、且确实是可配置的编码 agent 工具。**

### 2.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 主配置 | `~/.hermes/config.yaml`（`HERMES_HOME` 可整体迁移；profile 机制=每个 profile 一个 HERMES_HOME） | —（无项目级 config.yaml） | YAML | `model`、`terminal.*`（backend/cwd/timeout/home_mode/persistent_shell/docker_*/container_*）、`mcp_servers.*`、`memory.*`、`skills.*`、`agent.disabled_toolsets`、`tool_output.*`、`context_file_max_chars`、`file_read_max_chars`、`updates.*`、`worktree`/`worktree_sync`、`runtime.nofile_soft_limit`、`providers.*`（request_timeout_seconds 等）、`auxiliary.*`、`delegation.*`、`security.*`、`compression.*` 等；`${VAR}` 与 `${env:VAR}` 插值（未设置变量原样保留+警告） |
| 秘密 | `~/.hermes/.env`（`hermes config set KEY VAL` 自动路由：API key→.env，其余→config.yaml）；`~/.hermes/auth.json`（OAuth） | — | dotenv / JSON | 四级优先级：CLI 参数 > config.yaml > .env > 内建默认 |
| Managed scope（企业层） | `/etc/hermes/config.yaml` + `/etc/hermes/.env`（root 0755/0644；`HERMES_MANAGED_DIR` 可迁址） | — | YAML / dotenv | **leaf-level pin**：仅被点名的键不可覆盖（其余用户可控）；优先级高于用户 config/.env **甚至 shell 环境变量**；v1 仅 POSIX，无 MDM，文件权限即全部强制力 |
| 人格 | `~/.hermes/SOUL.md`（仅 HERMES_HOME 加载，不探工作目录；不存在时自动播种默认文件） | — | Markdown | 注入 system prompt slot #1 |
| 记忆 | `~/.hermes/memories/`（MEMORY.md、USER.md；agent 自维护） | — | Markdown | `memory.memory_enabled/user_profile_enabled/memory_char_limit(2200)/user_char_limit(1375)/write_approval`；开启 write_approval 后写入须 `/memory approve` 审批 |
| 上下文文件 | — | `.hermes.md`/`HERMES.md`（最高优先级）→ `AGENTS.md` → `CLAUDE.md` → `.cursorrules`/`.cursor/rules/*.mdc`（**first-match-wins，每会话只加载一种**）；AGENTS.md 支持 git root→cwd 链式合并（深层优先、去重、provenance header）+ **渐进式子目录发现**（agent 进入子目录时按需注入，8000 字符/文件） | Markdown | 无 frontmatter；全部经 prompt-injection 扫描（指令覆盖/隐藏注释与 div/凭据外泄/零宽字符） |
| Skills | `~/.hermes/skills/`（agent 用 `skill_manage` 工具自建自改进；agentskills.io 兼容；Skills Hub 社区） | 兼容读取 Claude Code 资产（经 `hermes import-agent claude-code` 迁移 skills+instructions+mcp） | 目录 + SKILL.md | frontmatter 可声明 config settings（落 `skills.config.<skill>.*` 命名空间）与 `required_environment_variables`；`skills.guard_agent_created`（危险关键词扫描，默认 off）、`skills.write_approval`（写入审批，默认 off，开启后落 `~/.hermes/pending/skills/` 走 `/skills approve`） |
| MCP | `~/.hermes/config.yaml` 的 `mcp_servers.*`；catalog 目录 `optional-mcps/<name>/manifest.yaml`（repo 内，PR 审核制）；token 存 `~/.hermes/mcp-tokens/<server>.json`（0600） | — | YAML | stdio：`command/args/env`；http：`url/headers`；公共：`enabled`、`timeout`、`connect_timeout`、`idle_timeout_seconds`/`max_lifetime_seconds`（stdio 回收，0=永不）、`supports_parallel_tool_calls`、`tools.{include,exclude(支持 fnmatch glob),prompts,resources}`、`auth: oauth` + `oauth.{client_id,client_secret,redirect_uri,redirect_port,redirect_host,client_name}`、`client_cert`（PEM 路径 / `[cert,key]` / `[cert,key,password]` mTLS）+ `client_key`、`identity_header.{name,value_from: static|profile,value}`、`sampling.{enabled,model,max_tokens_cap,timeout,max_rpm,max_tool_rounds,allowed_models,log_level}`、`elicitation.{enabled,timeout}`；`${VAR}`/`${userHome}`/`${workspaceFolder}` 等 Cursor 风格插值 |
| MCP catalog | repo `optional-mcps/`；`hermes mcp`（交互 picker）/`hermes mcp install <name>` | — | YAML manifest | `manifest_version`、`source`、`install.bootstrap`、`transport`、`tools.default_enabled`、`suggest.{keywords,hosts}`（Desktop 联想安装）；安装时探测工具清单供勾选（写 `tools.include`） |
| Cron | `~/.hermes/cron/`（定时任务，可投递到任意消息平台） | — | — | 内建 cron，配置面在 config.yaml |
| 消息平台 | config.yaml 平台段（20+：Telegram/Discord/Slack/WhatsApp/Signal/Matrix/Mattermost/Email/SMS/DingTalk/Feishu/WeCom/Weixin/QQ Bot/Yuanbao/BlueBubbles/Home Assistant/Teams/Google Chat…）；bot token 走 .env | — | YAML | 网关形态（与 OpenClaw 同族） |
| 终端后端 | config.yaml `terminal.backend`：`local|docker|ssh|modal|daytona|vercel_sandbox|singularity`（7 种） | — | YAML | 每后端一族键（`docker_*`、`container_cpu/memory/disk/persistent`、`singularity_image`、`vercel_runtime` 等）；`TERMINAL_*` env 逐键覆盖；SSH/Modal/Daytona teardown 时把远端 `~/.hermes` 状态**同步回宿主机** |
| Hermes 作为 MCP server | `hermes mcp serve`（stdio，10 工具：conversations_list/messages_read/messages_send/events_poll/events_wait/permissions_*） | — | — | 供 Claude Code/Cursor 等客户端挂载（配置写在对方工具里） |

来源：https://hermes-agent.nousresearch.com/docs/ 、https://hermes-agent.nousresearch.com/docs/user-guide/configuration 、https://hermes-agent.nousresearch.com/docs/user-guide/managed-scope 、https://hermes-agent.nousresearch.com/docs/user-guide/features/context-files 、https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/mcp.md

**跨工具迁移（官方一等命令）**：`hermes import-agent claude-code`——把 `~/.claude.json` 的 `mcpServers` 映射到 `mcp_servers`，并迁移 skills 与 instructions。来源：https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/mcp.md（页首 tip）

### 2.2 能力矩阵

| 能力 | 支持 | 说明 |
|------|------|------|
| instructions | ✅ | `.hermes.md` 自有指令文件名 + AGENTS.md 链 + CLAUDE.md/.cursorrules 兼容 + SOUL.md 全局人格 |
| mcp | ✅ | stdio/http + OAuth 全流程（DCR/PKCE/refresh/step-up）+ mTLS + 工具过滤 glob + sampling/elicitation + catalog 审批目录 |
| skills 类 | ✅ | agentskills.io 兼容 + **agent 自建自改进（学习循环）** + config settings 命名空间 + 写入审批/内容扫描双门 |
| rules（文件作用域） | ❌（无 glob 规则） | 上下文文件无 glob 维度 |
| workflows | ⚠️ | cron 定时任务（调度+平台投递）；无声明式工作流实体 |
| custom modes | ❌ | — |
| 独特机制 | **学习循环**（skill 自建/自改进/记忆 nudge/FTS5 跨会话召回/Honcho 用户建模）、**7 终端后端**（含 vercel_sandbox/singularity）、managed scope（leaf pin）、**hermes mcp serve**（自身当 server）、`import-agent` 迁移命令、profiles（HERMES_HOME 多开）、消息网关 20+ 平台 | — |

### 2.3 独特概念清单（相对已调研 16 工具）

1. **学习循环（self-improving）**：agent 用 `skill_manage` 从经验创建 skills、使用中自改进、定期 nudge 持久化知识——"agent 写自己的配置"（Windsurf memories 之后第二个机器生成内容源，且写的是 Skill 实体）。配套双安全门：`guard_agent_created`（内容扫描）与 `write_approval`（审批暂存区 `~/.hermes/pending/skills/`）。
2. **SOUL.md 全局人格槽位**：system prompt slot #1，仅 HERMES_HOME 加载——人格与项目指令分离的一等文件。
3. **上下文文件 first-match-wins 优先级**：`.hermes.md` > `AGENTS.md` > `CLAUDE.md` > `.cursorrules`——只加载一种（与多数工具的 concat 叠加相反）；自有指令文件名 `.hermes.md`。
4. **Managed scope leaf-level pin**：`/etc/hermes/` 只冻结被点名键，其余用户可控；优先级**压过 shell 环境变量**（独此一家的倒置）。
5. **MCP 外围字段最宽**：mTLS（`client_cert` 三形态）、`identity_header`、`idle/max_lifetime` 回收、`sampling`/`elicitation` 客户端能力配置、catalog manifest（`suggest.keywords/hosts` 联想安装）——MCP 配置面为 20 工具中最深。
6. **`hermes mcp serve`**：agent 自身作为 MCP server 暴露消息能力（与 JetBrains"IDE 即 server"同族，方向相反：这里是"网关即 server"）。
7. **7 种终端执行后端**（local/docker/ssh/modal/daytona/vercel_sandbox/singularity）+ 远端状态回同步。
8. **`hermes import-agent claude-code`**：官方配置迁移命令——cfg4ai 的竞品/同道证据。
9. **profiles = HERMES_HOME 多开**：一机多套完整 agent（配置/记忆/会话/skills 全隔离），profile distributions 可打包分享整个 agent。

### 2.4 IR 压力测试（IR-SCHEMA v0.3）

| 概念 | 能否表达 | 判定 |
|------|---------|------|
| config.yaml 全键（terminal.*/memory.*/tool_output.*/updates.* 等） | `Setting` ✅ | 标准字段 |
| managed scope（/etc/hermes/，leaf pin） | `managed` scope（只读不物化）✅；"leaf-level pin 压过 env"的语义差异由适配器记录 | 标准字段（v0.3 已解决） |
| SOUL.md / .hermes.md / AGENTS.md 链 / 渐进发现 | `Instruction` ✅；`.hermes.md` 记 origin.path；first-match-wins 由采集端记录胜出源；子树=subtree | 标准字段 |
| memories/（agent 自维护 MEMORY.md/USER.md + write_approval） | 机器生成内容（A 组 B5 同类：Windsurf memories）；write_approval 为 Setting | 弱击穿（既定同类，**不重复编号**） |
| skills（agent 自建 + skills.config 命名空间） | `Skill` ✅ + Setting ✅；`required_environment_variables`→x-hermes | 标准字段 + x- |
| **mcp `sampling`/`elicitation` 客户端能力配置** | McpServer 无对应字段（这是"server 反向请求 LLM/用户输入"的客户端策略） | 弱击穿（**C-3**：MCP 客户端能力面） |
| **mcp `client_cert`（mTLS 三形态）/`identity_header`** | McpServer 无对应字段 | 弱击穿（**C-3 合并**：传输层认证扩展） |
| **mcp `idle_timeout_seconds`/`max_lifetime_seconds`（stdio 回收）** | IR `timeout.{startup_ms,tool_sec}` 是调用超时；回收策略语义不同族 | 弱击穿（**C-4**：server 生命周期策略） |
| `tools.include/exclude`（fnmatch glob） | `enabled_tools`/`disabled_tools` ✅（D7 已建） | 标准字段 |
| `auth: oauth` + `oauth.*`（redirect_uri/client_name 等） | `oauth` 标准字段 ✅（D7）；Hermes 的 redirect_*/client_name 子键→x-hermes | 标准字段 + x- |
| cron（定时任务+平台投递） | 无调度实体（A 组 B10 同类：Cline cron） | 弱击穿（既定同类） |
| 消息平台配置（20+ 渠道 bot token 等） | Setting ✅（secret 走 secretref）；但"消息渠道接入"是全新配置域 | 标准字段（新配置域，**C-5 提示**：键空间大但无结构问题） |
| 终端后端配置族（docker_*/container_* 等） | Setting ✅ | 标准字段 |
| `hermes mcp serve`（自身为 server） | 非配置实体（配置写在客户端侧） | 范围外 |
| profiles（HERMES_HOME 多开） | cfg4ai profile 同构；采集根由 env 决定 | 适配器职责（Detect 需枚举 HERMES_HOME 候选） |
| catalog（optional-mcps manifest，PR 审核） | 远程分发渠道（B-2 同类） | 弱击穿（既定同类） |
| `hermes import-agent` | 迁移工具，非配置 | 不适用（但是 cfg4ai 价值佐证） |

### 2.5 真实样本

1. 官方 MCP 完整示例集（stdio/http/OAuth/mTLS/identity_header/过滤 glob/sampling/elicitation/回收）：https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/mcp.md
2. 官方配置示例 `cli-config.yaml.example`（全量注释）：https://github.com/NousResearch/hermes-agent/blob/main/cli-config.yaml.example
3. 官方 catalog manifest 目录（`optional-mcps/<name>/manifest.yaml`）：https://github.com/NousResearch/hermes-agent/tree/main/optional-mcps
4. 官方上下文文件示例（monorepo AGENTS.md 链 + frontend/backend 子目录例）：https://hermes-agent.nousresearch.com/docs/user-guide/features/context-files
5. 官方 managed scope 示例（`/etc/hermes/config.yaml` pin model/redact_secrets）：https://hermes-agent.nousresearch.com/docs/user-guide/managed-scope

### 2.6 时效状态

- 高度活跃：GitHub 231k stars / 46k forks（NousResearch/hermes-agent，本调研机读页面所示）；文档站双语言（en/zh-Hans），提供 llms.txt 与 llms-full.txt；Desktop 安装器 + install.sh/ps1 双通道。
- 社区周边已出现第三方生态（hermesatlas.com 220+ 工具地图、Skills Hub agentskills.io）。
- 配置面演进活跃：v0.7.0 pluggable memory、`sampling`/`elicitation` 为近版本新增；`HERMES_API_TIMEOUT` 等 legacy env 仍兼容但让位 `providers.*` 键。
- 无弃维护迹象。

---

## 卡片 3：OpenClaw（OpenClaw Foundation）

**一句话定位**：开源自托管多渠道 AI agent 网关（前身 Clawdbot，MIT，GitHub org `openclaw` 87 仓），单一 Gateway 进程桥接 20+ 聊天平台与内嵌 agent runtime（亦可外挂 claude/codex/opencode CLI 后端）；`~/.openclaw/openclaw.json`（JSON5，严格 schema 校验 + 热重载 + RPC 写配置）+ agent workspace（六注入文件）。**核实结论：项目真实存在且高度活跃，但用户假设需修正——它不是"对标/复现 Claude Code 的开源项目"，定位是个人 AI 助手网关；与 Claude 生态的关系是互操作（读 `.mcp.json`、`openclaw migrate codex`、外挂 CLI 后端），不是复刻。**

### 3.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 主配置 | `~/.openclaw/openclaw.json`（`OPENCLAW_CONFIG_PATH` 可指他处；**勿用 symlink**——原子替换会改写目标本体；`OPENCLAW_STATE_DIR` 迁数据根） | —（配置仅全局；agent workspace 目录承载项目语义） | **JSON5**（注释/尾逗号；顶层 `$schema` 唯一豁免键） | **严格 schema：未知键/类型错误→拒绝启动**；顶层族：`gateway.*`（port/auth/tailscale/TLS/push/reload）、`agents.{defaults,entries}`（workspace/model/models/skills/sandbox/subagents/heartbeat/groupChat/blockStreaming*）、`channels.<provider>.*`、`session.{dmScope,threadBindings,reset}`、`messages.{visibleReplies,groupChat.*}`、`tools.*`（profile/allow/deny/byProvider/toolsBySender/elevated/exec/web/media/sessions/codeMode/loopDetection/updatePlan/sandbox）、`skills.*`（entries/load/workshop/install）、`mcp.servers.*`、`cron.*`、`hooks.*`（webhook ingress）、`plugins.*`、`bindings[]`（agent 路由）、`models.{mode,providers.*}`（自定义 provider/baseUrl/api/compat）、`env.{vars,shellEnv}`、`identity`、`logging`、`discovery`、`browser`、`ui`；热重载默认 hybrid（安全项即改即生效，`gateway.*`/`plugins.load` 等自动重启）；`config.patch` RPC（baseHash 乐观锁 + `replacePaths` 数组防截断保护，30 req/min 限流） |
| 配置拆分 | `$include: "./file.json5"`（单文件替换所在对象）/数组（deep-merge 后写优先，≤10 层）；相对引用文件解析；confinement：必须解析在 openclaw.json 所在目录树下（`OPENCLAW_INCLUDE_ROOTS` 扩白名单，symlink 解析后复检） | — | JSON5 | OpenClaw 自有写入只回写被 include 的单文件段；根 include/数组 include/sibling 覆盖情形 fail-closed |
| 秘密 | SecretRef 对象 `{source: env|file|exec|store, provider, id}`（`secrets.providers` 定义后端）；`${VAR}` 插值（仅大写 `[A-Z_][A-Z0-9_]*`，缺失即 load 时报错）；`.env`（cwd + `~/.openclaw/.env`，不覆盖已有 env）；`env.shellEnv` 可从 login shell 导入缺失键 | — | JSON5 / dotenv | 支持 SecretRef 的字段有官方清单（SecretRef Credential Surface）：provider apiKey、skills.entries.*.apiKey、channels.googlechat.serviceAccount 等 |
| Agent workspace（注入文件） | `agents.defaults.workspace`（默认 `~/.openclaw/workspace`；per-agent `agents.entries.*.workspace`） | workspace 即项目语义 | Markdown | 六文件：`AGENTS.md`（操作指令+memory）、`SOUL.md`（人格/边界/语气）、`IDENTITY.md`（名字/emoji）、`USER.md`（用户画像）、`BOOTSTRAP.md`（一次性首 run 仪式，完成即删）、`MEMORY.md`（根长期记忆，存在才注入）；空文件跳过，大文件截断；缺文件注入占位行；`agents.defaults.skipBootstrap` 可关；workspace attestation 存 `~/.openclaw/state/openclaw.sqlite`（已 attested 的 workspace 消失时拒绝静默重播种） |
| Skills | 6 级优先级（高→低）：1. `<workspace>/skills`；2. `<workspace>/.agents/skills`；3. `~/.agents/skills`；4. `<state-dir>/skills`（managed）；5. bundled；6. `skills.load.extraDirs` + 插件 skills | 同左（workspace 内两级） | 目录 + SKILL.md（agentskills.io） | frontmatter：`name`/`description` 必填 + `homepage`/`user-invocable`/`disable-model-invocation`/`command-dispatch: tool`/`command-tool`/`command-arg-mode`；**gating**：`metadata.openclaw.{always,os,requires.{bins,anyBins,env,config},primaryEnv,install[]}`（legacy `metadata.clawdbot` 兼容）；`skills.entries.<name>.{enabled,apiKey,env,config}` 覆盖；`allowBundled` 白名单；per-agent allowlist（`agents.defaults.skills` / `agents.entries.*.skills`，**终态替换非合并**）；`skills.limits.maxSkillsPromptChars` 预算压缩 |
| Skill 分发 | ClawHub 注册表（clawhub.ai）：`openclaw skills install @owner/<slug>`（workspace）/`--global`（`~/.openclaw/skills`）；`skills-sh:`/`git:`/本地目录源；trust envelope `clawhub.skill.verify.v1` + `.clawhub/origin.json`；VirusTotal/ClawScan 扫描状态页 | — | — | `skills.install.allowUploadedArchives`（私有 zip 上传通道，默认关）；`security.installPolicy`（安装前本地策略命令，fail-closed） |
| Skill Workshop | 提案队列（agent 起草→人审）：`openclaw skills workshop list/inspect/evaluate/apply` | — | — | 提案不直接写 SKILL.md；`skills.workshop.allowSymlinkTargetWrites` |
| Node-hosted skills | 连接的 headless node 发布其 `~/.openclaw/skills`；断连即消；冲突时本地优先、node skill 加确定性前缀 | — | — | 经 `exec host=node node=<id>` 执行 |
| MCP | `mcp.servers.*`（openclaw.json）；**亦读取 Claude `.mcp.json` 与插件 manifest 来源**；统一暴露为 `bundle-mcp` 插件工具（命名 `<server>__<tool>`，server 前缀做 provider-safe 清洗：非 `[A-Za-z0-9_-]` 转 `-`，非字母开头加 `mcp-`） | — | JSON5 | `mcp.servers.<name>.{command,args,...}`（与 Claude mcpServers 同构方向）；sandbox 下须在 `tools.sandbox.tools.alsoAllow` 显式放行（`bundle-mcp`/`group:plugins`/server glob 如 `outlook__*`） |
| 工具策略 | `tools.profile: minimal|coding|messaging|full` + `tools.allow/deny`（大小写不敏感通配，deny 胜）+ `tools.byProvider` + `tools.toolsBySender`（`channel:/id:/e164:/username:/name:` 前缀键 + `*`）+ `tools.elevated`（沙箱外执行按渠道白名单）+ tool groups（`group:runtime/fs/sessions/memory/web/ui/automation/messaging/nodes/agents/media/openclaw/plugins`） | — | JSON5 | 策略在模型调用前执行；`allow` 与 `alsoAllow` 同 scope 互斥（校验拒绝） |
| Channels | `channels.<provider>.*`（whatsapp/telegram/discord/signal/slack/imessage/matrix/msteams/googlechat/feishu/mattermost/zalo…，各渠道多账号 `accounts.<id>`） | — | JSON5 | DM 策略 `dmPolicy: pairing|allowlist|open|disabled` + `allowFrom`；群策略 `groupPolicy` + `requireMention` + `mentionPatterns`（regex）；`messages.groupChat.unmentionedInbound`；`healthMonitor.enabled` |
| Multi-agent 路由 | `agents.entries.<id>`（独立 workspace/session）+ `bindings[]`（`{agentId, match:{channel, accountId,...}}`） | — | JSON5 | 会话隔离粒度 `session.dmScope: main|per-peer|per-channel-peer|per-account-channel-peer`；`tools.sessions.visibility: self|tree|agent|all` |
| 自动化 | `cron.*`（定时任务+sessionRetention）；`hooks.*`（**HTTP webhook ingress**：`enabled/token/path/defaultSessionKey/allowRequestSessionKey/allowedSessionKeyPrefixes/mappings[{match,action,agentId,sessionKey,sessionMode,deliver}]`，header-only 认证，含 Gmail 集成）；`agents.defaults.heartbeat.{every,target,directPolicy}` | — | JSON5 | hooks 语义=外部 HTTP 事件→注入 agent 会话（**非生命周期钩子**） |
| Plugins | `plugins.{entries,allow,deny,enabled,load,installs}`；ClawHub/npm/git/本地/压缩包安装；`openclaw.plugin.json` manifest（skills 目录、tools 契约） | — | JSON5 | 插件可带 tools/skills/channels/providers/hooks；SDK `api.registerTool(...)` |
| 沙箱 | `agents.defaults.sandbox.{mode: off|non-main|all, scope: session|agent|shared, workspaceRoot, docker.setupCommand}` | — | JSON5 | 与 tools 策略、elevated 三层分立；`sessionToolsVisibility` 沙箱内强制收窄 |

来源：https://docs.openclaw.ai/ 、https://docs.openclaw.ai/gateway/configuration 、https://docs.openclaw.ai/concepts/agent 、https://docs.openclaw.ai/tools/skills 、https://docs.openclaw.ai/gateway/config-tools 、https://docs.openclaw.ai/tools

**运行时归属证据（非"Claude Code 复刻"）**：官方明示"OpenClaw ships one **embedded agent runtime**: a built-in agent loop, tool wiring, and prompt assembly, distinct from delegating turns to an external harness process"；外挂后端走 bundled `coding-agent` skill（依赖 `claude`/`codex`/`opencode` 等 CLI，opt-in）；`openclaw migrate plan codex` / `openclaw migrate codex` 为 Codex 资产迁移命令；legacy `metadata.clawdbot` 块证明前身。来源：https://docs.openclaw.ai/concepts/agent 、https://docs.openclaw.ai/tools/skills

### 3.2 能力矩阵

| 能力 | 支持 | 说明 |
|------|------|------|
| instructions | ✅ | workspace 六注入文件（AGENTS.md/SOUL.md/IDENTITY.md/USER.md/BOOTSTRAP.md/MEMORY.md） |
| mcp | ✅ | `mcp.servers` + 读 Claude `.mcp.json` + sandbox 二次门控；统一 `bundle-mcp` 工具面 |
| skills 类 | ✅✅ | 6 级加载 + gating + per-agent allowlist + ClawHub 注册表（trust envelope）+ Skill Workshop 人审队列 + node-hosted |
| rules（文件作用域） | ❌（无 glob 维度） | skill gating 是环境依赖过滤，非文件 glob |
| workflows | ⚠️ | cron + heartbeat + webhook ingress + Task Flow；无声明式工作流实体 |
| custom modes | ❌ | `tools.profile` 四档为内置预设 |
| 独特机制 | **Gateway 常驻服务形态**（热重载+RPC 写配置+Control UI）、**channels 20+**、pairing/dmPolicy 接入控制、multi-agent bindings 路由、严格 schema（未知键拒启动）、`$include` 拆分、SecretRef 四后端、ClawHub 信任体系、`suggest_task`（agent 提议后续任务）、Steering queue | — |

### 3.3 独特概念清单（相对已调研 16 工具）

1. **Gateway 常驻服务形态**：配置不是"被读取"而是"被服务托管"——文件 watch 热重载（hybrid）、`config.patch` RPC（baseHash 乐观锁、replacePaths 防数组截断、限流 30/60s）、rejected 写落 `<path>.rejected.<timestamp>`、last-known-good 仅 `doctor --fix` 恢复。与 cfg4ai 文件级采集模型存在形态张力。
2. **严格 schema 校验**：未知键/类型错误→**拒绝启动**（唯一豁免 `$schema`）——对导出器是硬约束（写错键=搞挂用户服务）。
3. **`$include` 配置拆分**：单文件/数组 deep-merge/confinement/白名单 roots/自有写回分流——配置物理多文件、逻辑单树。
4. **workspace 六注入文件**（含 `BOOTSTRAP.md` 一次性仪式文件——完成即删，attestation 落 SQLite 防重播种）；"missing file 注入占位行"。
5. **Skill gating `metadata.openclaw.requires`**（bins/anyBins/env/config/os/always）+ `install[]` 安装器规格（brew/node/go/uv/download，UI 一键装依赖）。
6. **ClawHub 信任体系**：trust envelope、`.clawhub/origin.json`、VirusTotal/ClawScan 扫描、`security.installPolicy` 本地策略命令（fail-closed）——skill 供应链安全为 20 工具中最完整。
7. **Skill Workshop**：agent 起草 skill 提案→人审队列→apply——"agent 写配置"的治理流程（Hermes write_approval 同族，OpenClaw 做成队列产品）。
8. **channels 接入策略**：`dmPolicy: pairing|allowlist|open|disabled`、群 mention gating（regex mentionPatterns）、`toolsBySender` 请求者级工具策略——IM 接入控制是全新配置域。
9. **SecretRef 四后端对象**（env/file/exec/store + provider 注册表）与 marker 持久化（源标记不落明文）。
10. **multi-agent bindings 路由表** + `session.dmScope` 四档隔离 + `tools.sessions.visibility` 会话可见域——agent 间路由是一等配置。
11. **Node 体系**：headless node 发布 skills、macOS node 使 darwin-only skill 生效、`exec host=node` 远程执行。
12. **`suggest_task`/`dismiss_task`**：agent 提议后续任务（Control UI 可操作 chip，可开 managed worktree/云 worker）。

### 3.4 IR 压力测试（IR-SCHEMA v0.3）

| 概念 | 能否表达 | 判定 |
|------|---------|------|
| openclaw.json 全键（channels/agents/session/messages/tools/gateway/plugins…） | `Setting` ✅（键空间为 20 工具最大，但结构无问题） | 标准字段 |
| **严格 schema（未知键拒启动）** | 导出纪律问题：适配器 Capability 必须声明"只写已知键+版本护栏" | 适配器职责（高危 Warning） |
| **`$include` 多文件聚合** | `origin.path` 单路径假设；需逐文件采集或物化展开 | 弱击穿（**C-6**：Setting 来源多文件化） |
| workspace 六注入文件 | `Instruction` 多条目 ✅ | 标准字段 |
| **`BOOTSTRAP.md`（一次性，删后不重建+SQLite attestation）** | 机器生命周期状态混合体；Instruction 采集可，但"一次性"语义无字段 | 弱击穿（B5 机器生成内容同类扩展，**不重复编号**） |
| skills 6 级优先级 + gating + allowlist | `Skill` ✅ + Setting ✅（entries/allowlist）；`metadata.openclaw.requires/install`→x-openclaw | 标准字段 + x- |
| ClawHub / Skill Workshop / installPolicy | 远程分发+审批工作流状态 | 范围外（B-2/B-15 同类） |
| `mcp.servers` + 读 `.mcp.json` | `McpServer` ✅；`.mcp.json` 归属 claude-code 适配器（既定裁决） | 标准字段 |
| tools.profile/groups/allow/deny/byProvider/toolsBySender/elevated | K9/D17 既定：x- 承载策略 | 标准字段（x- 策略） |
| **channels 配置域**（dmPolicy/allowFrom/mentionPatterns/accounts） | Setting ✅；但"IM 渠道接入"语义跨工具无对应 | 标准字段（**C-5 提示**，与 Hermes 合并记录） |
| **hooks（HTTP webhook ingress mappings）** | IR `Hook` 实体是生命周期事件钩子（pre-tool-use 等）；OpenClaw hooks 是"外部 HTTP 端点→会话映射"，事件模型不同族 | 弱击穿（**C-7**：同名异义——建议以 Setting 承载，勿强行入 `hook.` 实体） |
| cron/heartbeat | 无调度实体（A 组 B10 同类） | 弱击穿（既定同类） |
| **SecretRef 四后端** `{source: env|file|exec|store}` | IR secretref 形态 `secretref://cfg4ai/<profile>/<entity>/<field>` + 后端 `keyring|file|none`；OpenClaw 的 exec/store provider 更宽 | 弱击穿（**C-8**：secret 后端词表需扩，或 x-openclaw 透传原对象） |
| multi-agent（entries+bindings+dmScope+visibility） | Agent 实体（PromptPack）只覆盖"agent 定义"；bindings/dmScope 属路由策略 | 标准字段 + Setting（bindings 为不透明 value） |
| JSON5（注释/尾逗号） | 保真分级既定：JSONC·TOML 注释键序不保证（免责） | 标准行为 |
| 热重载/RPC 写配置/rejected 文件/last-known-good | 服务运行时行为 | 范围外（但导出器必须知道：直接写文件会触发校验与热加载） |

### 3.5 真实样本

1. 官方最小配置与 `$include` 拆分示例：https://docs.openclaw.ai/gateway/configuration
2. 官方 channels 接入示例（`dmPolicy: pairing`、`allowFrom`、`requireMention`、`mentionPatterns`）：https://docs.openclaw.ai/gateway/configuration 与 https://docs.openclaw.ai/
3. 官方 skills 配置与 gating 示例（`metadata.openclaw.requires` + `install[]` + `skills.entries`）：https://docs.openclaw.ai/tools/skills
4. 官方工具策略与 sandbox MCP 放行示例（`tools.profile`、`tools.sandbox.tools.alsoAllow: ["bundle-mcp"]`、`mcp.servers`）：https://docs.openclaw.ai/gateway/config-tools
5. 官方 multi-agent 路由示例（`agents.entries` + `bindings[]`）：https://docs.openclaw.ai/gateway/configuration
6. 主仓与文档站：https://github.com/openclaw/openclaw 、https://docs.openclaw.ai/ ；官网 https://openclaw.ai ；维基条目 https://en.wikipedia.org/wiki/OpenClaw

### 3.6 时效状态

- 高度活跃：npm `openclaw@latest`、文档站持续更新（配置参考含 2026 年新键如 `blockStreaming*`、`suggest_task`）；非营利 OpenClaw Foundation 运营；ClawHub 注册表在线。
- 品牌沿革证据：legacy `metadata.clawdbot` 兼容块（前身 Clawdbot）已写入官方 gating 文档；Wikipedia 已有条目。
- 配置面风险点：**严格 schema** 意味着键集随版本快速扩张时，cfg4ai 导出物的版本护栏（MinVersion/MaxVersion）是硬需求；`openclaw doctor --fix` 承担了大量 legacy 键迁移职责（说明漂移快）。
- 无弃维护迹象。

---

## 卡片 4：Claude Desktop（Anthropic 官方桌面应用）

**一句话定位**：Anthropic 官方消费级桌面聊天应用（macOS/Windows），**遗漏确认属实**——它是 MCP 协议的"参考客户端"与 `mcpServers` 配置格式的原型来源，本地配置面高度聚焦（单文件 `claude_desktop_config.json` 的 `mcpServers` 键），远程能力走 UI 驱动的 Custom Connectors（不落文件）。与 Claude Code 是两个产品：Claude Desktop 无 instructions/skills/hooks/rules 体系。

### 4.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| MCP（本地 stdio） | **macOS**：`~/Library/Application Support/Claude/claude_desktop_config.json`；**Windows**：`%APPDATA%\Claude\claude_desktop_config.json`（经 设置→Developer→Edit Config 打开，不存在则创建） | —（无项目概念） | JSON | 顶层 `mcpServers.<name>`：`command`/`args`/`env`（官方 quickstart 与 troubleshooting 仅出现这三个字段）；`env` 常用于注入 API key 与 Windows 下补 `APPDATA` 路径 |
| MCP（远程） | **Custom Connectors：UI 配置**（设置→Connectors→Add custom connector→输入 `https://.../mcp` URL→OAuth/API key 流程），**不落公开文件路径**【存疑：connectors 的本机落盘位置官方未公开；OAuth token 存储位置同】 | — | — | 支持 per-tool 权限开关（连接器设置内 enable/disable 工具）；remote 传输为 Streamable HTTP（官方教程未提 SSE 配置入口） |
| 扩展（Bundles） | `.mcpb` 文件（zip：`manifest.json` + server 代码 + 依赖 + icon），双击经 Claude Desktop 安装；支持 Node.js（推荐，Desktop 内置 Node 运行时）/Python（`server.type: "uv"` 由宿主管理依赖，或传统 `python` 自打包）/binary | — | zip + manifest.json | **重大更名：DXT（Desktop Extensions）→ MCPB（MCP Bundles）**——仓库 `anthropics/dxt` 已迁 `modelcontextprotocol/mcpb`，`.dxt`→`.mcpb`，CLI `dxt`→`mcpb`，包 `@anthropic-ai/dxt`→`@anthropic-ai/mcpb`；manifest 全字段见 MANIFEST.md；Desktop 侧实现（加载/校验/自动更新/变量配置 UI/策展目录）部分开源于该仓 |
| 日志（运行时，不采集） | macOS `~/Library/Logs/Claude/`（`mcp.log` + `mcp-server-<name>.log`）；Windows `%APPDATA%\Claude\logs\` | — | 文本 | — |
| 其他顶级键 | 【存疑】官方文档（MCP quickstart/远程连接器教程/Anthropic 支持页）仅明确 `mcpServers`；社区报告过 `globalShortcut` 等顶级键，本次未找到官方来源确认 | — | JSON | — |
| Projects 自定义指令 | **云端功能**（claude.ai 账户内 Project：自定义指令+知识库文件），**非本地文件** | — | — | 不属配置采集面 |

来源：https://modelcontextprotocol.io/quickstart/user （= https://modelcontextprotocol.io/docs/2026-07-28/develop/connect-local-servers.md）、https://modelcontextprotocol.io/docs/2026-07-28/develop/connect-remote-servers.md 、https://github.com/modelcontextprotocol/mcpb 、https://support.anthropic.com/en/articles/11175166-getting-started-with-custom-connectors-using-remote-mcp

### 4.2 能力矩阵

| 能力 | 支持 | 说明 |
|------|------|------|
| instructions | ❌（无本地文件形态） | Projects 自定义指令在云端，无文件可采 |
| mcp | ✅（核心且唯一） | 本地 stdio（文件）+ 远程 connectors（UI）；MCPB 一键安装包 |
| skills 类 | ❌ | — |
| rules（文件作用域） | ❌ | — |
| workflows | ❌ | — |
| custom modes | ❌ | — |
| 独特机制 | **mcpServers 格式原型**（全行业事实标准起点）、**MCPB 扩展包**（一键安装+自动更新+变量配置 UI）、Custom Connectors（OAuth 全流程 UI 化）、per-tool 权限 UI | — |

### 4.3 独特概念清单（相对已调研 16 工具）

1. **`mcpServers` 格式原型**：`{"mcpServers": {"<name>": {"command","args","env"}}}` 这一形态由 Claude Desktop 确立，后被 Claude Code（`~/.claude.json`/`.mcp.json`）、Cursor、Cline、Roo、JetBrains（支持 "Import from Claude" 即读 `claude_desktop_config.json`）、Trae、OpenClaw、Hermes（`hermes import-agent`）、Grok（兼容加载 `~/.claude.json`）全行业继承——cfg4ai 的 McpServer 实体本质就是它的泛化。
2. **MCPB 扩展包**：zip + manifest.json，宿主一键安装、自动更新、变量配置 UI、策展目录——"MCP server 的应用商店分发格式"（唯一；其他工具的 skill marketplace 都是 prompt 包，MCPB 装的是可执行 server）。
3. **文件/UI 双轨配置**：本地 stdio 走 JSON 文件，远程 connectors 与 per-tool 权限走 UI（云端/内部存储）——配置面分裂为"可采集"与"不可采集"两半。
4. **无 instructions 体系**：与 20 工具中其余全部成员不同，Claude Desktop 没有任何本地指令文件概念（Projects 在云端）。
5. **Windows 路径陷阱的官方记录**：`${APPDATA}` 在 args 中不展开，须显式在 `env` 注入 `"APPDATA": "C:\\Users\\<user>\\AppData\\Roaming\\"`——适配器导出 Windows 目标时的实测坑（官方 troubleshooting 原文）。

### 4.4 IR 压力测试（IR-SCHEMA v0.3）

| 概念 | 能否表达 | 判定 |
|------|---------|------|
| `mcpServers`（command/args/env） | `McpServer` ✅ 完全同构（原型形态） | 标准字段 |
| 远程 Custom Connectors（UI/OAuth，无公开文件路径） | 无文件可采 | 范围外（**不采集清单**；若未来发现落盘位置再立 Location） |
| MCPB（.mcpb 包 + 安装状态） | 打包分发格式，非运行时配置；安装状态在 UI/内部存储 | 范围外（B-2 远程分发同类；可作为 blob 可选采集，价值低） |
| per-tool 权限开关（UI） | 运行时状态，无文件 | 范围外 |
| Projects 自定义指令（云端） | 非文件 | 范围外 |
| 其他顶级键【存疑】 | 待官方文档或实采确认 | 暂缓（差分采集规则可覆盖：与默认值不同才入库） |

**结论：Claude Desktop 对 IR 压力为零**——它恰好是 IR McpServer 实体的原型；采集面=单文件单键，是最简单的适配器形态。

### 4.5 真实样本

1. 官方 `claude_desktop_config.json` 完整示例（filesystem server，macOS/Windows 双版本 + `env.APPDATA` 补丁例）：https://modelcontextprotocol.io/quickstart/user
2. 官方远程 connector 流程（Add custom connector→URL→OAuth→per-tool 权限）：https://modelcontextprotocol.io/docs/2026-07-28/develop/connect-remote-servers.md 与 https://support.anthropic.com/en/articles/11175166-getting-started-with-custom-connectors-using-remote-mcp
3. MCPB 规范与示例（MANIFEST.md、`examples/hello-world-uv`）：https://github.com/modelcontextprotocol/mcpb
4. 官方 server 目录（配合 Desktop 使用）：https://github.com/modelcontextprotocol/servers

### 4.6 时效状态

- 活跃：MCP 官方文档以 Claude Desktop 为参考客户端持续更新（2026-07-28 文档版本线）；Desktop 内置 Node.js 运行时（MCPB Node 包开箱即用）。
- **重大变更：DXT→MCPB 更名**（仓库迁移至 modelcontextprotocol/mcpb，"IMPORTANT NOTICE" 置顶）——二手资料中的 `.dxt`/`dxt` CLI 说法已过时。
- JetBrains AI Assistant 的 "Import from Claude" 功能（读 `claude_desktop_config.json`）证明该文件已是行业互操作锚点。
- 注意区分：Hermes 官方文档把 `~/.claude/claude_desktop_config.json` 误称为 "Claude Code's" 配置（真实路径是 Claude Desktop 的 `%APPDATA%\Claude\` / `~/Library/Application Support/Claude/`）——**文件名撞车是社区常见混淆**，适配器文档应明确澄清两者关系。来源：https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/mcp.md（"MCP client configuration"一节）

---

## C 组汇总：适配器候选建议与 IR 击穿清单

### 1. 四工具核实结论与纳入建议

| 工具 | 存在性结论 | 配置形态 | 纳入建议 | 理由 |
|------|-----------|---------|---------|------|
| **Grok Build** | ✅ 存在（xAI 官方 CLI/TUI/ACP agent，非仅模型） | `~/.grok/config.toml`（TOML）+ 五层企业链 + hooks/skills/plugins/workflows(.rhai)/subagents/personas + 全量 Claude 兼容读取 | **纳入，P1** | 配置体系完整、官方文档齐全、增长势头强；与 Claude Code 的高兼容使双向迁移价值最大；TOML + 五层链 + entry-replace 是 IR v0.3 语义的良好试金石 |
| **Hermes Agent** | ✅ 存在（Nous Research 官方 agent，非仅模型） | `~/.hermes/`（config.yaml/.env/SOUL.md/memories/skills/cron）+ `/etc/hermes/` managed + MCP 深配置 + 消息网关 | **纳入，P2** | 全文件化配置对采集友好；`import-agent` 证明其自做迁移（cfg4ai 可互操作）；MCP 深字段（mTLS/sampling/elicitation/回收）是 IR 边界样本；学习循环产物（agent 自建 skills）需采集纪律 |
| **OpenClaw** | ✅ 存在（但**非 Claude Code 复刻**——多渠道 agent 网关，前身 Clawdbot） | `~/.openclaw/openclaw.json`（JSON5 严格 schema+热重载+RPC）+ workspace 六注入文件 + skills 6 级 + ClawHub | **纳入观察名单，P2 只读采集先行** | 配置面最大且严格 schema（导出风险高：写错键=拒启动）；Gateway 常驻服务形态与文件级采集有张力；建议 P2 仅做 collect/diff，export 待版本护栏成熟后开放 |
| **Claude Desktop** | ✅ 存在，**遗漏确认** | 单文件 `claude_desktop_config.json` 的 `mcpServers` 键（stdio）；远程 connectors/Projects 无文件 | **纳入，P1（轻量）** | mcpServers 原型形态，与 claude-code 适配器共享绝大部分代码，成本最低；connectors/Projects/MCPB 安装状态声明范围外 |

### 2. 不纳入项与理由（四工具范围内）

- **Grok Bot**（云端 AI teammate）：无本地配置面，不纳入。
- **MCPB 包管理**（.mcpb 安装/更新/变量配置）：分发格式非运行时配置，不纳入采集（可在文档登记为"Claude Desktop 生态知悉项"）。
- **ClawHub / Skills Hub / hermesatlas 等注册表交互**：远程分发渠道（B-2 既定同类），不纳入采集。
- **各工具运行时状态**（grok trusted_folders/auth.json、hermes auth.json/mcp-tokens、openclaw state/openclaw.sqlite、claude desktop logs）：信任/会话/日志，入不采集清单。

### 3. C 组新增 IR 击穿点（v0.3 之上，全部为弱击穿；无 K 级结构性击穿）

| 编号 | 击穿点 | 来源工具 | 说明与处置建议 |
|------|--------|---------|---------------|
| **C-1** | **managed 层内多级子层**：Grok 四层 managed 来源（/etc+user × managed+requirements）+ requirements 的"pin 最高优先级"语义 | Grok | `origin.scope: managed` 一层装不下优先级差；x-grok 记录原始层级序号即可，IR 不动 |
| **C-2** | **脚本即工作流**（.rhai，AI 起草、有界子代理编排） | Grok | Workflow 实体 `parameters/steps` 装不下脚本语义；脚本本体入 prompt.md/assets + x-grok；与 Goose recipe（B-8）同族但更重 |
| **C-3** | **MCP 客户端能力与传输认证扩展**：`sampling`/`elicitation` 配置、`client_cert`（mTLS）、`identity_header` | Hermes | McpServer 加可选字段的候选（第三工具出现同构再标准化）；当前 x-hermes |
| **C-4** | **stdio server 生命周期回收**（`idle_timeout_seconds`/`max_lifetime_seconds`） | Hermes | 与 `timeout.{startup_ms,tool_sec}` 调用超时不同族；x-hermes |
| **C-5** | **消息渠道接入配置域**（channels/dmPolicy/mentionPatterns/bot token） | OpenClaw、Hermes | 全新配置域但 Setting 可承载；若 cfg4ai 未来支持 gateway 类工具，考虑 `channel.` 实体（暂缓） |
| **C-6** | **`$include` 配置多文件聚合** | OpenClaw | Setting origin 单路径假设；适配器须逐文件采集（每文件一组 Setting 条目）并在导出时保持拆分布局；IR 记录 `origin.path` 为实际 include 文件 |
| **C-7** | **同名异义 hooks**：OpenClaw hooks = HTTP webhook ingress 映射，非生命周期事件钩子 | OpenClaw | 以 Setting（`setting.openclaw.hooks`）承载，**禁止**误映射入 `hook.` 实体；ADAPTERS 目标语义差异表登记 |
| **C-8** | **SecretRef 多后端对象**（env/file/exec/store + provider 注册表） | OpenClaw | IR secretref 后端词表 `keyring|file|none` 偏窄；导出到 OpenClaw 时映射为 `{source, provider, id}` 对象，采集时 x-openclaw 透传原对象；secret 后端扩展列为未来候选 |

**既有击穿的新证据（不重复编号）**：B-1/B-2（scope 与远程源：Grok 五层链、Hermes managed、OpenClaw ClawHub）、B5（机器生成内容：Hermes 学习循环产物、OpenClaw BOOTSTRAP.md）、B-10（cron：Hermes、OpenClaw）、B13（跨工具兼容读取归属：Grok 读 Claude 全量、OpenClaw 读 .mcp.json、Hermes import-agent）——三组调研后这些击穿的证据面继续扩大，D1/D4/D7/D16 的 v0.3 处置方向均被再次验证。

### 4. 对 cfg4ai 的增量启示

1. **适配器优先级**：Claude Desktop（P1，与 claude-code 共享 mcp 解析）→ Grok（P1，完整概念面）→ Hermes（P2，深 MCP 字段+学习循环纪律）→ OpenClaw（P2 只读先行）。
2. **导出安全新红线**：OpenClaw 严格 schema（未知键拒启动）确立"适配器版本护栏=服务可用性"的硬关联——`export` 前必须按 Capability.MinVersion/MaxVersion 校验，未知键绝不写入；建议 doctor 增"目标服务会因未知键拒启动"的最高级 Warning 模板。
3. **采集纪律新增**：
   - Hermes/OpenClaw 的 **agent 自建产物**（学习循环 skills、Skill Workshop 提案、BOOTSTRAP.md）需区分"人写"与"机写"：机写内容默认采集但标记 `x-*.generated: true`，sync 默认排除或需确认（B5 处置细则）；
   - Grok 兼容读取的 Claude 系文件**归属原工具适配器**，grok 适配器只采 `[compat.*]` 开关与 `.grok/` 自有路径（B13 裁决的执行样板）；
   - OpenClaw 采集须知：直接编辑 openclaw.json 会触发热重载与严格校验——collect 只读无虞，export 等价于"远程重启用户服务"，必须显式确认。
4. **互操作锚点确认**：`claude_desktop_config.json` 经 JetBrains"Import from Claude"、Hermes `import-agent`、Grok 兼容加载三条独立证据链确认为行业 MCP 配置锚点——cfg4ai 的 `export --target claude-desktop` 同时是向多工具辐射的最短路径。
5. **文件名撞车治理**：`claude_desktop_config.json`（Desktop）vs `~/.claude.json`（Code）vs `.mcp.json`（Code 项目级）——三者在社区文档中已被多次混淆（含 Hermes 官方文档）；cfg4ai CLI 输出与文档应统一使用全称（"Claude Desktop 配置"/"Claude Code 全局配置"/"Claude Code 项目 MCP"）。

### 附：来源 URL 总表

- Grok Build 总览：https://docs.x.ai/build/overview
- Grok Skills/Plugins/Marketplaces/Claude 兼容：https://docs.x.ai/build/features/skills-plugins-marketplaces
- Grok 规则（AGENTS.md）：https://docs.x.ai/build/features/project-rules
- Grok Hooks：https://docs.x.ai/build/features/hooks
- Grok Permissions：https://docs.x.ai/build/features/permissions
- Grok MCP：https://docs.x.ai/build/features/mcp-servers
- Grok Modes/Commands/Workflows：https://docs.x.ai/build/modes-and-commands
- Grok Headless/ACP：https://docs.x.ai/build/cli/headless-scripting
- Grok Enterprise（五层链/OIDC/requirements pin）：https://docs.x.ai/build/enterprise
- Hermes 文档首页：https://hermes-agent.nousresearch.com/docs/
- Hermes 配置：https://hermes-agent.nousresearch.com/docs/user-guide/configuration
- Hermes Managed Scope：https://hermes-agent.nousresearch.com/docs/user-guide/managed-scope
- Hermes 上下文文件：https://hermes-agent.nousresearch.com/docs/user-guide/features/context-files
- Hermes MCP（全字段）：https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/mcp.md
- Hermes 主仓：https://github.com/NousResearch/hermes-agent
- OpenClaw 文档首页：https://docs.openclaw.ai/
- OpenClaw 配置总览：https://docs.openclaw.ai/gateway/configuration
- OpenClaw Agent runtime（workspace 六文件）：https://docs.openclaw.ai/concepts/agent
- OpenClaw Skills：https://docs.openclaw.ai/tools/skills
- OpenClaw Tools/Providers 配置：https://docs.openclaw.ai/gateway/config-tools
- OpenClaw 主仓：https://github.com/openclaw/openclaw ；官网 https://openclaw.ai ；维基 https://en.wikipedia.org/wiki/OpenClaw
- Claude Desktop 本地 MCP 教程（配置路径+示例+日志+Windows APPDATA 坑）：https://modelcontextprotocol.io/quickstart/user
- Claude Desktop 远程 Connectors：https://modelcontextprotocol.io/docs/2026-07-28/develop/connect-remote-servers.md 、https://support.anthropic.com/en/articles/11175166-getting-started-with-custom-connectors-using-remote-mcp
- MCPB（原 DXT）规范与更名公告：https://github.com/modelcontextprotocol/mcpb
- MCP 官方 server 目录：https://github.com/modelcontextprotocol/servers
