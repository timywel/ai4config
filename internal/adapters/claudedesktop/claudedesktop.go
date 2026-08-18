// Package claudedesktop 是 Claude Desktop（桌面应用）的轻量适配器。
// 仅处理 claude_desktop_config.json 的 mcpServers（stdio）；远程连接器走 UI/OAuth 不落公开文件。
// 配置地图：docs/ADAPTERS.md §3.6 / research/tool-survey-c.md。
package claudedesktop

import (
	"context"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func init() { adapters.Register(&adapter{}) }

type adapter struct{}

func (a *adapter) Meta() adapters.ToolMeta {
	return adapters.ToolMeta{
		ID:          "claude-desktop",
		DisplayName: "Claude Desktop",
		MinVersion:  "1.0",
		MaxVersion:  "1.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportNone},
			ir.KindMCP:         {Level: adapters.SupportFull, Note: "claude_desktop_config.json mcpServers（stdio）"},
			ir.KindSkill:       {Level: adapters.SupportNone},
			ir.KindAgent:       {Level: adapters.SupportNone},
			ir.KindCommand:     {Level: adapters.SupportNone},
			ir.KindWorkflow:    {Level: adapters.SupportNone},
			ir.KindHook:        {Level: adapters.SupportNone},
			ir.KindSetting:     {Level: adapters.SupportPartial, Note: "仅 mcpServers 外的少数键"},
		},
	}
}

func (a *adapter) Import(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	return a.importLocation(ctx, loc)
}

func (a *adapter) Export(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	return a.exportBundle(ctx, b, opts)
}
