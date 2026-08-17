# 扩展工具配置地图调研（A 组）：Cursor / Windsurf / Aider / Cline / Roo Code

> 调研日期：2026-08-16 ｜ 调研人：生态调研扩展 A 组 ｜ 对应基线：IR-SCHEMA v0.2、ADAPTERS v0.2（基线五工具：claude-code / codex / copilot / zhanlu / gemini）
>
> 方法：官方文档实检（webfetch + 浏览器渲染）；所有结论标注来源 URL；无法核实的标【存疑】。
>
> ⚠️ 重大时效发现：**Windsurf 已被 Cognition 收购并更名为 Devin Desktop**，docs.windsurf.com 全站重定向至 docs.devin.ai；**Cline 官方宣布 `.clineignore` 将弃用**。详见各卡片 §6。

---

## 卡片 1：Cursor

### 1.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 项目规则（Project Rules） | —（官方文档无全局 rules 文件路径；用户规则在"自定义 → 规则"UI 中以纯文本保存，未公开文件位置【存疑：社区称存在 `~/.cursor/rules/`，现行官方文档无记载】） | `.cursor/rules/**/*.mdc`（必须 `.mdc` 扩展名；`.md` 会被忽略；支持子文件夹组织） | Markdown + YAML frontmatter | frontmatter：`description` / `globs`（逗号分隔多模式）/ `alwaysApply`（bool）。**现行官方文档只提这三个字段**；旧版 `type: Always/Auto/Agent Requested/Manual` 未在现行文档出现，UI"类型下拉菜单"改写三字段 |
| AGENTS.md | — | 项目根目录**及子目录**（官方明确"支持嵌套 AGENTS.md"，2026 年改进项） | 纯 Markdown | 无 frontmatter |
| 用户规则（User Rules） | "自定义 → 规则"UI（全局偏好纯文本） | — | 纯文本 | — |
| 团队规则（Team Rules） | Cursor 仪表盘（团队版/企业版） | — | 自由文本 + 可选 glob | 含"启用/强制执行（enforce）"标志；优先级 **团队 > 项目 > 用户** |
| MCP | `~/.cursor/mcp.json` | `.cursor/mcp.json` | JSON | 顶层 `mcpServers.<name>`；stdio：`type/command/args/env/envFile`；remote：`url/headers/auth`（静态 OAuth：`CLIENT_ID/CLIENT_SECRET/scopes`） |
| 远程规则导入 | — | `.cursor/rules/imported/<repoName>/…`（从 GitHub 仓库导入，保留相对路径，可同步） | `.mdc` | 同项目规则 |
| 其他（导航存在，未深抓） | Skills / Subagents / Hooks / Plugins 均有官方文档页（`/docs/skills`、`/docs/subagents`、`/docs/hooks`、`/docs/plugins`） | 同左 | — | 概念与 Claude Code 同族，非 Cursor 独有 |

来源：https://cursor.com/cn/docs/rules （英文原版 https://docs.cursor.com/docs/rules，SPA 需 JS 渲染）；https://cursor.com/cn/docs/mcp

**MCP 插值语法**（`command/args/env/url/headers` 五字段内解析）：`${env:NAME}`、`${userHome}`、`${workspaceFolder}`、`${workspaceFolderBasename}`、`${pathSeparator}`（或 `${/}`）。`envFile` 仅 stdio 可用。企业版另有 MCP 允许列表（命令/URL 模式 + 工具白名单 + 网络沙盒三档：允许全部/允许列表/全部拒绝/无沙盒）。来源：https://cursor.com/cn/docs/mcp

### 1.2 能力矩阵

| 能力 | 支持 | 说明 |
|------|------|------|
| instructions | ✅ | AGENTS.md（嵌套）+ 用户规则 + 团队规则 |
| mcp | ✅ | stdio / SSE / Streamable HTTP；OAuth；企业允许列表 |
| skills 类 | ✅ | 官方 Skills/Plugins/Subagents/Hooks（未展开） |
| rules（文件作用域） | ✅ | `globs` frontmatter；四种激活方式（始终/智能/按文件/手动 @） |
| workflows | ❌（无独立 workflow 实体；Hooks 除外） | — |
| custom modes | ❌ | — |
| 独特机制 | 团队规则（enforce）、远程规则导入（GitHub→`.cursor/rules/imported/`）、MCP Apps（工具返回交互式 UI）、nested AGENTS.md | — |

### 1.3 独特概念清单（相对基线五工具）

1. **规则四种激活模式**（始终应用 / 智能应用=按 description / 按文件 glob / 手动 @提及）——基线工具仅 Copilot `applyTo`（glob）与 always-on 两档。
2. **团队规则 + 强制执行标志**：第三管理平面（仪表盘），`enforce` 后用户不可禁用；优先级 团队 > 项目 > 用户。
3. **远程规则导入**：从任意 GitHub 仓库拉取 `.mdc` 到 `.cursor/rules/imported/<repoName>/`，保留相对路径——"规则订阅"语义。
4. **nested AGENTS.md**（子目录 AGENTS.md，官方明确支持）。
5. **MCP 静态 OAuth**（`auth.CLIENT_ID/CLIENT_SECRET/scopes`）与 MCP Apps 扩展。
6. `.cursorrules` 单文件为 legacy（Cline 仍在兼容读取它，见卡片 4）。

### 1.4 IR 压力测试（IR-SCHEMA v0.2）

| 概念 | 能否表达 | 判定 |
|------|---------|------|
| `.mdc` frontmatter `description` | Instruction 无 `description` 字段（仅 id/priority 等）；PromptPack 有 description 但 rules 不是 PromptPack | **x-cursor** |
| `globs` | `Instruction.file_patterns` ✅ 标准字段 | 标准字段 |
| `alwaysApply: true` | 可映射为缺省 always-on（无 file_patterns 即 always），语义可表达 | 标准字段（间接） |
| **智能应用（description 驱动，模型自行决定加载）** | IR Instruction 无"激活模式"字段；无法区分 always-on 与 model-decision | **击穿**（需新增 activation 字段，见击穿汇总 B1） |
| **手动应用（@提及才加载）** | 同上；PromptPack.trigger 有 `mention` 枚举但 rules 是 Instruction 不是 PromptPack | **击穿**（同 B1） |
| 团队规则（enforce、仪表盘来源、最高优先级） | `origin.scope` 仅 `global\|project`，无 team/managed 层；无 enforce 语义 | **击穿**（B2：第三层 scope） |
| nested AGENTS.md | `Instruction.subtree` ✅（Codex 已建此字段） | 标准字段 |
| 远程规则导入（来源 repo URL + 同步） | `Instruction.imports` 是本地 @path 引用，非远程订阅 | 半击穿：标准字段不符，`x-cursor.importedFrom` 可透传 |
| mcp `envFile` | `McpServer.env_file` ✅ | 标准字段 |
| mcp `auth`（OAuth 三元组） | McpServer 无 auth 段 | **击穿→x-cursor**（B3） |
| `${userHome}/${workspaceFolder}/${pathSeparator}` 插值 | IR §3.2 规则：同工具往返原样保留，跨工具不翻译 + Warning | 标准行为（Warning 路径已覆盖） |
| 企业 MCP 允许列表/网络沙盒 | 无对应实体 | x-cursor（策略类，见 B4） |

