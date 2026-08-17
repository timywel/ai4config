# IR 数据模型规范

> 版本：v0.3（吸收 R1 字段级调研 + R2 红队审计；决策依据见 [research/RESEARCH-SUMMARY.md](./research/RESEARCH-SUMMARY.md) D1–D17）｜ 序列化格式：YAML（指令体为 Markdown）

IR（Intermediate Representation）是所有工具配置的规范中间表示。设计原则：**可校验、可合并、可往返（边界见 §1.3）、Git 友好**。IR 是 SSOT 的索引与视图层；任何 IR 表达不了的内容由 blobs 兜底（保真保险丝）。

## 1. 通用约定

### 1.1 实体头（所有实体共有）

```yaml
id: mcp.filesystem            # <type>.<name>；解析规则：首个点号分隔 type，其余全部为 name（D2）
ir_version: 1
origin:
  tool: claude-code           # 已注册适配器 id
  path: "~/CLAUDE.md"         # raw 变量形态（~/ 或 %APPDATA%），跨机可移植
  scope: global               # 五层，见 §1.2（D1）
  collected_at: 2026-08-16T10:00:00+08:00
  raw_hash: sha256:...        # 源文件原始字节 sha256（增量比对）
  stored_hash: sha256:...     # 脱敏后入库内容 sha256
  raw_blob: sha256:...        # 可选：高注释密度源文件整体快照（overlay 兜底）
tombstone: false              # 墓碑（§2.3；判定前提见§2.3 防误判规则 D8）
x-claude-code: {}             # 扩展命名空间，只读透传
```

规则：
- `id` 在 profile 内唯一；name 段字符集 `[a-zA-Z0-9][a-zA-Z0-9._-]*`（放行点号与大写，承载 `chat.mcp.access`、`mcp_servers.<id>.*` 等真实键路径——D2）。
- **`x-<tool>` 生命周期**：绑定"最近一次源自该 tool 的 import"；异构再采集只更新标准字段与自身 x-，不触碰他工具 x-；导出回某工具时标准字段以最新值为准、x- 仅补特有字段。
- 时间 RFC 3339；文本 UTF-8/LF。hash 为条目级（规范化序列化后计算；raw_hash 例外，对原始字节）。
- **占位符回采保护（D8）**：导出物中的 secretref 占位符或空值被再次采集时，**永不覆盖**已有 secretref 条目——冲突记 Warning 并保持原值。

### 1.2 Scope 五层模型（D1）

```
managed  （企业下发/MDM：Claude managed、OpenCode MDM、Windsurf System、Gemini system-override；只读，默认不物化）
remote   （组织远程订阅：Cursor 团队规则、OpenCode remote .well-known）
local    （项目内私人层：settings.local.json、CLAUDE.local.md；sync 排除）
project  （项目层）
global   （用户层）
```

- 优先级：`managed > remote > local > project > global`（依据 RESEARCH-SUMMARY D1）。
- `origin.scope` 记录采集来源层；merge 时 concat 排序 = 低优先级在前（global→remote→project→local）；**managed 层采集仅供审计与差分比对，不参与导出物化**（除非适配器显式声明可写）。
- 替代 v0.2 的 `local: true` 布尔位（迁移：local=true → scope=local）。

### 1.3 词表与保真承诺边界

| 实体 | id 前缀 | 存储 | CLI --type |
|------|--------|------|-----------|
| Instruction | `instruction.` | `instructions/<name>.md` | instruction |
| McpServer | `mcp.` | `mcp.yaml` | mcp |
| Skill/Agent/Command/Workflow | `skill./agent./command./workflow.` | `<kind>s/<name>/` | skill/agent/command/workflow |
| Hook（v0.3 新增，D3） | `hook.` | `hooks.yaml` | hook |
| Setting | `setting.<tool>.<key>` | `settings.yaml` | setting |

保真分级：**字段级零丢失**（强承诺）/ JSONC·TOML 注释键序**不保证**（免责）/ `origin.raw_blob` + `render_mode: overlay` 可选兜底。

## 2. 合并语义

### 2.1 策略定义

```yaml
merge_semantics:
  merge-by-id: field-level-shallow   # 标量/object 字段覆盖；数组整体替换；未出现字段继承
  override: entry-replace
  concat: layer-ordered              # §1.2 层级序 + 层内 priority 升序 + origin.path 字典序
```

