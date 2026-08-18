# P1 验收记录 — cfg4ai M2 里程碑

> 日期：2026-08-18 ｜ 阶段：P1 生态扩展 ｜ 结论：**通过**

## 验收项核对（ARCHITECTURE §12 P1）

| 验收项 | 证据 | 结果 |
|--------|------|------|
| 七适配器 golden-file 全绿 | claudecode/codex/copilot/zhanlu/gemini/claude-desktop/grokbuild 各有 Import/Export/round-trip 测试，`go test ./...` 17 包全绿 | ✅ |
| Claude→Copilot e2e | `TestP1Acceptance_ClaudeToCopilot`：migrate code=0，3 文件（copilot-instructions.md 含指令、.vscode/mcp.json servers 键、settings.json） | ✅ |
| Codex→Zhanlu e2e | `TestP1Acceptance_CodexToZhanlu`：migrate code=0，2 文件（AGENTS.md 含指令、kilo.json mcpServers） | ✅ |
| relink 演示 | registry 测试：指纹归一 8 形态、link 合并/新建、relink 重定位、VerifyPath 路径劫持复核全绿 | ✅ |
| 麒麟 V10 冒烟 | **未执行**（需麒麟环境；CI 六 target 静态构建已断言 CGO_ENABLED=0，见备注） | ⏳ 待环境 |

## 新增适配器（5 个）

| 适配器 | 关键差异消化 |
|--------|-------------|
| copilot | `applyTo`↔file_patterns、servers 键、文件级 inputs/sandbox、点号 setting key |
| zhanlu | 防御式探测、zhanlu.json/kilo.json、.kilo/agent|command、skill 语义路由 |
| gemini | settings.json 嵌套键、顶级 mcpServers、trust/includeTools、timeout 毫秒 |
| claude-desktop | 轻量 mcpServers、局部 patch（保留既有键） |
| grokbuild | .grok/config.toml、subagents、SKILL.md |

## 过程中修正

- `adapters/all/all.go` 聚合注册遗漏（新适配器未登记导致"未注册"错误）——P1 验收 e2e 暴露并修复。
- collect 项目层探测基于进程 cwd，模拟环境测试时会采到真实项目配置（测试环境污染）——已在验收测试中通过 USERPROFILE 重定向规避；后续 collect 应支持显式 `--path` 隔离。

## 遗留（诚实清单）

1. 麒麟 V10 冒烟需在真实环境执行（当前仅 CI 静态构建断言）。
2. claude-desktop 远程连接器（UI/OAuth）不落公开文件，不在采集范围。
3. grokbuild 的 Rhai 脚本工作流进 x- 保留，未做语义转换。
4. 各适配器 managed 层（企业下发）只读采集，导出物化按设计不生成。

## 门禁

gofmt / go vet / go build（CGO_ENABLED=0）/ go test（17 包）全绿。