### 1.5 真实样本

1. 官方文档 frontmatter 四例（always / globs / description / manual），含 `.cursor/rules/frontend/components.mdc` 子目录组织示例。https://cursor.com/cn/docs/rules
2. 官方 mcp.json 示例：stdio（Node/Python）、remote（url+headers）、静态 OAuth（`auth.CLIENT_ID/CLIENT_SECRET/scopes`）、`${workspaceFolder}` 插值。https://cursor.com/cn/docs/mcp
3. 社区仓库 **PatrickJS/awesome-cursorrules**（40.6k stars，2026 年仍活跃维护，170 commits）：`rules/nextjs15-react19-vercelai-tailwind-cursorrules-prompt-file.mdc` 等数百个 `.mdc` 样本。https://github.com/PatrickJS/awesome-cursorrules
4. 官方推荐的社区站点 cursor.directory（MCP/规则集合，官方 MCP 文档提及）。https://cursor.directory 【存疑：第三方站点，非官方运营】

### 1.6 时效状态

- 活跃：2026 年文档已重构为新信息架构（Rules/Skills/Subagents/Hooks/Plugins/MCP 并列），提供官方中文版（cursor.com/cn/docs）。模型线活跃（Composer 2.5 等）。
- 变更点：rules 文档 URL 由旧 `/context/rules` 迁至 `/docs/rules`（旧地址 302 至首页）；`type` 字段从官方文档消失（社区旧样本仍大量存在，适配器需兼容读取）。
- 无弃维护迹象。

---

## 卡片 2：Windsurf（现 Devin Desktop）

> ⚠️ **品牌迁移**：docs.windsurf.com → docs.devin.ai；Linux 包名 `devin-desktop`（`windsurf` 为过渡依赖包）；配置根目录仍为 `~/.codeium/windsurf/`；规则首选目录由 `.windsurf/rules/` 迁为 `.devin/rules/`（`.windsurf/` 保留为 fallback）。来源：https://docs.devin.ai/（首页安装说明 Note）、https://docs.devin.ai/desktop/cascade/memories

### 2.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 规则（Rules） | `~/.codeium/windsurf/memories/global_rules.md`（单文件，always-on，**上限 6,000 字符**） | `.devin/rules/*.md`（首选）或 `.windsurf/rules/*.md`（fallback）；支持子目录与向上搜索至 git root；legacy 单文件 `.windsurfrules` 仍读；**每文件上限 12,000 字符** | Markdown + frontmatter | `trigger: always_on \| model_decision \| glob \| manual`；`globs: **/*.test.ts`（trigger=glob 时） |
| AGENTS.md | — | 工作区内任意目录（大小写不敏感 `AGENTS.md/agents.md`）；根目录 = always-on；子目录 = 自动 glob `<directory>/**` | 纯 Markdown | 无 frontmatter |
| Memories（自动生成） | `~/.codeium/windsurf/memories/`（按 workspace 关联，不跨工作区、不入库、仅本地） | — | 内部存储 | Cascade 自动生成或用户要求创建；⚠️ 仅 legacy Cascade agent；新默认 agent Devin Local **不支持**，官方引导迁移至 Skills |
| Workflows | `~/.codeium/windsurf/global_workflows/*.md` | `.windsurf/workflows/*.md`（注意：**未随 rules 迁至 `.devin/`**，文档仅列 `.windsurf/workflows/`）；子目录/父目录至 git root 均可发现；**每文件上限 12,000 字符** | Markdown（标题+描述+步骤） | `/[name]` slash 触发；**仅手动**；可互相调用；仅 legacy Cascade |
| MCP | `~/.codeium/windsurf/mcp_config.json`（仅 legacy Cascade；Devin Local 用 Devin CLI 配置） | — | JSON | `mcpServers.<name>`；stdio：`command/args/env`；remote：`serverUrl`（或 `url`）+ `headers`；`disabledTools` 数组 |
| System 层（企业，只读） | macOS `/Library/Application Support/Devin/rules/`（fallback `Windsurf/`）；Linux `/etc/devin/rules/`（fallback `/etc/windsurf/rules/`）；Windows `C:\ProgramData\Devin\rules\`（fallback `C:\ProgramData\Windsurf\rules\`）；workflows 同理 | — | `.md` | 与 workspace/global 合并，不覆盖；UI 显示 "System" 标签，用户不可删 |

来源：https://docs.devin.ai/desktop/cascade/memories、https://docs.devin.ai/desktop/cascade/workflows、https://docs.devin.ai/desktop/cascade/agents-md、https://docs.devin.ai/desktop/cascade/mcp

**MCP 细节**：传输支持 stdio / Streamable HTTP / SSE + 各传输 OAuth；插值 `${env:VAR}` 与 **`${file:/path/to/file}`（读文件内容，支持 `~`）**，作用于 `command/args/env/serverUrl/url/headers`；Cascade 工具总数上限 100；deeplink 一键安装 `windsurf://windsurf-mcp-registry?serverName=<name>`；企业 admin 可用 **regex 模式** 做 MCP allowlist（`^(?:pattern)$` 锚定、args 逐元素匹配、env 不参与匹配）。来源：https://docs.devin.ai/desktop/cascade/mcp

**Workflow 优先级**：System > Workspace > Global > Built-in（同名覆盖）。来源：https://docs.devin.ai/desktop/cascade/workflows

### 2.2 能力矩阵