**IR 语义唯一权威（D16）**：目标工具自身的合并语义差异（Claude settings 数组跨 scope 拼接去重、Cline skills 全局优先反转、Roo 同 slug 整体覆盖、Codex enabled 正极性）由**适配器在 Import/Export 双向转换中消化**，并列入 ADAPTERS「目标语义差异表」；IR 层语义不因目标工具改变。

### 2.2 manifest.yaml 与 merge_policy

```yaml
ir_version: 1                   # profile 整体版本；实体 ir_version ≤ 本版本
profile:
  name: global
  kind: global                  # global | project
merge_policy:                   # 可选覆盖默认
  instructions: concat          # concat | project-only | global-only
  mcp_servers: merge-by-id
  skills/agents/commands/workflows/hooks: merge-by-id
  settings: merge-by-id
```

启动时对 profile 执行链式 ir_version 迁移；实体版本高于实现版本则拒绝并提示升级。

### 2.3 删除传播（墓碑）与防误判（D8，红队 T-01/T-06/T-07 修复）

- **墓碑判定前提**：仅当源目录**存在且可读**、但上次采集自该 `(origin.tool, origin.path)` 的条目本次消失时，才标记 `tombstone: true`。**源目录不存在（盘掉线/未挂载/权限不足）→ 中止该 Location 采集并报 Warning，绝不标记墓碑**。
- **遮蔽规则**：项目层墓碑在 merged 视图中**遮蔽**全局同 id 条目（防"已删除复活"）。
- **reconcile 边界**：墓碑 reconcile 只作用于本次实际采集的 Location 对应的 `(origin.tool, origin.path)` 集合（多 clone 共存时互不踩踏）。
- **空集导出保护**：merged Bundle 条目数为 0 且目标已有文件时，必须 `--force` + 显式警告文案。
- `cfg4ai prune` 物理清除墓碑并级联清理 keyring 孤儿条目。

## 3. 实体 Schema

### 3.1 Instruction

```markdown
---
id: instruction.coding-style
ir_version: 1
name: 编码规范                 # 可选（D6；Copilot 语义路由运行时字段）
description: 团队 Go 编码规范   # 可选（D6；model-decision 路由依据）
activation: always            # always | glob | manual | model-decision（D4 统一枚举）
applies_to: [claude-code]     # 缺省 = [origin.tool]
file_patterns: ["**/*.py"]    # activation=glob 时生效（copilot applyTo/cursor globs）
subtree: ""                   # 子目录作用域（Codex 子目录 AGENTS.md、nested AGENTS.md）
priority: 100                 # 层内排序；默认 global=100/project=200/local=200/remote=150
language: zh
imports:                      # @path 引用结构化清单
  - { path: ./docs/style.md, blob: sha256:..., resolved: true }
roundtrip_policy: preserve    # preserve | inline
origin: { tool: claude-code, path: "~/CLAUDE.md", scope: global, collected_at: 2026-08-16T10:00:00+08:00, raw_hash: sha256:aaa, stored_hash: sha256:bbb }
---

# 编码规范
- 所有回复使用中文
```

导出物化：多条目拼接用 `<!-- cfg4ai:begin <id> -->`/`<!-- cfg4ai:end -->` 边界；布局由目标适配器唯一决定。

### 3.2 McpServer（v0.3 字段扩充，D7）

