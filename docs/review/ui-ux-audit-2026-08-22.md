# cfg4ai 桌面端 UI/UX 质量审计报告（2026-08-22）

## 审计方式
- 自建窗口截图工具（scripts/snapshot-app.ps1），对三主题（A 深色专业 / B 浅色清爽 / D 玻璃拟态）× 关键页面逐页截图审查
- WCAG 对比度计算（三主题 × 关键文本对）
- 全量 go test ./... + go vet + 构建验证

## 发现与修复

| # | 问题 | 级别 | 修复 |
|---|------|------|------|
| 1 | TextInverse 令牌缺失：chip/toast/nav 选中态误用 cs.Surface，D 主题下文字半透明不可见 | P0 | Colors 加 TextInverse（恒白），6 处误用全部改用它 |
| 2 | 健康分恒为 0：采集器 skill 缺 ir_version（42 ERROR）+ 桌面 Validate 未传 RegisteredTools | P0 | ir 包加 CurrentVersion 常量；采集器补齐；profile.Save 兜底 normalize；桌面传注册表 |
| 3 | 导航选中态与设计稿不符且对比度差（Accent 底白字 3.9:1） | P1 | 改设计稿风格：SurfaceHover 底 + Accent 文字 + Accent 左竖线 |
| 4 | 内容区无滚动（内容超高被裁切） | P1 | contentList 垂直滚动容器 |
| 5 | 快照页"保存定时计划"按钮渲染异常（紫条无文字） | P1 | 改 Flex 包装结构 |
| 6 | 密钥页长 secretref URL 溢出 | P1 | MaxLines=1 + Truncator 截断 |
| 7 | 仓库路径超长溢出 | P2 | 超长中间省略 + 限一行 |
| 8 | A 主题 TextSecondary 对比度 4.34（<4.5） | P2 | 提亮至 #B8C0DA（6.36:1） |
| 9 | D 主题 Accent 对比度 3.74（<4.5） | P2 | 提亮至 #BFA6FF（5.44:1） |
| 10 | D 玻璃卡片半透明感弱 | P2 | Surface/Border 透明度提高（8%/18%） |
| 11 | 删除操作无二次确认 | P1 | ConfirmModal 接入（删除/批量删除） |
| 12 | 异步操作无 loading 反馈 | P2 | 全局"处理中…"指示条 |

## 主题对比度结论（修复后）
- A 深色专业：全部 ≥4.5（正文 9.9 AAA）
- B 浅色清爽：全部 ≥5.4（正文 17.8 AAA）
- D 玻璃拟态：全部 ≥5.4（正文 13.9 AAA）

## 验证
- go test ./... 全绿（0 FAIL）
- 三主题仪表盘/实体/迁移/快照/密钥/一致性/历史/活动/发现页截图逐张审查通过
- 健康分恢复 100，校验 errors=0 warnings=0