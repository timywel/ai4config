# 字段清单：Zhanlu 湛卢（adapter id: `zhanlu`）

> 调研日期：2026-08-16 ｜ 调研人：P1 工具组（字段级调研）
> 方法：**本地实证**（无公开文档）。只读核查本机文件，敏感值一律 `<redacted>`。
> 实证基线：
> - `C:\Users\Wel\.config\zhanlu\zhanlu.json`（全局主配置）
> - `C:\Users\Wel\.config\zhanlu\.gitignore`
> - `C:\Users\Wel\.agents\`（`.skill-lock.json` + `skills/`，25 个 skill）
> - `F:\config-code\.zhanlu\agent-manager.json`（项目级实证样本）
> - 内置 skill 对照样本：`E:\Program Files\Zhanlu\resources\app\extensions\zhanlu\bin\zhanlu-plugins\zhanlu-sdd\skills\plan-review\SKILL.md`
> - 参照目录：`~/.claude`、`~/.codex`、`~/.copilot`、`~/.gemini` 顶层结构
>
> 承载状态图例：【标准字段】/【x- 承载】（`x-zhanlu`）/【无承载】。

## 0. 配置文件地图（实证 + 约定）

| 文件 / 目录 | scope | 状态 | IR 实体 |
|---|---|---|---|
| `~/.config/zhanlu/zhanlu.json` | global | ✅ 实证存在 | Setting |
| `~/.config/zhanlu/.gitignore` | global | ✅ 实证存在 | 【无承载】（提示该目录被 git 化管理；忽略 node/npm 生态文件与 `agent-manager.json`——暗示全局层会生成插件包管理文件与运行时状态文件） |
| `~/.config/zhanlu/zhanlu.jsonc` | global | ❓ 未见实证（ADAPTERS 标注待校准） | Setting |
| `~/.agents/skills/<name>/SKILL.md` | global | ✅ 实证 25 个 | Skill（PromptPack） |
| `~/.agents/.skill-lock.json` | global | ✅ 实证存在 | 【无承载】包管理锁定文件（见 §3） |
| `~/.agents/agents/`、`~/.agents/commands/` | global | ❓ 本机不存在（仅 `skills/` 一项内容） | Agent / Command |
| 项目根 `kilo.json` | project | ❓ 本项目未见（约定，待校准） | Setting |
| 项目根 `.zhanlu/agent-manager.json` | project | ✅ 实证存在 | 【无承载】Agent Manager 会话状态（见 §4） |
| `.kilo/agent/*.md` | project | 约定（环境信息明示 "Put new commands and agents in .kilo/"） | Agent |
| `.kilo/command/*.md` | project | 约定（本项目暂无实例） | Command |
| `<proj>/AGENTS.md` | project | 约定（环境信息明示） | Instruction |
| 全局 AGENTS.md | global | ❓ 约定位置待校准（`~/.agents/AGENTS.md` 本机不存在） | Instruction |

## 1. `zhanlu.json` 逐字段（实证）

本机实例仅含**一个顶级键** `permission`。任务预设的 `providers`/`mcp` 顶级键**本机未见**（ADAPTERS §3.4 "providers api_key 强制 secretref" 所述键在本机未配置；存在性待校准，见 §6 证伪）。

| 字段路径 | 类型 | 语义（推测） | IR 承载 |
|---|---|---|---|
| `permission` | object | 权限配置根 | 【标准字段】Setting `setting.zhanlu.permission`，不透明 value |
| `permission.bash` | object<pattern, verdict> | bash 命令权限表：键为命令 glob 模式（含空格分隔参数位，如 `"find * -name *"`、`"git log *"`、`"printenv PATH"`），值为裁决 | 【标准字段】同上 |
| `permission.bash.<pattern>` | enum string | 裁决值，本机 110 条实例**全部**为 `"allow"`；按行业惯例推测存在 `"deny"`/`"ask"` 值域（待校准） | 【标准字段】value 原样 |

实证要点：
- 模式语法为"命令 + 通配参数位"（`ls *`、`kubectl get *`），非正则；粒度到子命令与参数形态（`git stash list *` 与 `git log *` 分列）。
- 与 Claude `permissions.allow`（`Bash(npm run test:*)`）、Gemini `tools.allowed`（`run_shell_command(git)`）构成三工具权限映射族——语法互不相同，IR 层保持各不透明 value，翻译属导出器职责（同 gemini 击穿 #3）。
- 该文件在 `.gitignore` 管理下**未被忽略**（忽略清单仅 node 生态文件 + `agent-manager.json`），说明 `zhanlu.json` 属用户有意版本化资产——印证采集价值。

## 2. `~/.agents/skills/<name>/SKILL.md`（Skill 实体，实证 25 + 内置对照）

三类样本 frontmatter 比对：

| 样本 | frontmatter 键 | 形态 |
|---|---|---|
| 用户技能 `code-patterns`（及 bug-detective、architecture-design 等 24 个同批） | `name`、`description` | description 为 YAML 多行 literal block，内嵌"触发场景 + 触发词"结构化文本 |
| 内置插件 `zhanlu-sdd/plan-review` | `name`、`description` | description 为折叠多行英文（"Use when ... Also use when ..."） |
| 第三方 `find-skills`（vercel-labs/skills） | `name`、`description` | description 单行长字符串 |

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `name` | string | skill 标识，与目录名一致 | 【标准字段】→ PromptPack `name`（IR 校验规则 7 "id 末段=目录名"在此原生成立） |
| `description` | string | **触发语义载体**：模型据此做语义路由决定何时加载（"当需要 X 时自动使用此 Skill / Use when ..."） | 【标准字段】→ `description`；⚠️ 该字段在 zhanlu 是**运行时路由依据**而非纯展示，采集不得改写摘要 |
| 触发机制（无显式字段） | — | 无 `trigger`/`when` 键；触发 = LLM 对 description 的语义匹配 | 【x- 承载】IR `trigger.type` 词表 `slash-command\|mention\|manual\|hook` 无对应值；最近值为 `mention` 但不准确（击穿 #2） |
| 正文 | markdown | 指南/流程/清单 | 【标准字段】→ `prompt.md` |
| 附带文件（`scripts/`、`references/`、`assets/` 等） | 目录 | 技能资源 | 【标准字段】→ PromptPack `assets/`（内置 sdd 技能含多文件结构，形态吻合） |

## 3. `~/.agents/.skill-lock.json`（实证）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `version` | number | 锁文件格式版本（实证 `3`） | 【无承载】 |
| `skills.<name>.source` | string | 来源包（如 `vercel-labs/skills`） | 【无承载】包管理元数据 |
| `skills.<name>.sourceType` | string | 来源类型（`github`） | 【无承载】 |
| `skills.<name>.sourceUrl` | string | 仓库 URL | 【无承载】（可作 `origin` 补充信息，非必读） |
| `skills.<name>.skillFolderHash` | string | 目录 hash（实证为空串） | 【无承载】 |
| `skills.<name>.installedAt` / `updatedAt` | RFC3339 | 安装/更新时间 | 【无承载】 |
| `dismissed.findSkillsPrompt` | boolean | UI 提示关闭标记 | 【无承载】运行时状态 |

处置建议：**不采集**（锁定文件可由重装重建；类似 package-lock 之于源码）。若未来支持 skill 一键重装清单，可作为 `x-zhanlu` 附在 skill 实体上。

## 4. `.zhanlu/agent-manager.json`（项目级实证）

| 字段路径 | 类型 | 语义 | IR 承载 |
|---|---|---|---|
| `worktrees` | object | Agent Manager worktree 注册表（实证 `{}`） | 【无承载】运行时状态 |
| `sessions.<sessionId>` | object | 会话记录：`worktreeId`(null)、`createdAt`、`projectId`（URL-encoded 路径，如 `project:f%3A%5Cconfig-code`）、`rootPath`、`title` | 【无承载】运行时状态 |
| `taskReadMarkers.<sessionId>` | string | 任务已读游标（`m:msg_...\|t:...\|u:...\|r` 复合编码） | 【无承载】运行时状态 |
| `taskReadMarkersInitializedAt` | RFC3339 | 标记初始化时间 | 【无承载】 |

处置建议：**不采集**（纯运行时，类比 `~/.claude.json` projects 段；ADAPTERS 非专用文件局部 patch 原则同样适用——若需清理应由 prune 类命令处理）。

## 5. 参照目录顶层结构（供主项目参考，仅记结构）

- `~/.claude/`（✅ 存在，20 项）：`settings.json`（键：`env`——含 `ANTHROPIC_BASE_URL` 指向本机代理 `127.0.0.1:<port>`、`ANTHROPIC_AUTH_TOKEN: <redacted>`（占位式）、模型映射 `ANTHROPIC_DEFAULT_*_MODEL(_NAME)`、`API_TIMEOUT_MS`、`LANG` 等；`enabledPlugins`；`skipDangerousModePermissionPrompt`）、`config.json`（键：`primaryApiKey: <redacted>`）、`settings.local.json`、`skills/`、`plugins/`、`projects/`、`sessions/`、`backups/`、`cache/`、`history.jsonl`、`teams/` 等。**注意存在嵌套 `.claude/` 与 `nul` 异常项**（Windows 保留名文件，采集器需防御性跳过）。
- `~/.codex/`（✅ 存在）：仅 `skills/find-skills/`——无 `config.toml`/`AGENTS.md`/`auth.json`（本机 Codex 未配置主配置）。
- `~/.copilot/`（✅ 存在）：`ide/`、`skills/find-skills/`——尚无 `instructions/`、`agents/`、`mcp-config.json`（Agent Host 用户级目录未启用/未创建）。
- `~/.gemini/`（✅ 存在）：`antigravity/`（⚠️ Antigravity CLI 迁移已在本机落地）、`skills/find-skills/`。
- 横向发现：`find-skills` 被同一来源（`npx skills` 生态）铺进四个工具的 skills 目录——**多工具同 id 实体 reconcile 时不得跨工具合并**（IR-SCHEMA §2.1 "不跨工具自动合并条目"规则在此得到实证支撑）。

## 6. IR 击穿清单（zhanlu）

| # | 击穿点 | 等级 | 建议 |
|---|---|---|---|
| 1 | 任务预设/ADAPTERS §3.4 称 `zhanlu.json` 含 `providers`（api_key）与 `mcp` 段，**本机实证均未出现**；`zhanlu.jsonc` 变体、项目级 `kilo.json`、`.kilo/agent|command/*.md` 均无实例 | MAJOR（事实校准） | ADAPTERS §3.4 降级为"待校准"；适配器 Detect 实现按"键存在才采集"防御式编写，golden-file 暂缺 |
| 2 | skill 触发机制为 description 语义路由，IR `trigger.type` 词表无对应值 | minor | 词表增 `semantic`（或采集时置 `mention` + `x-zhanlu.routing: semantic`）；description 保真优先级提高（不得摘要改写） |
| 3 | `permission.bash` 值域仅实证 `"allow"`，`deny`/`ask` 存在性未证实 | minor | 适配器按开放枚举处理（未知值保留 + Warning），不做闭集校验 |
| 4 | 全局层运行时文件（`agent-manager.json` 被 `.gitignore` 显式忽略）与配置混居同一目录 | minor | 采集白名单制：仅 `zhanlu.json[c]`；`agent-manager.json` 永不入库 |
| 5 | 全局 agents/commands 目录、全局 AGENTS.md 位置均无实证 | 待校准 | 保留 ADAPTERS 待校准标记；P1 实现仅覆盖 skills + settings + 项目 AGENTS.md |

## 7. 真实样本（本地实证代替官方/社区样本）

1. **`zhanlu.json` 全量结构**（本机实证，值为真实裁决；无敏感项）：
   ```json
   {
     "permission": {
       "bash": {
         "ls *": "allow",
         "git log *": "allow",
         "find * -name *": "allow",
         "kubectl get *": "allow",
         "printenv PATH": "allow"
       }
     }
   }
   ```
   （实证共 110 条 pattern，值全为 `"allow"`；以上为节选。）
2. **`code-patterns/SKILL.md` frontmatter**（用户技能代表，24 个同型）：
   ```yaml
   ---
   name: code-patterns
   description: |
     当需要编写或审查代码规范时自动使用此 Skill。
     触发场景：
     - 代码规范
     - 命名规范
     触发词：代码规范、命名、注释、格式、代码风格
   ---
   ```
3. **`plan-review/SKILL.md` frontmatter**（内置插件技能代表）：
   ```yaml
   ---
   name: plan-review
   description: |
     Use when the user asks to review an execution plan, check if a plan is ready for
     execution, verify spec coverage, validate plan quality, or evaluate a plan file. ...
   ---
   ```
4. **`.skill-lock.json` 全量**（实证）：
   ```json
   { "version": 3,
     "skills": { "find-skills": { "source": "vercel-labs/skills", "sourceType": "github",
       "sourceUrl": "https://github.com/vercel-labs/skills.git", "skillFolderHash": "",
       "installedAt": "2026-01-27T01:49:24.127Z", "updatedAt": "2026-01-27T01:49:24.127Z" } },
     "dismissed": { "findSkillsPrompt": true } }
   ```
5. **`.zhanlu/agent-manager.json` 结构**（实证，session id 已缩写示意）：
   ```json
   { "worktrees": {},
     "sessions": { "ses_<id>": { "worktreeId": null, "createdAt": "2026-08-16T01:54:06.578Z",
       "projectId": "project:f%3A%5Cconfig-code", "rootPath": "f:\\config-code", "title": "New session - ..." } },
     "taskReadMarkers": { "ses_<id>": "m:msg_<id>|t:2026-08-16T05:03:07.913Z|u:2026-08-16T05:03:07.829Z|r" },
     "taskReadMarkersInitializedAt": "2026-08-16T01:54:06.543Z" }
   ```
6. **`~/.config/zhanlu/.gitignore` 全量**（实证）：`node_modules`、`package.json`、`package-lock.json`、`pnpm-lock.yaml`、`bun.lock`、`yarn.lock`、`.gitignore`、`agent-manager.json`。

## 8. 证伪结论：IR-SCHEMA v0.2 需要为 zhanlu 改什么

1. IR 本体**零改动可承载**当前实证面：`permission` → Setting 不透明 value；skill → PromptPack（`name`/`description`/正文/assets 全部对位）。
2. 需要动的是**词表与 ADAPTERS 而非实体结构**：
   - `trigger.type` 词表增 `semantic`（或明确 zhanlu skill 映射 `mention` + x- 标注）（击穿 #2）。
   - ADAPTERS §3.4 按实证降级：`providers`/`mcp`/`zhanlu.jsonc`/`kilo.json`/`.kilo/*` 全部"待校准"，Detect 防御式实现；采集白名单仅 `zhanlu.json` + `~/.agents/skills/` + 项目 `AGENTS.md`。
   - 明确不采集清单：`.skill-lock.json`、`agent-manager.json`、各工具 `skills/find-skills/` 之外的运行时目录。
3. 对 IR-SCHEMA §2.1 的一处实证确认：`find-skills` 同 id 存在于 4 个工具目录——reconcile "不跨工具合并"规则正确且必要，无需修改，建议在评审报告中引用本实证。
