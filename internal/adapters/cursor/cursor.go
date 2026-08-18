// Package cursor 是 Cursor 编辑器的适配器。
// 配置：.cursor/rules/*.mdc（description/globs/alwaysApply）、.cursor/mcp.json、.cursorrules（legacy）。
// 调研卡片：docs/research/tool-survey-a.md。
package cursor

import (
	"context"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func init() { adapters.Register(&adapter{}) }

type adapter struct{}

func (a *adapter) Meta() adapters.ToolMeta {
	return adapters.ToolMeta{
		ID:          "cursor",
		DisplayName: "Cursor",
		MinVersion:  "1.0",
		MaxVersion:  "1.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportFull, Note: ".cursor/rules/*.mdc（globs/alwaysApply）"},
			ir.KindMCP:         {Level: adapters.SupportFull},
			ir.KindSkill:       {Level: adapters.SupportNone},
			ir.KindAgent:       {Level: adapters.SupportNone},
			ir.KindCommand:     {Level: adapters.SupportNone},
			ir.KindWorkflow:    {Level: adapters.SupportNone},
			ir.KindHook:        {Level: adapters.SupportNone},
			ir.KindSetting:     {Level: adapters.SupportFull},
		},
	}
}

func (a *adapter) Import(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	return a.importLocation(ctx, loc)
}

func (a *adapter) Export(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	return a.exportBundle(ctx, b, opts)
}