```yaml
servers:
  - id: mcp.filesystem
    ir_version: 1
    name: filesystem              # 目标工具内原始键名（导出键名唯一来源）
    transport: stdio              # stdio | sse | http | ws（streamable-http 归一化为 http）
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/data"]
    cwd:                          # 可选（D7；codex/copilot/gemini 共有）
    env:
      API_TOKEN: secretref://cfg4ai/global/mcp.filesystem/env.API_TOKEN
      HOME_DIR: "{env:HOME}"      # 环境变量间接寻址规范形态（D7；codex env_vars/bearer_token_env_var 映射此形态）
    env_file:
    url:
    headers: {}
    headers_helper:               # 可选（D7；claude headersHelper 动态取头）
    timeout: { startup_ms: 5000, tool_sec: 60 }   # gemini 特有语义进 x-gemini（单位语义双失真不硬映射）
    enabled_tools: []             # 可选（D7；codex enabled_tools / gemini includeTools）
    disabled_tools: []            # 可选（codex disabled_tools / gemini excludeTools）
    trust:                        # 可选 bool（gemini trust：绕过确认）
    auto_approve: []              # 可选 string[]（D7；cline autoApprove/roo alwaysAllow 工具白名单）
    oauth:                        # 可选 object（cursor 静态 OAuth：client_id/scopes 等；密钥值一律 secretref）
    disabled: false               # codex enabled 正极性由适配器取反（D16）
    per_machine: false
    origin: { tool: claude-code, path: ".mcp.json", scope: project, collected_at: 2026-08-16T10:00:00+08:00, raw_hash: sha256:ccc, stored_hash: sha256:ddd }
    x-vscode: {}                  # 文件级 inputs[]/sandbox{} 归属：挂在 mcp.yaml 的 file_extensions（见下）
```

**文件级扩展位（修复含混）**：`mcp.yaml` 支持顶层 `file_extensions:` 键承载文件级数据（VS Code `inputs[]`/`sandbox{}` 等），不再挂到单个 server 的 x- 下。

### 3.3 Skill / Agent / Command / Workflow（PromptPack，v0.3 扩充 D5/D14）

```
skills/code-review/
├── skill.yaml  ├── prompt.md  └── assets/   # assets 物理存此；blobs 仅存采集快照
```

```yaml
# skill.yaml
id: skill.code-review
ir_version: 1
kind: skill
name: code-review
description: 本地代码评审            # model-decision 路由依据（zhanlu 语义路由=cursor 智能模式）
activation: model-decision           # D4：always | model-decision | glob | manual | scene
invocation: /review                  # 可选：调用名（slash-command/mention 形态）
file_patterns: []                    # activation=glob 时生效（skill paths、cursor globs）
scene:                               # 可选（trae scene：git_message 等场景标识）
model:                               # 可选 string|string[]（D5；claude subagent/copilot agent/roo mode）
tools: []                            # 可选 string[]（D5；工具白名单；disallowed 进 x-）
mcp_servers: []                      # 可选（D5；引用 mcp.* id 列表；内联定义进 x-，承载 amp skill↔MCP 绑定/copilot agent mcp-servers/claude frontmatter 内联）
user_invocable: true                 # 可选（D5；copilot user-invocable 等）
argument_hint:                       # 可选 string（copilot argument-hint）
# Workflow 专用可选字段（D14，承载 goose recipes 公共面）：
parameters: []                       # {name, description, required, default}
steps: []                            # 有序步骤（复杂编排/子配方/重试进 x-）
origin: { tool: zhanlu, path: ".kilo/agent/review.md", scope: project, collected_at: 2026-08-16T10:00:00+08:00, raw_hash: sha256:eee, stored_hash: sha256:fff }
x-zhanlu: { mode: primary }
```

- 合并语义：`merge-by-id`。assets 文件可带 `mode: 0o755`（仅 Unix）。
- 导出降级：ADAPTERS §5 两级规则（唯一定义处）。
- Roo custom modes 的 `groups+fileRegex` 权限维度、Zed agent.profiles 的工具集预设：**x- 承载**（权限模型标准化为未来候选，D17）。

### 3.4 Hook（v0.3 新增一等实体，D3）

```yaml
# hooks.yaml
hooks:
  - id: hook.pre-tool-guard
    ir_version: 1
    event: pre-tool-use            # 标准事件交集：session-start | session-end | pre-tool-use | post-tool-use | notification | stop | user-prompt-submit | pre-compact（工具特有事件进 x-）
    matcher: { tool: "Bash" }      # 事件匹配器（工具/模式，结构随事件）
    handler:
      type: command                # command | http | prompt | mcp_tool | agent
      command: ./scripts/guard.sh
      command_windows: ./scripts/guard.ps1   # 跨平台双命令（cfg4ai 核心场景）
      timeout_sec: 30
    origin: { tool: claude-code, path: "~/.claude/settings.json", scope: global, collected_at: 2026-08-16T10:00:00+08:00, raw_hash: sha256:ggg, stored_hash: sha256:hhh }
    x-codex: {}
```

