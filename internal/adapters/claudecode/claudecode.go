// Package claudecode 是 Claude Code 工具的适配器。
// 配置地图：docs/ADAPTERS.md §3.1；字段档案：docs/research/field-inventory-claude-code.md。
package claudecode

import (
	"context"
	"fmt"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func init() { adapters.Register(&adapter{}) }

// Import / Export 见 import.go / export.go。
func (a *adapter) Import(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	return a.importLocation(ctx, loc)
}

func (a *adapter) Export(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	return nil, fmt.Errorf("claudecode: Export 未实现（T7.3）")
}

type adapter struct{}

// Meta 工具元数据（能力矩阵：Claude Code 全实体支持）。
func (a *adapter) Meta() adapters.ToolMeta {
	return adapters.ToolMeta{
		ID:          "claude-code",
		DisplayName: "Claude Code",
		MinVersion:  "1.0",
		MaxVersion:  "2.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportFull},
			ir.KindMCP:         {Level: adapters.SupportFull},
			ir.KindSkill:       {Level: adapters.SupportFull},
			ir.KindAgent:       {Level: adapters.SupportFull},
			ir.KindCommand:     {Level: adapters.SupportPartial, Note: "commands 已并入 skills（legacy 兼容）"},
			ir.KindWorkflow:    {Level: adapters.SupportPartial, Note: "无独立 workflow 概念，降级为 command"},
			ir.KindHook:        {Level: adapters.SupportFull},
			ir.KindSetting:     {Level: adapters.SupportFull},
		},
	}
}
