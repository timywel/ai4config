// Package zhanlu 是 Zhanlu（湛卢）工具的适配器。
// 配置地图：docs/ADAPTERS.md §3.4；字段档案：docs/research/field-inventory-zhanlu.md（本机实证）。
package zhanlu

import (
	"context"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func init() { adapters.Register(&adapter{}) }

type adapter struct{}

// Meta 工具元数据。Zhanlu 生态：zhanlu.json 主配置、AGENTS.md、~/.agents/skills、
// 项目级 .kilo/agent|command。部分键结构待校准，探测需防御式（键缺失容忍）。
func (a *adapter) Meta() adapters.ToolMeta {
	return adapters.ToolMeta{
		ID:          "zhanlu",
		DisplayName: "Zhanlu (湛卢)",
		MinVersion:  "1.0",
		MaxVersion:  "1.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportFull},
			ir.KindMCP:         {Level: adapters.SupportPartial, Note: "zhanlu.json mcp 段结构待校准"},
			ir.KindSkill:       {Level: adapters.SupportFull, Note: "~/.agents/skills/<name>/SKILL.md"},
			ir.KindAgent:       {Level: adapters.SupportFull, Note: ".kilo/agent/*.md"},
			ir.KindCommand:     {Level: adapters.SupportFull, Note: ".kilo/command/*.md"},
			ir.KindWorkflow:    {Level: adapters.SupportNone, Note: "降级为 command/instruction"},
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
