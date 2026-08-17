# 字段清单：VS Code Copilot（adapter id: `copilot`）

> 调研日期：2026-08-16 ｜ 调研人：P1 工具组（字段级调研）
> 基线文档（官方快照日期 2026-08-12，文档站已从 `/docs/copilot/` 迁移至 `/docs/agent-customization/` 与 `/docs/agents/`）：
> - [custom-instructions](https://code.visualstudio.com/docs/agent-customization/custom-instructions)
> - [prompt-files](https://code.visualstudio.com/docs/agent-customization/prompt-files)
> - [custom-agents](https://code.visualstudio.com/docs/agent-customization/custom-agents)
> - [mcp-configuration](https://code.visualstudio.com/docs/agents/reference/mcp-configuration)
> - [mcp-servers](https://code.visualstudio.com/docs/agent-customization/mcp-servers)
> - [agent-host](https://code.visualstudio.com/docs/agents/concepts/agent-host)
>
> 承载状态图例：【标准字段】= IR-SCHEMA v0.2 标准字段直接承载；【x- 承载】= 需进 `x-copilot` 透传；【无承载】= IR 当前结构无处安放（结构性击穿）。

## 0. 配置文件地图

| 文件 / 目录 | scope | IR 实体 | 备注 |
|---|---|---|---|
| `.github/copilot-instructions.md` | project | Instruction | always-on，单文件 |
| `.github/instructions/**/*.instructions.md` | project | Instruction | 递归扫描；glob 条件生效 |
| `.claude/rules/**/*.md`（兼容） | project / global | Instruction | Claude 格式，frontmatter 用 `paths` 数组 |
| `AGENTS.md`（根 / 子目录，后者实验性） | project | Instruction | always-on；`chat.useNestedAgentsMdFiles` 控制嵌套 |
| `CLAUDE.md` / `CLAUDE.local.md`（兼容） | project / global | Instruction | always-on |
| user profile `*.instructions.md`（VS Code profile user data） | global | Instruction | ⚠️ Agent Host 启用后**不读**此位置 |
| `~/.copilot/instructions/` | global | Instruction | Agent Host 的原生读取位置 |
| `~/.claude/rules/`（兼容） | global | Instruction | Agent Host 亦读 |
| `.github/prompts/*.prompt.md` | project | Command（PromptPack） | slash command |
| user profile prompts 目录 | global | Command | ⚠️ Agent Host 不使用 prompt files（官方提供迁移为 skills） |
| `.github/agents/*.agent.md` | project | Agent（PromptPack） | `.github/agents` 下任意 `.md` 均被识别 |
| `.claude/agents/*.md`（兼容） | project | Agent | Claude 格式 frontmatter |
| `~/.copilot/agents/` | global | Agent | Agent Host 原生读取位置 |
| `.vscode/mcp.json` | project | McpServer | `servers` + `inputs` + `sandbox` 三段 |
| user profile `mcp.json` | global | McpServer | `MCP: Open User Configuration` |
| 工作区根 `.mcp.json` | project | McpServer | Agent Host 原生读取（跨工具惯例键名 `mcpServers`） |
| `~/.copilot/mcp-config.json` | global | McpServer | Agent Host 原生读取 |
| devcontainer.json `customizations.vscode.mcp.servers` | project | McpServer | 建容器时写入远端 mcp.json |
| `%APPDATA%\Code\User\settings.json` / `.vscode/settings.json` | global / project | Setting | `chat.*`、`github.copilot.*` 键 |

## 1. `*.instructions.md`（含 `copilot-instructions.md`）

frontmatter（YAML，全部可选）：

| 字段路径 | 类型 | 语义 | scope | IR 承载 |
|---|---|---|---|---|
| `name` | string | UI 显示名，缺省取文件名 | project / global | 【无承载】Instruction frontmatter 无 `name` 字段（id 末段承担命名，但显示名含空格/大小写时无法还原） |
| `description` | string | 悬停简介；**参与语义匹配**——agent 据此判断指令是否与当前任务相关 | project / global | 【无承载】Instruction 无 `description`；该字段有运行时语义（非纯展示），丢失会影响条件触发 |
| `applyTo` | string | glob（相对工作区根），`**` 为全部；逗号分隔多模式（如 `"**/*.ts,**/*.tsx"`）；缺省 = 不自动应用（仅手动附加） | project / global | 【标准字段】→ `file_patterns`（数组；适配器负责逗号拆分/合并） |
| `paths`（`.claude/rules` 变体） | string[] | Claude Rules 格式的 glob 数组，缺省 `**` | project / global | 【标准字段】→ `file_patterns`（数组天然对应；来源差异进 `x-copilot`） |
| 正文 `#tool:<tool-name>` 引用语法 | inline | 正文中引用 agent 工具（如 `#tool:web/fetch`） | — | 【标准字段】正文原样保留（IR 正文不改写）；跨工具导出按"不翻译，原样搬运 + Warning"规则处理 |

settings 内嵌 instructions（VS Code 1.102 起 code/test generation 已 deprecated，仅剩三类）：

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `github.copilot.chat.reviewSelection.instructions` | array[{`text`\|`file`}] | 代码评审指令（inline 文本或 .md 文件路径） | 【x- 承载】元素为 `{text}`/`{file}` 二选一对象，Setting value 可装但语义需适配器解释 |
| `github.copilot.chat.commitMessageGeneration.instructions` | 同上 | commit message 生成指令 | 【x- 承载】 |
| `github.copilot.chat.pullRequestDescriptionGeneration.instructions` | 同上 | PR 描述生成指令 | 【x- 承载】 |
| `github.copilot.chat.codeGeneration.instructions`（deprecated） | 同上 | 1.102 起废弃 | 【x- 承载】采集保留，导出记 Warning |
| `github.copilot.chat.testGeneration.instructions`（deprecated） | 同上 | 1.102 起废弃 | 【x- 承载】同上 |

## 2. `*.prompt.md`（slash command）

frontmatter（全部可选）：

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `name` | string | `/` 后的命令名，缺省取文件名 | 【标准字段】→ PromptPack `name` |
| `description` | string | 简介 | 【标准字段】→ PromptPack `description` |
| `argument-hint` | string | 输入框占位提示 | 【无承载】PromptPack 无对应字段 → `x-copilot` |
| `agent` | enum \| string | `ask`/`agent`/`plan` 或 custom agent 名；缺省 = 当前 agent；指定 `tools` 时缺省为 `agent` | 【无承载】PromptPack 无 agent 引用 → `x-copilot`（跨工具语义可类比 trigger，建议评估提升） |
| `model` | string | 运行模型；缺省 = 模型选择器当前值 | 【无承载】→ `x-copilot`（多工具共有语义，建议提升标准字段） |
| `tools` | string[] | 可用工具/工具集；`<server name>/*` 含整个 MCP server；运行时不存在的工具被忽略 | 【无承载】→ `x-copilot`（同上） |
| 正文 `${input:var}` / `${input:var:placeholder}` | inline | 交互输入占位 | 【x- 承载】正文原样保留 + 跨工具 Warning（IR-SCHEMA §3.2 已有同型规则） |
| 正文 Markdown 链接引用其他文件 | inline | 相对路径引用 | 【标准字段】正文保留；不进 `imports`（Copilot 链接非指令包含语义，适配器不解析） |

## 3. `*.agent.md`（custom agent）

frontmatter：

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `name` | string | agent 名，缺省取文件名 | 【标准字段】→ PromptPack `name` |
| `description` | string | 简介（chat 输入框占位文本） | 【标准字段】→ `description` |
| `argument-hint` | string | 输入提示 | 【无承载】→ `x-copilot` |
| `tools` | string[] | 可用工具（内置/工具集/MCP/扩展贡献） | 【无承载】→ `x-copilot` |
| `agents` | string[] | 可用 subagent 名单；`*` = 全部，`[]` = 禁止；需 `tools` 含 `agent` | 【无承载】→ `x-copilot` |
| `model` | string \| string[] | 单模型或**按优先级排序的候选数组**（依次尝试） | 【无承载】→ `x-copilot`（数组形态与 Gemini `model.name` 单值不同，提升标准字段需统一为数组） |
| `user-invocable` | boolean | 是否出现在 agents 下拉（缺省 `true`）；`false` = 仅 subagent/程序化调用 | 【无承载】→ `x-copilot` |
| `disable-model-invocation` | boolean | 禁止被其他 agent 作为 subagent 调用（缺省 `false`） | 【无承载】→ `x-copilot` |
| `infer` | boolean | **Deprecated**；由 `user-invocable` + `disable-model-invocation` 取代 | 【x- 承载】采集保留，导出记 Warning |
| `target` | enum | `vscode` \| `github-copilot`（运行环境） | 【无承载】→ `x-copilot` |
| `mcp-servers` | array[object] | 内嵌 MCP server 配置 JSON（仅 `target: github-copilot`） | 【无承载】→ `x-copilot`；注意这是**实体级内嵌 MCP 定义**，与 IR "McpServer 独立实体" 模型冲突，不拆分为独立 mcp 实体（避免 id 冲突），原样透传 |
| `handoffs` | array[object] | 会话间跳转按钮（guided workflow） | 【无承载】→ `x-copilot`（语义上是 workflow 雏形，跨工具迁移时可降级为 instruction 附录） |
| `handoffs[].label` | string | 按钮文本 | 【无承载】随 `handoffs` 透传 |
| `handoffs[].agent` | string | 目标 agent 标识 | 【无承载】同上 |
| `handoffs[].prompt` | string | 预填 prompt | 【无承载】同上 |
| `handoffs[].send` | boolean | 是否自动提交（缺省 `false`） | 【无承载】同上 |
| `handoffs[].model` | string | handoff 执行时模型，限定格式 `Model Name (vendor)` | 【无承载】同上 |
| `hooks`（Preview） | object | agent 级 hook（格式同 hooks 配置文件），需 `chat.useCustomAgentHooks` | 【无承载】→ `x-copilot`；IR 无 hook 实体（见击穿清单 #6） |

`.claude/agents/*.md`（Claude 格式变体）frontmatter：

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `name` | string | **必填** | 【标准字段】 |
| `description` | string | 功能描述 | 【标准字段】 |
| `tools` | string | **逗号分隔字符串**（如 `"Read, Grep, Glob, Bash"`），VS Code 映射为自有工具名 | 【x- 承载】形态差异（string vs array）需保留原格式 |
| `disallowedTools` | string | 逗号分隔黑名单 | 【无承载】→ `x-copilot` |

## 4. `mcp.json`（`.vscode/mcp.json` / user profile / `.mcp.json` / `~/.copilot/mcp-config.json`）

顶层三段：`servers`（object）、`inputs`（array）、`sandbox`（object，仅 macOS/Linux）。

### 4.1 `servers.<name>` — stdio 型

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `type` | enum `"stdio"` | 连接类型 | 【标准字段】→ `transport: stdio` |
| `command` | string | 可执行命令（PATH 或全路径） | 【标准字段】 |
| `args` | string[] | 参数 | 【标准字段】 |
| `cwd` | string | 工作目录，缺省 = workspace folder | 【无承载】IR McpServer 无 `cwd` → **建议提升标准字段**（Gemini 亦有同名字段，跨工具共有） |
| `env` | object<string,string\|number\|null> | 环境变量；值可为 number/null | 【标准字段】→ `env`（值类型需放宽或序列化为字符串，见击穿清单 #4） |
| `envFile` | string | .env 文件路径 | 【标准字段】→ `env_file` |
| `dev` | object | 开发模式：`watch`（glob\|glob[]，变更重启）、`debug`（Node/Python 调试器，仅 stdio） | 【无承载】→ `x-copilot` |
| `sandboxEnabled` | boolean | 沙箱运行（仅 macOS/Linux） | 【无承载】→ `x-copilot` |

### 4.2 `servers.<name>` — http/sse 型

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `type` | enum `"http"` \| `"sse"` | 连接类型（先尝试 HTTP Stream，回退 SSE） | 【标准字段】→ `transport` |
| `url` | string | 服务器 URL；支持 `unix:///path.sock`、`pipe:///pipe/name` 及 `#` 子路径 | 【标准字段】；unix socket / named pipe 形态跨工具不翻译，原样搬运 + Warning |
| `headers` | object | HTTP 头（认证等） | 【标准字段】 |
| `oauth` | object | OAuth 配置，配置后 VS Code 自动走浏览器授权流 | 【无承载】→ `x-copilot` |
| `oauth.clientId` | string | **必填**（oauth 存在时） | 【无承载】同上 |
| `oauth.enterpriseManaged` | boolean | （Preview）经企业 SSO（ID-JAG）静默认证，配合 `mcp.enterpriseManagedAuth.idp` 设置 | 【无承载】同上 |

### 4.3 `inputs[]`（敏感数据输入变量）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `inputs[].type` | enum | `promptString` \| `pickString` \| `command` | 【x- 承载】IR-SCHEMA §3.2 已示范 `x-vscode.inputs`；**文件级**数据挂 server 级 `x-` 的归属规则需 IR 明确（击穿清单 #3） |
| `inputs[].id` | string | `${input:<id>}` 引用键 | 【x- 承载】 |
| `inputs[].description` | string | 提示文案（promptString/pickString 必填） | 【x- 承载】 |
| `inputs[].default` | string | 默认值 | 【x- 承载】 |
| `inputs[].password` | boolean | 掩码输入（缺省 false） | 【x- 承载】与 secretref 抽取策略联动 |
| `inputs[].options` | array[string\|{label,value}] | pickString 选项 | 【x- 承载】 |
| `inputs[].command` | string | command 型：要执行的命令 ID | 【x- 承载】 |
| `inputs[].args` | string\|array\|object | command 型参数 | 【x- 承载】 |
| `${input:xxx}` / `${env:VAR}` / `${workspaceFolder}` 插值 | inline | 值内引用 | 【x- 承载】IR-SCHEMA §3.2 已有规则：同工具往返原样保留，跨工具不翻译 + Warning |

### 4.4 `sandbox`（顶层对象，仅 macOS/Linux）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `sandbox.filesystem.allowWrite` | string[] | 允许写路径（支持 `${workspaceFolder}`） | 【无承载】文件级对象，mcp.yaml 无顶层扩展位（击穿清单 #3） |
| `sandbox.filesystem.denyRead` | string[] | 禁止读路径 | 【无承载】同上 |
| `sandbox.filesystem.denyWrite` | string[] | 禁止写路径 | 【无承载】同上 |
| `sandbox.network.allowedDomains` | string[] | 允许域名（支持 `*.example.com` 通配） | 【无承载】同上 |
| `sandbox.network.deniedDomains` | string[] | 禁止域名 | 【无承载】同上 |

### 4.5 证伪记录（任务预设 vs 实检）

- 任务预设字段 `servers.*.gallery`：**现行官方参考页（2026-08-12 快照）字段表中无 `gallery`**。历史上 mcp.json 曾出现 `"gallery": true` 作为 gallery 安装来源标记；现行文档已不记载。处理：采集时若实测遇到，进 `x-copilot` 透传；字段清单不列标准支持。
- 任务预设 `.instructions.md` frontmatter 含 `agent` 字段：**现行文档 frontmatter 仅 `name`/`description`/`applyTo` 三键**（`.claude/rules` 变体另有 `paths`）。`agent` 字段存在于 `.prompt.md` 而非 `.instructions.md`。按实检修正。

## 5. `settings.json`（`chat.*` / `github.copilot.*`，VS Code 扁平 dotted 键）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `chat.instructionsFilesLocations` | object<path, boolean> | instructions 搜索位置开关（默认 `.github/instructions`、`.claude/rules`、`~/.copilot/instructions`、`~/.claude/rules`） | 【标准字段】Setting value 任意 object；**id 三段式与 dotted key 冲突**（击穿清单 #2） |
| `chat.promptFilesLocations` | object<path, boolean> | prompt 搜索位置 | 【标准字段】同上 |
| `chat.agentFilesLocations` | object<path, boolean> | agent 搜索位置 | 【标准字段】同上 |
| `chat.useAgentsMdFile` | boolean | AGENTS.md 开关 | 【标准字段】 |
| `chat.useNestedAgentsMdFiles` | boolean | 嵌套 AGENTS.md（实验） | 【标准字段】 |
| `chat.useClaudeMdFile` | boolean | CLAUDE.md 开关 | 【标准字段】 |
| `chat.includeApplyingInstructions` | boolean | applyTo 模式匹配指令开关 | 【标准字段】 |
| `chat.includeReferencedInstructions` | boolean | Markdown 链接引用指令开关 | 【标准字段】 |
| `chat.useCustomizationsInParentRepositories` | boolean | monorepo 父仓库发现 | 【标准字段】 |
| `chat.customizations.promptMigration.enabled` | boolean | prompt→skill 一次性迁移（实验） | 【标准字段】 |
| `chat.subagents.allowInvocationsFromSubagents` | boolean | 自引用/嵌套 subagent | 【标准字段】 |
| `chat.useCustomAgentHooks` | boolean | agent 级 hooks 开关（Preview） | 【标准字段】 |
| `chat.agentHost.enabled` | boolean | Agent Host 总开关 | 【标准字段】**迁移关键键**：决定采集根是 profile user data 还是 `~/.copilot/` |
| `chat.mcp.access` | enum/object | MCP server 可用性管控 | 【标准字段】 |
| `chat.mcp.discovery.enabled` | boolean/array | 从其他应用（如 Claude Desktop）发现 MCP 配置 | 【标准字段】**对 cfg4ai 是采集源信号** |
| `chat.mcp.autostart` | enum | 配置变更自动重启（实验） | 【标准字段】 |
| `chat.mcp.serverSampling` | object | 暴露给 MCP server sampling 的模型 | 【标准字段】 |
| `chat.mcp.apps.enabled` | boolean | MCP Apps（实验） | 【标准字段】 |
| `mcp.enterpriseManagedAuth.idp` | string | 企业 SSO IdP（配合 oauth.enterpriseManaged） | 【标准字段】 |
| `github.copilot.chat.organizationInstructions.enabled` | boolean | 组织级指令发现 | 【标准字段】 |
| `github.copilot.chat.organizationCustomAgents.enabled` | boolean | 组织级 agent 发现 | 【标准字段】 |

（注：组织级指令/agent 实体本体存于 GitHub 云端，cfg4ai **不采集**，仅记录开关设置。）

## 6. Agent Host 体系（`~/.copilot/`）

| 路径 | 语义 | IR 承载 |
|---|---|---|
| `~/.copilot/instructions/**/*.instructions.md` | 用户级指令（Agent Host 原生；递归扫描） | 同 §1 |
| `~/.copilot/agents/*.agent.md` | 用户级 agent | 同 §3 |
| `~/.copilot/mcp-config.json` | 用户级 MCP（Agent Host 原生；键名遵循跨工具惯例） | 同 §4 |
| 工作区根 `.mcp.json` | 项目级 MCP（Agent Host 原生，`mcpServers` 键） | 同 §4，导出键名映射 `servers` ↔ `mcpServers` 由适配器负责 |
| `~/.copilot/skills/`（本地实证存在，含 `find-skills/`） | Agent Host skills 目录 | Skill（PromptPack） |
| `~/.copilot/ide/`（本地实证存在） | IDE 联动状态，运行时数据 | 【无承载】不采集（类比 `~/.claude.json` 运行时状态） |

要点：Agent Host 启用时，VS Code 把 `.vscode/mcp.json` 的 servers **转发**给 Agent Host（含 `${input:}` 交互输入的 server 除外）；profile user data 变为 legacy 位置。采集器必须按 `chat.agentHost.enabled` 与文件实存情况双通道探测。

## 7. IR 击穿清单（copilot）

| # | 击穿点 | 等级 | 建议 |
|---|---|---|---|
| 1 | Instruction 缺 `name`/`description`：`description` 参与语义匹配（运行时语义），`name` 是含空格的显示名 | MAJOR | Instruction frontmatter 增加可选 `name`、`description` |
| 2 | Setting id 三段式 `setting.<tool>.<key>` 与 VS Code 扁平 dotted 键（`chat.mcp.access`）冲突：校验规则 1 限定 name 字符集 `[a-z0-9][a-z0-9-]*`，点号非法 | BLOCKER | id 解析改为"前两段定型，余下整体为 key-path"（允许点号）；或 key 段允许 `.` |
| 3 | 文件级配置（`inputs[]`、`sandbox{}`）无处挂靠：mcp.yaml 是 server 列表，`x-vscode.inputs` 示范把文件级数据塞进了单个 server 条目，归属含混 | MAJOR | IR 明确"文件级扩展"位（如 mcp.yaml 顶层 `x-vscode:` 与 `servers:` 平级），或约定挂在特定 server 并记录 |
| 4 | `env` 值允许 number/null，IR 未限定值类型；多工具 env 均为 string→string | minor | IR 注明 env 值规范化为 string（`null` = 显式置空语义进 `x-copilot`） |
| 5 | PromptPack 缺 `model`/`tools`/`agent`/`argument-hint`：prompt/agent 文件核心字段全靠 `x-copilot` | MAJOR | `model`（string\|string[]）、`tools`（string[]）提升为 PromptPack 标准可选字段；`agent`、`argument-hint`、handoffs/subagents/user-invocable 等留 `x-copilot` |
| 6 | `hooks`（agent 级，Preview）与 handoffs 的 workflow 语义：IR 无 hook 实体，workflow 仅 PromptPack 同构 | minor（功能 Preview） | 暂 `x-copilot`；hook 实体类型留 v0.3 评估 |
| 7 | `mcp-servers`（agent frontmatter 内嵌 MCP 定义）与 McpServer 独立实体模型冲突 | minor | 不拆分，整体 `x-copilot` 透传 |
| 8 | `target: github-copilot` 指向云 harness，实体运行在 GitHub 侧 | minor | `x-copilot.target`；导出到非 Copilot 工具时记 Warning |

## 8. 真实样本

1. **官方 `.instructions.md`（Python Standards）** — [custom-instructions](https://code.visualstudio.com/docs/agent-customization/custom-instructions)：
   ```markdown
   ---
   name: 'Python Standards'
   description: 'Coding conventions for Python files'
   applyTo: '**/*.py'
   ---
   # Python coding standards
   - Follow the PEP 8 style guide.
   ```
2. **官方 `.instructions.md`（多 glob 逗号分隔）** — 同上页：`applyTo: "**/*.ts,**/*.tsx"`（TypeScript/React 示例，正文含相对链接 `[general coding guidelines](./general-coding.instructions.md)`）。
3. **官方 `.prompt.md`（React 表单）** — [prompt-files](https://code.visualstudio.com/docs/agent-customization/prompt-files)：
   ```markdown
   ---
   agent: 'agent'
   model: GPT-4o
   tools: ['search/codebase', 'vscode/askQuestions']
   description: 'Generate a new React form component'
   ---
   ```
4. **官方 `.agent.md`（Planner，handoffs + model 数组）** — [custom-agents](https://code.visualstudio.com/docs/agent-customization/custom-agents)：
   ```markdown
   ---
   description: Generate an implementation plan ...
   name: Planner
   tools: ['web/fetch', 'search/codebase', 'search/usages']
   model: ['Claude Opus 4.5', 'GPT-5.2']  # Tries models in order
   handoffs:
     - label: Implement Plan
       agent: agent
       prompt: Implement the plan outlined above.
       send: false
   ---
   ```
5. **官方 `.agent.md`（hooks Preview）** — 同上页：`hooks: { PostToolUse: [{ type: command, command: "./scripts/format-changed-files.sh" }] }`。
6. **官方 `mcp.json`（inputs + stdio）** — [mcp-configuration](https://code.visualstudio.com/docs/agents/reference/mcp-configuration)：
   ```json
   {
     "inputs": [{ "type": "promptString", "id": "perplexity-key", "description": "Perplexity API Key", "password": true }],
     "servers": { "perplexity": { "type": "stdio", "command": "npx", "args": ["-y", "server-perplexity-ask"],
       "env": { "PERPLEXITY_API_KEY": "${input:perplexity-key}" } } }
   }
   ```
7. **官方 `mcp.json`（sandbox + OAuth）** — 同上页：`"sandbox": { "filesystem": { "allowWrite": ["${workspaceFolder}"], "denyRead": ["${userHome}/.ssh"] }, "network": { "allowedDomains": ["api.example.com"] } }`；OAuth 示例 `"oauth": { "clientId": "example-client-id" }`（slack server）。
8. **官方 devcontainer 内嵌 MCP** — [mcp-servers](https://code.visualstudio.com/docs/agent-customization/mcp-servers)：`customizations.vscode.mcp.servers.playwright = { "command": "npx", "args": ["-y", "@microsoft/mcp-server-playwright"] }`。
9. **社区样本仓库** — [github/awesome-copilot](https://github.com/github/awesome-copilot/tree/main)：官方背书社区合集，含大量 `.instructions.md` / `.prompt.md` / `.agent.md` 实例（按 `instructions/`、`prompts/`、`agents/` 目录组织），可作为适配器 golden-file 语料。

## 9. 证伪结论：IR-SCHEMA v0.2 需要为 copilot 改什么

1. Instruction 增加可选 `name`、`description`（击穿 #1）。
2. Setting id 语法放行 dotted key-path（击穿 #2，copilot/gemini 共同诉求）。
3. mcp.yaml 增加文件级扩展位或明确文件级 `x-` 归属规则（击穿 #3）。
4. McpServer 增加标准可选字段 `cwd`（copilot/gemini 共有）；`env` 值类型规则注明。
5. PromptPack 增加标准可选 `model`（string | string[]）、`tools`（string[]）；其余 copilot 特有键（`argument-hint`/`agent`/`agents`/`user-invocable`/`disable-model-invocation`/`target`/`mcp-servers`/`handoffs`/`hooks`/`infer`）列入 `x-copilot` 透传清单。
6. ADAPTERS §3.3 补录：Agent Host 双通道采集（`chat.agentHost.enabled` 决定）、`.mcp.json` ↔ `servers`/`mcpServers` 键名映射、`inputs/sandbox` 文件级承载约定、`gallery` 字段不列标准支持（遇实测进 `x-copilot`）。
