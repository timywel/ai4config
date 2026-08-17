# 字段清单：Claude Code（adapter id: `claude-code`）

> 调研日期：2026-08-16 ｜ 调研人：P0 工具组（字段级调研）
> 基线文档（官方文档站 2026-08-16 实时抓取，对应 Claude Code v2.1.x 系列）：
> - [settings](https://docs.anthropic.com/en/docs/claude-code/settings)（含 permissions/sandbox/worktree/plugin 小节）
> - [memory](https://docs.anthropic.com/en/docs/claude-code/memory)（CLAUDE.md / rules / auto memory）
> - [sub-agents](https://docs.anthropic.com/en/docs/claude-code/sub-agents)
> - [skills](https://code.claude.com/docs/en/skills)
> - [hooks](https://docs.anthropic.com/en/docs/claude-code/hooks)
> - [mcp](https://docs.anthropic.com/en/docs/claude-code/mcp)
> - [output-styles](https://docs.anthropic.com/en/docs/claude-code/output-styles) / [statusline](https://docs.anthropic.com/en/docs/claude-code/statusline)
>
> 承载状态图例：【标准字段】= IR-SCHEMA v0.2 标准字段直接承载；【x- 承载】= 需进 `x-claude-code` 透传；【无承载】= IR 当前结构无处安放（结构性击穿，见 §9 击穿清单）；【存疑】= 官方文档未明确，附核实方法。

## 0. 配置文件地图

| 文件 / 目录 | scope | IR 实体 | 备注 |
|---|---|---|---|
| `~/.claude/settings.json` | global（user） | Setting | 个人全项目设置 |
| `.claude/settings.json` | project | Setting | 团队共享（入版本库） |
| `.claude/settings.local.json` | local | Setting | 个人本项目（gitignored）；IR 用 `Setting.local: true` 承载 |
| `managed-settings.json` / plist / 注册表 / server-managed | managed | Setting | 组织强制策略；macOS `/Library/Application Support/ClaudeCode/`、Linux/WSL `/etc/claude-code/`、Windows `C:\Program Files\ClaudeCode\`；drop-in `managed-settings.d/*.json` 按字母序合并（标量后者胜、数组拼接去重、object 深合并） |
| `~/.claude.json` | global 运行时状态文件 | Setting（部分键）+ McpServer | OAuth 会话、user/local scope MCP servers、per-project 状态、缓存；**部分拥有权文件**，只允许读-改-写局部 patch |
| `.mcp.json`（项目根） | project | McpServer | 项目共享 MCP servers |
| `~/.claude/CLAUDE.md` | global | Instruction | 个人指令 |
| `./CLAUDE.md`、`./.claude/CLAUDE.md` | project | Instruction | 项目指令 |
| `./CLAUDE.local.md` | local | Instruction | 个人项目指令（gitignored） |
| managed 目录下 `CLAUDE.md` | managed | Instruction | 组织指令 |
| `.claude/rules/**/*.md`、`~/.claude/rules/*.md` | project / global | Instruction | 分主题规则，frontmatter `paths` 条件加载 |
| `~/.claude/agents/**/*.md`、`.claude/agents/**/*.md` | global / project | Agent（PromptPack） | 递归扫描，身份只看 frontmatter `name` |
| `~/.claude/skills/<name>/SKILL.md`、`.claude/skills/<name>/SKILL.md` | global / project | Skill（PromptPack） | 目录即包，可含支撑文件 |
| `.claude/commands/*.md`、`~/.claude/commands/*.md` | project / global | Command（PromptPack） | 已并入 skills 体系；同名 skill 优先 |
| `~/.claude/output-styles/*.md`、`.claude/output-styles/*.md` | global / project | 无实体（见 §7） | 输出风格 |
| 插件 `hooks/hooks.json` | plugin | 无实体（见 §2） | 插件 hooks |
| `~/.claude/projects/<project>/memory/` | global（机器本地） | 不采集（见 §3） | auto memory（MEMORY.md + topic 文件） |

优先级（高→低）：Managed > 命令行参数 > Local > Project > User。**数组键跨 scope 拼接去重**（例外：`fallbackModel` 整体取最高优先级层；`availableModels` 在 managed 层定义时锁定）。settings 文件热重载，`ConfigChange` hook 逐变更触发。

## 1. settings.json（全部键逐字段）

scope 列说明：未注明者 user/project/local/managed 均可读；标注 managed only 者仅在 managed 层生效。

### 1.1 顶层键（按字母序）

| 字段路径 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `advisorModel` | string | 服务端 advisor 工具模型（`"fable"`/`"opus"`/`"sonnet"` 或完整 ID；`/advisor` 自动写入） | 未设置=禁用 | 【x- 承载】 |
| `agent` | string | 主线程以指定 subagent 运行；`claude agents` 派发会话的默认 agent | 无 | 【x- 承载】 |
| `agentPushNotifEnabled` | boolean | Remote Control 下允许主动推手机通知 | `false` | 【x- 承载】 |
| `allowAllClaudeAiMcps` | boolean | managed-mcp.json 之外同时加载 claude.ai connectors | 未设置 | 【无承载】managed only → 击穿清单 B2 |
| `allowedChannelPlugins` | array<object> | channel 插件白名单 `[{marketplace, plugin}]` | undefined=默认表；[]=全禁 | 【无承载】managed only → B2 |
| `allowedHttpHookUrls` | array<string> | HTTP hook URL 白名单（支持 `*`；跨 source 合并） | undefined=不限；[]=全禁 | 【x- 承载】 |
| `allowedMcpServers` | array<object> | 用户可配置 MCP server 白名单 `[{serverName}]`；[]=锁定 | undefined=不限 | 【无承载】managed 策略层 → B2 |
| `allowManagedHooksOnly` | boolean | 仅允许 managed hooks（联动禁用 command-source 插件） | false | 【无承载】managed only → B2 |
| `allowManagedMcpServersOnly` | boolean | 仅认可 managed 的 allowedMcpServers | false | 【无承载】managed only → B2 |
| `allowManagedPermissionRulesOnly` | boolean | 禁止 user/project 定义 allow/ask/deny 规则 | false | 【无承载】managed only → B2 |
| `alwaysThinkingEnabled` | boolean | 默认开启 extended thinking | 未设置 | 【x- 承载】（IR-SCHEMA §1.1 已用作 x- 示例） |
| `apiKeyHelper` | string | 生成 auth 值的自定义命令（经系统 shell）；配 `CLAUDE_CODE_API_KEY_HELPER_TTL_MS` | 无 | 【x- 承载】敏感命令串，注意脱敏 |
| `askUserQuestionTimeout` | string | AskUserQuestion 自动继续等待 `"60s"/"5m"/"10m"/"never"`；仅 user settings | `"never"` | 【x- 承载】 |
| `attribution` | object | git commit/PR 署名定制，见 §1.3 | 见 §1.3 | 【x- 承载】 |
| `autoCompactEnabled` | boolean | 上下文接近上限时自动压缩 | `true` | 【x- 承载】 |
| `autoCompactWindow` | number | 自动压缩窗口阈值（token，100000–1000000） | 按模型调优 | 【x- 承载】 |
| `autoMemoryDirectory` | string | auto memory 自定义目录（绝对路径或 `~/` 开头） | `~/.claude/projects/<project>/memory/` | 【x- 承载】 |
| `autoMemoryEnabled` | boolean | 开关 auto memory | `true` | 【x- 承载】 |
| `autoMode` | object | auto mode 分类器规则：`environment/allow/soft_deny/hard_deny` 数组（prose 规则，`"$defaults"` 占位继承内置）；仅 user/`--settings`/managed | 无 | 【x- 承载】自由文本规则数组，无语义翻译（观察项） |
| `autoMode.classifyAllShell` | boolean | auto mode 下所有 shell 命令都过分类器 | `false` | 【x- 承载】 |
| `autoScrollEnabled` | boolean | fullscreen 渲染跟随输出滚动 | `true` | 【x- 承载】 |
| `autoUpdatesChannel` | string | 更新通道 `"stable"/"latest"` | `"latest"` | 【x- 承载】 |
| `availableModels` | array<string> | 可选模型白名单（主会话/subagent/skill/advisor） | 无 | 【x- 承载】 |
| `awaySummaryEnabled` | boolean | 离开数分钟后返回显示会话摘要 | true | 【x- 承载】 |
| `awsAuthRefresh` | string | 修改 `.aws` 目录的自定义脚本（Bedrock 凭证） | 无 | 【x- 承载】 |
| `awsCredentialExport` | string | 输出 AWS 凭证 JSON 的自定义脚本 | 无 | 【x- 承载】 |
| `axScreenReader` | boolean | 屏幕阅读器友好渲染 | false | 【x- 承载】 |
| `blockedMarketplaces` | array<object> | 插件市场黑名单；`github` source 支持 `"owner/*"` 通配 | 无 | 【无承载】managed only → B2/B3 |
| `browserExternalPageTools` | string | `"disabled"` 禁止 Claude 操作桌面 App 外部网页 | 无 | 【无承载】managed only → B2 |
| `channelsEnabled` | boolean | 组织级 channels 开关 | 视账号类型 | 【无承载】managed only → B2 |
| `claudeMd` | string | 内联 managed CLAUDE.md 内容（组织指令直接写进 settings） | 无 | 【无承载】managed only → 击穿清单 B7（Instruction 无法表达"settings 内联"来源形态） |
| `claudeMdExcludes` | array<string> | 跳过指定 CLAUDE.md 的 glob/绝对路径（按绝对路径匹配；跨层合并；managed policy 文件不可排除） | 无 | 【x- 承载】 |
| `cleanupPeriodDays` | number | 会话文件等数据保留天数，最小 1 | `30` | 【x- 承载】 |
| `companyAnnouncements` | array<string> | 启动时展示的公告（多条随机轮播） | 无 | 【x- 承载】 |
| `crossSessionInbound` | string | 跨会话消息入站策略 `"accept"/"hold"/"refuse"` | 按会话对决定 | 【x- 承载】 |
| `defaultShell` | string | 输入框 `!` 命令默认 shell `"bash"/"powershell"` | `"bash"`（Windows 无 Bash 时 `"powershell"`） | 【x- 承载】 |
| `deniedMcpServers` | array<object> | MCP server 黑名单 `[{serverName}]`，优先于白名单 | 无 | 【无承载】managed → B2 |
| `dialogExpiry` | string | 远程转发对话框超时 `"60s"/"5m"/"10m"/"never"` | `"5m"` | 【x- 承载】 |
| `disableAgentView` | boolean | 关闭 background agents 与 agent view | false | 【x- 承载】 |
| `disableAllHooks` | boolean | 禁用全部 hooks、自定义 statusLine、fileSuggestion | false | 【x- 承载】 |
| `disableArtifact` | boolean | 禁用 Artifact 工具 | false | 【x- 承载】 |
| `disableAutoMode` | string | `"disable"` 阻止激活 auto mode（也可置于 `permissions.disableAutoMode`） | 未设置 | 【x- 承载】 |
| `disableBrowserExternalNavigation` | boolean | 关闭桌面 App 外部浏览 | false | 【无承载】managed only → B2 |
| `disableBundledSkills` | boolean | 禁用随附 skills/workflows（内置命令保留可输入） | false | 【x- 承载】 |
| `disableClaudeAiConnectors` | boolean | 禁用 claude.ai MCP connectors 自动获取；任一 scope 的 `true` 生效 | false | 【x- 承载】 |
| `disableCommandPluginSources` | boolean | 禁止 command-source 插件安装/加载 | 跟随 allowManagedHooksOnly | 【无承载】managed only → B2 |
| `disableDeepLinkRegistration` | string | `"disable"` 不注册 `claude-cli://` 协议 | 未设置 | 【x- 承载】 |
| `disabledMcpjsonServers` | array<string> | 拒绝的 `.mcp.json` 服务器名单 | 无 | 【x- 承载】审批列表，非服务器定义一部分 |
| `disableMobileSimulatorTools` | boolean | 禁止 Claude 使用 iOS Simulator 工具 | false | 【无承载】managed only → B2 |
| `disableRemoteControl` | boolean | 禁用 Remote Control | false | 【x- 承载】 |
| `disableSideloadFlags` | boolean | 拒绝 `--plugin-dir/--plugin-url/--agents/--mcp-config` 启动旗标 | false | 【无承载】managed only → B2 |
| `disableSkillShellExecution` | boolean | 禁用 skills/commands 中 `` !`...` `` 与 ` ```! ` 内联 shell 执行 | false | 【x- 承载】 |
| `disableWorkflows` | boolean | 禁用动态 workflows | `false` | 【x- 承载】 |
| `editorMode` | string | 输入键位模式 `"normal"/"vim"` | `"normal"` | 【x- 承载】 |
| `effortLevel` | string | 跨会话保持的 effort `"low"/"medium"/"high"/"xhigh"` | 未设置 | 【x- 承载】 |
| `emojiCompletionEnabled` | boolean | `:` 短码 emoji 补全 | `true` | 【x- 承载】 |
| `enableAllProjectMcpServers` | boolean | 自动批准项目 `.mcp.json` 全部服务器 | false | 【x- 承载】 |
| `enableArtifact` | boolean | 用户级 Artifact 开关（project/local 中忽略） | 视账号 | 【x- 承载】 |
| `enabledMcpjsonServers` | array<string> | 批准的 `.mcp.json` 服务器名单 | 无 | 【x- 承载】 |
| `enforceAvailableModels` | boolean | availableModels 白名单扩展到 Default 选项 | false | 【无承载】managed → B2 |
| `env` | object<string,string> | 注入每个会话及子进程的环境变量；`""` 表示覆盖为空 | 无 | 【标准字段】Setting.value 通用 object 承载（语义=工具环境注入，不跨工具翻译） |
| `fallbackModel` | array<string> | 主模型过载回退链（≤3 个，`"default"` 展开；不跨文件合并，取最高优先级层整条） | 无 | 【x- 承载】 |
| `fastMode` | boolean | fast mode 开关 | 未设置 | 【x- 承载】 |
| `fastModePerSessionOptIn` | boolean | fast mode 不跨会话保持 | false | 【x- 承载】 |
| `feedbackSurveyRate` | number(0–1) | 会话质量调查出现概率；`0` 完全抑制 | 默认采样率 | 【x- 承载】 |
| `fileCheckpointingEnabled` | boolean | 编辑前快照文件供 `/rewind` | `true` | 【x- 承载】 |
| `fileSuggestion` | object | `@` 文件补全自定义命令 `{type:"command", command}` | 内置遍历 | 【x- 承载】 |
| `footerLinksRegexes` | array<object> | 页脚链接徽章 `[{type:"regex", pattern, url, label?}]`（命名捕获组填 `{name}` 占位）；仅 user/`--settings`/managed | 无 | 【x- 承载】 |
| `forceLoginMethod` | string | 限定登录方式 `claudeai/console/gateway` | 无 | 【x- 承载】 |
| `forceLoginGatewayUrl` | string | 预填并锁定 gateway URL | 无 | 【无承载】managed only → B2 |
| `forceLoginOrgUUID` | string \| array<string> | 限定登录组织 UUID | 无 | 【x- 承载】 |
| `forceRemoteSettingsRefresh` | boolean | 启动前必须成功拉取远端 managed settings（fail-closed） | false | 【无承载】managed only → B2 |
| `gcpAuthRefresh` | string | GCP ADC 刷新脚本 | 无 | 【x- 承载】 |
| `hooks` | object | 生命周期 hooks，结构见 §2 | 无 | 【无承载】→ 击穿清单 B1 |
| `httpHookAllowedEnvVars` | array<string> | HTTP hook headers 可插值的环境变量白名单（与各 hook 自身 allowedEnvVars 求交集） | undefined=不限 | 【x- 承载】 |
| `includeGitInstructions` | boolean | 系统提示含内置 commit/PR 指令与 git status 快照 | `true` | 【x- 承载】 |
| `inputNeededNotifEnabled` | boolean | Remote Control 下有待办输入时推手机通知 | `false` | 【x- 承载】 |
| `isolatePeerMachines` | boolean | 跨机器 SendMessage 需显式批准；任一 scope 的 true 生效 | false | 【x- 承载】 |
| `language` | string | 回复语言（如 `"japanese"`），兼管语音听写与会话标题语言 | 未设置 | 【x- 承载】（与 Instruction.language 维度不同：后者是指令文档语言标注） |
| `minimumVersion` | string | 自动更新版本下限（不阻止启动） | 无 | 【x- 承载】 |
| `model` | string | 默认模型覆盖 | 无 | 【标准字段】Setting key=model（跨工具翻译映射属导出器职责） |
| `modelOverrides` | object | Anthropic 模型 ID → 提供方模型 ID（如 Bedrock ARN）映射 | 无 | 【x- 承载】 |
| `otelHeadersHelper` | string | 生成动态 OTel headers 的脚本 | 无 | 【x- 承载】 |
| `outputStyle` | string | 输出风格名（内置或自定义） | `"default"` | 【x- 承载】（output style 本体见 §7） |
| `parentSettingsBehavior` | string | 嵌入宿主 supplied managed settings 处置 `"first-wins"/"merge"` | `"first-wins"` | 【无承载】managed only → B2 |
| `permissions` | object | 权限规则，结构见 §1.2 | 无 | 【标准字段】IR §3.4 示例已将 permissions 作为不透明 value |
| `plansDirectory` | string | plan 文件存放目录（相对项目根） | `~/.claude/plans` | 【x- 承载】 |
| `pluginSuggestionMarketplaces` | array<string> | 允许出现安装建议的市场名 | 无 | 【无承载】managed only → B2/B3 |
| `pluginTrustMessage` | string | 插件信任警告附加组织信息 | 无 | 【无承载】managed only → B2 |
| `policyHelper` | object | 启动时动态计算 managed settings 的可执行程序 `{path, timeoutMs, refreshIntervalMs}` | 无 | 【无承载】managed only → B2 |
| `preferredNotifChannel` | string | 通知方式 `"auto"/"terminal_bell"/"iterm2"/"iterm2_with_bell"/"kitty"/"ghostty"/"notifications_disabled"` | `"auto"` | 【x- 承载】 |
| `prefersReducedMotion` | boolean | 减少 UI 动画 | false | 【x- 承载】 |
| `processWrapper` | string | 企业启动器命令（包裹后台进程） | 无 | 【x- 承载】 |
| `promptSuggestionEnabled` | boolean | 输入框灰色预测建议 | `true` | 【x- 承载】 |
| `prUrlTemplate` | string | PR 徽章 URL 模板（`{host}/{owner}/{repo}/{number}/{url}` 占位） | 无 | 【x- 承载】 |
| `remote.defaultEnvironmentId` | string | 云会话默认环境 ID（点号嵌套键） | 无 | 【x- 承载】 |
| `remoteControlAtStartup` | boolean | 会话启动自动连接 Remote Control | 未设置（跟随组织默认） | 【x- 承载】 |
| `requiredMaximumVersion` | string | 允许启动的最高版本（超出即退出） | 无 | 【无承载】managed only → B2 |
| `requiredMinimumVersion` | string | 允许启动的最低版本 | 无 | 【无承载】managed only → B2 |
| `respectGitignore` | boolean | `@` 文件选择器遵守 .gitignore | `true` | 【x- 承载】 |
| `respondToBashCommands` | boolean | `!` 命令执行后 Claude 是否回应 | `true` | 【x- 承载】 |
| `showClearContextOnPlanAccept` | boolean | plan 接受页显示 clear context 选项 | `false` | 【x- 承载】 |
| `showThinkingSummaries` | boolean | 显示 thinking 摘要 | `false` | 【x- 承载】 |
| `showTurnDuration` | boolean | 显示回合耗时 | `true` | 【x- 承载】 |
| `skillListingBudgetFraction` | number | skill 清单占上下文窗口比例 | `0.01` | 【x- 承载】 |
| `skillListingMaxDescChars` | number | 单 skill description+when_to_use 截断字符数 | `1536` | 【x- 承载】 |
| `skillOverrides` | object<string,string> | 按 skill 名可见性覆盖 `"on"/"name-only"/"user-invocable-only"/"off"` | 无 | 【x- 承载】 |
| `skipWebFetchPreflight` | boolean | 跳过 WebFetch 域名安全预检 | false | 【x- 承载】 |
| `spinnerTipsEnabled` | boolean | spinner 提示语 | `true` | 【x- 承载】 |
| `spinnerTipsOverride` | object | 自定义提示 `{tips: [], excludeDefault: bool}` | 无 | 【x- 承载】 |
| `spinnerVerbs` | object | 自定义动作动词 `{mode:"replace"/"append", verbs:[]}` | 无 | 【x- 承载】 |
| `sshConfigs` | array<object> | 桌面环境 SSH 连接 `[{id,name,sshHost,sshPort?,sshIdentityFile?,startDirectory?}]`；仅 managed/user | 无 | 【x- 承载】 |
| `statusLine` | object | 自定义状态行 `{type:"command", command, padding?, refreshInterval?, hideVimModeIndicator?}` | 无 | 【x- 承载】（脚本本体是文件资产，见 §7） |
| `strictKnownMarketplaces` | array<object> | 插件市场白名单（source 类型 github/git/url/npm/file/directory/hostPattern/pathPattern/settings；精确匹配 + owner 通配） | undefined=不限；[]=锁定 | 【无承载】managed only → B2/B3 |
| `strictPluginOnlyCustomization` | boolean \| array<string> | 锁定 skills/agents/hooks/mcp 只能来自插件或 managed | false | 【无承载】managed only → B2/B3 |
| `subagentStatusLine` | object | subagent 任务展示行重写命令 `{type:"command", command}` | 无 | 【x- 承载】 |
| `switchModelsOnFlag` | boolean | 安全分类器标记时自动切回退模型 | `true` | 【x- 承载】 |
| `syntaxHighlightingDisabled` | boolean | 禁用语法高亮 | false | 【x- 承载】 |
| `teammateMode` | string | agent team 展示 `"in-process"/"auto"/"tmux"/"iterm2"` | `"in-process"` | 【x- 承载】 |
| `terminalProgressBarEnabled` | boolean | 终端进度条 | `true` | 【x- 承载】 |
| `theme` | string | 主题 `"auto"/"dark"/"light"/"dark-daltonized"/"light-daltonized"/"dark-ansi"/"light-ansi"/"custom:<slug>"` | `"dark"` | 【标准字段】theme 为多工具通用键，可走 Setting 翻译表 |
| `tui` | string | 渲染器 `"fullscreen"/"default"` | 未设置 | 【x- 承载】 |
| `ultracode` | boolean | 会话开启 ultracode（不从 settings.json 读取，仅 `/effort`、`--settings`、SDK） | 无 | 【无承载】不持久化于文件，采集无源 → B10 |
| `useAutoModeDuringPlan` | boolean | plan 模式使用 auto mode 语义；不读共享 project settings | `true` | 【x- 承载】 |
| `verbose` | boolean | 完整工具输出 | `false` | 【x- 承载】 |
| `viewMode` | string | 启动 transcript 视图 `"default"/"verbose"/"focus"` | 未设置 | 【x- 承载】 |
| `vimInsertModeRemaps` | object<string,string> | vim 插入模式双键序列映射（目标仅 `"<Esc>"`）；仅 user/`--settings`/managed | 无 | 【x- 承载】 |
| `voice` | object | 语音听写 `{enabled, mode:"hold"/"tap", autoSubmit}` | 无 | 【x- 承载】 |
| `voiceEnabled` | boolean | `voice.enabled` 的遗留别名 | 无 | 【x- 承载】 |
| `wheelScrollAccelerationEnabled` | boolean | 滚轮加速 | `true` | 【x- 承载】 |
| `workflowKeywordTriggerEnabled` | boolean | 输入关键词 `ultracode` 触发动态 workflow | `true` | 【x- 承载】 |
| `workflowSizeGuideline` | string | 动态 workflow 规模建议 `"unrestricted"/"small"/"medium"/"large"` | `"medium"` | 【x- 承载】 |
| `wslInheritsWindowsSettings` | boolean | WSL 读取 Windows 策略链 | false | 【无承载】Windows managed only → B2 |

### 1.2 permissions 子结构

| 字段路径 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `permissions.allow[]` | array<string> | 允许规则，语法 `Tool` 或 `Tool(specifier)`；`mcp__<server>__<tool-glob>` 中 server 段须为字面量 | 无 | 【标准字段】作为 `setting.claude-code.permissions` 不透明 value（IR §3.4 示例）；规则语法 Claude 专有，不跨工具翻译 |
| `permissions.ask[]` | array<string> | 询问规则 | 无 | 【标准字段】同上 |
| `permissions.deny[]` | array<string> | 拒绝规则（先于 ask/allow 评估；`"*"`/`"mcp__*"` 全拒） | 无 | 【标准字段】同上 |
| `permissions.additionalDirectories[]` | array<string> | 额外工作目录（文件访问；不加载其 `.claude/` 配置） | 无 | 【标准字段】同上 |
| `permissions.defaultMode` | string | 新会话权限模式 `default/acceptEdits/plan/auto/dontAsk/bypassPermissions/manual(别名)` | 内置默认 | 【标准字段】同上 |
| `permissions.disableAutoMode` | string | `"disable"`；同顶层 `disableAutoMode` | 无 | 【标准字段】同上 |
| `permissions.disableBypassPermissionsMode` | string | `"disable"` 禁用 bypassPermissions | 无 | 【标准字段】同上 |
| `permissions.skipDangerousModePermissionPrompt` | boolean | 跳过进入 bypass 前的确认（project settings 中被忽略） | 无 | 【标准字段】同上 |

规则求值顺序：deny → ask → allow，首个命中生效（与规则特异性无关）。规则语法形态：`Bash(npm run *)`、`Read(./.env)`、`WebFetch(domain:example.com)`、`Agent(agent_type)` 等；已废弃的 `ignorePatterns` 由 `permissions.deny` 的 `Read(...)` 规则取代。

### 1.3 attribution 子结构

| 字段路径 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `attribution.commit` | string | commit 署名（git trailers）；空串隐藏 | `Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>`（随模型变） | 【x- 承载】 |
| `attribution.pr` | string | PR 描述署名；空串隐藏 | `🤖 Generated with [Claude Code](https://claude.com/claude-code)` | 【x- 承载】 |
| `attribution.sessionUrl` | boolean | commit/PR 是否附 claude.ai 会话链接 | `true` | 【x- 承载】 |

注：`attribution` 优先于已废弃的 `includeCoAuthoredBy`。

### 1.4 sandbox 子结构（sandbox.*）

| 字段路径 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `sandbox.enabled` | boolean | bash 沙箱（macOS/Linux/WSL2） | `false` | 【x- 承载】 |
| `sandbox.failIfUnavailable` | boolean | 沙箱不可用时启动即失败 | `false` | 【x- 承载】 |
| `sandbox.autoAllowBashIfSandboxed` | boolean | 沙箱内自动批准 bash | `true` | 【x- 承载】 |
| `sandbox.excludedCommands[]` | array<string> | 沙箱外运行的命令 | 无 | 【x- 承载】 |
| `sandbox.allowUnsandboxedCommands` | boolean | 允许 `dangerouslyDisableSandbox` 逃逸 | `true` | 【x- 承载】 |
| `sandbox.filesystem.allowWrite[]` | array<string> | 沙箱可写路径（跨 scope 合并；并入 Edit allow 规则路径） | 无 | 【x- 承载】 |
| `sandbox.filesystem.denyWrite[]` | array<string> | 禁写路径 | 无 | 【x- 承载】 |
| `sandbox.filesystem.denyRead[]` | array<string> | 禁读路径 | 无 | 【x- 承载】 |
| `sandbox.filesystem.allowRead[]` | array<string> | denyRead 区域内重开读 | 无 | 【x- 承载】 |
| `sandbox.filesystem.allowManagedReadPathsOnly` | boolean | 仅认可 managed 的 allowRead | `false` | 【无承载】managed only → B2 |
| `sandbox.filesystem.disabled` | boolean | 关文件系统隔离保留网络隔离（仅 user/managed/`--settings`） | `false` | 【x- 承载】 |
| `sandbox.credentials.files[]` | array<object> | 凭证文件保护 `[{path, mode:"deny"/"mask", extract?, onExtractNoMatch?, decode?, maskClaims?, maskDuplicates?, injectHosts?}]` | 无 | 【无承载】→ B4（深度嵌套凭证遮蔽策略，先 x- 透传） |
| `sandbox.credentials.envVars[]` | array<object> | 凭证环境变量保护 `[{name, mode, extract?, onExtractNoMatch?, decode?, maskClaims?, injectHosts?}]` | 无 | 【无承载】→ B4 |
| `sandbox.credentials.allowPlaintextInject` | boolean | 允许明文 HTTP 注入 | `false` | 【x- 承载】 |
| `sandbox.credentials.awsPairs[]` | array<object> | AWS SigV4 重签组 `[{accessKeyIdVar, secretAccessKeyVar, sessionTokenVar?}]` | 无 | 【无承载】→ B4 |
| `sandbox.credentials.sigv4` | object | `{streaming, presigned, sigv4a}` 各取 `deny/passthrough` | deny | 【无承载】→ B4 |
| `sandbox.network.allowUnixSockets[]` | array<string> | 可用 Unix socket 路径（macOS） | 无 | 【x- 承载】 |
| `sandbox.network.allowAllUnixSockets` | boolean | 放开全部 Unix socket | `false` | 【x- 承载】 |
| `sandbox.network.allowLocalBinding` | boolean | 允许绑定 localhost 端口（macOS） | `false` | 【x- 承载】 |
| `sandbox.network.allowMachLookup[]` | array<string> | 额外 XPC/Mach 服务名（macOS，尾部 `*` 前缀匹配） | 无 | 【x- 承载】 |
| `sandbox.network.allowedDomains[]` | array<string> | 出站域名白名单（`*.example.com`、IPv6 `[::1]:443` 形式） | 无 | 【x- 承载】 |
| `sandbox.network.deniedDomains[]` | array<string> | 出站域名黑名单（优先于白名单） | 无 | 【x- 承载】 |
| `sandbox.network.strictAllowlist` | boolean | 白名单外直接拒（不询问）；仅 user/managed/`--settings` | `false` | 【x- 承载】 |
| `sandbox.network.allowManagedDomainsOnly` | boolean | 仅认可 managed 域名规则 | `false` | 【无承载】managed only → B2 |
| `sandbox.network.httpProxyPort` | number | 自带 HTTP 代理端口 | 内置代理 | 【x- 承载】 |
| `sandbox.network.socksProxyPort` | number | 自带 SOCKS5 代理端口 | 内置代理 | 【x- 承载】 |
| `sandbox.network.tlsTerminate` | object | 沙箱代理 TLS 终结 `{}` 或 `{caCertPath, caKeyPath}` | 无 | 【x- 承载】 |
| `sandbox.enableWeakerNestedSandbox` | boolean | Docker 非特权环境弱化沙箱（降安全） | `false` | 【x- 承载】 |
| `sandbox.enableWeakerNetworkIsolation` | boolean | 放开系统 TLS 信任服务（macOS，降安全） | `false` | 【x- 承载】 |
| `sandbox.allowAppleEvents` | boolean | 允许发送 Apple Events（macOS，移除代码执行隔离） | `false` | 【x- 承载】 |
| `sandbox.bwrapPath` | string | bwrap 二进制绝对路径 | 自动探测 | 【无承载】managed only → B2 |
| `sandbox.socatPath` | string | socat 二进制绝对路径 | 自动探测 | 【无承载】managed only → B2 |

路径前缀语义：`/` 绝对、`~/` 家目录、`./` 或无前缀=项目根（project settings）或 `~/.claude`（user settings）；旧式 `//path` 仍兼容；尾部 `/` 与 `/**` 会被归一化。

### 1.5 worktree 子结构（worktree.*）

| 字段路径 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `worktree.baseRef` | string | 新 worktree 分支起点 `"fresh"/"head"` | `"fresh"` | 【x- 承载】 |
| `worktree.symlinkDirectories[]` | array<string> | 从主仓库软链进 worktree 的目录 | 无 | 【x- 承载】 |
| `worktree.sparsePaths[]` | array<string> | sparse-checkout 目录 | 无 | 【x- 承载】 |
| `worktree.bgIsolation` | string | 后台会话隔离 `"worktree"/"none"` | `"worktree"` | 【x- 承载】 |

另有项目根 `.worktreeinclude` 文件（拷贝 gitignored 文件入新 worktree）——非 settings 键，独立惯例文件。【无承载】（IR 无对应实体；观察项，可用 PromptPack assets 兜底）。

### 1.6 插件设置（settings.json 内）

| 字段路径 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `enabledPlugins` | object<string,boolean> | `"plugin@marketplace": true/false` | 缺省随插件 defaultEnabled | 【x- 承载】（插件是 IR 未建模实体 → B3） |
| `pluginConfigs` | object | 插件 userConfig 收集的非敏感选项 `{plugin@marketplace: {options: {...}}}`；仅 user/`--settings`/managed | 无 | 【x- 承载】 |
| `extraKnownMarketplaces` | object | 额外市场 `{name: {source: {...}, autoUpdate?}}`；source 类型 github/git/url/npm/file/directory/hostPattern/settings；github/git 支持 `ref`/`path`/`skipLfs`；url 支持 `headers`（含 `${VAR}` 插值） | 无 | 【无承载】→ B3（插件市场注册表，IR 无实体） |
| `strictKnownMarketplaces` | array<object> | managed 白名单（见 §1.1） | 无 | 【无承载】→ B2/B3 |

### 1.7 ~/.claude.json（global config settings + 运行时状态）

settings 页确认**仅**存在于此文件的键（写入 settings.json 会被静默忽略）：

| 字段路径 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `autoConnectIde` | boolean | 外部终端启动时自动连 IDE | `false` | 【x- 承载】回写目标是 ~/.claude.json，适用部分拥有权原则 |
| `autoInstallIdeExtension` | boolean | VS Code 终端内自动装 IDE 扩展 | `true` | 【x- 承载】 |
| `diffTool` | string | diff 展示 `"auto"/"terminal"` | `"auto"` | 【x- 承载】 |
| `externalEditorContext` | boolean | Ctrl+G 外部编辑器预填上次回复 | `false` | 【x- 承载】 |
| `permissionExplainerEnabled` | boolean | Ctrl+E 权限解释 | `true` | 【x- 承载】 |
| `teammateDefaultModel` | string \| null | agent team 队友默认模型 | null=继承 | 【x- 承载】 |

运行时状态结构（部分拥有权文件，适配器只允许读-改-写局部 patch）：

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `mcpServers` | object | user scope MCP 服务器（结构同 §4） | 【标准字段】McpServer 实体，scope=global |
| `projects.<abs-path>.mcpServers` | object | local scope MCP 服务器（仅该项目加载） | 【标准字段】McpServer 实体；【存疑】IR scope 枚举只有 global/project，此形态建议 scope=project + `x-claude-code.project_path` 记录挂载点（核实：实采样本验证键路径形态） |
| `projects.<abs-path>.disabledMcpServers[]` | array<string> | 按项目禁用服务器（opt-out 列表，`/mcp` 面板落点） | 【x- 承载】 |
| `projects.<abs-path>.enabledMcpServers[]` | array<string> | 按项目启用默认关的内置服务器（opt-in） | 【x- 承载】 |
| `projects.<abs-path>.allowedTools[]` 等 per-project 状态 | array | 记住的权限批准、信任设置 | 【无承载】→ B8（运行时状态，非配置；建议不采集） |
| OAuth 会话、各类缓存 | — | 登录态与缓存 | 【无承载】→ B8（凭证走 secretref/keyring，不入库） |

【存疑】`~/.claude.json` 完整键表无官方文档，社区观察到的顶层键还有 `userID`、`hasCompletedOnboarding`、`autoUpdates`、`installMethod` 等。核实方法：本地实采一份 `~/.claude.json` 做键枚举（注意脱敏）。

### 1.8 环境变量（择要）

来源：各页交叉引用（完整表在 [env-vars](https://docs.anthropic.com/en/docs/claude-code/env-vars)，本次未逐字段抓取——【存疑】补抓该页即可闭合）。

常用项：`ANTHROPIC_MODEL`、`ANTHROPIC_DEFAULT_*_MODEL` 家族、`ANTHROPIC_BASE_URL`、`CLAUDE_CODE_API_KEY_HELPER_TTL_MS`、`CLAUDE_CODE_AUTO_COMPACT_WINDOW`、`CLAUDE_CONFIG_DIR`、`CLAUDE_CODE_DISABLE_AUTO_MEMORY`、`CLAUDE_CODE_EFFORT_LEVEL`、`CLAUDE_CODE_SUBAGENT_MODEL`、`CLAUDE_PROJECT_DIR`（hooks/stdio server 注入）、`ENABLE_TOOL_SEARCH`（true/auto/auto:N/false）、`MCP_TIMEOUT`、`MCP_TOOL_TIMEOUT`、`MAX_MCP_OUTPUT_TOKENS`、`MAX_THINKING_TOKENS`、`DISABLE_AUTOUPDATER`、`DISABLE_AUTO_COMPACT`、`DISABLE_DOCTOR_COMMAND`、`CLAUDE_CODE_SYNC_SKILLS` 等。

IR 承载：环境变量经 `env` 键入 settings.json 后按 §1.1 `env` 处理；纯 shell 环境不属配置文件采集范围。

## 2. hooks 结构（settings.json `hooks` 键 + 插件 hooks/hooks.json）

### 2.1 事件全集（31 个）

| 事件 | matcher 过滤对象 | 取值示例 |
|---|---|---|
| `SessionStart` | 启动方式 | `startup`/`resume`/`clear`/`compact`/`fork` |
| `Setup` | 触发旗标 | `init`/`maintenance` |
| `UserPromptSubmit` | 无 matcher | — |
| `UserPromptExpansion` | 命令名 | skill/command 名 |
| `PreToolUse` | 工具名 | `Bash`、`Edit\|Write`、`mcp__.*` |
| `PermissionRequest` | 工具名 | 同上 |
| `PermissionDenied` | 工具名 | 同上 |
| `PostToolUse` | 工具名 | 同上 |
| `PostToolUseFailure` | 工具名 | 同上 |
| `PostToolBatch` | 无 | — |
| `Notification` | 通知类型 | `permission_prompt`/`idle_prompt`/`auth_success`/`elicitation_*`/`agent_needs_input`/`agent_completed` |
| `MessageDisplay` | 无 | — |
| `SubagentStart` / `SubagentStop` | agent type | `general-purpose`/`Explore`/`Plan`/自定义名/插件名 `^my-plugin:reviewer$` |
| `TaskCreated` / `TaskCompleted` | 无 | — |
| `Stop` | 无 | — |
| `StopFailure` | 错误类型 | `rate_limit`/`overloaded`/`authentication_failed`/`oauth_org_not_allowed`/`billing_error`/`invalid_request`/`model_not_found`/`server_error`/`max_output_tokens`/`unknown` |
| `TeammateIdle` | 无 | — |
| `InstructionsLoaded` | 加载原因 | `session_start`/`nested_traversal`/`path_glob_match`/`include`/`compact` |
| `ConfigChange` | 配置源 | `user_settings`/`project_settings`/`local_settings`/`policy_settings`/`skills` |
| `CwdChanged` | 无 | — |
| `DirectoryAdded` | 添加方式 | `slash_command`/`register_repo_root` |
| `FileChanged` | 字面文件名监视列表 | `.envrc\|.env` |
| `WorktreeCreate` / `WorktreeRemove` | 无 | — |
| `PreCompact` / `PostCompact` | 压缩触发 | `manual`/`auto` |
| `Elicitation` / `ElicitationResult` | MCP server 名 | 已配置服务器名 |
| `SessionEnd` | 结束原因 | `clear`/`resume`/`logout`/`prompt_input_exit`/`bypass_permissions_disabled`/`other` |

IR 承载：【无承载】→ B1（IR 无 Hook 实体；当前只能整体作为 `setting.claude-code.hooks` 的不透明 value，丢失事件/matcher 结构语义）。

### 2.2 配置结构

```
hooks.<Event>[] = { matcher?: string, hooks: Handler[] }
```

- `matcher` 语义：`"*"`/空/省略=全匹配；仅含字母数字 `_-` 空格 `,` `|` = 精确串或列表；含其他字符 = JavaScript 非锚定正则。`FileChanged`/`StopFailure` 精确集更窄（仅字母数字 `_|`）。
- Handler 公共字段：`type`（必填：`command|http|mcp_tool|prompt|agent`）、`if`（权限规则语法单条过滤器，仅工具事件）、`timeout`（秒；command/http/mcp_tool 默认 600，prompt 30，agent 60；UserPromptSubmit 降为 30，MessageDisplay 降为 10，SessionEnd 共享 1.5s 预算上限 60）、`statusMessage`、`once`（仅 skill frontmatter hooks）。
- command handler：`command`（必填）、`args`（存在则 exec 直起不走 shell）、`async`、`asyncRewake`（exit 2 唤醒 Claude）、`shell`（bash/powershell）。
- http handler：`url`（必填）、`headers`（值支持 `$VAR`/`${VAR}` 插值）、`allowedEnvVars`（可插值变量白名单）。
- mcp_tool handler：`server`（必填；插件服务器用 `plugin:<plugin>:<server>`）、`tool`（必填）、`input`（`${path}` 从输入 JSON 取值）。
- prompt/agent handler：`prompt`（必填，`$ARGUMENTS` 占位）、`model`（可选）。
- 路径占位符：`${CLAUDE_PROJECT_DIR}`、`${CLAUDE_PLUGIN_ROOT}`、`${CLAUDE_PLUGIN_DATA}`。
- 输入：stdin JSON（`tool_name`、`tool_input`、`session_id`、`transcript_path`、`cwd`、`permission_mode`、`hook_event_name`、`agent_id`/`agent_type` 等，随事件而异）。
- 输出：exit code（0=成功且 stdout JSON 可解析；2=阻塞错误，stderr 喂给 Claude；其他=非阻塞错误）或 HTTP 响应 JSON；`hookSpecificOutput` 携带各事件决策字段（`permissionDecision: allow/deny/ask`、`additionalContext`、`updatedInput` 等）。
- 位置：settings 各层、插件 `hooks/hooks.json`（可选顶层 `description`）、skill/subagent frontmatter（同样格式；skill hooks 注册后存活至会话结束，subagent hooks 仅其运行期；项目 subagent frontmatter hooks 需 workspace trust）。

IR 承载判定：事件名/matcher/handler 结构在 IR 中无任何对应实体；PromptPack.trigger 枚举仅 `slash-command|mention|manual|hook`，无事件载荷。判定【无承载】→ B1。

## 3. CLAUDE.md / rules / memory（Instruction 对应物）

### 3.1 文件与加载语义

| 项 | 语义 | IR 承载 |
|---|---|---|
| `~/.claude/CLAUDE.md` | user 层指令，全项目加载 | 【标准字段】Instruction（scope=global） |
| `./CLAUDE.md`、`./.claude/CLAUDE.md` | project 层 | 【标准字段】（scope=project） |
| `./CLAUDE.local.md` | local 层（gitignored） | 【标准字段】Instruction；但 IR scope 枚举无 local →【存疑】建议 origin.path 保留原名 + x- 标 `local: true`（Setting 有 local 字段，Instruction 没有）→ B14 |
| managed `CLAUDE.md` | 组织层 | 【标准字段】+ x- 标 `managed: true`【存疑】（同 B2 scope 扩展） |
| 目录向上遍历拼接 | 从文件系统根到 cwd 全部 CLAUDE.md/CLAUDE.local.md 拼接（根在前），同目录 local 在后 | 【标准字段】Instruction concat 合并语义可表达排序；"向上逐目录发现"的物化布局由适配器决定，IR 不建模 |
| 子目录 CLAUDE.md 惰性加载 | 读取子目录文件时才加载其 CLAUDE.md | 【标准字段】subtree 字段近似承载；on-demand 语义 x- 补充 |
| `.claude/rules/**/*.md`（递归发现） | 项目规则目录 | 【标准字段】多条 Instruction |
| `~/.claude/rules/*.md` | 用户规则（先于项目规则加载） | 【标准字段】 |
| rules frontmatter `paths` | glob 列表，命中才加载；支持 brace expansion（预算：1000 展开模式/4MiB）；`[` 需转义 | 【标准字段】file_patterns |
| `@path` import | 相对/绝对路径，递归最大深度 4 跳；代码 span/围栏块内不解析；反引号包裹可字面化 | 【标准字段】imports[]（path/blob/resolved）+ roundtrip_policy=preserve |
| 外部 import 批准 | 项目文件 import 解析到工作目录外时首次弹批准框 | 【无承载】→ B8（运行时信任状态） |
| 块级 HTML 注释剥离 | `<!-- -->` 注释不进上下文（代码块内除外） | 【标准字段】文档层语义，不影响采集；导出 overlay 模式可保留（IR §1.3 已覆盖） |
| `#` 快捷记忆 | 输入 `#` 开头快速追加到 memory 文件 | 【无承载】交互行为，非配置字段（不处理） |
| `` !`command` `` 注入 | skills/commands 的动态上下文注入语法；CLAUDE.md 正文不执行 | 【无承载】该语法仅对 skill/command 有效；跨工具导出时不翻译（观察项，Instruction 正文原样保留即可） |
| auto memory：`MEMORY.md` 加载上限 200 行/25KB、topic 文件、`modified` frontmatter 时间戳 | 机器本地自学习记忆 | 【无承载】→ B9（机器本地派生数据，建议不采集；如采集则 Instruction + x-） |
| subagent memory 目录 | `~/.claude/agent-memory/<name>/`、`.claude/agent-memory[-local]/<name>/` | 【无承载】→ B9 |

### 3.2 与 AGENTS.md 的互操作

官方推荐 `CLAUDE.md` 内 `@AGENTS.md` import 或 symlink；`/init` 可读 Cursor/Copilot/AGENTS.md 等并并入。→ 对 IR 的意义：imports[] 已能表达 `@AGENTS.md` 引用；跨工具"同一物理文件服务两个工具"的诉求由采集侧按 `(origin.tool, origin.path, id)` 分立条目解决（IR §2.1 reconcile 规则已覆盖）。

## 4. .mcp.json / MCP 服务器条目

### 4.1 装载位置与 scope

| scope | 存储 | 团队共享 |
|---|---|---|
| local（默认） | `~/.claude.json` 的 `projects.<path>.mcpServers` | 否 |
| project | 项目根 `.mcp.json` 的 `mcpServers` | 是 |
| user | `~/.claude.json` 顶层 `mcpServers` | 否 |
| plugin | 插件 `.mcp.json` 或 plugin.json 内联 | 随插件 |
| managed | `managed-mcp.json` | 组织 |

同名冲突优先级：local > project > user > plugin > connector；整条定义取高优先级源，**字段不跨 scope 合并**（⚠️ 与 IR merge-by-id 的 field-level-shallow 语义不同，适配器在采集多 scope 时需注意：Claude 侧语义是 entry-replace）。

### 4.2 服务器条目字段（mcpServers.<name>.*）

| 字段 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `type` | string | `stdio`/`http`/`sse`/`ws`；`streamable-http` 为 `http` 别名；无 type 且有 url = 配置错误 | stdio | 【标准字段】transport（枚举需扩 `ws` → §10 建议） |
| `command` | string | stdio 可执行文件 | 无 | 【标准字段】 |
| `args[]` | array<string> | 参数 | 无 | 【标准字段】 |
| `env` | object | 注入服务器进程的环境变量 | 无 | 【标准字段】（secretref 抽取点） |
| `url` | string | http/sse/ws 端点 | 无 | 【标准字段】 |
| `headers` | object | 静态 HTTP 头 | 无 | 【标准字段】 |
| `headersHelper` | string | 动态头生成命令（shell 执行，10s 超时，输出 JSON object；每次连接重跑；401/403 后自动重跑重试一次） | 无 | 【无承载】→ B5（McpServer 无"动态凭证命令"字段；可 x- 透传，导出他工具无语义） |
| `timeout` | number(ms) | 单工具调用墙钟超时（<1000 忽略；兼作 idle 下限） | `MCP_TOOL_TIMEOUT`（默认约 28h） | 【标准字段】timeout.tool_sec（注意 ms→s 换算） |
| `alwaysLoad` | boolean | 工具不延迟加载，启动即入上下文 | false | 【无承载】→ B6（建议 x-；工具发现策略，工具间不可译） |
| `oauth.clientId` | string | 预配置 OAuth client ID | 无 | 【x- 承载】 |
| `oauth.callbackPort` | number | 固定回调端口 | 随机 | 【x- 承载】 |
| `oauth.authServerMetadataUrl` | string(https) | 覆盖 OAuth 元数据发现 | 无 | 【x- 承载】 |
| `oauth.scopes` | string | 空格分隔 scope 限定 | 服务器决定 | 【x- 承载】 |
| （无）`env_file` | — | Claude Code 无此字段（VS Code 概念），IR 的 env_file 在 Claude 侧导出应忽略 | — | n/a |
| （无）per-server 启动超时 | — | Claude 无 per-server startup timeout 字段；仅全局 `MCP_TIMEOUT` 环境变量 | 5s 连接超时 | 【存疑】IR `timeout.startup_ms` 在 Claude 侧无逐服务器对应，只能映射全局 env（粒度损失）→ 击穿观察项 |

### 4.3 插值与占位符

- `.mcp.json`/`~/.claude.json` 服务器条目的 `command`/`args`/`env`/`url`/`headers` 支持 `${VAR}` 与 `${VAR:-default}` 展开；未设置且无默认时原样保留并告警。→ IR §3.2 已对 VS Code 插值立规（同工具往返原样保留，跨工具不翻译），Claude `${VAR:-default}` 带默认值语法需同样覆盖【标准字段 + 适配器规则】。
- 插件服务器额外占位符：`${CLAUDE_PLUGIN_ROOT}`、`${CLAUDE_PLUGIN_DATA}`、`${CLAUDE_PROJECT_DIR}`。
- 服务器名保留字：`workspace`、`claude-in-chrome`、`computer-use`、`Claude Preview`、`Claude Browser`（重名会被跳过/拒绝）。

### 4.4 审批与治理

- 项目 `.mcp.json` 服务器交互会话首次需批准；`claude mcp reset-project-choices` 重置；`enabledMcpjsonServers`/`disabledMcpjsonServers`/`enableAllProjectMcpServers` 三个 settings 键管理（见 §1.1）；未信任文件夹中，已提交到仓库的批准被忽略。
- `disabledMcpServers`/`enabledMcpServers`（`~/.claude.json` 按项目）：`/mcp` 面板的开关落点。
- 工具名形态：`mcp__<server>__<tool>`；插件服务器：`mcp__plugin_<plugin>_<server>__<tool>`。
- 远端服务器 `cached` 状态、`MCP_DISCOVERY_CACHE=0` 关闭发现缓存（v2.1.221+）。

## 5. Subagents（agents/*.md frontmatter）

文件：`~/.claude/agents/**/*.md`、`.claude/agents/**/*.md`（递归扫描；身份只看 name，不看子路径）、插件 `agents/`（子路径进入 scoped id `plugin:dir:name`）、managed 目录 `.claude/agents/`、`--agents` CLI JSON（含 `prompt` 字段，仅会话级）。

frontmatter 字段（仅 `name`、`description` 必填）：

| 字段 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `name` | string | 小写字母+连字符唯一标识；禁 `:` | 必填 | 【标准字段】PromptPack.name |
| `description` | string | 委派时机描述 | 必填 | 【标准字段】description |
| `tools` | 逗号分隔 string | 工具白名单；支持 `mcp__<server>`/`mcp__<server>__*`；`Agent(worker, researcher)` 限定可 spawn 类型 | 继承全部 | 【x- 承载】PromptPack 无 tools 字段 → §10 建议升级标准字段 |
| `disallowedTools` | 逗号分隔 string | 工具黑名单（先于 tools 应用） | 无 | 【x- 承载】同上 |
| `model` | string | `sonnet/opus/haiku/fable`/完整 ID/`inherit` | `inherit` | 【x- 承载】PromptPack 无 model 字段 → §10 建议升级标准字段 |
| `permissionMode` | string | `default/acceptEdits/auto/dontAsk/bypassPermissions/plan/manual` | 继承主会话 | 【x- 承载】 |
| `maxTurns` | number | 最大 agentic 回合数 | 无 | 【x- 承载】 |
| `skills` | list | 启动时预载的 skill 名单（全文注入） | 无 | 【x- 承载】 |
| `mcpServers` | list | 服务器名引用或内联定义（同 .mcp.json schema，含 stdio/http/sse/ws） | 无 | 【x- 承载】内联 MCP 定义嵌在 agent 内，IR McpServer 是独立实体 → 结构冲突（B11） |
| `hooks` | object | subagent 作用域生命周期 hooks（同 §2 结构；Stop→SubagentStop 转换） | 无 | 【无承载】→ B1 |
| `memory` | string | `user/project/local` 持久记忆目录 | 无 | 【x- 承载】 |
| `background` | boolean | 强制后台运行 | false | 【x- 承载】 |
| `effort` | string | `low/medium/high/xhigh/max` | 继承会话 | 【x- 承载】 |
| `isolation` | string | `worktree`=临时 git worktree 隔离 | 无 | 【x- 承载】 |
| `color` | string | 展示色 `red/blue/green/yellow/purple/orange/pink/cyan` | 无 | 【x- 承载】 |
| `initialPrompt` | string | 作为主会话 agent 运行时的首条用户消息 | 无 | 【x- 承载】 |

正文 = system prompt。插件 subagent 忽略 `hooks`/`mcpServers`/`permissionMode` 三个字段。

## 6. Skills（SKILL.md frontmatter）与 commands

位置：`~/.claude/skills/<name>/SKILL.md`、`.claude/skills/<name>/SKILL.md`（含嵌套目录惰性加载，重名时以 `apps/web:deploy` 限定名区分）、企业层、插件 `skills/`（命名空间 `plugin:name`）、`~/.claude/skills/synced/`（claude.ai 同步，`synced` 为保留目录名）。skill 目录内可含任意支撑文件（scripts/templates/examples）。`.claude/commands/*.md` 与 skills 合并（同名 skill 优先；commands 文件忽略 `name`/`paths` frontmatter）。

frontmatter 字段（全部可选；`description` 推荐；布尔接受 yes/no/on/off/1/0/true/false）：

| 字段 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `name` | string | 展示名（个人/项目 skill 的命令名取目录名；插件 skill 的 name 替换命令末段） | 目录名 | 【标准字段】name |
| `description` | string | 触发判定依据（与 when_to_use 合计截断 1536 字符） | 正文首段 | 【标准字段】description |
| `when_to_use` | string | 补充触发上下文 | 无 | 【x- 承载】可并入 description，但往返需保真 → x- |
| `argument-hint` | string | 自动补全参数提示如 `[issue-number]` | 无 | 【x- 承载】 |
| `arguments` | string \| list | 命名位置参数（`$name` 替换） | 无 | 【x- 承载】 |
| `disable-model-invocation` | boolean | 禁止 Claude 自动加载 | `false` | 【x- 承载】trigger.type=manual 近似，但保留原键 |
| `user-invocable` | boolean | false 则用户不可 `/name` 调用 | `true` | 【x- 承载】 |
| `allowed-tools` | string \| list | 该 skill 回合内免询问的工具 | 无 | 【x- 承载】 |
| `disallowed-tools` | string \| list | skill 激活期移除的工具 | 无 | 【x- 承载】 |
| `model` | string | skill 激活期模型覆盖 | 继承 | 【x- 承载】 |
| `effort` | string | effort 覆盖 | 继承 | 【x- 承载】 |
| `context` | string | `fork`=在分叉 subagent 中运行 | 无 | 【x- 承载】 |
| `agent` | string | context:fork 时用的 subagent 类型 | 无 | 【x- 承载】 |
| `background` | boolean | context:fork 时是否后台（false=等待结果） | `true` | 【x- 承载】 |
| `hooks` | object | 调用后注册的生命周期 hooks（支持 once） | 无 | 【无承载】→ B1 |
| `paths` | string \| list | 命中文件 glob 才自动加载 | 无 | 【x- 承载】IR 的 file_patterns 在 Instruction 上，PromptPack 无 → §10 建议 |
| `shell` | string | 内联命令用 `bash`/`powershell` | bash | 【x- 承载】 |
| `metadata` | map | 自由键值（Claude Code 不消费） | 无 | 【x- 承载】 |
| `license` | string | 许可证（Agent Skills 规范字段） | 无 | 【x- 承载】 |
| `compatibility` | string(≤500) | 环境要求（Agent Skills 规范字段） | 无 | 【x- 承载】 |

正文动态特性：`` !`command` `` 与 ` ```! ` 块（调用时执行注入）、`$ARGUMENTS`/`$ARGUMENTS[N]`/`$N`/`$name`/`${CLAUDE_SESSION_ID}`/`${CLAUDE_EFFORT}` 替换、`@` 文件引用。commands 文件支持同样 frontmatter（除 name/paths）。

## 7. Output styles 与 statusLine

Output style 文件（`~/.claude/output-styles/*.md`、`.claude/output-styles/*.md`、managed、插件 output-styles/）frontmatter：

| 字段 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `name` | string | 风格名 | 文件名 | 【x- 承载】IR 无 OutputStyle 实体；可用 PromptPack 近似或 Setting.outputStyle 仅存引用名 → B12 |
| `description` | string | 选择器展示 | 无 | 【x- 承载】 |
| `keep-coding-instructions` | boolean | 保留内置软件工程系统提示 | `false` | 【x- 承载】 |
| `force-for-plugin` | boolean | 插件启用即强制该风格 | `false` | 【x- 承载】 |

statusLine 设置对象（§1.1 已列）：`type:"command"`、`command`、`padding`（默认 0）、`refreshInterval`（秒，≥1）、`hideVimModeIndicator`。脚本经 stdin 收 JSON（model/workspace/cost/context_window/rate_limits/session_id 等字段），stdout 即展示内容；`subagentStatusLine` 同构。脚本本体是独立文件资产 → IR 可用 PromptPack assets 或 Setting.value 内联路径，脚本文件本体【无承载】→ B12（需 assets 机制挂接）。

## 8. Plugins 体系（实体级缺口）

- 插件 = 可分发包，内含 skills/agents/hooks/MCP servers/output-styles；清单元件 `.claude-plugin/plugin.json`（本次未逐字段抓 plugins-reference 页。【存疑】plugin.json 完整键表：已知含 `name`、`mcpServers`、`hooks`、userConfig、defaultEnabled、relevance 等；核实方法：抓 https://docs.anthropic.com/en/docs/claude-code/plugins-reference）。
- marketplace.json：市场清单（source 类型见 §1.6）。
- settings 侧的 `enabledPlugins`/`extraKnownMarketplaces`/`pluginConfigs`/`strictKnownMarketplaces` 见 §1.1/§1.6。

IR 承载：【无承载】→ B3（IR 无 Plugin/Marketplace 实体；MVP 建议不采集插件本体，仅 x- 透传 settings 中的开关状态）。

## 9. IR 击穿清单（无承载字段汇总 + 处理建议）

| # | 字段/结构 | 类别 | 建议处理 |
|---|---|---|---|
| B1 | `hooks` 全结构（31 事件 × matcher × 5 种 handler × if/timeout/async/shell 等；含 settings、插件 hooks.json、skill/agent frontmatter 三种载体） | 实体缺口 | 短期：`setting.claude-code.hooks` 不透明 value + x-；长期：IR 增加 `hook.` 前缀标准实体（事件名标准化层 + 工具特有事件进 x-） |
| B2 | managed 策略键家族（`allowManagedHooksOnly`、`allowedMcpServers`、`deniedMcpServers`、`allowManagedMcpServersOnly`、`allowManagedPermissionRulesOnly`、`strictKnownMarketplaces`、`blockedMarketplaces`、`requiredMinimumVersion`、`requiredMaximumVersion`、`forceLoginGatewayUrl`、`policyHelper`、`parentSettingsBehavior`、`wslInheritsWindowsSettings`、`enforceAvailableModels`、`allowAllClaudeAiMcps`、`allowedChannelPlugins`、`channelsEnabled`、`disableCommandPluginSources`、`disableSideloadFlags`、`disableMobileSimulatorTools`、`disableBrowserExternalNavigation`、`browserExternalPageTools`、`pluginSuggestionMarketplaces`、`pluginTrustMessage`、`forceRemoteSettingsRefresh`、sandbox 的 `allowManaged*`/`bwrapPath`/`socatPath` 等） | scope 维度缺口 | IR `origin.scope` 增加 `managed` 枚举值；这些键以 Setting + scope=managed + x- 承载，sync 默认排除 |
| B3 | 插件体系（`enabledPlugins`、`pluginConfigs`、`extraKnownMarketplaces`、plugin.json、marketplace.json） | 实体缺口 | MVP 不建模 Plugin 实体；settings 键 x- 透传；插件落地的 servers/skills/agents 以其落地形态正常采集 |
| B4 | `sandbox.credentials.*`（files/envVars 的 mask/extract/decode/maskClaims/injectHosts、awsPairs、sigv4） | 深度结构 | x- 透传；IR 不标准化凭证遮蔽策略（工具间差异过大） |
| B5 | `.mcp.json` 服务器 `headersHelper`（动态头命令） | McpServer 字段缺口 | McpServer 增加可选 `headers_helper` 标准字段候选（Codex 有 `bearer_token_env_var`/`env_http_headers` 间接对应）或 x- 透传 |
| B6 | `.mcp.json` 服务器 `alwaysLoad`（工具即时加载） | McpServer 字段缺口 | x- 透传（工具发现策略，工具间不可译） |
| B7 | `claudeMd`（managed settings 内联 CLAUDE.md） | 形态冲突 | 仅存 Setting x-；不升格为 Instruction |
| B8 | `~/.claude.json` 运行时状态（OAuth、`projects.*` 信任/批准、caches）；外部 import 批准状态 | 非配置数据 | 明确不采集（凭证走 keyring；信任状态不回迁） |
| B9 | auto memory / agent memory 目录（MEMORY.md、topic 文件） | 派生数据 | 默认不采集；提供 opt-in 开关按 Instruction 采集 |
| B10 | `ultracode` 等非文件持久化键 | 无源 | 不处理 |
| B11 | agent frontmatter 内联 `mcpServers` | 结构冲突 | x- 透传整段；不拆成 McpServer 实体（生命周期绑定 agent） |
| B12 | output style 文件、statusline 脚本等独立文件资产 | 资产挂接 | 复用 PromptPack assets 机制；`outputStyle`/`statusLine` 设置键 x- 透传 |
| B13 | IR `transport` 枚举缺 `ws`；`timeout.startup_ms` 在 Claude 无 per-server 对应 | schema 微调 | 见 §11 修改建议 |
| B14 | `CLAUDE.local.md` 的 local scope（Instruction 无 local 标记） | scope 缺口 | 同 B2 一并解决（scope 枚举扩展或实体级 local 布尔位） |

## 10. 真实样本

1. **官方 settings.json 示例**（来源：https://docs.anthropic.com/en/docs/claude-code/settings 页内示例）：
```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "permissions": {
    "allow": ["Bash(npm run lint)", "Bash(npm run test *)", "Read(~/.zshrc)"],
    "deny": ["Bash(curl *)", "Read(./.env)", "Read(./.env.*)", "Read(./secrets/**)"]
  },
  "env": { "CLAUDE_CODE_ENABLE_TELEMETRY": "1", "OTEL_METRICS_EXPORTER": "otlp" },
  "companyAnnouncements": ["Welcome to Acme Corp! ..."]
}
```
另注：官方 JSON Schema 发布于 https://json.schemastore.org/claude-code-settings.json （编辑器自动补全用，字段略滞后于文档）。

2. **官方 .mcp.json 示例**（来源：https://docs.anthropic.com/en/docs/claude-code/mcp）：
```json
{
  "mcpServers": {
    "api-server": {
      "type": "http",
      "url": "${API_BASE_URL:-https://api.example.com}/mcp",
      "headers": { "Authorization": "Bearer ${API_KEY}" }
    }
  }
}
```
以及 `oauth: {"clientId": "...", "callbackPort": 8080}`、`headersHelper` 示例（见该页 "Use dynamic headers" 小节）。

3. **官方 subagent 示例**（来源：https://docs.anthropic.com/en/docs/claude-code/sub-agents）：
```markdown
---
name: code-reviewer
description: Reviews code for quality and best practices
tools: Read, Glob, Grep
model: sonnet
---
You are a code reviewer. ...
```
以及内联 mcpServers 示例（`browser-tester`：内联 playwright stdio 定义 + 引用 github）。

4. **官方 SKILL.md 示例**（来源：https://code.claude.com/docs/en/skills）：
```yaml
---
description: Summarizes uncommitted changes and flags anything risky. ...
---
## Current changes
!`git diff HEAD`
```
展示 `` !`...` `` 动态注入语法；另有 `context: fork` + `disable-model-invocation: true` 的 deploy 示例。

5. **官方 hooks 示例**（来源：https://docs.anthropic.com/en/docs/claude-code/hooks）：
```json
{
  "hooks": {
    "PreToolUse": [
      { "matcher": "Bash",
        "hooks": [
          { "type": "command", "if": "Bash(rm *)",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/block-rm.sh", "args": [] }
        ] }
    ]
  }
}
```
同页还有 http hook（url/headers/allowedEnvVars）与 mcp_tool hook（server/tool/input）完整示例。

6. **官方 statusline 脚本示例**（来源：https://docs.anthropic.com/en/docs/claude-code/statusline）：jq 解析 stdin JSON 的多语言（Bash/Python/Node）脚本集合，含 context 进度条、git 状态、成本、OSC 8 链接。

7. **官方 MDM 部署模板**（settings 页引用）：https://github.com/anthropics/claude-code/tree/main/examples/mdm —— Jamf/Iru(Kandji)/Intune/Group Policy 的 managed settings 起始模板。

8. **社区聚合仓库**：https://github.com/hesreallyhim/awesome-claude-code —— 社区 CLAUDE.md / commands / hooks / statusline 样本索引（【存疑】具体文件级样本建议从该仓 awesome 列表二次选取，如 wong2 的 claude-code-config 类 dotfiles 仓；核实方法：按列表链接逐个确认存活）。

## 11. 证伪回答：若 IR 只服务 Claude Code + Codex CLI，v0.2 需改什么（Claude 侧输入）

1. **scope 枚举扩展**：`origin.scope` 当前 `global|project`，需加 `local`（settings.local.json、CLAUDE.local.md、.claude/settings.local.json 形态）与 `managed`（managed-settings 家族）。Setting 已有 `local` 布尔位，Instruction/PromptPack/McpServer 均无。
2. **新增 Hook 实体（或明确长期降级策略）**：Claude 31 事件与 Codex 11 事件高度重叠（PreToolUse/PostToolUse/SessionStart/SessionEnd/SubagentStart/SubagentStop/UserPromptSubmit/Stop/PreCompact/PostCompact/PermissionRequest），值得 `hook.` 前缀标准实体（event/matcher/handlers[]），工具特有事件（Claude 的 Notification/InstructionsLoaded/ConfigChange 等）与 handler 类型（http/mcp_tool/prompt/agent）进 x-。
3. **McpServer.transport 枚举加 `ws`**；`streamable-http` 作为 `http` 别名在适配器归一化（Claude 接受别名写法）。
4. **McpServer 增加 `headers_helper`（标准字段候选）**；`always_load` 走 x-；`timeout.startup_ms` 标注"Claude 无 per-server 对应，导出忽略或映射全局 env `MCP_TIMEOUT`"。
5. **PromptPack 增加 `tools`/`disallowedTools`/`model` 可选标准字段**：Claude subagent（tools/disallowedTools/model）、skill（allowed-tools/disallowed-tools/model）与 Codex agent role（config_file 内同概念）、apps/mcp per-tool approval 都共享"工具集+模型"维度，属可标准化公共语义。
6. **PromptPack 增加 `file_patterns`（paths）**：skill `paths` frontmatter 与 Instruction.file_patterns 同构，当前 PromptPack 无此字段。
7. **合并语义冲突记录**：Claude settings 数组键跨 scope 拼接去重、MCP 同名条目 entry-replace；IR merge-by-id 是"数组整体替换"。适配器须在采集 Claude 多 scope settings 时按 Claude 语义预合并（或分条存储并在导出时重放 scope 语义），ADAPTERS 需写明。
8. **`~/.claude.json` 部分拥有权**：IR §3.4 的回写路由 (b) 已覆盖原则，但需在该文件上落实"局部 patch"适配器契约（含 projects.<path> 子树的条目级定位）。
