# 字段清单：Codex CLI（adapter id: `codex`）

> 调研日期：2026-08-16 ｜ 调研人：P0 工具组（字段级调研）
> 基线文档（developers.openai.com 官方文档站 2026-08-16 实时抓取；文档站注明 Markdown 版本可通过 URL 追加 `.md` 获取）：
> - [config-reference](https://developers.openai.com/codex/config-file/config-reference)（config.toml + requirements.toml 全键）
> - [config-advanced](https://developers.openai.com/codex/config-file/config-advanced)（profiles / 项目层 / model_providers / shell_environment_policy / otel）
> - [agents-md](https://developers.openai.com/codex/agent-configuration/agents-md)
> - [build-skills](https://developers.openai.com/codex/build-skills)
> - [hooks](https://developers.openai.com/codex/hooks)
> - [rules](https://developers.openai.com/codex/agent-configuration/rules)（execpolicy / Starlark）
>
> 承载状态图例：【标准字段】= IR-SCHEMA v0.2 标准字段直接承载；【x- 承载】= 需进 `x-codex` 透传；【无承载】= IR 当前结构无处安放（结构性击穿，见 §8 击穿清单）；【存疑】= 官方文档未明确，附核实方法。

## 0. 配置文件地图

| 文件 / 目录 | scope | IR 实体 | 备注 |
|---|---|---|---|
| `~/.codex/config.toml`（`$CODEX_HOME/config.toml`） | global（user） | Setting + McpServer | 用户级主配置 |
| `<repo>/.codex/config.toml`（项目根到 cwd 逐级） | project | Setting + McpServer | 仅项目 trusted 时加载；以下键在项目层被忽略并告警：`openai_base_url`、`chatgpt_base_url`、`apps_mcp_product_sku`、`model_provider`、`model_providers`、`notify`、`profile`、`profiles`、`experimental_realtime_ws_base_url`、`otel` |
| `~/.codex/<name>.config.toml` | global（profile 层） | Setting | `--profile <name>` 选择；叠加于 base user config 之上、project/CLI 之下（Codex 0.134.0+；旧 `[profiles.<name>]` 表与顶层 `profile=` 选择器已废弃） |
| `requirements.toml`（系统/MDM/云端下发） | managed（admin） | Setting | 管理员强制约束，用户不可覆盖 |
| `~/.codex/AGENTS.md` / `~/.codex/AGENTS.override.md` | global | Instruction | override 优先于 base |
| 项目根到 cwd 各目录 `AGENTS.override.md` / `AGENTS.md` / `project_doc_fallback_filenames` 所列名 | project | Instruction | 每目录最多取一个文件；根→cwd 顺序拼接 |
| `.agents/skills/<name>/SKILL.md`（cwd 到 repo 根逐级）、`~/.agents/skills/`、`/etc/codex/skills/` | project / global / admin | Skill（PromptPack） | 另有系统内置（SYSTEM）；支持 symlink |
| `~/.codex/hooks.json`、`<repo>/.codex/hooks.json`、config.toml 内联 `[hooks]` | global / project | 无实体（见 §5） | 同层两种表示并存则合并并告警 |
| `~/.codex/rules/*.rules`、`<repo>/.codex/rules/*.rules`、Team Config 位置 | global / project / managed | 无实体（见 §6） | Starlark 语言 execpolicy |
| 插件 `.codex-plugin/plugin.json` + 插件内 `hooks/hooks.json`、`skills/`、MCP 配置 | plugin | 无实体（见 §8-B3） | OpenAI/ChatGPT 通用插件目录分发 |
| `~/.codex/auth.json`、`history.jsonl`、`log/`、`sessions`、`sqlite` 状态库 | 运行时 | 不采集 | 凭证与状态，非配置 |

层级优先级（低→高）：系统默认 < user config < profile 文件 < project config（逐目录就近覆盖）< CLI `-c/--config`；managed requirements 在各自维度强制收敛。项目层仅在 trust 后加载（`projects.<path>.trust_level` 控制）。

## 1. config.toml（全部键逐字段）

来源：config-reference 页全表。类型列为原文 "Type / Values"。

### 1.1 顶层与 agents/apps 家族

| 字段路径 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `model` | string | 使用的模型（如 `gpt-5.5`） | 无 | 【标准字段】Setting key=model（跨工具翻译映射属导出器职责） |
| `model_provider` | string | `model_providers` 中的 provider id | `openai` | 【x- 承载】 |
| `model_reasoning_effort` | enum | `minimal/low/medium/high/xhigh`（Responses API） | 无 | 【x- 承载】 |
| `model_reasoning_summary` | enum | `auto/concise/detailed/none` | 无 | 【x- 承载】 |
| `model_supports_reasoning_summaries` | boolean | 强制是否发送 reasoning 元数据 | 无 | 【x- 承载】 |
| `model_verbosity` | enum | `low/medium/high`（GPT-5 Responses API） | 模型默认 | 【x- 承载】 |
| `model_context_window` | number | 活动模型上下文窗口 token 数 | 无 | 【x- 承载】 |
| `model_auto_compact_token_limit` | number | 触发自动历史压缩的 token 阈值 | 模型默认 | 【x- 承载】 |
| `model_auto_compact_token_limit_scope` | enum | 阈值计量口径 `total/body_after_prefix` | `total` | 【x- 承载】 |
| `model_catalog_json` | string(path) | 启动加载的 JSON 模型目录路径（profile 文件可覆盖） | 无 | 【x- 承载】 |
| `model_instructions_file` | string(path) | 替代内置指令的文件（替代 AGENTS.md 机制） | 无 | 【x- 承载】 |
| `instructions` | string | 保留字段（未来用） | 无 | 【x- 承载】 |
| `developer_instructions` | string | 注入会话的额外 developer 指令 | 无 | 【x- 承载】 |
| `compact_prompt` | string | 内联覆盖历史压缩 prompt | 无 | 【x- 承载】 |
| `experimental_compact_prompt_file` | string(path) | 从文件加载压缩 prompt（实验） | 无 | 【x- 承载】 |
| `review_model` | string | `/review` 使用的模型覆盖 | 会话模型 | 【x- 承载】 |
| `service_tier` | string | 新回合的服务层（`fast` 映射 `priority`） | 无 | 【x- 承载】 |
| `plan_mode_reasoning_effort` | enum | Plan 模式专用 reasoning 覆盖 `none/minimal/low/medium/high/xhigh` | 内置预设 | 【x- 承载】 |
| `personality` | enum | 沟通风格 `none/friendly/pragmatic` | 无 | 【x- 承载】 |
| `agents` | table | 多 agent 设置与自定义角色声明（标量键名为保留字，不能用作角色名） | — | 【x- 承载】（agent role 本体见 §7） |
| `agents.enabled` | boolean | 多 agent 工具开关 | true | 【x- 承载】 |
| `agents.<name>.config_file` | string(path) | 该角色的 TOML 配置层路径（相对声明它的配置文件解析） | 无 | 【x- 承载】 |
| `agents.<name>.description` | string | 角色用途说明（选择/spawn 时展示给模型） | 无 | 【x- 承载】 |
| `agents.default_subagent_model` | string | spawn agent 默认模型 | 无 | 【x- 承载】 |
| `agents.default_subagent_reasoning_effort` | string | spawn agent 默认 reasoning effort | 无 | 【x- 承载】 |
| `agents.interrupt_message` | boolean | agent 回合被打断时记录模型可见消息 | true | 【x- 承载】 |
| `agents.max_concurrent_threads_per_session` | number | 并发 spawn 线程上限（不含主线程） | 内置 | 【x- 承载】 |
| `agents.max_threads` | number | 上者遗留别名 | 无 | 【x- 承载】 |
| `approval_policy` | enum/object | `untrusted/on-request/never` 或 `{ granular = { sandbox_approval, rules, mcp_elicitations, request_permissions, skill_approval } }`；`on-failure` 已废弃 | 无 | 【x- 承载】（与 Claude `permissions.defaultMode` 语义近但不可直译，映射表属导出器） |
| `approval_policy.granular.*`（5 键） | boolean | 各类审批提示是否放行（sandbox_approval/rules/mcp_elicitations/request_permissions/skill_approval） | 无 | 【x- 承载】 |
| `approvals_reviewer` | enum | `user/auto_review`（auto_review 用 reviewer subagent 审批准请求） | `user` | 【x- 承载】 |
| `auto_review.policy` | string | 自动评审的本地 Markdown 策略（managed `guardian_policy_config` 优先） | 无 | 【x- 承载】 |
| `apps._default.enabled` | boolean | 全部 app 默认启用态 | true | 【x- 承载】 |
| `apps._default.approvals_reviewer` | enum | app 工具审批默认 reviewer | 继承顶层 | 【x- 承载】 |
| `apps._default.default_tools_approval_mode` | enum | `auto/prompt/writes/approve` | 无 | 【x- 承载】 |
| `apps._default.destructive_enabled` | boolean | `destructive_hint=true` 工具默认放行 | 无 | 【x- 承载】 |
| `apps._default.open_world_enabled` | boolean | `open_world_hint=true` 工具默认放行 | 无 | 【x- 承载】 |
| `apps.<id>.enabled` | boolean | 按 id 启停 app/connector | true | 【x- 承载】 |
| `apps.<id>.approvals_reviewer` / `.default_tools_approval_mode` / `.default_tools_enabled` / `.destructive_enabled` / `.open_world_enabled` | 同 _default | 按 app 覆盖 | 无 | 【x- 承载】 |
| `apps.<id>.tools.<tool>.approval_mode` | enum | 单工具审批行为覆盖 `auto/prompt/writes/approve` | 无 | 【x- 承载】 |
| `apps.<id>.tools.<tool>.enabled` | boolean | 单工具启停覆盖 | 无 | 【x- 承载】 |

### 1.2 sandbox / permissions / rules 相关键

| 字段路径 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `sandbox_mode` | enum | `read-only/workspace-write/danger-full-access` | 无 | 【x- 承载】（与 Claude sandbox.enabled 等无直译，映射表属导出器） |
| `sandbox_workspace_write.writable_roots[]` | array<string> | workspace-write 模式额外可写根 | 无 | 【x- 承载】 |
| `sandbox_workspace_write.network_access` | boolean | 沙箱内出站网络 | false | 【x- 承载】 |
| `sandbox_workspace_write.exclude_slash_tmp` | boolean | 可写根排除 `/tmp` | 无 | 【x- 承载】 |
| `sandbox_workspace_write.exclude_tmpdir_env_var` | boolean | 可写根排除 `$TMPDIR` | 无 | 【x- 承载】 |
| `default_permissions` | string | 默认权限 profile 名（内置 `:read-only/:workspace/:danger-full-access` 或自定义 `[permissions.<name>]`）；勿与 `sandbox_mode`/`sandbox_workspace_write` 同用 | 无 | 【x- 承载】 |
| `permissions.<name>.description` | string | 命名 profile 描述（不经 extends 继承） | 无 | 【x- 承载】 |
| `permissions.<name>.extends` | string | 父 profile（另一命名 profile 或 `:read-only/:workspace`；`:danger-full-access`/未定义/循环被拒绝） | 无 | 【x- 承载】 |
| `permissions.<name>.filesystem` | table | 文件系统权限表；键为绝对路径或 `:minimal/:workspace_roots` 等特殊 token | 无 | 【x- 承载】 |
| `permissions.<name>.filesystem.<path-or-glob>` | enum/table | `"read"/"write"/"deny"` 或嵌套子表 | 无 | 【x- 承载】 |
| `permissions.<name>.filesystem.":workspace_roots".<subpath-or-glob>` | enum | 相对各 workspace root 的权限（`"."` 表示根本身；glob 可 deny 读） | 无 | 【x- 承载】 |
| `permissions.<name>.filesystem.glob_scan_max_depth` | number | deny-read glob 展开最大深度 | 无 | 【x- 承载】 |
| `permissions.<name>.workspace_roots.<path>` | boolean | 纳入 profile 的 workspace root 集 | 无 | 【x- 承载】 |
| `permissions.<name>.network.enabled` | boolean | 该 profile 命令网络访问（不启动代理） | 无 | 【x- 承载】 |
| `permissions.<name>.network.mode` | enum | 代理模式 `limited/full` | 无 | 【x- 承载】 |
| `permissions.<name>.network.domains.<pattern>` | enum | 域名规则 `allow/deny`（精确/`*.x.com` 子域/`**.x.com`  apex+子域/全局 `*`；deny 优先） | 无 | 【x- 承载】 |
| `permissions.<name>.network.unix_sockets.<path>` | enum | Unix socket 规则 `allow/deny` | 无 | 【x- 承载】 |
| `permissions.<name>.network.allow_local_binding` / `.allow_upstream_proxy` / `.dangerously_allow_all_unix_sockets` / `.dangerously_allow_non_loopback_proxy` / `.enable_socks5` / `.enable_socks5_udp` / `.proxy_url` / `.socks_url` | 同 features.network_proxy.* | profile 级覆盖 | 见 §1.5 | 【x- 承载】 |
| `projects.<path>.trust_level` | string | `"trusted"/"untrusted"`；untrusted 跳过项目 `.codex/` 层（config/hooks/rules） | 无 | 【无承载】→ B8（信任状态存于用户 config 内，但是运行时信任数据；建议不采集或 x- 标记 sync 排除） |
| `allow_login_shell` | boolean | shell 工具可用 login shell 语义 | true | 【x- 承载】 |
| `shell_environment_policy.inherit` | enum | 子进程环境继承基线 `all/core/none` | 无 | 【x- 承载】 |
| `shell_environment_policy.set` | map<string,string> | 显式注入的环境值（在排除之后应用） | 无 | 【x- 承载】 |
| `shell_environment_policy.filters` | map<string,enum> | 大小写不敏感模式过滤 `include/exclude`（含 `*`/`?`） | 无 | 【x- 承载】 |
| `shell_environment_policy.ignore_default_excludes` | boolean | true=保留名字含 KEY/SECRET/TOKEN 的变量 | true | 【x- 承载】 |
| `shell_environment_policy.exclude` / `.include_only` | array<string> | 遗留数组形式（勿与 filters 同层混用） | 无 | 【x- 承载】 |
| `shell_environment_policy.experimental_use_profile` | boolean | 子进程使用用户 shell profile | 无 | 【x- 承载】 |
| `windows.sandbox` | enum | Windows 原生沙箱 `unelevated/elevated` | 无 | 【x- 承载】 |
| `windows.sandbox_private_desktop` | boolean | 沙箱子进程运行于私有桌面 | 无 | 【x- 承载】 |
| `windows_wsl_setup_acknowledged` | boolean | Windows onboarding 确认记录 | 无 | 【无承载】运行时状态 → B8 |
| `computer_use.windows.always_allowed_app_ids` | array<string> | Computer Use 免提示打开的 Windows 应用 id | 无 | 【x- 承载】 |

### 1.3 mcp_servers 家族（`mcp_servers.<id>.*`）

| 字段路径 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `mcp_servers.<id>.command` | string | stdio 启动命令 | 无 | 【标准字段】McpServer.command |
| `mcp_servers.<id>.args` | array<string> | 命令参数 | 无 | 【标准字段】args |
| `mcp_servers.<id>.env` | map<string,string> | 转发给 stdio 服务器的环境变量 | 无 | 【标准字段】env（secretref 抽取点） |
| `mcp_servers.<id>.env_vars` | array<string\|object> | 额外环境变量白名单 `["NAME"]` 或 `{name, source:"local"/"remote"}`（remote 仅 executor 远程 stdio） | 无 | 【无承载】→ B5（IR env 是值表，Codex 另有"允许继承的变量名清单"维度） |
| `mcp_servers.<id>.cwd` | string | stdio 服务器工作目录 | 无 | 【无承载】→ B5（IR McpServer 无 cwd 字段） |
| `mcp_servers.<id>.url` | string | streamable HTTP 端点 | 无 | 【标准字段】url（transport=http） |
| `mcp_servers.<id>.http_headers` | map<string,string> | 静态 HTTP 头 | 无 | 【标准字段】headers |
| `mcp_servers.<id>.env_http_headers` | map<string,string> | 值取自环境变量的 HTTP 头 `{头名: 环境变量名}` | 无 | 【无承载】→ B5（间接寻址，IR headers 仅存字面值/secretref） |
| `mcp_servers.<id>.bearer_token_env_var` | string | 提供 bearer token 的环境变量名 | 无 | 【无承载】→ B5（同上；语义可用 env_http_headers.Authorization 近似但不等价） |
| `mcp_servers.<id>.auth` | enum | HTTP 服务器认证回退 `oauth/chatgpt` | `oauth` | 【x- 承载】 |
| `mcp_servers.<id>.oauth_resource` | string | RFC 8707 OAuth resource 参数 | 无 | 【x- 承载】 |
| `mcp_servers.<id>.scopes` | array<string> | OAuth 请求 scope | 无 | 【x- 承载】（与 Claude oauth.scopes 数组/字符串形态差异，适配器转换） |
| `mcp_servers.<id>.enabled` | boolean | 不删配置地禁用服务器 | true | 【标准字段】disabled（极性相反，适配器取反——IR §3.2 已立规） |
| `mcp_servers.<id>.required` | boolean | 启用但初始化失败则启动/resume 失败 | false | 【无承载】→ B5 |
| `mcp_servers.<id>.startup_timeout_sec` | number | 启动超时（默认 10s） | 10 | 【标准字段】timeout.startup_ms（s→ms 换算，IR §3.2 已立规） |
| `mcp_servers.<id>.startup_timeout_ms` | number | 上者的毫秒别名 | 无 | 【标准字段】同上 |
| `mcp_servers.<id>.tool_timeout_sec` | number | 单工具超时（默认 60s） | 60 | 【标准字段】timeout.tool_sec |
| `mcp_servers.<id>.enabled_tools` | array<string> | 工具允许列表 | 无 | 【无承载】→ B5 |
| `mcp_servers.<id>.disabled_tools` | array<string> | 工具拒绝列表（在 enabled_tools 之后应用） | 无 | 【无承载】→ B5 |
| `mcp_servers.<id>.default_tools_approval_mode` | enum | 该服务器工具默认审批 `auto/prompt/writes/approve` | 无 | 【无承载】→ B5 |
| `mcp_servers.<id>.tools.<tool>.approval_mode` | enum | 单工具审批覆盖 | 无 | 【无承载】→ B5 |
| `mcp_servers.<id>.experimental_environment` | enum | 实验性放置 `local/remote`（remote 经 executor 起 stdio） | local | 【x- 承载】 |
| `mcp_oauth_callback_port` | integer | MCP OAuth 本地回调固定端口 | 随机 | 【x- 承载】（Claude 对应物在 per-server oauth.callbackPort，粒度不同） |
| `mcp_oauth_callback_url` | string | MCP OAuth 回调 base URL 覆盖 | 无 | 【x- 承载】 |
| `mcp_oauth_credentials_store` | enum | MCP OAuth 凭证存储 `auto/file/keyring` | 无 | 【x- 承载】 |
| `plugins.<plugin>.mcp_servers.<server>.enabled` | boolean | 插件捆绑服务器的启停覆盖（不改插件清单） | 无 | 【x- 承载】（插件维度 → B3 关联） |
| `plugins.<plugin>.mcp_servers.<server>.enabled_tools` / `.disabled_tools` / `.default_tools_approval_mode` / `.tools.<tool>.approval_mode` | 同 mcp_servers 对应物 | 插件服务器工具治理 | 无 | 【x- 承载】 |

### 1.4 model_providers 家族

| 字段路径 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `model_providers.<id>` | table | 自定义 provider（内置 `openai/ollama/lmstudio` 保留不可覆盖） | 无 | 【无承载】→ B6（IR 无 Provider 实体；与 Setting 的 key=value 平铺模型不兼容的嵌套表家族，可作 `setting.codex.model_providers` 不透明 value 兜底） |
| `model_providers.<id>.name` | string | 展示名 | 无 | 【x- 承载】（同上 value 内） |
| `model_providers.<id>.base_url` | string | API base URL | 无 | 【x- 承载】 |
| `model_providers.<id>.env_key` | string | 提供 API key 的环境变量名 | 无 | 【x- 承载】 |
| `model_providers.<id>.env_key_instructions` | string | API key 配置指引 | 无 | 【x- 承载】 |
| `model_providers.<id>.experimental_bearer_token` | string | 直接 bearer token（不推荐） | 无 | 【x- 承载】敏感，secretref 抽取 |
| `model_providers.<id>.requires_openai_auth` | boolean | 使用 OpenAI 认证 | false | 【x- 承载】 |
| `model_providers.<id>.wire_api` | enum | 仅 `responses` | `responses` | 【x- 承载】 |
| `model_providers.<id>.http_headers` | map<string,string> | 静态请求头 | 无 | 【x- 承载】 |
| `model_providers.<id>.env_http_headers` | map<string,string> | 环境变量取值请求头 | 无 | 【x- 承载】 |
| `model_providers.<id>.query_params` | map<string,string> | 附加 query 参数 | 无 | 【x- 承载】 |
| `model_providers.<id>.auth.command` / `.args` / `.cwd` / `.timeout_ms`(默认 5000) / `.refresh_interval_ms`(默认 300000，0=仅重试后刷新) | 命令型 bearer token 获取 | 无 | 【x- 承载】 |
| `model_providers.<id>.request_max_retries` | number | HTTP 重试 | 4 | 【x- 承载】 |
| `model_providers.<id>.stream_max_retries` | number | SSE 重试 | 5 | 【x- 承载】 |
| `model_providers.<id>.stream_idle_timeout_ms` | number | SSE 空闲超时 | 300000 | 【x- 承载】 |
| `model_providers.<id>.supports_standalone_web_search` | boolean | 声明支持独立 web search 端点 | false | 【x- 承载】 |
| `model_providers.<id>.supports_websockets` | boolean | 支持 Responses API WebSocket 传输 | 无 | 【x- 承载】 |
| `model_providers.amazon-bedrock.aws.profile` / `.region` | string | 内置 Bedrock provider 的 AWS profile/region | 无 | 【x- 承载】 |
| `openai_base_url` | string | 内置 openai provider base URL 覆盖 | 无 | 【x- 承载】 |
| `chatgpt_base_url` | string | ChatGPT 登录流 base URL 覆盖 | 无 | 【x- 承载】 |
| `oss_provider` | enum | `--oss` 默认本地 provider `lmstudio/ollama` | 询问 | 【x- 承载】 |
| `forced_login_method` | enum | 限定认证方式 `chatgpt/api` | 无 | 【x- 承载】 |
| `forced_chatgpt_workspace_id` | string(uuid) | 限定 ChatGPT workspace | 无 | 【x- 承载】 |
| `cli_auth_credentials_store` | enum | CLI 凭证存储 `file/keyring/auto` | 无 | 【x- 承载】 |

### 1.5 features / tui / otel / notice / history / 其余顶层键

| 字段路径 | 类型 | 语义 | 默认值 | IR 承载 |
|---|---|---|---|---|
| `features.apps` | boolean | app/connector 集成 | true（stable） | 【x- 承载】 |
| `features.code_mode.enabled` | boolean | code mode（开发中） | false | 【x- 承载】 |
| `features.code_mode.direct_only_tool_namespaces` / `.excluded_tool_namespaces` | array<string> | code mode 工具命名空间治理 | 无 | 【x- 承载】 |
| `features.enable_request_compression` | boolean | zstd 压缩流式请求体 | true（stable） | 【x- 承载】 |
| `features.fast_mode` | boolean | 模型目录服务层选择（Fast-tier 命令） | true（stable） | 【x- 承载】 |
| `features.goals` | boolean | 持久化 goals 与自动续作 | true（stable） | 【x- 承载】 |
| `features.hooks` | boolean | 生命周期 hooks 总开关（`codex_hooks` 为废弃别名） | true | 【x- 承载】 |
| `features.memories` | boolean | Memories 功能 | false | 【x- 承载】 |
| `features.multi_agent` | boolean | 多 agent 协作工具（spawn_agent/send_input/resume_agent/wait_agent/close_agent） | true（stable） | 【x- 承载】 |
| `features.network_proxy` | boolean\|table | 沙箱命令网络代理（experimental）；table 形态含下述子键 | false | 【x- 承载】 |
| `features.network_proxy.enabled` | boolean | 命令网络访问时启动代理（不开则不强制 profile 域名规则） | false | 【x- 承载】 |
| `features.network_proxy.domains` | map<string,enum> | 域名策略 `allow/deny`（unset=不放行任何外部目标） | unset | 【x- 承载】 |
| `features.network_proxy.unix_sockets` | map<string,enum> | Unix socket 策略 | unset | 【x- 承载】 |
| `features.network_proxy.allow_local_binding` / `.allow_upstream_proxy`(true) / `.dangerously_allow_all_unix_sockets`(false) / `.dangerously_allow_non_loopback_proxy`(false) / `.enable_socks5`(true) / `.enable_socks5_udp`(true) | boolean | 代理行为细项 | 括号内 | 【x- 承载】 |
| `features.network_proxy.proxy_url` / `.socks_url` | string | 监听地址 | `http://127.0.0.1:3128` / `http://127.0.0.1:8081` | 【x- 承载】 |
| `features.personality` | boolean | personality 选择控件 | true（stable） | 【x- 承载】 |
| `features.prevent_idle_sleep` | boolean | 回合运行中防系统休眠（experimental） | false | 【x- 承载】 |
| `features.remote_plugin` | boolean | 远程插件目录 | true（stable） | 【x- 承载】 |
| `features.rollout_budget.enabled` / `.limit_tokens` / `.prefill_token_weight`(1.0) / `.reminder_interval_tokens`(10% of limit) / `.sampling_token_weight`(1.0) | boolean/integer/number | rollout 预算追踪（开发中） | false | 【x- 承载】 |
| `features.shell_snapshot` | boolean | shell 环境快照加速 | true（stable） | 【x- 承载】 |
| `features.shell_tool` | boolean | 默认 shell 工具 | true（stable） | 【x- 承载】 |
| `features.skill_mcp_dependency_install` | boolean | 允许提示安装 skill 缺失的 MCP 依赖 | true（stable） | 【x- 承载】 |
| `features.unified_exec` | boolean | 统一 PTY exec 工具 | true（Windows 除外） | 【x- 承载】 |
| `features.web_search` / `.web_search_cached` / `.web_search_request` | boolean | 遗留开关；优先用顶层 `web_search` | 无 | 【x- 承载】采集保留，导出记 Warning |
| `feedback.enabled` | boolean | `/feedback` 反馈提交 | true | 【x- 承载】 |
| `analytics.enabled` | boolean | 匿名使用指标 | 客户端默认 | 【x- 承载】 |
| `file_opener` | enum | 引用打开 URI scheme `vscode/vscode-insiders/windsurf/cursor/none` | `vscode` | 【x- 承载】 |
| `hide_agent_reasoning` | boolean | TUI 与 `codex exec` 隐藏 reasoning 事件 | 无 | 【x- 承载】 |
| `show_raw_agent_reasoning` | boolean | 展示原始 reasoning 内容 | 无 | 【x- 承载】 |
| `history.persistence` | enum | 会话 transcript 持久化 `save-all/none` | 无 | 【x- 承载】 |
| `history.max_bytes` | number | history 文件体积上限（丢弃最旧条目） | 无 | 【x- 承载】 |
| `notify` | array<string> | 通知命令（收 JSON payload） | 无 | 【x- 承载】（项目层被忽略） |
| `log_dir` | string(path) | 日志目录；显式设置同时启用 `codex-tui.log` 明文日志 | `$CODEX_HOME/log` | 【x- 承载】 |
| `sqlite_home` | string(path) | SQLite 状态库目录 | 无 | 【x- 承载】 |
| `background_terminal_max_timeout` | number | 后台终端空轮询最大窗口（ms） | 300000 | 【x- 承载】 |
| `check_for_update_on_startup` | boolean | 启动检查更新 | true | 【x- 承载】 |
| `disable_paste_burst` | boolean | 关闭 TUI 突发粘贴检测 | 无 | 【x- 承载】 |
| `tool_output_token_limit` | number | 单条工具输出入历史的 token 预算 | 无 | 【x- 承载】 |
| `tool_suggest.disabled_tools` / `.discoverables` | array<table> | 工具建议治理；条目 `{type:"connector"/"plugin", id}` | 无 | 【x- 承载】 |
| `tools.view_image` | boolean | 本地图片附件工具 | 无 | 【x- 承载】 |
| `tools.web_search` | boolean\|object | web search 工具；object 形态 `{context_size:"low/medium/high", allowed_domains:[...], location:{country,region,city,timezone}}` | 无 | 【x- 承载】 |
| `web_search` | enum | `disabled/cached/indexed/live` | `cached`（full access 时 `live`） | 【x- 承载】 |
| `suppress_unstable_features_warning` | boolean | 抑制不稳定特性警告 | 无 | 【x- 承载】 |
| `notice.hide_full_access_warning` / `.hide_gpt-5.1-codex-max_migration_prompt` / `.hide_gpt5_1_migration_prompt` / `.hide_rate_limit_model_nudge` / `.hide_world_writable_warning` | boolean | 各类提示的确认记录 | 无 | 【无承载】→ B8（运行时状态） |
| `notice.model_migrations` | map<string,string> | 已确认的模型迁移 old→new 映射 | 无 | 【无承载】→ B8 |
| `experimental_use_unified_exec_tool` | boolean | unified exec 遗留名 | 无 | 【x- 承载】导出记 Warning |
| `desktop.custom_file_handlers.<id>` 及 `.command`(必)/`.label`(必)/`.icon`(必)/`.args`([])/`.input`(`path/json_argument/json_stdin`)/`.supports_ssh`(false) | table | 桌面 App "Open in" 自定义目标（仅 user 层） | 无 | 【x- 承载】 |
| `skills.config` | array<object> | `[[skills.config]]` 按路径启停 skill | 无 | 【x- 承载】 |
| `skills.config.<index>.path` | string(path) | 指向含 SKILL.md 的目录 | 无 | 【x- 承载】 |
| `skills.config.<index>.enabled` | boolean | 启停 | 无 | 【x- 承载】 |
| `project_doc_fallback_filenames` | array<string> | AGENTS.md 缺失时的候选文件名 | 无 | 【x- 承载】 |
| `project_doc_max_bytes` | number | 项目指令组合读取上限 | 32 KiB | 【x- 承载】 |
| `project_root_markers` | array<string> | 项目根标记文件名（`[]`=不向上找） | `[".git"]` 语义默认 | 【x- 承载】 |
| `hooks`（内联表） | table | 内联生命周期 hooks，结构见 §5 | 无 | 【无承载】→ B1 |
| `tui` | table | TUI 选项总表 | 无 | 【x- 承载】 |
| `tui.alternate_screen` | enum | alt-screen `auto/always/never`（auto 在 Zellij 中跳过） | `auto` | 【x- 承载】 |
| `tui.animations` | boolean | 终端动画 | true | 【x- 承载】 |
| `tui.keymap.<context>.<action>` | string\|array<string> | 键位绑定（context：global/chat/composer/editor/vim_normal/vim_operator/vim_text_object/pager/list/approval；空数组=解绑） | 无 | 【无承载】→ B7（开放二维键位表，Setting 平铺模型可装但键名含动态段，建议 x- 整体透传） |
| `tui.model_availability_nux.<model>` | integer | 启动 tooltip 内部状态 | 无 | 【无承载】→ B8 |
| `tui.notification_condition` | enum | 通知时机 `unfocused/always` | `unfocused` | 【x- 承载】 |
| `tui.notification_method` | enum | 通知方式 `auto/osc9/bel` | `auto` | 【x- 承载】 |
| `tui.notifications` | boolean\|array<string> | TUI 通知开关/事件类型限定 | 无 | 【x- 承载】 |
| `tui.raw_output_mode` | boolean | 原始滚动模式（`/raw` 或 alt-r 切换） | false | 【x- 承载】 |
| `tui.resume_cwd` | enum | 恢复/fork 会话的工作目录 `current/session` | 询问 | 【x- 承载】 |
| `tui.show_tooltips` | boolean | 欢迎页 onboarding 提示 | true | 【x- 承载】 |
| `tui.status_line` | array<string>\|null | 页脚状态行项 id 有序列表；null=禁用 | 无 | 【x- 承载】（与 Claude statusLine 命令脚本形态不同：Codex 是内置项编排） |
| `tui.terminal_title` | array<string>\|null | 终端标题项编排 | `["spinner","project"]` | 【x- 承载】 |
| `tui.theme` | string | 语法高亮主题（kebab-case） | 无 | 【x- 承载】 |
| `tui.vim_mode_default` | boolean | composer 默认 vim normal 模式 | false | 【x- 承载】 |
| `otel.environment` | string | OTel 事件环境标签 | `dev` | 【x- 承载】（项目层忽略整个 otel） |
| `otel.exporter` | enum/object | 日志导出器 `none/otlp-http/otlp-grpc` 或带端点对象 | `none` | 【x- 承载】 |
| `otel.exporter.<id>.endpoint` / `.headers` / `.protocol`(binary/json) / `.tls.ca-certificate` / `.tls.client-certificate` / `.tls.client-private-key` | 导出器端点配置 | 无 | 【x- 承载】 |
| `otel.metrics_exporter` | enum | 指标导出 `none/statsig/otlp-http/otlp-grpc` | `statsig` | 【x- 承载】 |
| `otel.trace_exporter` 及 `<id>` 子键 | 同 exporter | trace 导出 | 无 | 【x- 承载】 |
| `otel.log_user_prompt` | boolean | 导出原始用户 prompt（脱敏开关） | false | 【x- 承载】 |
| `memories.consolidation_model` / `.extract_model` | string | 记忆整合/抽取模型覆盖 | 无 | 【x- 承载】 |
| `memories.generate_memories` | boolean | 新线程参与记忆生成 | true | 【x- 承载】 |
| `memories.use_memories` | boolean | 未来会话注入已有记忆 | true | 【x- 承载】 |
| `memories.disable_on_external_context` | boolean | 含 MCP/web/tool search 的线程不生成记忆（旧别名 `no_memories_if_mcp_or_web_search`） | false | 【x- 承载】 |
| `memories.max_raw_memories_for_consolidation` | number | 原始记忆保留数（上限 4096） | 256 | 【x- 承载】 |
| `memories.max_rollout_age_days` | number | 线程参与生成的最大年龄（clamp 0–90） | 30 | 【x- 承载】 |
| `memories.max_rollouts_per_startup` | number | 每次启动处理的 rollout 候选数（上限 128） | 16 | 【x- 承载】 |
| `memories.max_unused_days` | number | 记忆未用天数淘汰阈值（clamp 0–365） | 30 | 【x- 承载】 |
| `memories.min_rate_limit_remaining_percent` | number | 生成记忆的速率窗口余量下限（clamp 0–100） | 25 | 【x- 承载】 |
| `memories.min_rollout_idle_hours` | number | 线程参与生成的最小空闲时间（clamp 1–48） | 6 | 【x- 承载】 |

【存疑】顶层 `profile` / `profiles` 键：config-reference 的项目层忽略清单提及二者，但 config-advanced 明确 Codex 0.134.0+ 已废弃 `[profiles.<name>]` 表与顶层 `profile=` 选择器，由 `~/.codex/<name>.config.toml` 文件机制取代。采集旧版本配置文件时仍可能遇到，适配器应识别并记迁移 Warning（核实：openai/codex 仓库 CHANGELOG 0.134.0）。

## 2. requirements.toml（admin 强制层）

来源：config-reference 页 requirements.toml 全表。admin 强制约束，用户不可覆盖；位置见 [managed-configuration](https://developers.openai.com/codex/enterprise/managed-configuration)。

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `allow_appshots` / `allow_remote_control` | boolean | 关闭 Appshots / 设备远程控制 | 【无承载】→ B2（managed 维度） |
| `allow_login_shell` | boolean | 强制 login shell 策略 | 【无承载】→ B2 |
| `allow_managed_hooks_only` | boolean | 跳过 user/project/session/plugin hooks，仅 managed | 【无承载】→ B2 |
| `allowed_approval_policies` / `allowed_approvals_reviewers` / `allowed_sandbox_modes` / `allowed_web_search_modes` | array<string> | 各键的允许值清单 | 【无承载】→ B2 |
| `allowed_permission_profiles` / `allowed_permission_profiles.<name>` | table<boolean> | 权限 profile 完整白名单（缺省/false 即拒，含未来新增 profile） | 【无承载】→ B2 |
| `apps.<id>.enabled` / `apps.<id>.tools.<tool>.approval_mode` | — | app 级强制 | 【无承载】→ B2 |
| `check_for_update_on_startup` / `log_dir` / `sqlite_home` / `model_catalog_json` | — | 强制路径/更新检查 | 【无承载】→ B2 |
| `computer_use.allow_locked_computer_use` | boolean | 锁屏后禁止 Computer Use（macOS managed） | 【无承载】→ B2 |
| `default_permissions` | string | managed 默认权限 profile（须被 allowed_permission_profiles 允许） | 【无承载】→ B2 |
| `enforce_residency` | string | 数据驻留要求（当前仅 `us`） | 【无承载】→ B2 |
| `experimental_network.*`（enabled/allowed_domains/denied_domains/domains/unix_sockets/allow_local_binding/allow_upstream_proxy/dangerously_*/http_port/socks_port/managed_allowed_domains_only） | table | 管理员网络要求（可在无 features.network_proxy 时启动代理） | 【无承载】→ B2 |
| `features.<name>` / `features.apps` / `.browser_use` / `.browser_use_external` / `.browser_use_full_cdp_access` / `.computer_use` / `.fast_mode` / `.guardian_approval` / `.in_app_browser` / `.in_app_updates` / `.memories` / `.multi_agent` / `.plugin_sharing` / `.plugins` / `.remote_plugin` / `.workspace_dependencies` | boolean | 特性钉死 | 【无承载】→ B2 |
| `feedback.enabled` | boolean | 反馈提交强制 | 【无承载】→ B2 |
| `guardian_policy_config` | string | managed 自动评审 Markdown 策略（优先于本地 auto_review.policy） | 【无承载】→ B2 |
| `hooks`（含 `hooks.managed_dir` macOS/Linux、`hooks.windows_managed_dir` Windows、内联事件表） | table | managed hooks（脚本由企业另行分发，绝对路径） | 【无承载】→ B1+B2 |
| `marketplaces.restrict_to_allowed_sources` | boolean | 市场来源限制总开关 | 【无承载】→ B2/B3 |
| `marketplaces.allowed_sources.<name>` 及 `.source`(git/host_pattern/local)/`.url`/`.ref`/`.host_pattern`/`.path` | table | 市场来源白名单规则 | 【无承载】→ B2/B3 |
| `mcp_servers.<id>.identity.command` / `.url`（string 或 `{match:exact/prefix/regex, value/expression}` 表；command 表形态含 `.executable` + `.args[]{match,value/expression}` 有序匹配器） | string/table | MCP 服务器白名单（名称+身份双匹配；不在表内即禁用） | 【无承载】→ B2 |
| `plugins.<plugin>.mcp_servers.<server>.identity.*` | 同上 | 插件捆绑服务器白名单 | 【无承载】→ B2/B3 |
| `models.new_thread.model` / `.model_reasoning_effort` / `.service_tier` | string | 新线程 managed 默认值（显式 CLI 覆盖优先） | 【无承载】→ B2 |
| `permissions.<name>` / `permissions.filesystem.deny_read` | table / array<string> | admin 定义权限 profile（同名冲突拒绝）；强制读拒绝（用户不可弱化） | 【无承载】→ B2 |
| `remote_sandbox_config[]`（`.hostname_patterns` + `.allowed_sandbox_modes`） | array<table> | 按主机名的沙箱模式覆盖 | 【无承载】→ B2 |
| `rules.prefix_rules[]`（`.pattern[]{token/any_of}` + `.decision`(prompt/forbidden) + `.justification`） | array<table> | 强制命令规则（仅可 restrictive；与 .rules 文件合并） | 【无承载】→ B2+B6 |
| `windows.allowed_sandbox_implementations` / `windows.sandbox_private_desktop` | — | Windows 沙箱强制 | 【无承载】→ B2 |

## 3. AGENTS.md（Instruction 对应物）

来源：agents-md 页。

| 项 | 语义 | IR 承载 |
|---|---|---|
| `~/.codex/AGENTS.md` | 全局指令 | 【标准字段】Instruction（scope=global） |
| `~/.codex/AGENTS.override.md` | 全局覆盖（存在则取代 base；全局层只取第一个非空文件） | 【标准字段】（另一条 Instruction；override 语义 x- 标注，priority 调整） |
| 项目发现：项目根（`project_root_markers`，默认 `.git`）→ cwd 逐目录检查 `AGENTS.override.md` → `AGENTS.md` → `project_doc_fallback_filenames`；每目录最多一个 | 链式拼接 | 【标准字段】Instruction + subtree；【存疑】"override 取代同目录 base" 的成对语义 IR 无表达 → x- 标 `overrides: <relpath>`（观察项） |
| 合并：根→cwd 顺序、空行连接、跳空文件、总量上限 `project_doc_max_bytes`（默认 32 KiB） | 拼接语义 | 【标准字段】Instruction concat（layer-ordered 两段式与 Codex 的"根→叶"近似但 IR 以 scope 分层，Codex 以目录层级分层 → 适配器注意排序映射） |
| `## Code Review Rules` 小节 | GitHub code review 规则（就近生效） | 【标准字段】正文原样保留（文档层语义，不抽取） |
| 运行时记忆 Memories（`~/.codex/memories` 体系） | 自动学习记忆（features.memories 开关） | 【无承载】→ B9（派生数据，默认不采集） |

## 4. Skills（Codex 体系）

来源：build-skills 页。遵循 [Agent Skills 开放标准](https://agentskills.io)。

| 项 | 语义 | IR 承载 |
|---|---|---|
| 位置：`.agents/skills/`（cwd→repo 根逐级）、`~/.agents/skills/`、`/etc/codex/skills/`、系统内置 | 四级 scope | 【标准字段】Skill（PromptPack）；`/etc/codex/skills` 属 admin/managed 层 → B2 关联 |
| `SKILL.md` frontmatter `name` + `description` | **两者必填**（与 Claude Code 全可选不同） | 【标准字段】name/description |
| skill 目录支撑文件 `scripts/`、`references/`、`assets/` | 包内容 | 【标准字段】PromptPack assets |
| `agents/openai.yaml`：`interface.display_name/short_description/icon_small/icon_large/brand_color/default_prompt` | 桌面 App UI 元数据 | 【x- 承载】 |
| `agents/openai.yaml`：`policy.allow_implicit_invocation`（默认 true） | 隐式调用开关（显式 `$skill` 仍可用） | 【x- 承载】（与 Claude `disable-model-invocation` 极性相反，映射表属导出器） |
| `agents/openai.yaml`：`dependencies.tools[]`（`{type:"mcp", value, description, transport, url}`） | 声明 MCP 依赖 | 【x- 承载】（与 McpServer 实体的联动为引用语义，不展开） |
| 调用方式：`/skills` 菜单、`$name` 显式提及、模型按 description 隐式选择 | 触发 | 【标准字段】trigger（slash-command 形态差异：Codex 用 `$` 前缀，适配器映射） |
| `[[skills.config]]`（config.toml）path+enabled | 启停覆盖 | 【x- 承载】（见 §1.5） |
| 初始清单预算：≤2% 上下文窗口或 8000 字符 | 运行语义 | n/a（非配置字段） |
| 插件分发（plugins 打包 skills + connectors） | 分发形态 | 【无承载】→ B3 |

## 5. Hooks（hooks.json / 内联 [hooks]）

来源：hooks 页。

### 5.1 事件全集（11 个）

| 事件 | matcher 过滤对象 | 取值 |
|---|---|---|
| `SessionStart` | source | `startup`/`resume`/`clear`/`compact` |
| `SessionEnd` | reason | 当前仅 `other`；始终同步执行（默认 timeout 1s，上限 3s） |
| `UserPromptSubmit` | 不支持 matcher | — |
| `PreToolUse` | 工具名 | `Bash`（含 unified exec）、`apply_patch`（也可用 `Edit`/`Write`）、`mcp__<server>__<tool>`、`update_plan` 等本地函数工具、`Agent`（spawn_agent 亦匹配） |
| `PermissionRequest` | 工具名 | 同上（hosted tools 如 WebSearch 不经过） |
| `PostToolUse` | 工具名 | 同上 |
| `PreCompact` / `PostCompact` | 压缩触发 | `manual`/`auto` |
| `SubagentStart` / `SubagentStop` | agent_type | 取决于 subagent |

### 5.2 结构与 handler 字段

- 文件形态：`hooks.json`（可选顶层 `description`）或 config.toml 内联 `[[hooks.<Event>]]` + `[[hooks.<Event>.hooks]]`；同层两种并存则合并并告警。
- 位置：`~/.codex/`、`<repo>/.codex/`（trusted 才加载项目层）、requirements.toml（managed）、插件 `hooks/hooks.json` 或 plugin.json `hooks` 键（`./` 路径/数组/内联对象）。
- handler 字段：`type`（当前仅 `command` 实际执行；`prompt`/`agent` 解析但跳过）、`command`、`commandWindows`（TOML 别名 `command_windows`）、`timeout`（秒，默认 600；SessionEnd 默认 1 上限 3）、`statusMessage`、`additionalContextLimit`（默认 2500 token 阈值，0=全量直给模型；超量写盘 `<temp_dir>/hook_outputs/...` 并给摘要）、`async`（后台运行，每会话并发上限 8；SessionEnd 除外）。
- matcher：正则字符串；`"*"`/空/省略=全匹配。
- 输入（stdin JSON 公共字段）：`session_id`、`transcript_path`、`cwd`、`hook_event_name`、`model`（Codex 扩展）、`turn_id`（回合域）、`permission_mode`（`default/acceptEdits/plan/dontAsk/bypassPermissions`）。
- 输出：公共 `continue`/`stopReason`/`systemMessage`/`suppressOutput`（后者解析未实现）；`hookSpecificOutput.additionalContext`（SessionStart/SubagentStart 等注入 developer 上下文）；PreToolUse/PermissionRequest 仅支持 systemMessage（返回不支持字段=hook 失败但放行）。
- 信任机制：非 managed command hook 须经 review+trust（按 hook 定义 hash 记账；变更后重新审核）；`/hooks` 菜单管理；`--dangerously-bypass-hook-trust` 单次跳过。
- 插件 hook 环境变量：`PLUGIN_ROOT`、`PLUGIN_DATA`（Codex 扩展；同时设 `CLAUDE_PLUGIN_ROOT`/`CLAUDE_PLUGIN_DATA` 兼容）。

IR 承载：【无承载】→ B1（同 Claude hooks；Codex 事件集是 Claude 的子集，标准化可行）。

## 6. Rules（.rules / execpolicy）

来源：rules 页（experimental）。

| 项 | 语义 | IR 承载 |
|---|---|---|
| `~/.codex/rules/*.rules`、`<repo>/.codex/rules/*.rules`（trusted）、Team Config 位置 | 命令出站（沙箱外执行）规则 | 【无承载】→ B6 |
| 语言：Starlark（无副作用子集） | `prefix_rule(pattern, decision, justification, match, not_match)` | 【无承载】→ B6 |
| `pattern`：命令前缀 token 列表；元素为字面串或 `["view","list"]` 备选 | execvp 参数列表语义 | 【无承载】→ B6 |
| `decision`：`allow`（默认）/`prompt`/`forbidden`；多规则命中取最严（forbidden>prompt>allow）；requirements 中仅允许 prompt/forbidden | 决策 | 【无承载】→ B6 |
| `match`/`not_match`：加载时校验的内联用例 | 自测试 | 【无承载】→ B6 |
| shell 包装命令分割：`bash -lc` 等仅含安全算符（`&&`/`||`/`;`/`|`）的线性链用 tree-sitter 拆分逐条评估；含重定向/替换/变量/通配/控制流则整体作为单命令评估 | 求值语义 | 【无承载】→ B6 |
| `codex execpolicy check --rules <file> -- <cmd>` | 规则测试 CLI | n/a（非配置） |
| TUI 允许列表落点：`~/.codex/rules/default.rules` | 交互写入 | 【无承载】→ B6 |

## 7. Subagents / agent roles（`[agents]` 表）

来源：config-reference `agents.*` 键 + config-advanced（细节页 /codex/agent-configuration/subagents 本次未逐字段抓取——【存疑】角色 TOML 层的完整键表：已知可含模型/reasoning/指令等覆盖，核实方法：补抓该页或 openai/codex 仓库文档）。

| 项 | 语义 | IR 承载 |
|---|---|---|
| `agents.<name>`（自定义角色）+ `agents.<name>.config_file`（指向独立 TOML 层）+ `agents.<name>.description` | 角色声明；角色配置是**另一个 TOML 文件** | 【x- 承载】角色本体近似 PromptPack(agent)；config_file 外链结构 IR 无对应 → x- 存路径引用，目标文件可作为 Setting/资产采集（观察项 B11 关联） |
| 内置多 agent 工具（spawn_agent 等）与 `agents.default_subagent_model` 等 | 见 §1.1 | 【x- 承载】 |
| 无 frontmatter 文件形态（与 Claude `agents/*.md` 不同） | Codex 角色=配置层，不是 Markdown prompt 包 | 【无承载】（形态差异记录；导出 Claude subagent 时需合成 Markdown，属适配器降级规则） |

## 8. IR 击穿清单（无承载字段汇总 + 处理建议）

| # | 字段/结构 | 类别 | 建议处理 |
|---|---|---|---|
| B1 | hooks 全结构（11 事件 × matcher × handler 字段；hooks.json/内联/managed/插件四载体） | 实体缺口 | 同 Claude B1：新增 `hook.` 标准实体（Codex 事件集≈Claude 子集，标准化层用 Codex 集合，Claude 特有事件进 x-）；短期 `setting.codex.hooks` 不透明 value |
| B2 | requirements.toml 全部键（约 40 键：allowed_* 白名单家族、features 钉死、experimental_network、marketplaces、mcp_servers.identity、rules.prefix_rules、models.new_thread 等） | scope/实体缺口 | IR `origin.scope` 增加 `managed`；requirements 键以 Setting + scope=managed + x- 承载，sync 默认排除；`mcp_servers.<id>.identity` 的"身份匹配器"语义不进 McpServer 实体（属策略而非定义） |
| B3 | 插件体系（plugin.json、插件捆绑 skills/hooks/MCP、`plugins.<plugin>.mcp_servers.*` 治理键、`marketplaces.*`） | 实体缺口 | MVP 不建模 Plugin；settings 键 x- 透传；插件落地内容按落地形态采集 |
| B4 | （Codex 侧无对应 Claude B4 的凭证遮蔽深度结构；Codex 的凭证保护依赖 shell_environment_policy 过滤与沙箱） | — | 记录形态差异即可 |
| B5 | `mcp_servers.<id>` 的 `cwd`/`env_vars`/`env_http_headers`/`bearer_token_env_var`/`required`/`enabled_tools`/`disabled_tools`/`default_tools_approval_mode`/`tools.<tool>.approval_mode` | McpServer 字段缺口 | 建议 McpServer 增加标准字段候选：`cwd`、`enabled_tools`/`disabled_tools`（工具过滤维度 Claude 没有但属通用语义）；`env_http_headers`/`bearer_token_env_var`（环境变量间接寻址）需 IR 决定是否引入"env 引用"值形态（与 secretref 并列）；审批治理键 x- 透传 |
| B6 | .rules / execpolicy（Starlark prefix_rule 体系） | 实体缺口 | MVP：作为独立文件资产整体采集（blob 化 + origin 记录），不做字段级结构化；长期可评估 `rule.` 实体（与 Claude permissions.allow/deny 的 Bash 规则存在语义重叠区，但 Codex 的 token 级前缀匹配与 Claude 的 glob 匹配不可互译） |
| B7 | `tui.keymap.<context>.<action>` 二维开放键位表 | 动态键空间 | x- 整体透传（Setting.value 可装，键名动态段无校验问题；标注即可） |
| B8 | 运行时状态键（`notice.*`、`tui.model_availability_nux.*`、`windows_wsl_setup_acknowledged`、`projects.<path>.trust_level`） | 非配置数据 | 不采集；`trust_level` 若采集须 x- 且 sync 排除（信任是机器本地决策） |
| B9 | Memories 派生数据（`~/.codex/memories`） | 派生数据 | 默认不采集（同 Claude B9） |
| B10 | profile 文件机制（`~/.codex/<name>.config.toml`） | 结构映射 | 【标准字段】可映射为 IR 的 profile 概念（cfg4ai profile ≈ Codex profile 文件层）；采集时按独立 profile 处理，origin.path 记录文件名 |

## 9. 真实样本

1. **官方 profile 文件示例**（来源：https://developers.openai.com/codex/config-file/config-advanced）：
```toml
# ~/.codex/deep-review.config.toml
model = "gpt-5.5"
model_reasoning_effort = "xhigh"
approval_policy = "on-request"
model_catalog_json = "/Users/me/.codex/model-catalogs/deep-review.json"
```

2. **官方自定义 provider 示例**（来源：config-advanced 页）：
```toml
model = "gpt-5.6-terra"
model_provider = "proxy"
[model_providers.proxy]
name = "OpenAI using LLM proxy"
base_url = "http://proxy.example.com"
env_key = "OPENAI_API_KEY"
[model_providers.proxy.auth]
command = "/usr/local/bin/fetch-codex-token"
args = ["--audience", "codex"]
timeout_ms = 5000
refresh_interval_ms = 300000
```

3. **官方 sandbox/approval 组合示例**（来源：config-advanced 页）：
```toml
approval_policy = "untrusted"
approvals_reviewer = "user"
sandbox_mode = "workspace-write"
allow_login_shell = false
[sandbox_workspace_write]
exclude_tmpdir_env_var = false
exclude_slash_tmp = false
writable_roots = ["/Users/YOU/.pyenv/shims"]
network_access = false
[auto_review]
policy = """Use your organization's automatic review policy."""
```

4. **官方 hooks.json 示例**（来源：https://developers.openai.com/codex/hooks）：
```json
{
  "description": "Optional lifecycle hooks for this workspace.",
  "hooks": {
    "SessionStart": [
      { "matcher": "startup|resume",
        "hooks": [
          { "type": "command",
            "command": "python3 ~/.codex/hooks/session_start.py",
            "statusMessage": "Loading session notes",
            "additionalContextLimit": 5000 }
        ] }
    ]
  }
}
```
同页含内联 TOML 形态（`[[hooks.PreToolUse]]` + `[[hooks.PreToolUse.hooks]]`）与 managed hooks（requirements.toml `hooks.managed_dir`）示例。

5. **官方 .rules 示例**（来源：https://developers.openai.com/codex/agent-configuration/rules）：
```python
prefix_rule(
    pattern = ["gh", "pr", "view"],
    decision = "prompt",
    justification = "Viewing PRs is allowed with approval",
    match = ["gh pr view 7888", "gh pr view --repo openai/codex"],
    not_match = ["gh pr --repo openai/codex view 7888"],
)
```

6. **官方 AGENTS.md 示例**（来源：https://developers.openai.com/codex/agent-configuration/agents-md）：`~/.codex/AGENTS.md` 的 "Working agreements" 小节与 `services/payments/AGENTS.override.md` 嵌套覆盖示例；`## Code Review Rules` 小节示例。

7. **官方 skill 示例与 curated 仓库**：https://github.com/openai/skills （build-skills 页引用：`skills/.curated/gh-fix-ci`、`skills/.curated/pdf`、`skills/.curated/linear`）；含 `agents/openai.yaml` 的 interface/policy/dependencies 完整示例。

8. **官方 otel 配置示例**（来源：config-advanced 页）：
```toml
[otel]
exporter = { otlp-http = {
  endpoint = "https://otel.example.com/v1/logs",
  protocol = "binary",
  headers = { "x-otlp-api-key" = "${OTLP_TOKEN}" }
}}
```

9. **开源仓库**：https://github.com/openai/codex —— Codex CLI 本体仓库（含 docs/ 与源码中 config 键定义 `codex-rs/`，otel 指标名集中于 `codex-rs/otel/src/metrics/names.rs`）。【存疑】仓库内 `docs/config.md` 类文档与文档站的同步时差，核实方法：对比仓库 main 分支文档。

## 10. 证伪回答：若 IR 只服务 Claude Code + Codex CLI，v0.2 需改什么（Codex 侧输入）

1. **scope 枚举扩展**（与 Claude 侧同一诉求）：至少增加 `managed`（requirements.toml / `/etc/codex/skills`）与 `local` 概念；Codex 的 project 层还附带"trusted 才加载"语义，信任状态（`projects.<path>.trust_level`）不应入库。
2. **新增 Hook 实体**：Codex 11 事件基本为 Claude 子集，`hook.` 实体 + handler 字段（command/timeout/statusMessage/async/additionalContextLimit）可标准化；Codex 的 `commandWindows` 双平台命令字段值得进标准层（跨平台配置是 cfg4ai 的核心场景）。
3. **McpServer 字段补充**：`cwd`、`enabled_tools`/`disabled_tools`（通用语义）；`startup_timeout` 双向换算规则已在 IR §3.2 立规（秒↔毫秒）但需覆盖 Codex 的 `startup_timeout_ms` 别名；Codex 无 `disabled` 而是 `enabled`（IR 已立规取反）；Codex 无 `env_file`（VS Code 概念，Codex 导出忽略）；`env_http_headers`/`bearer_token_env_var` 的"环境变量间接寻址"形态需 IR 决策（建议作为 `headers` 值的 `{env: VAR}` 结构化占位或 x-）。
4. **Instruction 合并语义**：Codex 是"项目根→cwd 逐目录拼接 + override 文件取代同目录 base"，与 IR 的 scope 两段式分层不同；`AGENTS.override.md` 的成对取代语义需 x- 或 merge_policy 注记。
5. **profile 概念对齐**：Codex profile 文件（`~/.codex/<name>.config.toml`）与 cfg4ai profile 是近同构概念，ADAPTERS 应声明映射（一个 Codex profile 文件 ≈ 一个 cfg4ai named profile 的一层）。
6. **Setting 实体的键空间开放度**：Codex config.toml 含大量动态键段（`mcp_servers.<id>.*`、`apps.<id>.*`、`tui.keymap.<context>.<action>`、`notice.*`），IR Setting 的 `setting.<tool>.<key>` 三段式 id 需明确 `<key>` 允许含点号路径与动态段（当前校验规则 name 字符集 `[a-z0-9][a-z0-9-]*` 与动态段有张力 → 建议放宽或规范化为首段静态前缀）。
7. **rules/execpolicy 暂以 blob 资产处理**：MVP 不做字段级结构化（见 B6）。
8. **`disabled` 极性之外的第二组极性翻转**：Codex `skills.config[].enabled`、`mcp_servers.<id>.enabled` 均为正极性，与 IR `disabled` 负极性约定相反——适配器取反规则需在 ADAPTERS 逐一列明（IR §3.2 只写了 MCP 一例）。
