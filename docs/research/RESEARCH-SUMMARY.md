# R1+R2 调研与对抗审计汇总

> 时间：2026-08-16 ｜ 范围：字段级调研（5 工具）+ 生态广度调研（11 工具）+ 红队 FMEA 与对抗用例
> 档案索引：`field-inventory-{claude-code,codex,copilot,gemini,zhanlu}.md`、`tool-survey-{a,b}.md`、`adversarial-cases.md`、`../review/REDTEAM.md`

## 1. 调研覆盖矩阵（回答"调研了什么"）

| 维度 | 覆盖 | 深度 |
|------|------|------|
| 字段级 inventory | Claude Code(~130+键/hooks 31 事件)、Codex(~160 键)、Copilot(~89)、Gemini(~240)、Zhanlu(22，本地实证) | 逐字段：类型/语义/默认值/scope/IR 承载状态 |
| 生态广度 | Cursor、Windsurf(→Devin)、Aider、Cline、Roo Code（tool-survey-a）；OpenCode、Amp、Goose、Zed、JetBrains(Junie/AI Assistant)、Trae（tool-survey-b）；Claude Desktop、Grok Build、Hermes、OpenClaw（tool-survey-c，2026-08-16 应用户指认补调） | 配置地图+能力矩阵+独特概念+IR 压力测试 |
| 红队对抗 | 12 组件 43 条 FMEA + 38 条对抗用例 | 26 条 v0.2 未定义行为被定位 |
| 真实样本 | 各档案"真实样本"小节（官方示例+社区仓库，URL+片段） | 样本量仍偏少，P0 期间持续扩充 |

**未覆盖声明（诚实边界）**：Claude env-vars 完整键表、plugins-reference 的 plugin.json 键表、Codex subagent TOML 层键表未逐字段抓取（档案内附核实方法）；`~/.claude.json` 完整键表无官方文档，需 P0 实采脱敏后枚举；Zhanlu 无公开文档，本机实证仅见 `permission` 顶级键，ADAPTERS §3.4 的 providers/mcp 段描述已降级"待校准"。

## 2. IR 结构性击穿汇总（合并去重，9 类）

| # | 击穿点 | 证据来源 | v0.3 处置 |
|---|--------|---------|-----------|
| K1 | **层级模型不足**：scope 仅 global/project——Claude local scope、Gemini 四层（system-defaults/override）、Cursor 团队规则、Windsurf System 企业层、OpenCode remote/MDM、Roo 全局 modes | 全部五路专家共识 | **D1：scope 扩展为五层** `managed>remote>local>project>global` |
| K2 | **Setting id 与点号 key 冲突**：`chat.mcp.access`、`mcp_servers.<id>.*` 等 250+ 真实键违反 `[a-z0-9-]` 字符集（红队 F1.5 同判） | 专家B、E | **D2：name 段字符集放行 `.`，解析规则改为首个点号分隔 type** |
| K3 | **Hook 无实体**：Claude 31 事件、Codex 11、Junie 7、Trae 兼容 Claude——事件模型事实趋同，标准化条件成熟 | 专家A、B、D | **D3：新增 `hook.` 一等实体**（标准事件交集+工具特有进 x-） |
| K4 | **激活/触发模型缺失**：Cursor 四模式、Windsurf trigger、JetBrains 五态、Zhanlu 语义路由、Trae scene——IR 无统一承载 | 专家B、C、D | **D4：统一 `activation` 枚举**（always/model-decision/glob/manual/scene）+ `invocation` |
| K5 | **PromptPack 公共字段缺口**：model/tools/mcp-servers/user-invocable/argument-hint（Copilot agent、Claude subagent、Roo mode、Amp skill↔MCP 绑定共有） | 专家A、B、D | **D5：增 5 个标准可选字段** |
| K6 | **McpServer 字段缺口**：cwd、enabled/disabled_tools、trust、auto_approve 白名单、oauth、headers_helper、env 间接寻址 `{env:VAR}` | 专家A、B、C、D | **D7：全部增为标准可选字段** |
| K7 | **Workflow 实体太薄**：Goose recipes（Jinja 参数化+子编排+重试）远超当前定义 | 专家D | **D14：增 parameters/steps/retry 可选字段**，复杂编排进 x- |
| K8 | **合并语义不可假设**：Cline skills 全局>项目（反转）、Roo 同 slug 整体覆盖、Claude settings 数组拼接去重、Codex enabled 正极性 | 专家A、C | **D16：ADAPTERS 增"目标语义差异表"**，IR 语义唯一权威、适配器负责双向转换 |
| K9 | **权限/审批模型无公共维度**：Roo groups+fileRegex、Cline 权限、Cursor/Windsurf allowlist、Zed profiles | 专家C、D | **暂 x- 承载**（P2 工具为主），记录为标准化候选；ignore 文件家族同理以 Setting 不透明 value 承载（D15） |

