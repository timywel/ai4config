package migrate

import (
	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

// filterForTarget 过滤：--only 限定 + applies_to 过滤（W2 步骤；--include-foreign 纳入异构条目）。
func (e *Engine) filterForTarget(b *ir.Bundle, target adapters.ToolID, req ExportRequest) (*ir.Bundle, []ir.Warning) {
	var warnings []ir.Warning
	out := *b // 浅拷贝结构，逐字段过滤

	only := map[ir.EntityKind]bool{}
	for _, k := range req.Only {
		only[k] = true
	}
	want := func(kind ir.EntityKind) bool {
		return len(only) == 0 || only[kind]
	}

	// applies_to 过滤：条目 applies_to 非空且不含目标且不含 all，且未 --include-foreign → 跳过
	keepByAppliesTo := func(inst *ir.Instruction) bool {
		if req.IncludeForeign || len(inst.AppliesTo) == 0 {
			return true
		}
		for _, t := range inst.AppliesTo {
			if t == string(target) || t == "all" {
				return true
			}
		}
		warnings = append(warnings, ir.Warning{
			Kind:    "skip",
			Entity:  inst.ID,
			Message: "applies_to 不含目标工具，已跳过（--include-foreign 可纳入）",
		})
		return false
	}

	if !want(ir.KindInstruction) {
		out.Instructions = nil
	} else {
		filtered := out.Instructions[:0]
		for _, inst := range out.Instructions {
			if keepByAppliesTo(&inst) {
				filtered = append(filtered, inst)
			}
		}
		out.Instructions = filtered
	}
	if !want(ir.KindMCP) {
		out.MCPServers = nil
	}
	if !want(ir.KindSkill) {
		out.Skills = nil
	}
	if !want(ir.KindAgent) {
		out.Agents = nil
	}
	if !want(ir.KindCommand) {
		out.Commands = nil
	}
	if !want(ir.KindWorkflow) {
		out.Workflows = nil
	}
	if !want(ir.KindHook) {
		out.Hooks = nil
	}
	if !want(ir.KindSetting) {
		out.Settings = nil
	}
	return &out, warnings
}

// applyCapabilities 能力矩阵降级（ADAPTERS §5 两级规则）。
// 目标不支持的实体 → 最近概念映射 / instruction 附录 / 跳过，全部记 Warning。
func (e *Engine) applyCapabilities(b *ir.Bundle, caps adapters.CapabilitySet) (*ir.Bundle, []ir.Warning) {
	var warnings []ir.Warning
	out := *b

	level := func(kind ir.EntityKind) adapters.Capability {
		if c, ok := caps[kind]; ok {
			return c
		}
		return adapters.Capability{Level: adapters.SupportNone}
	}

	// workflow：目标无 workflow → 降级为 command（最近概念）或 instruction 附录
	if len(out.Workflows) > 0 {
		if level(ir.KindWorkflow).Level == adapters.SupportNone {
			for _, wf := range out.Workflows {
				if level(ir.KindCommand).Level != adapters.SupportNone {
					wf.Kind = ir.KindCommand
					out.Commands = append(out.Commands, wf)
					warnings = append(warnings, ir.Warning{Kind: "degrade", Entity: wf.ID, Message: "目标无 workflow，降级为 command"})
				} else {
					out.Instructions = append(out.Instructions, packToInstruction(wf))
					warnings = append(warnings, ir.Warning{Kind: "degrade", Entity: wf.ID, Message: "目标无 workflow/command，降级为 instruction 附录"})
				}
			}
			out.Workflows = nil
		}
	}

	// command：目标无 command → 降级为 skill 或 prompt（instruction 附录）
	if len(out.Commands) > 0 && level(ir.KindCommand).Level == adapters.SupportNone {
		for _, c := range out.Commands {
			if level(ir.KindSkill).Level != adapters.SupportNone {
				c.Kind = ir.KindSkill
				out.Skills = append(out.Skills, c)
				warnings = append(warnings, ir.Warning{Kind: "degrade", Entity: c.ID, Message: "目标无 command，降级为 skill"})
			} else {
				out.Instructions = append(out.Instructions, packToInstruction(c))
				warnings = append(warnings, ir.Warning{Kind: "degrade", Entity: c.ID, Message: "目标无 command/skill，并入 instruction 附录"})
			}
		}
		out.Commands = nil
	}

	// mcp：目标无 MCP 支持 → 跳过 + Warning
	if len(out.MCPServers) > 0 && level(ir.KindMCP).Level == adapters.SupportNone {
		for _, m := range out.MCPServers {
			warnings = append(warnings, ir.Warning{Kind: "degrade", Entity: m.ID, Message: "目标不支持 MCP，已跳过"})
		}
		out.MCPServers = nil
	}

	// settings：Partial/None 时由适配器自行决定（此处仅提示）
	if len(out.Settings) > 0 && level(ir.KindSetting).Level == adapters.SupportNone {
		for _, s := range out.Settings {
			warnings = append(warnings, ir.Warning{Kind: "degrade", Entity: s.ID, Message: "目标不支持 setting 迁移，已跳过"})
		}
		out.Settings = nil
	}

	return &out, warnings
}

// packToInstruction 把 PromptPack 降级为 instruction 附录（frontmatter 信息并入正文）。
func packToInstruction(p ir.PromptPack) ir.Instruction {
	body := "\n<!-- cfg4ai:degraded " + string(p.Kind) + " " + p.ID + " -->\n"
	if p.Description != "" {
		body += "**" + p.Description + "**\n\n"
	}
	body += p.Body
	return ir.Instruction{
		Header: ir.Header{
			ID:         "instruction.degraded-" + p.ID,
			IRVersion:  p.IRVersion,
			Origin:     p.Origin,
			Extensions: p.Extensions,
		},
		Activation: ir.ActivationAlways,
		AppliesTo:  []string{"all"},
		Body:       body,
	}
}
