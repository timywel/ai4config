// Package copilot 是 VS Code Copilot 工具的适配器。
// 配置地图：docs/ADAPTERS.md §3.3；字段档案：docs/research/field-inventory-copilot.md。
package copilot

import (
	"context"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func init() { adapters.Register(&adapter{}) }

type adapter struct{}

// Meta 工具元数据。Copilot 生态关键差异：.github/ 三件套、mcp.json 用 servers 键、
// 文件级 inputs/sandbox、instructions 的 applyTo 文件作用域。
func (a *adapter) Meta() adapters.ToolMeta {
	return adapters.ToolMeta{
		ID:          "copilot",
		DisplayName: "VS Code Copilot",
		MinVersion:  "1.99",
		MaxVersion:  "1.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportFull},
			ir.KindMCP:         {Level: adapters.SupportFull},
			ir.KindSkill:       {Level: adapters.SupportPartial, Note: "无独立 skill，instructions 近似承载"},
			ir.KindAgent:       {Level: adapters.SupportFull, Note: ".github/agents/*.agent.md"},
			ir.KindCommand:     {Level: adapters.SupportFull, Note: ".github/prompts/*.prompt.md"},
			ir.KindWorkflow:    {Level: adapters.SupportNone, Note: "无 workflow，降级为 command/instruction"},
			ir.KindHook:        {Level: adapters.SupportNone, Note: "无 hook 体系，跳过+Warning"},
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
