// Package grokbuild 是 xAI Grok Build 工具的适配器。
// 配置：~/.grok/config.toml + 项目 .grok/config.toml、hooks（14 事件）、SKILL.md、subagents、MCP。
// 调研卡片：docs/research/tool-survey-c.md。
package grokbuild

import (
	"context"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func init() { adapters.Register(&adapter{}) }

type adapter struct{}

// Meta 工具元数据。Grok Build 生态：TOML 配置、14 hook 事件、零配置兼容读取 Claude Code 生态。
func (a *adapter) Meta() adapters.ToolMeta {
	return adapters.ToolMeta{
		ID:          "grokbuild",
		DisplayName: "Grok Build (xAI)",
		MinVersion:  "1.0",
		MaxVersion:  "1.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportFull},
			ir.KindMCP:         {Level: adapters.SupportFull},
			ir.KindSkill:       {Level: adapters.SupportFull, Note: "SKILL.md 目录形态"},
			ir.KindAgent:       {Level: adapters.SupportPartial, Note: "subagents 目录"},
			ir.KindCommand:     {Level: adapters.SupportPartial},
			ir.KindWorkflow:    {Level: adapters.SupportPartial, Note: "Rhai 脚本工作流进 x-grokbuild"},
			ir.KindHook:        {Level: adapters.SupportFull, Note: "14 事件"},
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
