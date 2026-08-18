// Package all 聚合全部适配器（集中 blank import，由 cmd/cfg4ai 引用）。
// 未被 import 的适配器包其 init() 不会执行——新增适配器必须在此登记。
package all

import (
	_ "github.com/timywel/ai4config/internal/adapters/claudecode"
	_ "github.com/timywel/ai4config/internal/adapters/claudedesktop"
	_ "github.com/timywel/ai4config/internal/adapters/codex"
	_ "github.com/timywel/ai4config/internal/adapters/copilot"
	_ "github.com/timywel/ai4config/internal/adapters/gemini"
	_ "github.com/timywel/ai4config/internal/adapters/grokbuild"
	_ "github.com/timywel/ai4config/internal/adapters/zhanlu"
)