| 能力 | 支持 | 说明 |
|------|------|------|
| instructions | ✅ | rules + AGENTS.md（嵌套自动作用域）+ global_rules.md |
| mcp | ✅ | stdio / HTTP / SSE + OAuth + 双插值语法 |
| skills 类 | ✅ | 官方 Skills 页（`/desktop/cascade/skills`，未展开）；官方建议新投资方向 |
| rules（文件作用域） | ✅ | `trigger: glob` + `globs` |
| workflows | ✅ | `.windsurf/workflows/*.md`，手动 slash 触发 |
| custom modes | ❌ | — |
| 独特机制 | **Memories（AI 自动生成）**、**System 企业层**、字符上限、`${file:}` 插值、MCP regex allowlist | — |

### 2.3 独特概念清单

1. **Memories**：AI 在对话中自动生成并持久化的上下文，存 `~/.codeium/windsurf/memories/`，workspace 关联、不入 git、机器本地。基线五工具无对应物（最接近的是 Claude Code 的自动 memory，但官方形态不同）。
2. **System（Enterprise）层规则/工作流**：OS 级第三配置层（`/etc/…`、`C:\ProgramData\…`），IT 部署、用户只读，与用户规则"合并而非覆盖"。
3. **activation mode 四态** `trigger: always_on|model_decision|glob|manual`（与 Cursor 四模式同构，且官方明确各模式的 context 成本）。
4. **字符上限入规范**：global 6,000 / 文件 12,000 / workflow 12,000。
5. **`${file:/path}` 插值**：从文件读秘密值（基线工具只有 `${env:}`/`${input:}`）。
6. **AGENTS.md 子目录自动 glob 化**：`<directory>/**` 由引擎按位置推断，无需 frontmatter。
7. 企业 MCP regex allowlist（命令/参数逐元素正则匹配）。

### 2.4 IR 压力测试

| 概念 | 能否表达 | 判定 |
|------|---------|------|
| `trigger: glob` + `globs` | `Instruction.file_patterns` ✅ | 标准字段 |
| `trigger: always_on` | 缺省（无 file_patterns） | 标准字段（间接） |
| `trigger: model_decision`（仅 description 常驻，正文按需加载） | 无字段 | **击穿**（B1） |
| `trigger: manual`（@rule-name） | Instruction 无 mention 触发 | **击穿**（B1） |
| **Memories**（自动生成、workspace 关联、非用户编写） | Instruction 是用户静态内容；memories 语义为"机器生成的动态事实库"，无作者/自动生成/召回策略字段 | **击穿**（B5；可降级为 instruction + `x-windsurf`，但丢失自动生成语义与同步排除语义） |
| **System 层** | `origin.scope` 仅两层 | **击穿**（B2） |
| 字符上限（6k/12k） | 无容量约束概念 | 半击穿：导出需按目标能力告警（建议入适配器 Capability.Note / doctor 校验） |
| AGENTS.md 子目录自动作用域 | `Instruction.subtree` ✅ | 标准字段 |
| Workflow 实体 | `Workflow`（PromptPack）✅；`trigger.type: slash-command` ✅ | 标准字段 |
| Workflow 四级优先级（System>Workspace>Global>Built-in） | merge-by-id 只定义 global→project 方向 | **击穿**（B2/B6：多层与方向） |
| `${file:}` 插值 | IR 仅规定 `${input:}`/`${env:}` 原样搬运；`${file:}` 可同样搬运 | 标准行为（Warning 路径）；**但它天然映射 secretref file 后端，是启示不是击穿** |
| `serverUrl` 键名 | IR 标准字段 `url` | 适配器映射（x-windsurf 保留原键名） |
| `disabledTools`、MCP regex allowlist | 无字段 | **击穿→x-windsurf**（B3/B4） |

### 2.5 真实样本

1. 官方 glob 规则示例：`---\ntrigger: glob\nglobs: **/*.test.ts\n---\nAll test files must use describe/it blocks …`。https://docs.devin.ai/desktop/cascade/memories
2. 官方 workflow 示例 `/address-pr-comments`（完整步骤文本，`gh api --paginate … jq …`）。https://docs.devin.ai/desktop/cascade/workflows
3. 官方 `mcp_config.json` 示例（GitHub server npx/docker 两式、`${env:AUTH_TOKEN}`、`${file:~/.secrets/api_key.txt}`）。https://docs.devin.ai/desktop/cascade/mcp
4. 官方规则模板库：https://windsurf.com/editor/directory（官方 memories 文档推荐）

### 2.6 时效状态

- **重大变更**：Windsurf 归 Cognition 后产品更名 **Devin Desktop**；新默认 agent "Devin Local" 不支持 memories 与 workflows，官方提供 `Devin: Open Cascade Migration Wizard` 引导迁移到 Skills；legacy Cascade 继续维护上述机制。
- 适配器含义：路径双轨（`.devin/` 优先、`.windsurf/` fallback；`~/.codeium/windsurf/` 配置根未变）；memories/workflows 属"冻结中的 legacy 能力"，采集可、新建导出应谨慎并 Warning。

---

## 卡片 3：Aider

### 3.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 主配置 | 用户主目录 `.aider.conf.yml` | git 仓库根 `.aider.conf.yml`、cwd `.aider.conf.yml`（加载顺序 home→git root→cwd，**后加载优先**；`--config` 指定则只读该文件） | YAML | 全量 CLI 选项的键（`model`、`read`、`lint-cmd`、`auto-commits`、`map-tokens`、`watch-files` 等数百项，官方样例列出全部合法键） |
| 指令（conventions） | — | 任意 Markdown，惯例名 `CONVENTIONS.md`；经 `.aider.conf.yml` 的 `read: [CONVENTIONS.md, …]` 常驻，或 CLI `--read` / 会话内 `/read` 加载（**read-only 上下文 + 可命中 prompt 缓存**） | Markdown | 无 frontmatter |
| 忽略文件 | — | git 根 `.aiderignore`（配置键 `aiderignore`，gitignore 语法） | 文本 | 同 .gitignore 语法 |
| 秘密 | `.env`（git 根，配置键 `env-file`）；YAML 中仅建议放 OpenAI/Anthropic key，其余走 .env / `set-env` / `api-key provider=<key>` | 同左 | dotenv | `KEY=value` |
| 运行时产物（不采集） | `.aider.input.history`、`.aider.chat.history.md`（`input-history-file`/`chat-history-file` 可改名）、`.aider.llm.history`、`.aider.tags.cache` 等 | 同左 | 文本 | — |
| MCP | ❌ 不支持 | — | — | — |

