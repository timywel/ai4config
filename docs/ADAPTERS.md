# 适配器规范与工具配置地图

> 版本：v0.3（吸收 R1 字段级调研，档案见 `docs/research/`）｜ 对应架构文档 §6
>
> 核查基线：§3.1–§3.5 经官方文档实检 + 逐字段 inventory（`research/field-inventory-*.md`）；§3.6 扩展工具已完成调研卡片（`research/tool-survey-{a,b}.md`）。

## 1. 适配器接口

```go
// internal/adapters/adapter.go

type ToolID string

type ToolMeta struct {
    ID           ToolID
    DisplayName  string
    MinVersion   string
    MaxVersion   string
    Capabilities CapabilitySet
}

type SupportLevel int
const (
    SupportNone SupportLevel = iota
    SupportPartial             // 配 Note 说明边界
    SupportFull
)

type EntityKind string // instruction | mcp | skill | agent | command | workflow | hook | setting

type CapabilitySet map[EntityKind]Capability // Capability{ Level SupportLevel; Note string }

type Location struct {
    Scope   ir.Scope // 五层：managed | remote | local | project | global
    Root    string
    Version string
    Running bool     // 目标进程运行中（热重载提示依据）
}

type WrittenFile struct {
    Path string
    Hash string // 写出内容 sha256（Verify 与导出清单用）
}

type ExportOpts struct {
    ProjectRoot string
    DryRun      bool
    Force       bool
    // AIAssist 已移除——AI 转换由引擎层主导
}

type Adapter interface {
    Meta() ToolMeta
    Detect(ctx context.Context) ([]Location, error)
    Import(ctx context.Context, loc Location) (*ir.Bundle, error)
    Export(ctx context.Context, b *ir.Bundle, opts ExportOpts) ([]WrittenFile, error)
}
```

注册：各包 `init()` 调 `adapters.Register(...)`；`internal/adapters/all/all.go` 聚合 blank import，由 `cmd/cfg4ai` 引用。

## 2. 适配器实现规范

1. **只读 Detect**；进程检测 best-effort。
2. **保真 Import**：无法映射进 x-；保真边界 IR-SCHEMA §1.3。
3. **写入走 `internal/atomicfile`**（写入协议 ARCHITECTURE §5.3），禁手写。
4. **版本护栏**：超出 [Min,Max] 告警继续。
5. **Golden-file 双向测试**，显式覆盖 YAML 注释/键序保留。
6. **幂等**：重复 Export 第二次全 no-op。
7. **回写路由**：声明 `default_write_target`；非专用文件只许局部 patch。
8. **物化布局唯一决定权**（§3 各表"导出布局"列）。
9. **符号链接**：采集 lstat 不跟随；写入 EvalSymlinks 穿透。
10. **目标语义差异消化（D16）**：IR 语义唯一权威。目标工具的合并/极性/覆盖语义差异由适配器双向转换，并登记到下表：

| 工具 | 差异点 | 适配器处理 |
|------|--------|-----------|
| claude-code | settings 数组跨 scope 拼接去重（非整体替换） | Import 时按来源拆分标记；Export 时按目标语义渲染拼接 |
| codex | `enabled` 正极性 vs IR `disabled` | 双向取反（不止 MCP 一处，逐实体列明于适配器 README） |
| codex | AGENTS.override.md 成对取代 + 逐目录拼接 | Import 映射 Instruction subtree/priority；override 文件记 x-codex |
| cline | skills 全局优先于项目（优先级反转） | Import 时 priority 按目标实际语义标定 |
| roo | 同 slug mode 整体覆盖（非浅合并） | Import/Export 按整体覆盖转换 |
| gemini | `security.allowedEnvironmentVariables` 新旧键名并存 | 双键兼容读取，Export 写新键 |
| copilot | `env` 值允许 number/null（非纯 string） | 类型规范化并记 x-copilot |

## 3. 工具配置地图（逐字段细节见 research/field-inventory-*.md）

### 3.1 Claude Code（`claude-code`）— P0 ｜字段档案 `field-inventory-claude-code.md`（~130+ settings 键、hooks 31 事件、subagent 16 字段、SKILL.md 20 字段）