**收敛信号（归纳正确的部分）**：SKILL.md 目录形态全行业收敛（agentskills.io）；AGENTS.md 嵌套/子目录作用域四工具支持（`subtree` 设计被验证）；MCP 三键名映射（mcpServers/servers/mcp_servers）稳定。

## 3. 红队 Top 威胁与处置（对应 D8–D13）

| 威胁 | 级联链 | v0.3 处置 |
|------|--------|-----------|
| T-01 盘掉线→全量墓碑→清空配置 | 读取失败误判删除→export 空集写空 | **D8**：源目录不存在=中止采集（非墓碑）；项目层墓碑遮蔽全局同 id；空集导出需 `--force` |
| T-02 路径复用劫持项目身份 | rename 原地重建→collect 采错 profile | **D10**：collect 路径命中仍需指纹复核（first_commit 比对） |
| T-03 占位符回采断链→keyring 级联清理 | none 后端导出空值→回采覆盖 secretref | **D8 补**：导出物中占位符/空值回采时**永不覆盖**已有 secretref（冲突→Warning） |
| T-04 exports/ 归属真空→换机信任链断裂 | 不在白名单也不在排除清单 | **D9**：exports/ 入白名单；换机/重定位后 doctor 引导 rebase；hash 对比前字节级规范化（CRLF/BOM） |
| T-05 自由文本真 secret 经 sync 上远端 | 正文贴 key→Warning→CI 放行 5→push | **D11**：sync push 前置全仓扫描，命中即阻断或显式确认 |
| T-06 多 clone 互踩墓碑 | origin.path 相对形态相同→轮流墓碑 | **D10 补**：reconcile 限定本次实际采集的 Location 边界 |
| T-09 ai.base_url 投毒 | 恶意 pull 改端点→配置出域 | **D12**：AI 配置段变更后下次 --ai 强制重新 consent |
| T-10 blobs 不同步→overlay 悬空 | 换机查无 blob 无降级 | **D13**：blob 悬空降级 preserve/全量重渲染+Warning |

## 4. v0.3 设计决策表（17 项，依据可追溯）

| # | 决策 | 依据 |
|---|------|------|
| D1 | scope 五层 `managed>remote>local>project>global`；managed 默认只读不物化 | K1；Cursor 团队>项目、Claude local>project、OpenCode MDM 最高 |
| D2 | id 首个点号分隔 type，name 段放行 `.` 与大写 | K2 |
| D3 | hook 一等实体：event（标准交集）+matcher+handler{type,command,command_windows,url,prompt} | K3 |
| D4 | `activation` 统一枚举 always/model-decision/glob/manual/scene；command 用 `invocation` 存调用名 | K4 |
| D5 | PromptPack 增 model/tools/mcp_servers/user_invocable/argument_hint（均可选） | K5 |
| D6 | Instruction 增 name/description（Copilot 语义路由运行时字段） | 专家B |
| D7 | McpServer 增 cwd/enabled_tools/disabled_tools/trust/auto_approve/oauth/headers_helper；env 间接寻址规范 `{env:VAR}` | K6 |
| D8 | 删除语义三修复：目录不存在=中止；项目墓碑遮蔽全局；空集导出 --force；占位符回采不覆盖 | T-01/T-03/T-07 |
| D9 | exports/ 入 sync 白名单 + rebase + hash 规范化 + 确认选项集写死 | T-04/T-08 |
| D10 | collect 路径命中仍指纹复核；reconcile 限定本次 Location 边界 | T-02/T-06 |
| D11 | sync push preflight 全仓敏感扫描，命中阻断 | T-05 |
| D12 | AI 配置变更强制重新 consent | T-09 |
| D13 | blob 悬空降级链定义 | T-10 |
| D14 | Workflow 增 parameters/steps/retry | K7 |
| D15 | ignore 文件家族以 Setting 不透明 value 承载（`setting.<tool>.ignorefile`） | K9 |
| D16 | ADAPTERS 增目标语义差异表（数组拼接/极性/整体覆盖），IR 语义唯一权威 | K8 |
| D17 | 权限/审批模型暂不标准化，记录为未来候选 | K9（控制 v0.3 爆炸半径） |

## 5. 对原两问题的最终回答

**"归纳/汇总/输出的机制是什么"（v0.3 修正版）**：
- 归纳 = 并集调研（16 工具）→ 交集泛化为标准字段、差集入 x-、blob 兜底；每新工具必须回答"IR 要改什么"（证伪纪律）
- 汇总 = 五层 scope 的 merge-by-id/concat/遮蔽语义（IR-SCHEMA §2）
- 输出 = 能力矩阵（SupportLevel）驱动的投影 + 降级携带 + Verify round-trip 自检 + 导出清单信任链
- 验证 = golden-file（含注释保留用例）+ 对抗用例库 38 条 + 真实样本回归

**剩余已知风险**：Zhanlu 键结构待校准（本机实证仅 permission 键）；插件体系（K1 之外最大的私有化区域）全部走 x-，跨工具迁移时降级为 Warning——这是有意的边界，不是遗漏。