来源：https://aider.chat/docs/config/aider_conf.html（含官方全量样例）、https://aider.chat/docs/usage/conventions.html

**其他关键键**（均可在 `.aider.conf.yml`）：`lint-cmd`（按语言多值）、`auto-lint`、`test-cmd`、`auto-test`、`commit-prompt`、`attribute-author/committer/co-authored-by`（git 署名）、`watch-files`（监听代码中的 AI 注释触发）、`subtree-only`、`add-gitignore-files`、`cache-prompts`、`alias`、`model-settings-file`（`.aider.model.settings.yml`）、`model-metadata-file`（`.aider.model.metadata.json`）。

### 3.2 能力矩阵

| 能力 | 支持 | 说明 |
|------|------|------|
| instructions | ✅（弱形态） | CONVENTIONS.md 经 `read` 常驻；无 always-on 项目指令文件的专属地位 |
| mcp | ❌ | 官方文档无 MCP |
| skills 类 | ❌ | — |
| rules（文件作用域） | ❌ | 无 glob 作用域规则 |
| workflows | ❌ | 无命名 workflow；`--message/--load` 批处理属 CLI 用法 |
| custom modes | ⚠️ 近似 | `chat modes`（code/ask/architect 等，`/chat-mode` 或 `--chat-mode`）为内置模式，`architect` + `editor-model` 组合可配置，但非用户自定义实体 |
| 独特机制 | read-only 文件约定、git 深度集成（auto-commits/dirty-commits/署名）、repomap（`map-tokens`）、watch-files AI 注释、lint/test 自动循环 | — |

### 3.3 独特概念清单

1. **read-only 上下文文件**：`read:`/`--read`/`/read` 把文件标记为"只读、仅供上下文"（且利于 prompt 缓存）——"会话资源及其读写权限"概念。
2. **`.aiderignore`**：gitignore 语法控制 aider 编辑/可见范围。
3. **git 署名策略**：`attribute-author/committer/co-authored-by`、`aider: ` 前缀——配置驱动 git 提交身份。
4. **watch-files**：监听源码中 `AI!` 风格注释触发动作（配置开关）。
5. **三层配置文件叠加**（home → git root → cwd，后者胜）——"项目层"实际是两个候选位置。
6. 配置即全部：几乎全部行为（含 lint/test 命令、颜色、分析开关）都在一份 YAML 中。

### 3.4 IR 压力测试

| 概念 | 能否表达 | 判定 |
|------|---------|------|
| `.aider.conf.yml` 全部键 | `Setting` 实体（`setting.aider.<key>`，`value` 原样嵌套）✅ | 标准字段 |
| `read: [CONVENTIONS.md]` | 两种建法：(a) 作为 `setting.aider.read` 条目 ✅；(b) 文件本体作为 Instruction | 标准字段；**但"read-only 上下文"语义在 (b) 建法下无字段**——半击穿（B7：资源只读性） |
| CONVENTIONS.md 本体 | `Instruction` ✅ | 标准字段 |
| `.aiderignore` | IR 七实体无 ignore/access 实体 | **击穿**（B8） |
| 三层叠加（home/git root/cwd） | `origin.path` 记录实际来源即可；merge 语义 setting merge-by-id ✅ | 标准行为 |
| git 署名/lint-cmd 等 | Setting ✅ | 标准字段 |
| `.aider.chat.history.md` 等运行时产物 | 不应采集 | 适配器排除清单（非 schema 问题） |

**结论：Aider 对 IR 压力最小**——其模型是"单文件全量 settings + 自由 markdown 引用"，几乎全部落入 Setting/Instruction。

### 3.5 真实样本

1. 官方全量样例 `.aider.conf.yml`（列出全部合法键，可下载）：https://github.com/Aider-AI/aider/blob/main/aider/website/assets/sample.aider.conf.yml （文档页：https://aider.chat/docs/config/aider_conf.html）
2. 官方社区仓库 **Aider-AI/conventions**（203 stars）：`golang/CONVENTIONS.md`、`nextjs-ts/CONVENTIONS.md` 等，README 明示 `aider --read-only CONVENTIONS.md` 与 `.aider.conf.yml` 配置法。https://github.com/Aider-AI/conventions
3. 官方 conventions 文档示例（httpx/types 规约 + `read: [CONVENTIONS.md, anotherfile.txt]`）。https://aider.chat/docs/usage/conventions.html

### 3.6 时效状态

- 活跃开源项目（github.com/Aider-AI/aider，文档站持续更新；文档内示例版本号为 v0.24.2-dev，属旧示例快照，非最新版）。
- 无 MCP 支持迹象；无弃维护公告。

---

## 卡片 4：Cline

