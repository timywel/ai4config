# cfg4ai 总体架构设计

> 版本：v0.3（吸收 R1 字段级调研 + R2 红队审计；决策依据 [research/RESEARCH-SUMMARY.md](./research/RESEARCH-SUMMARY.md) D1–D17）｜ 状态：已评审 ｜ 正式名：**cfg4ai**（2026-08-17 定名，GitHub 零占用验证）
> 关联文档：[IR-SCHEMA.md](./IR-SCHEMA.md) ｜ [CLI-SPEC.md](./CLI-SPEC.md) ｜ [ADAPTERS.md](./ADAPTERS.md) ｜ 评审：[review/REVIEW-REPORT.md](./review/REVIEW-REPORT.md) ｜ 红队：[review/REDTEAM.md](./review/REDTEAM.md)

## 1. 背景与目标

### 1.1 问题

AI 编码工具（Claude Code、Codex CLI、VS Code Copilot、Zhanlu、Gemini CLI、Cursor……）各自维护一套私有配置格式与目录约定。用户在工具间切换、重装系统、更换设备时，配置迁移完全靠手工，成本高且易丢失。

### 1.2 目标

| # | 目标 | 说明 |
|---|------|------|
| G1 | 统一采集与整理 | 纳入 SSOT 管理（含删除传播与防误判，IR-SCHEMA §2.3） |
| G2 | 全局/项目双层模型 | 用户/项目双层继承；采集侧识别五层 scope（IR-SCHEMA §1.2） |
| G3 | 项目关联与重定位 | 项目目录迁移/改名后凭指纹重新关联 |
| G4 | 跨工具迁移交付 | 规则引擎确定性转换 + AI 语义级转换 |
| G5 | 跨平台 | Windows、macOS、Linux（含麒麟，amd64/arm64/loong64） |

### 1.3 非目标

- 实时双向同步；团队权限体系；secret 明文托管
- **企业管理层（managed）与远程订阅层（remote）的写入**：采集可读，物化归企业/组织渠道
- 可执行插件（TS/Starlark）的跨工具语义翻译（IR 保真存储 + 降级 Warning，见 ADAPTERS §5）

## 2. 核心设计：Hub-and-Spoke + IR

```
Claude Code ──┐
Codex CLI  ───┤ Importer →  [ IR / SSOT 仓库 ]  → Exporter ├──→ Claude Code
Copilot    ───┤                                          ├──→ Codex
Zhanlu     ───┘                                          ├──→ 任意新工具
```

第一性原理支撑（RESEARCH-SUMMARY §5）：
1. **信息守恒**：blobs/raw 快照是事实层，IR 是索引与视图——IR 表达不了的内容 blob 兜底（保真保险丝）。
2. **IR 表达力上限 = 调研完备性**：16 工具调研档案见 `docs/research/`；每接入新工具必须回答"IR 要改什么"（证伪纪律）。
3. **导出 = 能力矩阵驱动的投影**，降级是信息携带而非丢弃。
4. **正确性靠证伪机制逼近**：round-trip diff + golden-file + 对抗用例库（38 条）+ 真实样本回归。

保真分级与 `x-<tool>` 生命周期：IR-SCHEMA §1.3/§1.1。

## 3. 分层模型

profile 分全局/项目两层（物化语义）；采集侧识别五层 scope（`managed>remote>local>project>global`，IR-SCHEMA §1.2）。merge 语义见 IR-SCHEMA §2（merge-by-id=浅字段级；concat=层级序两段式；**墓碑遮蔽**：项目层墓碑遮蔽全局同 id）。**IR 语义唯一权威**：目标工具合并语义差异（数组拼接/极性/整体覆盖/优先级反转）由适配器双向转换消化（ADAPTERS §2 差异表）。

导出物化布局由目标适配器唯一决定（ADAPTERS §3 "导出布局"列）。

## 4. 项目注册表与指纹关联

`registry.yaml` 维护 `project_id ↔ 路径 ↔ profile`：

```yaml
projects:
  - id: prj_01JXYZ...
    name: config-code
    paths: [F:\config-code]     # 历史路径，重定位追加
    fingerprint:
      git_remote: github.com/user/config-code   # 规范化：去协议/去 .git/host 小写/scp 转标准
      root_name: config-code
      first_commit: 9fceb02...
    profile: profiles/projects/prj_01JXYZ...
    same_remote_as: []
    linked_tools: [claude-code, zhanlu]
```