| 实体 | 全局 | 项目 | 导出布局 |
|------|------|------|---------|
| instruction | `~/.claude/CLAUDE.md` | `<proj>/CLAUDE.md`、`<proj>/.claude/CLAUDE.md`（向上逐级拼接）、`CLAUDE.local.md`（scope=local） | 全局→`~/.claude/CLAUDE.md`；项目→`<proj>/CLAUDE.md` 单文件物化（边界注释拼接） |
| settings | `~/.claude/settings.json`；`~/.claude.json`（局部 patch 专用） | `.claude/settings.json`、`.claude/settings.local.json`（scope=local） | 按 origin.path 回写；新条目→default_write_target |
| mcp | `~/.claude.json`（user scope；local scope 按项目路径隔离） | `<proj>/.mcp.json`（`mcpServers`） | user→`~/.claude.json` 局部 patch；项目→`.mcp.json` 整写 |
| agents / skills | `~/.claude/agents/*.md`、`~/.claude/skills/<name>/SKILL.md` | `.claude/agents/`、`.claude/skills/` | 同路径 |
| commands（legacy，已并入 skills） | ⚠️ `~/.claude/commands/*.md` 现行文档无依据 | `.claude/commands/*.md`（兼容，同名 skill 优先） | 默认导出为 skills |
| hooks | settings.json `hooks` 键（31 事件×matcher×handler） | 同左（项目 settings） | 回写 settings.json hooks 键（局部 patch） |
| rules | `~/.claude/rules/*.md` | `.claude/rules/*.md`（frontmatter `paths`→file_patterns） | 同路径 |