### 4.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| 规则（Rules） | `~/.cline/rules/`；另兼容 `~/Documents/Cline/Rules`（Windows `Documents\Cline\Rules`） | `.clinerules/`（目录，`.md`/`.txt`，数字前缀可选）或单文件 `.clinerules`；**并自动兼容读取** `.cursorrules`、`.windsurfrules`、`AGENTS.md`、`~/.agents/AGENTS.md` | Markdown + 可选 YAML frontmatter | 条件规则 frontmatter：`paths: ["src/**", "*.config.js"]`（glob 数组；`paths: []` = 永不激活；无 frontmatter = 常驻）；UI 可逐条 toggle |
| Skills | `~/.cline/skills/`（Windows `%USERPROFILE%\.cline\skills\`） | `.cline/skills/`（推荐）、`.clinerules/skills/`、`.claude/skills/`（兼容） | 目录 + `SKILL.md` | frontmatter `name`（须与目录同名）、`description`（≤1024 字符）；**同名时全局优先于项目**（与多数工具相反！） |
| MCP | `~/.cline/data/settings/cline_mcp_settings.json`（CLI 向导写 `~/.cline/mcp.json`【存疑：config 页与 MCP 页路径表述并存，以 `cline config mcp` 实测为准】） | —（官方文档未列项目级 mcp 文件） | JSON | `mcpServers.<name>`：`command/args/env` 或 `url/headers`；`type: streamableHttp \| sse`（缺省按 legacy sse）；**`disabled: false`、`autoApprove: []`** |
| Workflows | `~/.cline/data/workflows/`；兼容 `~/Documents/Cline/Workflows` | （`.cline/` 布局中未列 workflows；官方仅公开目录位置，文件格式未展开【存疑】） | — | — |
| Hooks / Plugins / Cron / Agents | `~/.cline/hooks/`、`~/.cline/plugins/`（.js/.ts）、`~/.cline/cron/`、`~/.cline/agents/`；兼容 `~/Documents/Cline/{Hooks,Plugins}` | `.cline/hooks/`、`.cline/plugins/`、`.cline/cron/`、`.cline/agents/` | 代码/配置 | hooks/plugins 为**可执行代码**；cron 为定时任务 spec |
| `.clineignore` | — | 项目根 `.clineignore` | 文本（gitignore 语法） | ⚠️ **官方宣布即将弃用**；仅控制自动加载（`@` 提及与 shell 可绕过）；替代方案为 SDK 插件（`gitignore-read-files-guard.ts`，beforeTool 钩子阻断 read/edit） |
| 数据/设置 | `~/.cline/data/settings/providers.json`、`global-settings.json`；`CLINE_DATA_DIR` 可整体迁移 | — | JSON | 含 API keys（明文风险） |

来源：https://docs.cline.bot/customization/cline-rules.md、https://docs.cline.bot/customization/skills.md、https://docs.cline.bot/mcp/mcp-overview.md、https://docs.cline.bot/getting-started/config.md、https://docs.cline.bot/customization/clineignore.md

**权限相关环境变量**：`CLINE_COMMAND_PERMISSIONS='{"allow":["npm *","git *"],"deny":["rm -rf *"],"allowRedirects":false}'`（deny 优先；设置 allow 后白名单外全拒）。来源：https://docs.cline.bot/getting-started/config.md

### 4.2 能力矩阵

| 能力 | 支持 | 说明 |
|------|------|------|
| instructions | ✅ | rules（目录+单文件+跨工具兼容读取 AGENTS.md 等） |
| mcp | ✅ | stdio / streamableHttp / sse；CLI 向导管理 |
| skills 类 | ✅ | SKILL.md 目录形态，斜杠命令可显式触发 |
| rules（文件作用域） | ✅ | frontmatter `paths:` glob 数组（基于消息/打开页签/编辑中文件判定激活） |
| workflows | ✅（细节未公开） | 全局目录存在，格式【存疑】 |
| custom modes | ❌（Plan/Act 为内置双模） | — |
| 独特机制 | hooks/plugins（可执行扩展）、cron 定时任务、Memory Bank（文档约定的结构化记忆文件集）、`.clineignore`（将弃用）、`autoApprove` 工具白名单 | — |

### 4.3 独特概念清单

1. **`.clineignore`**（将弃用）与插件化访问控制（beforeTool 钩子）——ignore 概念 + 可执行访问策略。
2. **MCP `autoApprove` 数组**：每 server 的工具级自动批准白名单。
3. **hooks / plugins / cron**：生命周期钩子与可执行插件（.js/.ts）、定时任务——超越 prompt 的"代码级扩展"实体。
4. **跨工具规则兼容读取**：`.cursorrules`、`.windsurfrules`、`AGENTS.md`、`~/.agents/AGENTS.md`、`.claude/skills/`——一个工具主动聚合他工具配置。
5. **Skills 同名全局优先于项目**（官方明确，与 Roo/惯例相反）。
6. 全局规则目录在 `Documents\Cline\Rules`（Windows）——非常规路径（非 dotfile）。
7. Memory Bank 方法论（`memory-bank/` 文档集约定，社区最佳实践页）。

### 4.4 IR 压力测试

| 概念 | 能否表达 | 判定 |
|------|---------|------|
| rules 目录/单文件/多源聚合 | `Instruction` 多条目 + origin.path ✅；`paths:` → `file_patterns` ✅ | 标准字段 |
| 规则 UI toggle（启用/禁用状态） | 无"条目级 enabled"字段 | 半击穿：`x-cline.enabled`（B9） |
| Skills（SKILL.md 目录） | `Skill`（PromptPack）✅ 完全同构 | 标准字段 |
| **Skills 全局>项目** | IR merge-by-id 固定"项目覆盖全局" | **击穿**（B6：合并方向反转） |
| `autoApprove`（每 server 工具白名单） | McpServer 无审批字段 | **击穿→x-cline**（B3） |
| `type: streamableHttp` | IR transport 枚举 `stdio\|sse\|http` | 适配器映射（http ↔ streamableHttp），x- 保留原值 |
| hooks / plugins / cron | PromptPack trigger 有 `hook` 枚举，但 hook 本体（事件、可执行代码）无 schema；cron 无对应物 | **击穿**（B10：新实体或 x-cline 大包） |
| `.clineignore` | 无 ignore 实体 | **击穿**（B8） |
| `CLINE_COMMAND_PERMISSIONS`（allow/deny JSON 策略） | 无策略实体 | **击穿→x-cline**（B4） |
| `~/Documents/Cline/Rules` 兼容路径 | origin.path 记录 | 标准行为 |

### 4.5 真实样本

1. 官方条件规则四例（frontend/backend/testing/docs，`paths:` frontmatter + glob 语义表）。https://docs.cline.bot/customization/clineignore.md → 正页 https://docs.cline.bot/customization/cline-rules.md
2. 官方 `.clineignore` 模板（node_modules/build/env/大文件分类清单）。https://docs.cline.bot/customization/clineignore.md
3. 官方 MCP 配置例：STDIO（含 `disabled`/`autoApprove`）与 `streamableHttp`（含 `headers`）。https://docs.cline.bot/mcp/mcp-overview.md
4. 官方访问控制插件示例：https://github.com/cline/cline/blob/main/sdk/examples/plugins/gitignore-read-files-guard.ts （文档内直接引用，可用 `cline plugin install <url> --cwd .` 安装）
5. 官方 skill 完整例（`data-analysis/SKILL.md` + docs/templates/scripts 分包）。https://docs.cline.bot/customization/skills.md

### 4.6 时效状态

- 高度活跃并扩张：已有 VS Code/JetBrains 扩展、CLI（`npm i -g cline`）、Kanban、SDK、企业版（SSO/远程配置/OTel）。
- **重大变更预告**：`.clineignore` 官方标注 "deprecate soon"，迁移方向为 SDK 插件 + `.gitignore`；配置结构正向 `~/.cline/data/` 收敛。
- 对 cfg4ai：cline 适配器应把 `.clineignore` 标为 legacy（采集 Warning），并预期 workflows/hooks 文档进一步完善。

---

## 卡片 5：Roo Code

### 5.1 配置地图

| 实体 | 全局路径 | 项目路径 | 文件格式 | 键结构 |
|------|---------|---------|---------|--------|
| Custom Modes | VS Code 扩展 globalStorage 下 `settings/custom_modes.yaml`（首选；`custom_modes.json` 启动时自动迁移为 YAML 并保留原文件）【存疑：globalStorage 绝对路径官方未列，由"Edit Global Modes"按钮打开】 | 项目根 `.roomodes`（YAML 或 JSON，先按 YAML 解析；无自动迁移，UI 编辑后存为 YAML） | YAML/JSON | 顶层 `customModes: []`；字段：`slug`（`^[a-zA-Z0-9-]+$`）、`name`、`description`、`roleDefinition`、`whenToUse`、`customInstructions`、`groups`（`read/edit/command/mcp` 字符串或 `["edit",{fileRegex,description}]` 元组）；`source` 由系统写入 |
| 规则（通用） | `~/.roo/rules/`（Win `%USERPROFILE%\.roo\rules\`） | `.roo/rules/`（目录递归、按文件名字母序拼接）；fallback 单文件 `.roorules`；legacy `.clinerules` | `.md`/`.txt` | 无 frontmatter；系统排除 `.DS_Store/*.bak/*.tmp` 等；符号链接支持（深度≤5） |
| 规则（mode 专属） | `~/.roo/rules-{slug}/` | `.roo/rules-{slug}/`；fallback `.roorules-{slug}`；legacy `.clinerules-{slug}` | 同上 | 目录优先于单文件；与 customInstructions 拼接（文件内容在后） |
| AGENTS.md | — | 工作区根 `AGENTS.md`（fallback `AGENT.md`；设置 `roo-cline.useAgentRules: false` 关闭） | Markdown | 无 frontmatter |
| Skills | `~/.roo/skills/{name}/SKILL.md`（高优先级）、`~/.agents/skills/`（跨 agent 共享） | `.roo/skills/`、`.agents/skills/`；mode 专属 `skills-{slug}/`（全局与项目均可） | 目录 + SKILL.md | frontmatter `name`（1–64 字符 kebab-case，须与目录同名）、`description`（1–1024 字符）；遵循 agentskills.io；**8 级覆盖优先级**（项目 `.roo/skills-{mode}/` 最高 → 全局 `.agents/skills/` 最低） |
| MCP | 全局 `mcp_settings.json`（VS Code globalStorage，经"Edit Global MCP"打开） | `.roo/mcp.json`（项目优先） | JSON | `mcpServers.<name>`：`command/args/cwd/env`、remote `type: streamable-http\|sse` + `url/headers`；公共：`alwaysAllow[]/disabled/timeout(1–3600 秒)/disabledTools[]/watchPaths[]`；args 支持 `${env:VAR}` |
| `.rooignore` | — | 工作区根 `.rooignore`（热重载；文件自身隐式忽略） | 文本（gitignore 语法） | 工具级强制：`read_file/write_to_file/apply_diff` 直接阻断，`execute_command` 对预定义命令检查；`showRooIgnoredFiles` 控制 🔒 显示 |
| Mode 导入导出 | 单文件 YAML：`customModes[]` + `rulesFiles[]`（`relativePath`+`content` 内嵌），slug 改名时自动重写 rules 路径 | 同左 | YAML | 便携打包格式（非运行时实体） |

来源：https://docs.roocode.com/features/custom-modes/、https://docs.roocode.com/features/custom-instructions/、https://docs.roocode.com/features/mcp/using-mcp-in-roo/、https://docs.roocode.com/features/rooignore/、https://docs.roocode.com/features/skills/

**合并/优先级**：modes 为 项目 `.roomodes` > 全局 `custom_modes.yaml` > 内置（同 slug 整体覆盖所有属性）；rules 为 全局先加载、项目后加载且冲突时项目优先，mode 专属先于通用，目录法优先于单文件 fallback；系统提示拼接顺序官方有完整模板（Language Preference → Global Instructions → Mode Instructions → rules-{slug} → .rooignore 说明 → AGENTS.md → rules → .roorules）。

### 5.2 能力矩阵

| 能力 | 支持 | 说明 |
|------|------|------|
| instructions | ✅ | rules 目录/单文件 + AGENTS.md + Prompts tab 文本 |
| mcp | ✅ | stdio / streamable-http / sse；项目级 `.roo/mcp.json` |
| skills 类 | ✅ | SKILL.md；项目>全局、`.roo/`>`.agents/`、mode 专属目录 |
| rules（文件作用域） | ❌（无 glob 触发；作用域单位为 mode） | mode 是 Roo 的"作用域"机制 |
| workflows | ❌（无独立 workflow；有 Slash Commands 与 Boomerang 编排） | — |
| custom modes | ✅（核心特色） | 全字段见上 |
| 独特机制 | **custom modes**（含 fileRegex 编辑白名单）、`.rooignore` 工具级强制、mode/rules 导入导出包、跨 agent 共享目录 `.agents/`、`alwaysAllow`/`watchPaths`/`disabledTools` | — |

### 5.3 独特概念清单

1. **Custom Modes 全字段模型**：`slug/name/description/roleDefinition/whenToUse/customInstructions/groups`——同一实体融合"人格 prompt + 工具组权限 + 文件编辑正则白名单（`fileRegex`）+ 编排提示（whenToUse 供 Orchestrator 选 mode）"。
2. **`.roomodes` 项目文件**（无扩展名、YAML/JSON 自适应）与全局 `custom_modes.yaml`。
3. **`.rooignore` 工具级强制**（不同于 Cline 的"仅自动加载过滤"）。
4. **mode 绑定 rules 目录**：`rules-{slug}/`、三级 fallback（目录→`.roorules-{slug}`→`.clinerules-{slug}`）。
5. **跨 agent 共享目录**：`.agents/skills/`、`~/.agents/skills/`（与其他工具共享技能库），8 级覆盖优先级。
6. **MCP `alwaysAllow`（工具白名单）/`watchPaths`（变更自动重启）/`cwd`/`timeout` 秒数单值**。
7. **mode + rules 的导入导出打包 YAML**（rulesFiles 内嵌 content、slug 改名自动重写路径）。
8. Sticky Models（每个 mode 记忆上次所用模型——运行时状态，非配置）。

### 5.4 IR 压力测试

| 概念 | 能否表达 | 判定 |
|------|---------|------|
| `roleDefinition` / `customInstructions` / `whenToUse` | PromptPack（Agent）有 prompt.md + description；role/whenToUse 无标准字段 | 半击穿：主体可入 prompt.md，`whenToUse` 需 `x-roocode` |
| **`groups`（工具组权限 + `fileRegex` 编辑白名单）** | IR 无任何"实体携带权限策略"概念（Setting 里没有、PromptPack 没有） | **击穿**（B4：权限模型缺失，本组最严重击穿之一） |
| mode 与 rules-{slug}/ 的绑定关系 | 无"PromptPack ↔ 指令集"关联字段 | **击穿→x-roocode**（B11：实体间引用） |
| `.roo/rules/` 目录聚合（递归、字母序、跨全局+项目拼接） | Instruction 多条目 ✅ | 标准字段 |
| 三级 fallback（目录/单文件/legacy） | 采集时按胜出源记录 origin | 标准行为 |
| 同 slug 项目**整体覆盖**全局（所有属性） | IR merge-by-id 是 field-level-shallow（项目未出现字段继承全局），**Roo 是整条替换** | **击穿**（B6 变种：合并粒度冲突——IR 浅合并 vs Roo 整体覆盖；导出需 entry-replace 语义） |
| `.rooignore` | 无 ignore 实体 | **击穿**（B8） |
| MCP `alwaysAllow`/`disabledTools`/`watchPaths`/`cwd` | McpServer 无这些字段 | **击穿→x-roocode**（B3） |
| `timeout` 单值秒（1–3600） | IR `timeout.startup_ms/tool_sec` 结构化 | 半击穿：粒度不对齐，可映射 tool_sec，x- 保留原值（B12） |
| `.agents/skills/` 共享目录（跨工具命名空间） | origin.tool 单值；共享目录归属模糊 | 半击穿：采集归属策略问题（B13） |
| mode 导入导出包 | 非运行时实体，无需 IR 表达 | 不适用 |
| 8 级覆盖优先级 | merge 仅两层+方向固定 | **击穿**（B6） |

### 5.5 真实样本

1. 官方 `custom_modes.yaml` 完整示例（docs-writer：含 `description/whenToUse/customInstructions/groups` 元组 fileRegex）。https://docs.roocode.com/features/custom-modes/
2. 官方 `.roo/mcp.json` 示例集：STDIO 全字段（`cwd/alwaysAllow/disabled/timeout/watchPaths/disabledTools`）、`${env:VAR}` args 插值、streamable-http、sse、Windows `cmd /c` 与 mise/asdf 版本管理器适配例。https://docs.roocode.com/features/mcp/using-mcp-in-roo/
3. 官方 mode 导出 YAML 格式（`rulesFiles: [{relativePath, content}]`）。https://docs.roocode.com/features/custom-modes/
4. Roo Code Marketplace（官方 mode 分发渠道，文档内提及）：https://docs.roocode.com/features/marketplace
5. 官方 `.rooignore` 语法与工具交互矩阵（含错误消息原文）。https://docs.roocode.com/features/rooignore/

### 5.6 时效状态

- 活跃：GitHub 15.4k stars（RooCodeInc/Roo-Code），文档站 2026-05-15 仍有更新；VS Code Marketplace 安装量 574.1k。
- 演进方向：custom modes 由 JSON 迁移到 YAML（全局自动迁移、项目手动）；规则目录化（`.roo/rules/` 取代 `.roorules`）；skills 对齐 agentskills.io 跨工具规范。
- 无弃维护迹象。

---

## IR 击穿汇总（按严重度排序）

> 判定口径：标准字段 = IR v0.2 现有字段可无损表达；x- = 需工具命名空间透传（不丢但无跨工具语义）；**击穿 = 标准模型无法表达，且涉及多工具或结构性缺失**。

| 编号 | 击穿点 | 涉及工具 | 说明 |
|------|--------|---------|------|
| **B1** | **Instruction 激活模式缺失**：always-on 之外的三态——model_decision（description 驱动按需加载）/ glob / manual（@提及）——IR 只有 `file_patterns` 一档 | Cursor、Windsurf（`trigger` 四态）、Cline（`paths`+toggle） | Cursor/Windsurf 规则的核心语义；建议 Instruction 增 `activation: always \| auto \| glob \| manual` + `description` |
| **B2** | **scope 只有 global/project 两层**：Windsurf System 企业层（`/etc/devin/rules/`、`C:\ProgramData\…`）、Cursor 团队规则（enforce）构成第三/第四管理平面 | Windsurf、Cursor（基线 Claude managed 层同源） | `origin.scope` 枚举与 merge_policy 都需扩展（`system/managed/team`），且该层通常**只读不导出** |
| **B4** | **权限/审批模型整体缺失**：Roo mode `groups`+`fileRegex`（编辑白名单）、Cline `CLINE_COMMAND_PERMISSIONS`（allow/deny）、Cursor 企业 MCP 允许列表+网络沙盒、Windsurf MCP regex allowlist | Roo、Cline、Cursor、Windsurf | IR 七实体无 Policy/Permission 概念；这是五工具共同浮现的方向，建议立项新实体（`policy.`）或最低限度 x- 透传 + 文档免责 |
| **B3** | **McpServer 外围字段簇**：`autoApprove`/`alwaysAllow`（工具白名单）、`disabledTools`、`watchPaths`、`cwd`、`auth`（OAuth 三元组）、`serverUrl` 别名 | Cline、Roo、Cursor、Windsurf | 单工具可 x-，但 `autoApprove/alwaysAllow` 两工具同语义不同名，值得升标准字段或维护映射表 |
| **B6** | **合并方向/粒度不可假设**：Cline skills **全局>项目**（反转）；Roo 同 slug mode **整体覆盖**（vs IR field-level-shallow）；Roo skills 8 级、Windsurf workflow 4 级优先级 | Cline、Roo、Windsurf | `merge_policy` 需 per-tool 方向声明；`override: entry-replace` 语义已在词表但需接 mode 类实体 |
| **B8** | **ignore 文件家族无实体**：`.clineignore`（仅自动加载过滤）/`.rooignore`（工具级强制）/`.aiderignore`（编辑范围）——同语法三强度 | Cline、Roo、Aider（+Cursor `.cursorignore` 同类） | 建议新实体 `ignore.`（patterns + enforcement 强度枚举）；注意 Cline 即将弃用 .clineignore |
| **B5** | **机器生成内容**：Windsurf memories（AI 自写、workspace 关联、不入库）——作者性/易失性/同步排除语义无字段 | Windsurf | 可降级 instruction+x-，但需 `volatile`/作者标记防误同步 |
| **B10** | **可执行扩展实体**：Cline hooks/plugins（.js/.ts，可执行代码）、cron specs | Cline（Cursor hooks 同族） | PromptPack trigger=hook 只有枚举没有实体；安全采集（代码即攻击面）需专门设计 |
| **B7** | **资源只读性/上下文角色**：Aider `read:` 文件（read-only 上下文+缓存优化） | Aider | 作为 Setting 条目可过；作为 Instruction 则缺 `role: read-only-context` 之类字段 |
| **B9** | **条目级 enabled/toggle 状态**（UI 开关不落文件内容） | Cline（rules/skills toggle） | x- 可存，但状态存工具数据库而非文件——**部分信息根本不在文件系统**，采集边界需注明 |
| **B11** | **实体间引用**：Roo mode ↔ `rules-{slug}/` 目录绑定 | Roo | IR 无跨实体 link 字段（PromptPack→Instruction 集） |
| **B12** | **timeout 粒度不对齐**：Roo 单值秒 vs IR `startup_ms/tool_sec` 结构 | Roo（Codex 双值与 IR 对齐） | 适配器映射 + x- 保留原值即可，低危 |
| **B13** | **跨工具共享命名空间**：`.agents/skills/`、`~/.agents/AGENTS.md` 被多工具共读，origin.tool 归属模糊 | Roo、Cline（AGENTS.md 生态） | reconcile 按 (tool,path,id) 不跨工具合并的规则在同一路径被两工具声明时需裁决策略 |

**未击穿的重要收敛信号**：AGENTS.md（嵌套/自动作用域）已被 Cursor、Windsurf、Roo、Cline 全部支持——IR 的 `Instruction.subtree` 设计被验证正确；SKILL.md 目录形态（agentskills.io）在 Cline/Roo/Cursor 间收敛——PromptPack 的 Skill 实体设计被验证正确。

## 对 cfg4ai 的启示

1. **IR v0.3 候选变更**（按优先级）：
   - Instruction 增 `activation`（always/auto/glob/manual）+ `description`（B1，三工具刚需）；
   - `origin.scope` 与合并体系增第三层（system/managed/team，默认只读）（B2）；
   - 立项 `policy.` 实体或在 PromptPack/McpServer 加 `permissions` 子结构（B4/B3 的 `alwaysAllow` 升标准字段）；
   - 新实体 `ignore.`（patterns + enforcement: filter-auto-load \| block-tools \| edit-scope）（B8）；
   - `merge_policy` 支持 per-tool 方向与粒度声明（B6）。
2. **适配器层即可吸收、不必动 IR 的**：`serverUrl`/`streamableHttp`/`streamable-http` 键名与枚举映射、Roo timeout 单值映射、`${file:}` 原样搬运（并可提供 secretref file 后端的平滑升级路径）。
3. **采集纪律新增项**：
   - Cline/Roo 的 VS Code globalStorage（`mcp_settings.json`、custom_modes.yaml、toggle 状态）属"数据库内配置"，与文件配置并存——Detect 需声明两类 Location；
   - Cline hooks/plugins 是**可执行代码**，采集要过代码级安全提示，导出默认需显式确认；
   - Windsurf 双轨路径（`.devin/` vs `.windsurf/`、`/etc/devin` vs `/etc/windsurf`）与 memories/workflows 的 legacy 冻结状态，版本护栏 + Warning；
   - Aider 运行时产物清单（`.aider.chat.history.md` 等）入排除表。
4. **导出策略**：AGENTS.md 已成为五工具 + 基线多工具共同读取的"最低公分母"，cfg4ai 可增加 `export --target agents-md` 的零适配器普惠路径（自动作用域=subtree，always-on=根文件）。
5. **时效监控**：Windsurf→Devin 迁移、Cline `.clineignore` 弃用、Cursor rules `type` 字段消失——三者都印证 ADAPTERS §3 "待校准"机制与 golden-file 雷达的必要性；建议把本次五卡片并入 ADAPTERS §3.6 候选扩展表时同步标注这些变动点。

---

### 附：来源 URL 总表

- Cursor rules：https://cursor.com/cn/docs/rules （英文 https://docs.cursor.com/docs/rules）
- Cursor MCP：https://cursor.com/cn/docs/mcp
- Windsurf/Devin memories & rules：https://docs.devin.ai/desktop/cascade/memories
- Windsurf/Devin workflows：https://docs.devin.ai/desktop/cascade/workflows
- Windsurf/Devin AGENTS.md：https://docs.devin.ai/desktop/cascade/agents-md
- Windsurf/Devin MCP：https://docs.devin.ai/desktop/cascade/mcp
- Windsurf/Devin 首页（更名证据）：https://docs.devin.ai/
- Aider YAML 配置：https://aider.chat/docs/config/aider_conf.html
- Aider conventions：https://aider.chat/docs/usage/conventions.html
- Aider 社区 conventions 仓库：https://github.com/Aider-AI/conventions
- Cline rules：https://docs.cline.bot/customization/cline-rules.md
- Cline skills：https://docs.cline.bot/customization/skills.md
- Cline MCP：https://docs.cline.bot/mcp/mcp-overview.md
- Cline config/storage：https://docs.cline.bot/getting-started/config.md
- Cline .clineignore（弃用预告）：https://docs.cline.bot/customization/clineignore.md
- Cline 访问控制插件示例：https://github.com/cline/cline/blob/main/sdk/examples/plugins/gitignore-read-files-guard.ts
- Roo custom modes：https://docs.roocode.com/features/custom-modes/
- Roo custom instructions：https://docs.roocode.com/features/custom-instructions/
- Roo MCP：https://docs.roocode.com/features/mcp/using-mcp-in-roo/
- Roo .rooignore：https://docs.roocode.com/features/rooignore/
- Roo skills：https://docs.roocode.com/features/skills/
- 社区样本 awesome-cursorrules：https://github.com/PatrickJS/awesome-cursorrules
