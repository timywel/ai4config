// Package windsurf 是 Windsurf（→Devin）编辑器的适配器。
// 配置：.windsurf/rules/、.windsurfrules（legacy）、.devin/ 双读（品牌迁移）、workflows、mcp_config.json。
// 调研卡片：docs/research/tool-survey-a.md。
package windsurf

import (
	"context"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func init() { adapters.Register(&adapter{}) }

type adapter struct{}

func (a *adapter) Meta() adapters.ToolMeta {
	return adapters.ToolMeta{
		ID:          "windsurf",
		DisplayName: "Windsurf / Devin",
		MinVersion:  "1.0",
		MaxVersion:  "1.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportFull, Note: ".windsurf/rules/ 与 .devin/ 双读"},
			ir.KindMCP:         {Level: adapters.SupportFull, Note: "mcp_config.json"},
			ir.KindSkill:       {Level: adapters.SupportNone},
			ir.KindAgent:       {Level: adapters.SupportNone},
			ir.KindCommand:     {Level: adapters.SupportNone},
			ir.KindWorkflow:    {Level: adapters.SupportPartial, Note: ".windsurf/workflows/*.md"},
			ir.KindHook:        {Level: adapters.SupportNone},
			ir.KindSetting:     {Level: adapters.SupportPartial},
		},
	}
}

func (a *adapter) Import(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	return a.importLocation(ctx, loc)
}

func (a *adapter) Export(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	return a.exportBundle(ctx, b, opts)
}
