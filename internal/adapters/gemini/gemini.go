// Package gemini 是 Gemini CLI 工具的适配器。
// 配置地图：docs/ADAPTERS.md §3.5；字段档案：docs/research/field-inventory-gemini.md。
package gemini

import (
	"context"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func init() { adapters.Register(&adapter{}) }

type adapter struct{}

// Meta 工具元数据。Gemini 生态：settings.json 嵌套结构、顶级 mcpServers、GEMINI.md。
// 时效注意：官方已启动向 Antigravity CLI 过渡（2026-06-18 起），见 ADAPTERS §3.5。
func (a *adapter) Meta() adapters.ToolMeta {
	return adapters.ToolMeta{
		ID:          "gemini",
		DisplayName: "Gemini CLI",
		MinVersion:  "0.1",
		MaxVersion:  "0.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportFull},
			ir.KindMCP:         {Level: adapters.SupportFull},
			ir.KindSkill:       {Level: adapters.SupportNone, Note: "无 skill 体系"},
			ir.KindAgent:       {Level: adapters.SupportNone, Note: "降级为 instruction 附录"},
			ir.KindCommand:     {Level: adapters.SupportNone, Note: "降级为 instruction 附录"},
			ir.KindWorkflow:    {Level: adapters.SupportNone, Note: "降级为 instruction 附录"},
			ir.KindHook:        {Level: adapters.SupportNone, Note: "跳过+Warning"},
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
