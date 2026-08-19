# P3 验收记录 — cfg4ai M4 里程碑（平台化）

> 日期：2026-08-18 ｜ 阶段：P3 平台化 ｜ 结论：**通过**

## 验收项核对（ARCHITECTURE §12 P3）

| 验收项 | 证据 | 结果 |
|--------|------|------|
| 第三方适配器插件接入示例 | `TestPluginHostRoundTrip`：编译 demo 插件进程，host↔plugin net/rpc over stdio 互通（Meta/Detect/Import/Export 全通） | ✅ |
| GUI 主界面可用 | `TestServerServesIndexAndAPI`：本地 Web 界面（net/http 零外部依赖，CGO_ENABLED=0）首页 + /api/entities | ✅ |
| 团队 profile 共享 | `TestTeamLayerMerges`：remote 层（profiles/team/*）参与合并导出，concat 层级序正确 | ✅ |

## P3 新增能力

| 模块 | 内容 |
|------|------|
| plugin | go-plugin net/rpc over stdio；AdapterRPC 四方法 JSON 传输；host（LoadPlugin）+ SDK（plugin.Serve）；第三方任意语言可对等实现 |
| gui | 本地 Web 界面（标准库 net/http，不引 Wails/Fyne 的 CGO/WebView 依赖，守住麒麟静态分发纪律） |
| 团队共享 | remote 层 profile（profiles/team/*）经 sync 白名单共享，参与合并 |

## 设计决策

- **GUI 选本地 Web 界面而非 Wails/Fyne**：Wails 需 WebView2（Windows 系统组件）、Fyne 需 OpenGL（CGO），都会破坏 CGO_ENABLED=0 静态分发纪律。net/http + 浏览器是零依赖、跨平台、麒麟兼容的最稳形态。
- **插件选 net/rpc 而非 gRPC**：go-plugin 的 net/rpc 模式无需 protobuf 依赖链，数据经 JSON 传输规避 gob 对 any 的限制。

## 遗留（诚实清单）

1. 插件暂无版本协商/签名校验（加载任意二进制有供应链风险，P3 后续加固）。
2. GUI 当前为只读浏览；collect/export 的 Web 操作交互待增强。
3. 团队共享的 remote 层当前为本地目录约定；真正的远程订阅下发（pull 团队 profile）依赖 sync 流程。

## 门禁

gofmt / go vet / go build（CGO_ENABLED=0）/ go test（全部包）全绿。