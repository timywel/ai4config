# cfg4ai

AI 编码工具配置的采集、治理与迁移系统（Go CLI + 桌面/Web 界面）。

把散落在 Claude Code、Codex、VS Code Copilot、Zhanlu、Gemini、Cursor 等 13 个 AI 编码工具里的个人配置（CLAUDE.md / AGENTS.md / MCP / skills / hooks / settings）统一采集进单一事实源（SSOT）仓库，经中间表示（IR）无损互转并交付到任意已接入工具。

## 安装

Windows（PowerShell）：

```powershell
irm https://raw.githubusercontent.com/timywel/ai4config/main/scripts/install.ps1 | iex
```

Linux / macOS / 麒麟：

```sh
curl -fsSL https://raw.githubusercontent.com/timywel/ai4config/main/scripts/install.sh | sh
```

或从 [Releases](https://github.com/timywel/ai4config/releases) 下载对应平台的二进制/安装包（zip / tar.gz / deb / rpm）。

## 快速上手

```sh
cfg4ai scan                                   # 发现本机已装的 AI 工具配置
cfg4ai collect                                # 采集进统一仓库（自动脱敏）
cfg4ai list                                   # 查看已采集内容
cfg4ai export --to codex --dry-run            # 预览交付到 Codex（不落盘）
cfg4ai migrate --from claude-code --to codex  # 一步迁移
cfg4ai gui                                    # 本地 Web 界面
cfg4ai-desktop                                # 原生桌面窗口（Gio）
```

## 特性

- 13 个内置工具适配器 + 外置进程插件机制（任意语言可扩展）
- 全局/项目五层配置模型，项目指纹关联与重定位
- 写入协议（原子写+快照回滚）、secret 三级后端降级链、敏感扫描
- 跨平台：Windows / macOS / Linux（含麒麟 Kylin、龙芯 loong64）
- AI 辅助语义迁移（consent 链 + 出域脱敏）

## 文档

- `docs/ARCHITECTURE.md` 总体架构
- `docs/IR-SCHEMA.md` 数据模型
- `docs/CLI-SPEC.md` 命令规范
- `docs/ADAPTERS.md` 适配器规范
- `docs/cfg4ai-design.html` 单文件设计总览

## License

MIT