managed 层（`/Library/...`、`C:\Program Files\ClaudeCode\` 等）：只读采集，scope=managed，不物化。插件体系（plugin.json/marketplace.json/enabledPlugins）：x- 承载。
已知含明文文件：`~/.claude.json`、`.mcp.json`。

### 3.2 Codex CLI（`codex`）— P0 ｜字段档案 `field-inventory-codex.md`（config.toml ~160 键、mcp_servers 23 键、requirements.toml ~40 键）

| 实体 | 全局 | 项目 | 导出布局 |
|------|------|------|---------|
| instruction | `~/.codex/AGENTS.md`（`AGENTS.override.md` 优先） | 逐目录拼接、就近优先；上限 `project_doc_max_bytes`（32KiB） | 全局→`~/.codex/AGENTS.md`；项目按 `subtree` 还原目录 |
| settings / mcp | `~/.codex/config.toml`（`[mcp_servers.<id>]`） | `<proj>/.codex/config.toml`（trusted-gate；机器级键不可覆盖） | TOML 整块重写+快照兜底；项目级跳过机器级键+Warning |
| hooks | config.toml `hooks` / `hooks.json`（11 事件） | 项目 hooks（默认不执行语义记 x-codex） | 回写 config.toml |
| rules/execpolicy | Starlark `prefix_rule` 体系 | 同左 | **x-codex 保真存储，不跨工具翻译**（可执行策略降级 Warning） |
| skills / profiles | `~/.codex/skills/`、`~/.codex/<name>.config.toml` profiles | 项目级同构 | profile 文件映射 cfg4ai profile（近同构，映射表写入适配器 README） |
| 认证（secret） | `~/.codex/auth.json` | — | 仅 secretref 抽取 |

managed：`requirements.toml`（~40 键白名单/钉死）scope=managed 只读采集。运行时状态（notice.*/trust_level/auto-memories）不采集。
已知含明文文件：`auth.json`、`config.toml`。

### 3.3 VS Code Copilot（`copilot`）— P1 ｜字段档案 `field-inventory-copilot.md`（~89 字段）

| 实体 | 全局 | 项目 | 导出布局 |
|------|------|------|---------|
| instruction | profile user data `*.instructions.md`；Agent Host：`~/.copilot/instructions`（兼容 `~/.claude/rules`）；settings 内嵌 instructions 自 1.102 deprecated（仅 3 类残留） | `.github/copilot-instructions.md`、`.github/instructions/*.instructions.md`（frontmatter `name/description/applyTo`） | 全局→user data instructions；项目→`.github/instructions/` |
| prompts | user profile prompts 目录（Agent Host 不使用） | `.github/prompts/*.prompt.md` | 同路径 |
| agents | — | `.github/agents/*.agent.md`（24 字段含 handoffs/mcp-servers→PromptPack 标准字段） | 同路径 |
| mcp | user profile `mcp.json`；Agent Host：`~/.copilot/mcp-config.json` | `.vscode/mcp.json`（`servers`+文件级 `inputs`/`sandbox`→mcp.yaml `file_extensions`） | 全局→user mcp.json；项目→`.vscode/mcp.json`（JSONC 注释免责） |
| settings | `%APPDATA%\Code\User\settings.json` | `.vscode/settings.json` | 按 origin.path 回写 |

已知含明文文件：`.vscode/mcp.json`、user `mcp.json`。

### 3.4 Zhanlu 湛卢（`zhanlu`）— P1 ｜字段档案 `field-inventory-zhanlu.md`（本地实证 22 字段）

| 实体 | 全局 | 项目 | 导出布局 |
|------|------|------|---------|
| 主配置 | `~/.config/zhanlu/zhanlu.json`（✅ 实证；本机仅见 `permission` 顶级键，providers/mcp 段**待校准**） | `.zhanlu/`（✅ 实证含 agent-manager.json——运行时锁文件**不采集**）；`kilo.json` 待校准 | 按 origin.path 回写；防御式 Detect（键缺失容忍） |
| instruction | 全局 AGENTS.md 约定（待校准） | `<proj>/AGENTS.md` | 同路径 |
| skills | `~/.agents/skills/<name>/SKILL.md`（✅ 实证 25 个；`activation: model-decision` 语义路由） | 项目级（待校准） | 同路径 |
| agents / commands | `~/.agents/`（待校准） | `.kilo/agent/*.md`、`.kilo/command/*.md`（待校准） | 同路径 |

锁文件（.skill-lock.json、agent-manager.json）不采集。已知含明文文件：`zhanlu.json`（providers api_key，若存在）。

### 3.5 Gemini CLI（`gemini`）— P1 ｜字段档案 `field-inventory-gemini.md`（~240 键）

> ⚠️ 时效：官方公告将过渡为 Antigravity CLI（2026-06-18 起）。排期与版本护栏需关注。

| 实体 | 全局 | 项目 | 导出布局 |
|------|------|------|---------|
| instruction | `~/.gemini/GEMINI.md` | `<proj>/GEMINI.md`（父目录向上+JIT；`context.fileName` 可自定义） | 同路径单文件 |
| settings / mcp | `~/.gemini/settings.json`（顶级 `mcpServers`） | `.gemini/settings.json` | 局部 patch；四层模型中 system-defaults/override 两层 scope=managed 只读 |

已知含明文文件：`settings.json`（mcpServers env）。

### 3.6 扩展工具（调研已完成；卡片见 research/tool-survey-{a,b,c}.md）

- **P1 追加**（2026-08-16 补充调研，tool-survey-c.md）：
  - `claude-desktop`（Claude Desktop 桌面应用）：`claude_desktop_config.json`（Win `%APPDATA%\Claude\`、macOS `~/Library/Application Support/Claude/`），文件面仅 `mcpServers`（stdio）；远程连接器走 UI/OAuth 不落公开文件；扩展格式 DXT 已更名 MCPB（`.mcpb`）。与 claudecode 适配器共享 MCP 适配代码，轻量接入。
  - `grokbuild`（xAI Grok Build）：`~/.grok/config.toml` + 项目 `.grok/config.toml`（向上至 git root）+ 五层企业配置链（含 requirements pin 层）；hooks 14 事件、SKILL.md、subagents、Rhai 脚本工作流（进 x-）、MCP（stdio/http/OAuth）；零配置兼容读取 Claude Code 生态——双向迁移价值高。
- **P2 候选**：Cursor / Windsurf→Devin / Aider / Cline / Roo Code（tool-survey-a）；OpenCode / Amp / Goose / Zed / JetBrains Junie / Trae（tool-survey-b）；Hermes（MCP 客户端字段最深：mTLS/sampling/elicitation；含"机写内容"采集纪律问题）/ OpenClaw（多渠道 agent 网关；JSON5 严格 schema+热重载，export 风险高，**只读采集先行**，观察名单）。
- 接入前按卡片复核时效项（Windsurf 双目录、Cline .clineignore legacy、Zed Rules 已废弃、Claude Desktop 与 Claude Code 产品边界等）。

## 4. 新增适配器 Checklist

- [ ] 官方文档核对 + 逐字段 inventory（补齐 research/field-inventory-\<tool\>.md）+ 回答"IR 要改什么"（证伪纪律）
- [ ] 实现 `Adapter` 四方法（含 ctx）+ CapabilitySet（SupportLevel）
- [ ] x- 字段清单；`default_write_target`；局部 patch 清单；§2.10 差异表登记
- [ ] golden-file 双向（含注释/键序保留用例）+ 对抗用例库中该工具相关条目回归
- [ ] 敏感字段清单；含明文文件清单
- [ ] 平台路径矩阵（win/mac/linux+麒麟冒烟）
- [ ] doctor 可见

## 5. 能力降级对照（导出时，唯一定义处）

| IR 实体 | 目标不支持时降级为 |
|---------|-------------------|
| skill / agent / command / workflow | 两级规则：有最近概念→映射该形态；无→instruction 附录（含 frontmatter），记 Warning |
| mcp | 跳过 + Warning |
| settings（无对应键） | 跳过 + Warning |
| workflow 复杂编排（steps/parameters 超目标能力） | 正文内联描述 + Warning |
| hook（目标无 hook 体系 / 事件不可翻译） | x- 原样保留 + Warning；不生成目标文件 |
| 可执行策略/插件（codex execpolicy、cline/amp TS 插件） | **不翻译**，x- 保真存储 + Warning（有意边界，ARCHITECTURE §1.3） |
| instruction.file_patterns | 保留正文，作用域说明并入 frontmatter 注释 + Warning |
| activation=scene（目标无场景概念） | 降为 model-decision + Warning |

降级结果在 export 输出的 Warnings 中逐条列出，可知、可回滚。
