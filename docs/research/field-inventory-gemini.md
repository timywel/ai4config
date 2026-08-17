# 字段清单：Gemini CLI（adapter id: `gemini`）

> 调研日期：2026-08-16 ｜ 调研人：P1 工具组（字段级调研）
> 基线文档：
> - [configuration reference](https://www.geminicli.com/docs/reference/configuration/)（页面快照 "Last updated: Jul 17, 2026"）
> - [GEMINI.md](https://www.geminicli.com/docs/cli/gemini-md/)（"Last updated: Jun 18, 2026"）
>
> ⚠️ 时效警告：官方公告 unpaid/Google One 层 Gemini CLI 将于 2026-06-18 起过渡为 Antigravity CLI（本机 `~/.gemini/` 下已实证存在 `antigravity/` 目录）。
> 承载状态图例：【标准字段】/【x- 承载】（`x-gemini`）/【无承载】。
> settings.json 为 JSON；字符串值支持 `$VAR_NAME`、`${VAR_NAME}`、`${VAR_NAME:-DEFAULT}` 环境变量插值（采集时原样保留，跨工具不翻译 + Warning，同 IR-SCHEMA §3.2 插值规则）。

## 0. 配置文件地图与层级

| 文件 | 层 | scope | IR 实体 |
|---|---|---|---|
| `/etc/gemini-cli/system-defaults.json`（Win: `C:\ProgramData\gemini-cli\system-defaults.json`，macOS: `/Library/Application Support/GeminiCli/system-defaults.json`） | L2 系统默认 | **managed** | Setting（【无承载】IR scope 仅 global\|project，见击穿清单 #2） |
| `~/.gemini/settings.json` | L3 用户 | global | Setting / McpServer（`mcpServers` 键） |
| `<proj>/.gemini/settings.json` | L4 项目 | project | Setting / McpServer |
| `/etc/gemini-cli/settings.json`（Win/macOS 同型） | L5 系统覆盖（最高优先） | **managed** | Setting（【无承载】同 #2） |
| `~/.gemini/GEMINI.md` | — | global | Instruction |
| `<proj>/GEMINI.md`（cwd 及祖先目录向上搜索拼接 + JIT 子目录） | — | project | Instruction（`subtree` 承载子目录条目） |
| `<proj>/.gemini/sandbox-macos-<name>.sb`、`.gemini/sandbox.Dockerfile` | — | project | 【无承载】沙箱定义文件，非 settings（可作 PromptPack assets 之外的二进制资产，建议不采集） |
| `~/.gemini/tmp/<project_hash>/shell_history` | — | 运行时 | 【无承载】不采集 |

环境变量层（L6）与 CLI 参数层（L7）不入库；其中 `GEMINI_API_KEY`、`GOOGLE_API_KEY` 等属敏感值，**只抽 secretref 引用场景，不采集值**。

## 1. `settings.json` 顶级键总览（v0.3.0+ 嵌套结构，共 24 个顶级键 / 约 236 个叶子键）

`policyPaths`、`adminPolicyPaths`、`general`、`output`、`ui`、`ide`、`privacy`、`billing`、`model`、`modelConfigs`、`agents`、`context`、`tools`、`mcp`、`useWriteTodos`、`security`、`advanced`、`experimental`、`skills`、`hooksConfig`、`hooks`、`contextManagement`、`admin`、`mcpServers`、`telemetry`。

除 `mcpServers`（→ McpServer 实体）与 `hooks`（语义近独立实体）外，其余整体映射为 Setting 条目：`setting.gemini.<key-path>`，`value` 为不透明嵌套值。统一击穿：dotted key-path vs 三段式 id（同 copilot 击穿，见 IR 击穿清单 #1）。下表"IR 承载"列不再逐条重复该说明。

## 2. 逐分类字段

### 2.1 `policyPaths` / `adminPolicyPaths`

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `policyPaths` | array | 追加加载的策略文件/目录；需重启 | 【标准字段】 |
| `adminPolicyPaths` | array | 追加加载的管理员策略文件/目录；需重启 | 【标准字段】managed 层语义，见击穿 #2 |

### 2.2 `general`（22 键）

| 字段路径 | 类型 | 语义 / 缺省 | IR 承载 |
|---|---|---|---|
| `general.preferredEditor` | enum | 16 个内置编辑器标识（`vscode`…`emacsclient`）；缺省用 `$VISUAL/$EDITOR` | 【标准字段】 |
| `general.openEditorInNewWindow` | boolean | VS Code 系编辑器新窗口打开；`false` | 【标准字段】 |
| `general.vimMode` | boolean | Vim 键位；`false` | 【标准字段】 |
| `general.defaultApprovalMode` | enum | `default`/`auto_edit`/`plan`（`yolo` 仅 CLI）；`"default"` | 【标准字段】权限语义，跨工具不可翻译，value 透传 |
| `general.devtools` | boolean | 启动 DevTools；`false` | 【标准字段】 |
| `general.enableAutoUpdate` | boolean | 自动更新；`true` | 【标准字段】 |
| `general.enableAutoUpdateNotification` | boolean | 更新提示；`true` | 【标准字段】 |
| `general.enableNotifications` | boolean | 终端通知；`false` | 【标准字段】 |
| `general.notificationMethod` | enum | `auto`/`osc9`/`osc777`/`bell` | 【标准字段】 |
| `general.checkpointing.enabled` | boolean | 会话检查点；`false`，需重启 | 【标准字段】 |
| `general.plan.enabled` | boolean | Plan Mode；`true`，需重启 | 【标准字段】 |
| `general.plan.directory` | string | 规划产物目录；缺省系统临时目录 | 【标准字段】 |
| `general.plan.modelRouting` | boolean | 规划/实施自动 Pro/Flash 切换；`true` | 【标准字段】 |
| `general.retryFetchErrors` | boolean | fetch failed 重试；`true` | 【标准字段】 |
| `general.maxAttempts` | number | 主模型请求最大尝试，≤10；`10` | 【标准字段】 |
| `general.debugKeystrokeLogging` | boolean | 按键调试日志；`false` | 【标准字段】 |
| `general.sessionRetention.enabled` | boolean | 会话自动清理；`true` | 【标准字段】 |
| `general.sessionRetention.maxAge` | string | 保留期（`30d`/`24h`…）；`"30d"` | 【标准字段】 |
| `general.sessionRetention.maxCount` | number | 保留会话数上限 | 【标准字段】 |
| `general.sessionRetention.minRetention` | string | 最短保留（安全下限）；`"1d"` | 【标准字段】 |
| `general.topicUpdateNarration` | boolean | Topic & Update 沟通模型；`true` | 【标准字段】 |
| `general.logRagSnippets` | boolean | RAG 片段落盘调试；`false` | 【标准字段】 |

### 2.3 `output`（1 键）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `output.format` | enum | `text`/`json`；`"text"` | 【标准字段】 |

### 2.4 `ui`（41 键，节选式全列）

| 字段路径 | 类型 | 语义 / 缺省 | IR 承载 |
|---|---|---|---|
| `ui.theme` | string | 主题名 | 【标准字段】 |
| `ui.customThemes` | object | 自定义主题定义；`{}` | 【标准字段】 |
| `ui.autoThemeSwitching` / `ui.terminalBackgroundPollingInterval` | boolean / number | 依终端背景自动换主题；`true` / `60` | 【标准字段】 |
| `ui.debugRainbow` / `ui.renderProcess` / `ui.terminalBuffer` / `ui.useAlternateBuffer` / `ui.useBackgroundColor` / `ui.incrementalRendering` | boolean | 渲染管线开关组（多需重启） | 【标准字段】 |
| `ui.hideWindowTitle` / `ui.hideBanner` / `ui.hideTips` / `ui.hideContextSummary` / `ui.hideFooter` / `ui.showShortcutsHint` | boolean | 界面元素显隐组 | 【标准字段】 |
| `ui.inlineThinkingMode` | enum | `off`/`full` | 【标准字段】 |
| `ui.showStatusInTitle` / `ui.dynamicWindowTitle` | boolean | 窗口标题状态；`false`/`true` | 【标准字段】 |
| `ui.showHomeDirectoryWarning` / `ui.showCompatibilityWarnings` | boolean | 警告类；均 `true`，需重启 | 【标准字段】 |
| `ui.escapePastedAtSymbols` | boolean | 粘贴时转义 `@` 防 @path 展开；`false` | 【标准字段】 |
| `ui.compactToolOutput` | boolean | 工具输出紧凑格式；`true` | 【标准字段】 |
| `ui.footer.items` / `.showLabels` / `.hideCWD` / `.hideSandboxStatus` / `.hideModelInfo` / `.hideContextPercentage` | array / boolean×5 | footer 定制组 | 【标准字段】 |
| `ui.collapseDrawerDuringApproval` | boolean | 审批时收起抽屉；`true` | 【标准字段】 |
| `ui.showMemoryUsage` / `ui.showLineNumbers` / `ui.showCitations` / `ui.showModelInfoInChat` / `ui.showUserIdentity` / `ui.showSpinner` | boolean | 信息显示组 | 【标准字段】 |
| `ui.loadingPhrases` | enum | `tips`/`witty`/`all`/`off`；`"off"` | 【标准字段】 |
| `ui.customWittyPhrases` | array | 自定义加载语；`[]` | 【标准字段】 |
| `ui.errorVerbosity` | enum | `low`/`full`；`"low"` | 【标准字段】 |
| `ui.accessibility.enableLoadingPhrases` | boolean | **@deprecated** → `ui.loadingPhrases` | 【标准字段】deprecated 标记进 value 注释/Warning |
| `ui.accessibility.screenReader` | boolean | 屏幕阅读器纯文本；`false`，需重启 | 【标准字段】 |

### 2.5 `ide` / `privacy` / `billing`（6 键）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `ide.enabled` | boolean | IDE 集成模式；`false`，需重启 | 【标准字段】 |
| `ide.hasSeenNudge` | boolean | 已见集成提示（**运行时状态混入**） | 【标准字段】建议采集跳过或标注运行时（击穿 #5） |
| `privacy.usageStatisticsEnabled` | boolean | 使用统计；`true`，需重启 | 【标准字段】 |
| `billing.overageStrategy` | enum | `ask`/`always`/`never` | 【标准字段】 |
| `billing.vertexAi.requestType` | enum | `dedicated`/`shared`（X-Vertex-AI-LLM-Request-Type 头） | 【标准字段】 |
| `billing.vertexAi.sharedRequestType` | enum | `priority`/`flex` | 【标准字段】 |

### 2.6 `model` / `modelConfigs`（14 键）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `model.name` | string | 会话模型 | 【标准字段】（对照 copilot agent `model`，提升 PromptPack 字段时对齐） |
| `model.maxSessionTurns` | number | 会话轮次上限，-1 无限 | 【标准字段】 |
| `model.summarizeToolOutput` | object | 按工具 token 预算（如 `{"run_shell_command":{"tokenBudget":2000}}`） | 【标准字段】 |
| `model.compressionThreshold` | number | 上下文压缩触发比例；`0.5` | 【标准字段】 |
| `model.disableLoopDetection` | boolean | 关循环检测 | 【标准字段】 |
| `model.skipNextSpeakerCheck` | boolean | 跳过下一发言者检查；`true` | 【标准字段】 |
| `modelConfigs.aliases` | object | 命名模型预设，`extends` 继承链（内建 40+ 别名默认值巨大） | 【标准字段】**内建默认值不采集**（击穿 #4） |
| `modelConfigs.customAliases` | object | 用户自定义别名（合并覆盖内建） | 【标准字段】采集重点 |
| `modelConfigs.customOverrides` | array | 用户自定义覆盖 | 【标准字段】采集重点 |
| `modelConfigs.overrides` | array | 按 model/alias 主键匹配的条件覆盖 | 【标准字段】 |
| `modelConfigs.modelDefinitions` | object | 模型元数据注册表（tier/family/features） | 【标准字段】内建默认值不采集 |
| `modelConfigs.modelIdResolutions` | object | 模型名→具体 ID 的上下文解析规则 | 【标准字段】同上 |
| `modelConfigs.classifierIdResolutions` | object | 分类器层级→模型 ID 解析 | 【标准字段】同上 |
| `modelConfigs.modelChains` | object | 可用性回退链 | 【标准字段】同上 |

### 2.7 `agents`（10 键）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `agents.overrides` | object | 按 agent 覆盖（禁用/自定义 model config/run config）；`{}` | 【标准字段】 |
| `agents.browser.sessionMode` | enum | `persistent`/`isolated`/`existing` | 【标准字段】 |
| `agents.browser.headless` | boolean | 无头浏览器 | 【标准字段】 |
| `agents.browser.profilePath` | string | 浏览器 profile 目录（**机器相关路径** → 联动 IR `per_machine` 思路，采集时标注） | 【标准字段】 |
| `agents.browser.visualModel` | string | 视觉分析模型（设置即启用 analyze_screenshot） | 【标准字段】 |
| `agents.browser.allowedDomains` | array | 浏览器 agent 域名白名单（支持通配） | 【标准字段】 |
| `agents.browser.disableUserInput` | boolean | 自动化期间禁用户输入；`true` | 【标准字段】 |
| `agents.browser.maxActionsPerTask` | number | 单任务工具调用硬上限；`100` | 【标准字段】 |
| `agents.browser.confirmSensitiveActions` | boolean | 敏感动作需确认 | 【标准字段】 |
| `agents.browser.blockFileUploads` | boolean | 硬阻断上传 | 【标准字段】 |

### 2.8 `context`（13 键）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `context.fileName` | string \| string[] | 上下文文件名（默认 `GEMINI.md`；可 `["AGENTS.md","CONTEXT.md","GEMINI.md"]`） | 【标准字段】**导出布局关键键**：决定 Instruction 物化文件名 |
| `context.importFormat` | string | memory 导入格式 | 【标准字段】 |
| `context.includeDirectoryTree` | boolean | 首请求附目录树；`true` | 【标准字段】 |
| `context.discoveryMaxDirs` | number | memory 搜索目录上限；`200` | 【标准字段】 |
| `context.memoryBoundaryMarkers` | array | 向上遍历终止标记；`[".git"]`，空数组禁父遍历 | 【标准字段】 |
| `context.includeDirectories` | array | 附加 workspace 目录（缺失告警跳过） | 【标准字段】 |
| `context.loadMemoryFromIncludeDirectories` | boolean | `/memory reload` 是否扫描 include 目录 | 【标准字段】 |
| `context.fileFiltering.respectGitIgnore` | boolean | 遵守 .gitignore；`true` | 【标准字段】 |
| `context.fileFiltering.respectGeminiIgnore` | boolean | 遵守 .geminiignore；`true` | 【标准字段】 |
| `context.fileFiltering.enableFileWatcher` | boolean | @ 补全文件监听（实验） | 【标准字段】 |
| `context.fileFiltering.enableRecursiveFileSearch` | boolean | @ 引用递归搜索；`true` | 【标准字段】 |
| `context.fileFiltering.enableFuzzySearch` | boolean | 模糊搜索；`true` | 【标准字段】 |
| `context.fileFiltering.customIgnoreFilePaths` | array | 追加 ignore 文件（数组序=优先级） | 【标准字段】 |

### 2.9 `tools`（18 键）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `tools.sandbox` | string\|boolean | 沙箱环境（`docker`/`podman`/`lxc`/`windows-native` 或 profile 路径） | 【标准字段】 |
| `tools.sandboxAllowedPaths` | array | 沙箱可访问附加路径 | 【标准字段】 |
| `tools.sandboxNetworkAccess` | boolean | 沙箱联网；`false` | 【标准字段】 |
| `tools.shell.enableInteractiveShell` | boolean | node-pty 交互 shell；`true` | 【标准字段】 |
| `tools.shell.backgroundCompletionBehavior` | enum | `silent`/`inject`/`notify` | 【标准字段】 |
| `tools.shell.pager` | string | 分页命令；`cat` | 【标准字段】 |
| `tools.shell.showColor` | boolean | shell 输出着色；`true` | 【标准字段】 |
| `tools.shell.inactivityTimeout` | number | 无输出超时秒；`300` | 【标准字段】 |
| `tools.shell.enableShellOutputEfficiency` | boolean | 输出效率优化；`true` | 【标准字段】 |
| `tools.core` | array | 内建工具白名单（限制全集） | 【标准字段】权限语义，跨工具不翻译 |
| `tools.allowed` | array | 免确认工具（如 `["run_shell_command(git)"]`） | 【标准字段】对照 zhanlu `permission.bash`、claude `permissions.allow`——**权限三元组映射候选**，见击穿 #3 |
| `tools.confirmationRequired` | array | 强制确认清单（优先于 allowed/core） | 【标准字段】同上 |
| `tools.exclude` | array | 发现排除清单 | 【标准字段】 |
| `tools.discoveryCommand` | string | 工具发现命令 | 【标准字段】 |
| `tools.callCommand` | string | 工具调用命令（stdin JSON 协议） | 【标准字段】 |
| `tools.useRipgrep` | boolean | ripgrep 搜索；`true` | 【标准字段】 |
| `tools.truncateToolOutputThreshold` | number | 输出截断字符数；`40000` | 【标准字段】 |
| `tools.disableLLMCorrection` | boolean | 关编辑工具 LLM 自纠错；`true` | 【标准字段】 |

### 2.10 `mcp` / `useWriteTodos`（4 键）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `mcp.serverCommand` | string | 启动 MCP server 的命令 | 【标准字段】 |
| `mcp.allowed` | array | MCP server 白名单 | 【标准字段】实体级过滤，非 McpServer 字段 |
| `mcp.excluded` | array | MCP server 黑名单 | 【标准字段】 |
| `useWriteTodos` | boolean | 启用 write_todos 工具；`true` | 【标准字段】 |

### 2.11 `security`（15 键）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `security.toolSandboxing` | boolean | 工具级沙箱（隔离单工具） | 【标准字段】 |
| `security.disableYoloMode` | boolean | 禁 YOLO（即使 CLI flag） | 【标准字段】 |
| `security.disableAlwaysAllow` | boolean | 禁"Always allow"选项 | 【标准字段】 |
| `security.enablePermanentToolApproval` | boolean | 启用"未来会话均允许" | 【标准字段】 |
| `security.autoAddToPolicyByDefault` | boolean | 低风险工具默认入策略（受信 workspace） | 【标准字段】 |
| `security.blockGitExtensions` | boolean | 阻断 Git 来源扩展 | 【标准字段】 |
| `security.allowedExtensions` | array | 扩展 Regex 白名单（覆盖 blockGitExtensions） | 【标准字段】 |
| `security.folderTrust.enabled` | boolean | 文件夹信任；`true` | 【标准字段】 |
| `security.environmentVariableRedaction.enabled` | boolean | 环境变量脱敏；`false` | 【标准字段】与 cfg4ai 敏感扫描（IR-SCHEMA §5.4）语义联动 |
| `security.environmentVariableRedaction.allowed` | array | 免脱敏变量 | 【标准字段】 |
| `security.environmentVariableRedaction.blocked` | array | 强制脱敏变量 | 【标准字段】 |
| `security.auth.selectedType` | string | 当前认证类型 | 【标准字段】**认证状态混入**，见击穿 #5 |
| `security.auth.enforcedType` | string | 强制认证类型（不匹配则重新认证） | 【标准字段】managed 语义 |
| `security.auth.useExternal` | boolean | 外部认证流 | 【标准字段】 |
| `security.enableConseca` | boolean | 上下文感知安全检查器（LLM 动态策略） | 【标准字段】 |

（另：文档"Environment variable redaction"小节出现旧键名 `security.allowedEnvironmentVariables`/`blockedEnvironmentVariables`，与正文 `environmentVariableRedaction.allowed/blocked` 不一致——按正文新键名为准，旧键名采集时进 `x-gemini` 并记 Warning。）

### 2.12 `advanced`（5 键）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `advanced.autoConfigureMemory` | boolean | Node 内存自动配置（**仅全局层生效**，项目层忽略） | 【标准字段】层级限制进元数据注释 |
| `advanced.dnsResolutionOrder` | string | DNS 解析顺序 | 【标准字段】 |
| `advanced.excludedEnvVars` | array | 项目上下文排除变量；`["DEBUG","DEBUG_MODE"]` | 【标准字段】 |
| `advanced.ignoreLocalEnv` | boolean | 忽略项目 .env | 【标准字段】 |
| `advanced.bugCommand` | object | /bug 命令配置 | 【标准字段】 |

### 2.13 `experimental`（33 键，分组全列）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `experimental.gemma` | boolean | Gemma 4 模型访问；`true` | 【标准字段】 |
| `experimental.voiceMode` + `voice.activationMode`/`backend`/`whisperModel`/`stopGracePeriodMs` | boolean/enum×3/number | 语音输入组 | 【标准字段】 |
| `experimental.adk.agentSessionNoninteractiveEnabled`/`InteractiveEnabled`/`SubagentEnabled` | boolean×3 | ADK 会话实现组 | 【标准字段】 |
| `experimental.enableAgents` | boolean | 本地/远程 subagent；`true` | 【标准字段】 |
| `experimental.worktrees` | boolean | Git worktree 并行管理 | 【标准字段】 |
| `experimental.extensionManagement`/`extensionConfig`/`extensionRegistry`/`extensionRegistryURI`/`extensionReloading` | boolean×4/string | 扩展体系组 | 【标准字段】 |
| `experimental.useOSC52Paste`/`useOSC52Copy` | boolean×2 | OSC52 剪贴板 | 【标准字段】 |
| `experimental.taskTracker` | boolean | 任务跟踪工具 | 【标准字段】 |
| `experimental.modelSteering` | boolean | 工具执行中模型引导 | 【标准字段】 |
| `experimental.directWebFetch` | boolean | 绕过 LLM 摘要的抓取 | 【标准字段】 |
| `experimental.dynamicModelConfiguration` | boolean | settings 驱动模型配置 | 【标准字段】 |
| `experimental.gemmaModelRouter.enabled`/`autoStartServer`/`binaryPath`/`classifier.host`/`classifier.model` | boolean×2/string×3 | 本地 Gemma 路由组（binaryPath 机器相关） | 【标准字段】 |
| `experimental.stressTestProfile` | boolean | 压测 profile | 【标准字段】 |
| `experimental.autoMemory` | boolean | 后台自动抽取 memory/skill（.patch 入 inbox 待审） | 【标准字段】 |
| `experimental.generalistProfile`/`powerUserProfile` | boolean×2 | 行为 profile | 【标准字段】 |
| `experimental.contextManagement` | boolean | 上下文管理总开关 | 【标准字段】 |
| `experimental.topicUpdateNarration` | boolean | **Deprecated** → `general.topicUpdateNarration` | 【标准字段】deprecated Warning |

### 2.14 `skills` / `hooksConfig` / `hooks`（16 键）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `skills.enabled` | boolean | Agent Skills 总开关；`true` | 【标准字段】 |
| `skills.disabled` | array | 禁用 skill 名单 | 【标准字段】实体级启停 |
| `hooksConfig.enabled` | boolean | hooks 系统总开关；`true` | 【标准字段】 |
| `hooksConfig.disabled` | array | 按名禁用 hook | 【标准字段】 |
| `hooksConfig.notifications` | boolean | hook 执行可视提示 | 【标准字段】 |
| `hooks.BeforeTool` / `AfterTool` / `BeforeAgent` / `AfterAgent` / `Notification` / `SessionStart` / `SessionEnd` / `PreCompress` / `BeforeModel` / `AfterModel` / `BeforeToolSelection` | array×11 | 11 个事件点的 hook 定义数组（结构见 hooks reference） | 【x- 承载】**IR 无 hook 实体**；作为 Setting 不透明 value 可保真，但跨工具无翻译路径（击穿 #6） |

### 2.15 `contextManagement`（10 键）

| 字段路径 | 类型 | 语义 / 缺省 | IR 承载 |
|---|---|---|---|
| `contextManagement.historyWindow.maxTokens` / `.retainedTokens` | number | 压缩触发/保留 token；`150000`/`40000` | 【标准字段】 |
| `contextManagement.messageLimits.normalMaxTokens` / `.retainedMaxTokens` / `.normalizationHeadRatio` | number | 单轮预算/截断上限/头部保留比；`2500`/`12000`/`0.25` | 【标准字段】 |
| `contextManagement.tools.distillation.maxOutputTokens` / `.summarizationThresholdTokens` | number | 工具输出蒸馏；`10000`/`20000` | 【标准字段】 |
| `contextManagement.tools.outputMasking.protectionThresholdTokens` / `.minPrunableThresholdTokens` / `.protectLatestTurn` | number×2/boolean | 输出遮蔽组；`50000`/`30000`/`true` | 【标准字段】 |

### 2.16 `admin`（6 键，企业管控）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `admin.secureModeEnabled` | boolean | 禁 YOLO 与"Always allow" | 【标准字段】managed 语义（击穿 #2） |
| `admin.extensions.enabled` | boolean | 禁扩展安装使用；`true` | 【标准字段】 |
| `admin.mcp.enabled` | boolean | 禁 MCP；`true` | 【标准字段】 |
| `admin.mcp.config` | object | 管理员 MCP 白名单 | 【标准字段】 |
| `admin.mcp.requiredConfig` | object | 管理员强制注入的 MCP | 【标准字段】 |
| `admin.skills.enabled` | boolean | 禁 skills；`true` | 【标准字段】 |

### 2.17 `mcpServers.<SERVER_NAME>`（12 字段 → McpServer 实体）

命名约束：server alias **避免下划线**（策略引擎按首个下划线解析 FQN `mcp_<alias>_<tool>`，会静默误判）→ 采集/导出校验 Warning。三者至少其一：`command`/`url`/`httpUrl`；并存时优先级 `httpUrl` > `url` > `command`。

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `command` | string | stdio 启动命令 | 【标准字段】（`command` ⇒ `transport: stdio` 推导） |
| `args` | string[] | 参数 | 【标准字段】 |
| `env` | object | 环境变量（**已知含明文** → 强制 secretref 扫描） | 【标准字段】 |
| `cwd` | string | 工作目录 | 【无承载】IR McpServer 无 `cwd`（同 copilot，建议提升标准字段） |
| `url` | string | **SSE** 传输 URL | 【标准字段】→ `transport: sse` + `url` |
| `httpUrl` | string | **streamable HTTP** 传输 URL | 【标准字段】→ `transport: http` + `url`（导出按 transport 回写 `httpUrl`/`url`，适配器职责） |
| `headers` | object | HTTP 头 | 【标准字段】 |
| `timeout` | number | **请求超时（毫秒）** | 【x- 承载】IR `timeout.{startup_ms,tool_sec}` 语义不对应（gemini 为单请求超时、毫秒）；映射进 `tool_sec` 会失真（击穿 #7） |
| `trust` | boolean | 信任该 server，**绕过全部工具确认** | 【无承载】安全敏感行为开关，IR 无对应 → `x-gemini`（击穿 #8） |
| `description` | string | 展示用描述 | 【无承载】McpServer 无 description → `x-gemini` |
| `includeTools` | array | 工具白名单（仅列出的可用） | 【无承载】→ `x-gemini` |
| `excludeTools` | array | 工具黑名单（**优先于 includeTools**） | 【无承载】→ `x-gemini` |

### 2.18 `telemetry`（8 键）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `telemetry.enabled` / `traces` | boolean | 遥测开关/详细 trace | 【标准字段】 |
| `telemetry.target` | string | `local`/`gcp` | 【标准字段】 |
| `telemetry.otlpEndpoint` / `otlpProtocol` | string | OTLP 端点/协议（`grpc`/`http`） | 【标准字段】 |
| `telemetry.logPrompts` | boolean | 日志含 prompt 内容（**隐私敏感**） | 【标准字段】导出跨工具时记 Warning |
| `telemetry.outfile` | string | local 输出文件 | 【标准字段】 |
| `telemetry.useCollector` | boolean | 外部 collector | 【标准字段】 |

## 3. `GEMINI.md`（Instruction 实体）

| 维度 | 语义 | IR 承载 |
|---|---|---|
| 加载层级 | global `~/.gemini/GEMINI.md` → workspace 及祖先向上拼接 → JIT（工具访问目录时扫描该目录及祖先至 trusted root） | 【标准字段】scope + concat；JIT 条目对应 `subtree` |
| 正文 `@file.md` import | 相对/绝对路径模块化导入（Memory Import Processor，memport） | 【标准字段】→ `imports[]` + `roundtrip_policy`（v0.2 已闭环，三工具中最吻合） |
| `context.fileName` 自定义文件名 | 可改为 AGENTS.md 等或多名 | 【标准字段】Setting `setting.gemini.context.fileName`；导出布局由适配器读取该设置决定 |
| `/memory show`/`reload` | 运行时查看/重载 | 【无承载】运行时命令，不采集 |
| 数量指示 | footer 显示已加载文件数 | 【无承载】运行时 |

## 4. IR 击穿清单（gemini）

| # | 击穿点 | 等级 | 建议 |
|---|---|---|---|
| 1 | Setting id 三段式 vs dotted key-path（`general.vimMode` 等约 236 个叶子键） | BLOCKER | 同 copilot #2，统一修 id 语法 |
| 2 | 四层 settings（system-defaults / user / project / system override）+ `admin.*` managed 语义：IR scope 仅 `global\|project` | MAJOR | scope 枚举增 `managed`（只读采集、导出跳过），或进 `x-gemini.layer`；`admin.*` 条目建议默认不导出 |
| 3 | `tools.allowed`/`confirmationRequired` 与 zhanlu `permission.bash`、claude `permissions.allow` 构成跨工具权限映射族，但语法各异（`run_shell_command(git)` vs `Bash(npm run test:*)` vs glob pattern） | MAJOR | v0.2 保持各进 Setting 不透明 value；跨工具翻译表属导出器职责（IR-SCHEMA §3.4 已声明），需在 ADAPTERS 落实映射函数 |
| 4 | `modelConfigs.*` 内建默认值巨大（文档默认展开数百行）：全量采集会污染 profile | MAJOR | 采集器仅记录与内建默认的**差分**（custom* 键 + 被覆盖键）；diff 基线需随版本护栏更新 |
| 5 | 运行时状态混入 settings（`ide.hasSeenNudge`、`security.auth.selectedType`）：合并语义会把运行时状态当配置传播 | minor | 采集标注 `x-gemini.runtime: true`，导出策略默认跳过 |
| 6 | `hooks.*` 11 事件点数组：IR 无 hook 实体 | MAJOR | v0.2 作 Setting 不透明 value + `x-gemini`；v0.3 评估 hook 实体（copilot agent `hooks`、zhanlu plugin hooks 同诉求） |
| 7 | `mcpServers.*.timeout`（请求级、毫秒）vs IR `timeout.{startup_ms,tool_sec}`（启动/工具级、秒） | minor | 保留 `x-gemini.timeout` 原值；标准字段不强行映射（防止单位/语义双重失真） |
| 8 | `trust`/`includeTools`/`excludeTools`/`description`（MCP 实体级） | minor | `x-gemini` 透传；`trust` 导出到他工具时**必须记安全 Warning** |
| 9 | `.geminiignore` / `.gemini/sandbox.*` 文件资产无实体对应 | minor | 不采集或作 Instruction assets；ADAPTERS 明确 |

## 5. 真实样本

1. **官方完整 settings.json 示例（v0.3.0 嵌套结构）** — [configuration](https://www.geminicli.com/docs/reference/configuration/#example-settingsjson)：
   ```json
   { "general": { "vimMode": true, "sessionRetention": { "enabled": true, "maxAge": "30d", "maxCount": 100 } },
     "ui": { "theme": "GitHub", "hideBanner": true, "customWittyPhrases": ["..."] },
     "tools": { "sandbox": "docker", "exclude": ["write_file"] },
     "mcpServers": { "mainServer": { "command": "bin/mcp_server.py" } },
     "telemetry": { "enabled": true, "target": "local", "otlpEndpoint": "http://localhost:4317", "logPrompts": true },
     "model": { "name": "gemini-1.5-pro-latest", "maxSessionTurns": 10,
       "summarizeToolOutput": { "run_shell_command": { "tokenBudget": 100 } } },
     "context": { "fileName": ["CONTEXT.md", "GEMINI.md"], "loadFromIncludeDirectories": true,
       "fileFiltering": { "respectGitIgnore": false } },
     "advanced": { "excludedEnvVars": ["DEBUG", "DEBUG_MODE", "NODE_ENV"] } }
   ```
2. **官方 GEMINI.md 示例（TypeScript Library）** — [gemini-md](https://www.geminicli.com/docs/cli/gemini-md/)：含 `## General Instructions` / `## Coding Style` 分节的纯 Markdown（无 frontmatter——与 copilot `.instructions.md` 形态差异显著）。
3. **官方 GEMINI.md @import 示例** — 同上页：
   ```markdown
   # Main GEMINI.md file
   This is the main content.
   @./components/instructions.md
   @../shared/style-guide.md
   ```
4. **官方 `context.fileName` 自定义示例** — 同上页：`{ "context": { "fileName": ["AGENTS.md", "CONTEXT.md", "GEMINI.md"] } }`。
5. **官方环境变量脱敏配置示例** — [configuration](https://www.geminicli.com/docs/reference/configuration/#environment-variable-redaction)：`{ "security": { "allowedEnvironmentVariables": ["MY_PUBLIC_KEY"], "blockedEnvironmentVariables": ["INTERNAL_IP_ADDRESS"] } }`（注意此示例用旧键名，见 §2.11 附注）。
6. **官方 mcpServers 字段与优先级说明** — [configuration #mcpServers](https://www.geminicli.com/docs/reference/configuration/#mcpservers)：`httpUrl` > `url` > `command` 优先级、`mcp_` FQN 下划线禁令。
7. **官方 JSON Schema（golden-file 对齐源）** — `https://raw.githubusercontent.com/google-gemini/gemini-cli/main/schemas/settings.schema.json`（文档明示可用于编辑器校验；适配器 golden-file 应以此回归）。
8. **官方自定义沙箱 Dockerfile 示例** — [configuration #sandboxing](https://www.geminicli.com/docs/reference/configuration/#sandboxing)：`.gemini/sandbox.Dockerfile`（`FROM gemini-cli-sandbox` + `BUILD_SANDBOX=1 gemini -s`）。

## 6. 证伪结论：IR-SCHEMA v0.2 需要为 gemini 改什么

1. Setting id 放行 dotted key-path（击穿 #1，最高优先）。
2. scope 枚举增 `managed`（或等效标记）承载 system-defaults/system override 与 `admin.*`（击穿 #2）。
3. McpServer 增标准 `cwd`；`timeout` 保持 x-gemini 原值不映射（击穿 #7）。
4. 采集器规范补"内建默认值差分采集"条款（`modelConfigs.*`）（击穿 #4）——这实际是 ADAPTERS 层规则，IR 侧需在 §3.4 Setting 注明"工具内建默认不入库"。
5. `x-gemini` 透传清单：`mcpServers.*.{trust,description,includeTools,excludeTools,timeout}`、旧键名 `security.{allowed,blocked}EnvironmentVariables`、deprecated `experimental.topicUpdateNarration`、`ui.accessibility.enableLoadingPhrases`。
6. GEMINI.md 侧无需改动：`imports`/`subtree`/concat 已完整覆盖（v0.2 验证通过的最佳样本）。