- 优先级：规范化 git_remote > root_name+first_commit > root_name。
- **二次判别**：指纹命中后需 first_commit 一致+用户确认才合并，否则新建 pid 记 `same_remote_as`。
- **路径命中也须指纹复核（D10，红队 T-02 修复）**：collect/link 时即使路径直接命中注册项，仍比对 first_commit；不匹配 → 警告并建议 relink（防 rename 原地重建劫持）。
- 并发防护：§7 锁机制。

## 5. 迁移引擎

### 5.1 管线

```
Load → Merge(五层，含墓碑遮蔽) → Map → Assist(可选，引擎层) → Render → Verify(两级) → Write(写入协议)
```

职责切分：引擎负责 Merge/Map/Assist/Verify；适配器只做 Import 与 Render/Write，不接触 AI。

Verify 两级：格式校验（必做）；round-trip 自检（best-effort，`Export` 返回 `[]WrittenFile` → 重新 Import → 语义 diff → Warnings）。

**空集导出保护（D8，红队 T-01 修复）**：merged Bundle 条目数为 0 且目标已有文件 → 必须 `--force` + 显式警告文案，否则拒绝执行。

### 5.2 规则优先，AI 兜底

| 转换类型 | 方式 |
|---------|------|
| 字段映射/格式重组 | 规则引擎+模板 |
| 语义改写/冲突建议/语言适配 | AI 接口 |

- provider 可插拔（OpenAI 兼容），默认引导本地/私有端点。
- **确认机制**：AI 结果必须确认后落盘；`--yes` 不豁免；无人值守显式 `--ai-approve`（记决策日志）。
- **无 AI 也可用**：未配置 AI 时系统完全可用，语义级转换退化为"原样搬运 + 警告"。
- **出域 consent**：首次使用显式同意；**AI 配置段（provider/base_url/model）变更后下次 `--ai` 强制重新 consent（D12，红队 T-09 修复——防 sync pull 投毒端点）**；脱敏范围=secret+内网地址+可配置正则；日志默认只记元数据，记原文需 `ai.log_payload=true` 且强制 gitignore；企业端点 allowlist。

### 5.3 写入协议（强制规范）

- **原子粒度**：单文件原子（temp+rename）；批量非原子，快照补偿（全部 temp 就位→逐一 rename→失败逆序清理+快照恢复）。
- temp 与目标**同卷同目录**；写后 Sync+rename+父目录 Sync（Unix）。
- Windows `SHARING_VIOLATION`/`ACCESS_DENIED` 指数退避重试，终失败报"文件被占用"及路径。
- symlink 写入前 `EvalSymlinks` 穿透；快照记录链接关系。
- 统一 `internal/atomicfile` 实现，适配器禁手写。
- **外来内容识别与确认选项集（D9 写死）**：凭 `exports/<tool>/<scope>/manifest.yaml` 判定；hash 对比前做字节级规范化（CRLF→LF、去 BOM，防确认疲劳）。确认选项固定为 `overwrite / skip / view-diff / backup-overwrite`；`--force` = 全部 `backup-overwrite`。
- **IDE 热重载**：Detect 进程检测；export/restore 时提示"需重启/Reload Window 生效"。

## 6. 适配器规范

```go
type Adapter interface {
    Meta() ToolMeta
    Detect(ctx context.Context) ([]Location, error)                        // 只读；含进程运行检测
    Import(ctx context.Context, loc Location) (*ir.Bundle, error)
    Export(ctx context.Context, b *ir.Bundle, opts ExportOpts) ([]WrittenFile, error)
}
```

要点：全方法带 ctx；能力矩阵 `map[EntityKind]Capability`（SupportLevel None/Partial/Full+Note，EntityKind 含 hook）；`Export` 返回写入清单供 Verify。详见 [ADAPTERS.md](./ADAPTERS.md)。

## 7. 存储布局（SSOT 仓库）

```
$CFG4AI_HOME/
├── config.yaml              # AI provider、secrets 后端、merge 默认策略
├── registry.yaml
├── profiles/{global,projects/<pid>}/   # manifest + instructions/ + mcp.yaml + <kinds>/ + hooks.yaml + settings.yaml
├── exports/<tool>/<scope>/manifest.yaml   # 导出清单（sync 白名单内，D9）
├── snapshots/<timestamp>/   # manifest+blob 引用（去重）
├── blobs/<sha256>/          # 脱敏后内容寻址（不入库）
├── secrets.age              # 可选加密文件后端（0600，gitignore 强制）
├── cache/  logs/            # 不入库
└── .lock                    # 仓库级写锁
```