- 合并语义：`merge-by-id`。事件名不可翻译时保留 x- 原样 + Warning。

### 3.5 Setting

```yaml
entries:
  - id: setting.copilot.chat.mcp.access   # 三段式+点号 key 路径（D2）
    ir_version: 1
    key: chat.mcp.access                  # 目标文件内的点号路径（嵌套表/键）
    value: registry
    tool_scope: [copilot]
    origin: { ... }
  - id: setting.claude-code.permissions
    key: permissions
    value: { allow: ["Bash(npm run test:*)"] }   # 工具特有结构=不透明 value，不参与跨工具翻译
    tool_scope: [claude-code]
  - id: setting.cline.ignorefile          # ignore 文件家族承载形态（D15）
    key: ignorefile
    value: ["secrets/**", "*.env"]        # 行列表；强制强度差异进 x-
    tool_scope: [cline]
```

- 合并语义：`merge-by-id`（条目级）。
- **回写路由**：origin.path 写回原文件；新条目写适配器 `default_write_target`；非专用文件（`~/.claude.json` 等含运行时状态）只许局部 patch；`export --project` 物化分流 `materialize: inherited-skip（默认）| inherited-inline`。
- **运行时状态/派生数据不采集**（D8 澄清）：OAuth 会话、`projects.*.trust_level`、`notice.*`、auto-memories、`modelConfigs` 内建默认值等——差分采集（仅与默认值不同的才入库）。

### 3.6 Secret 引用

```yaml
value: secretref://cfg4ai/<profile>/<entity-id>/<field>   # key 字符集 [a-zA-Z0-9./_-]（放行大写：env 变量名如 API_TOKEN）
```

- 三级后端降级链（ARCHITECTURE §9）；实体记录 `secret_backend: keyring|file|none`。
- 单条上限 2KB；实体删除/改名/prune 级联清理。
- **回采保护**（§1.1）：导出物中的占位符/空值再采集时不覆盖已有 secretref。

## 4. Bundle（迁移管线内存模型）

```go
type Bundle struct {
    IRVersion int
    Scope      Scope            // global | project | local | remote | managed | merged（merged 仅用于 export，不回写）
    Instructions []Instruction
    MCPServers []MCPServer
    Skills/Agents/Commands/Workflows []PromptPack
    Hooks      []Hook
    Settings   []SettingEntry
    Warnings   []Warning        // 非空 → CLI 退出码 5
}
```

- merged Bundle 条目 origin 取胜出层；墓碑遮蔽在 merge 阶段完成（§2.3）。
- 引擎内建 `map[id]entity` 索引。
- Export 前按能力矩阵（SupportLevel）检查降级，记入 Warnings。

## 5. 校验规则（Import/Export 均执行，v0.3 共 12 条）

1. `id` 唯一；首个点号分隔 type 与 name；name 字符集 `[a-zA-Z0-9][a-zA-Z0-9._-]*`；Setting 必须 `setting.<tool>.<key>` 形态且 tool 已注册。
2. `transport` 枚举合法；`stdio` 必有 `command`，`sse/http/ws` 必有 `url`。
3. `x-<tool>`、`applies_to`、`origin.tool`、`tool_scope` 取值为已注册适配器 id（或 `all`）；未知告警保留。
4. 敏感扫描分级：结构化字段命中→默认抽取可否决；自由文本命中→仅 Warning 不改写。规则库外置+豁免清单。
5. frontmatter 必填字段齐全（`id`、`ir_version`）。
6. secretref 解析：export 查询失败→Warning+占位符导出；doctor 报 dangling 清单。
7. PromptPack/Instruction 的 id 末段等于所在目录/文件名（规范化等价）。
8. `merge_policy` 键为已知实体类型；instructions 仅 `concat|project-only|global-only`。
9. `imports` 引用图无环；解析失败 Warning 不阻断。
10. 墓碑不参与导出；遮蔽规则在 merge 阶段生效；墓碑标记前提满足 §2.3 防误判规则。
11. 实体 `ir_version` ≤ manifest 版本 ≤ 实现版本；高于实现版本拒绝并提示升级。
12. `activation`/`event`/`handler.type` 为词表内枚举；scene 仅当 activation=scene 时有意义；未知值进 x- 并 Warning。
