// Package codex 是 OpenAI Codex CLI 工具的适配器。
// 配置地图：docs/ADAPTERS.md §3.2；字段档案：docs/research/field-inventory-codex.md。
package codex

import (
	"context"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func init() { adapters.Register(&adapter{}) }

type adapter struct{}

// Meta 工具元数据（能力矩阵；Codex 与 Claude 生态的关键差异：TOML 格式、enabled 正极性、逐目录 AGENTS.md）。
func (a *adapter) Meta() adapters.ToolMeta {
	return adapters.ToolMeta{
		ID:          "codex",
		DisplayName: "Codex CLI",
		MinVersion:  "0.30",
		MaxVersion:  "0.x",
		Capabilities: adapters.CapabilitySet{
			ir.KindInstruction: {Level: adapters.SupportFull},
			ir.KindMCP:         {Level: adapters.SupportFull},
			ir.KindSkill:       {Level: adapters.SupportPartial, Note: "~/.codex/skills 目录"},
			ir.KindAgent:       {Level: adapters.SupportPartial, Note: "subagent 体系较新，字段映射 Partial"},
			ir.KindCommand:     {Level: adapters.SupportNone, Note: "无 command 概念，降级为 prompt/skill"},
			ir.KindWorkflow:    {Level: adapters.SupportNone, Note: "无独立 workflow，降级处理"},
			ir.KindHook:        {Level: adapters.SupportFull},
			ir.KindSetting:     {Level: adapters.SupportFull},
		},
	}
}

// Import / Export 见 import.go / export.go。
func (a *adapter) Import(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	return a.importLocation(ctx, loc)
}

func (a *adapter) Export(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	return a.exportBundle(ctx, b, opts)
}