- **并发**：写操作持 `.lock`（gofrs/flock）；读快照读；stale lock 检测入 doctor；sync 持锁。
- **权限**：目录 0700/文件 0600 写后校验；云同步/共享挂载强 Warning；缓存/快照/blobs 落 `%LOCALAPPDATA%`。
- **快照/GC**：范围=SSOT 全量+（export 触发时）目标配置区；retention 默认 20 份+按天去重；blob 标记-清除。

## 8. 跨平台适配

| 差异点 | 策略 |
|--------|------|
| 路径 | `platform/paths` 封装（XDG/%APPDATA%/Application Support）；IR 内部 `/` |
| 换行符 | 保持源风格（采集探测），未知则 LF；转换 opt-in；仓库 `.gitattributes` 钉 `eol=lf` |
| 符号链接 | 采集 lstat 不跟随（越权防护）；写入穿透；export 回 Unix 还原、Windows 复制+Warning |
| 权限位 | IR `mode`（八进制，仅 Unix） |
| 长路径 | `longPathAware` manifest；doctor 报基准路径长度 |
| 麒麟 | linux 构建 + **CGO_ENABLED=0 强制纪律**；V10 服务器（headless）+UKUI 桌面冒烟矩阵 |
| 架构 | windows/amd64、darwin/amd64、darwin/arm64、linux/amd64、linux/arm64、linux/loong64（未实测标注） |

**构建纪律**：发布强制 `CGO_ENABLED=0`；CI 对全部 target 断言 static（ldd 检查）。纯 Go 模式 DNS/用户解析差异在 doctor/测试矩阵留痕。

**发布工程**：macOS 签名+notarytool 公证；Windows 代码签名；无证书期走 Homebrew/Scoop/winget。

## 9. 安全设计

- **Secret 三级降级链**：系统 keyring（99designs/keyring）→ 加密文件（secrets.age，CI/headless 出路）→ none（占位符）。IR 记录 `secret_backend`；doctor 输出状态。
- **脱敏入库管线**：blob 存脱敏后内容；双 hash（raw_hash 增量比对/stored_hash 指向 blob）；落盘强制"先扫描替换→落盘→零命中校验"。
- **敏感扫描分级**：结构化字段命中默认抽取（可否决）；自由文本命中仅 Warning（IR-SCHEMA §5 规则 4）。
- **sync 白名单**：`profiles/`、`registry.yaml`、`config.yaml`、`exports/`（D9——导出清单随库走，换机/重定位后 doctor 引导 rebase 路径重写）；其余强制 gitignore（sync init 自动写入基线）。
- **sync preflight（D11，红队 T-05 修复）**：push 前全仓敏感扫描（含自由文本），命中即阻断或显式确认——CI 不得对 sync 放行退出码 5。
- **blob 悬空降级（D13，红队 T-10 修复）**：blobs 不入库；换机后 `imports[].blob` 悬空 → 导出降级为 preserve 正文（引用路径仍在）+Warning；`raw_blob` 悬空 → 回退全量重渲染+Warning。
- AI consent 见 §5.2；keyring 上限/级联清理见 IR-SCHEMA §3.6。

## 10. 技术选型（已定：Go）

> 决策依据 [review/GO-VS-RUST.md](./review/GO-VS-RUST.md)，2026-08-16 拍板。

| 项 | 选择 | 备注 |
|----|------|------|
| 语言 | **Go 1.23+** | 已决策；代价清单已吸收（GO-VS-RUST §6） |
| CLI / TUI | spf13/cobra ／ charmbracelet/bubbletea(P2) | — |
| YAML | gopkg.in/yaml.v3 | Node API 保注释；golden-file 显式覆盖注释保留 |
| TOML | pelletier/go-toml/v2 | 不保注释→整块重写+快照兜底 |
| Diff | sergi/go-diff | — |
| Keyring | 99designs/keyring | file 后端降级刚需 |
| Git | go-git | 纯 Go |
| 文件锁 | gofrs/flock | — |
| 加密文件后端 | age（filippo.io/age） | — |
| AI provider | OpenAI 兼容协议自封装 | 默认引导本地端点 |
| 测试 | testing + testscript + golden-file + **对抗用例回归**（38 条） | adversarial-cases.md |
| 发布 | goreleaser | 六 target |

