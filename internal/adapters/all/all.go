// Package all 聚合全部适配器（集中 blank import，由 cmd/cfg4ai 引用）。
// 未被 import 的适配器包其 init() 不会执行——新增适配器必须在此登记。
package all

import (
// P0 适配器（实现后取消注释）：
// _ "github.com/timywel/ai4config/internal/adapters/claudecode"
// _ "github.com/timywel/ai4config/internal/adapters/codex"
)