## 11. Go 模块结构

```
github.com/<org>/cfg4ai
├── cmd/cfg4ai/main.go
├── internal/
│   ├── core/
│   │   ├── ir/         # IR 模型、校验（12 条）、merge（map[id] 索引、墓碑遮蔽）
│   │   ├── profile/    # 五层 scope 读写、物化、ir_version 链式迁移
│   │   ├── registry/   # 注册表、指纹规范化、二次判别、路径命中复核
│   │   ├── migrate/    # 管线编排（Merge/Map/Assist/Verify/Write、空集保护）
│   │   ├── aiassist/   # AI provider、consent 状态机（配置变更强制重确认）
│   │   └── secrets/    # 三级后端、敏感扫描、脱敏管线、sync preflight
│   ├── adapters/
│   │   ├── all/        # 聚合 blank import
│   │   ├── claudecode/  codex/  copilot/  zhanlu/  gemini/ ...
│   │   └── registry.go
│   ├── platform/paths/
│   ├── atomicfile/     # 写入协议唯一实现
│   └── store/          # SSOT、快照、blob、锁、导出清单（rebase）
└── docs/
```

依赖方向：`cmd → migrate → {adapters, aiassist, profile, store}`；`adapters → ir`；`aiassist → ir`。无环。不设 `pkg/`（P3 外置进程插件不需要公开 Go API）。

## 12. 路线图

| 阶段 | 内容 | 验收标准 |
|------|------|---------|
| P0 | IR v0.3 模型 + store（锁/快照/写入协议/脱敏管线）+ claudecode/codex 适配器（含 hooks）+ `collect`/`export` + 双层 merge + 最小 `link` | Claude↔Codex 指令/MCP/hooks 互转；字段级 round-trip 无丢失；**对抗用例库 P0 子集回归通过**；双层继承 e2e |
| P1 | copilot/zhanlu/gemini 适配器 + claude-desktop（轻量 MCP 适配，与 claudecode 共享代码）+ grokbuild + 指纹 relink + 独立 `diff` + secrets 三级后端完整化 | 七适配器 golden-file 全绿 + Claude→Copilot、Codex→Zhanlu e2e；relink 成功；麒麟 V10 冒烟 |
| P2 | aiassist 完整 consent 链 + TUI + `sync`（白名单+preflight）+ 扩展适配器（cursor/windsurf/aider/cline/roo/opencode…，调研卡片已备：research/tool-survey-{a,b}.md） | AI 转换确认链完整；sync preflight 阻断演示通过；TUI 完成 collect/export 全流程 |
| P3 | 团队 profile 共享、GUI、外置进程插件（hashicorp/go-plugin，gRPC over stdio——不用 Go plugin，不支持 Windows；顺带允许非 Go 语言适配器） | 第三方适配器接入示例 |

**时效跟踪**（适配器排期依据）：Gemini CLI→Antigravity 过渡（2026-06-18 起）；Windsurf→Devin Desktop 品牌迁移（适配器 `.devin//.windsurf/` 双读）；Cline `.clineignore` 弃用预告（采集标 legacy）；Zed Rules 已废弃（拆 Skills+Instructions）；Claude commands 并入 skills（legacy 兼容）。

## 13. 风险与开放问题

1. **工具格式漂移**：版本护栏+golden-file 雷达；时效实例见 §12。
2. **IR 表达力**：五层 scope+hook 实体后覆盖率大幅提升（20 工具调研验证：5 个逐字段 + 15 个调研卡片）；残余私有化区（插件体系/权限模型）走 x-，属有意边界（D17）。
3. **红队残余风险**：Top10 已全部处置（D8–D13）；完整 FMEA 与残余评级见 [review/REDTEAM.md](./review/REDTEAM.md)，对抗用例回归纳入 CI（38 条）。
4. **AI 转换不可控** → consent+确认+`--ai-approve`+配置变更重确认。
5. **IDE 热重载** → 进程检测+提示。
6. **开放问题**：profile 远程订阅（团队下发）；watch 模式；~~语言选型~~（已定 Go，2026-08-16）；~~项目命名~~（已定 cfg4ai，2026-08-17）。
