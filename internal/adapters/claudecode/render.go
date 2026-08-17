package claudecode

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/timywel/ai4config/internal/core/ir"
)

// 渲染器：IR → Claude Code 文件格式（ADAPTERS §3.1 导出布局）。

// renderClaudeMD 多条 instruction 物化为单个 CLAUDE.md（边界注释拼接，采集时可还原）。
func renderClaudeMD(instructions []ir.Instruction) string {
	var sb strings.Builder
	for i, inst := range instructions {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("<!-- cfg4ai:begin %s -->\n", inst.ID))
		sb.WriteString(strings.TrimRight(inst.Body, "\n"))
		sb.WriteString(fmt.Sprintf("\n<!-- cfg4ai:end %s -->\n", inst.ID))
	}
	return sb.String()
}

// renderSettingsJSON 把 settings + hooks 组装为 settings.json。
// permissions/env 等不透明 value 原样还原；hooks 由 IR Hook 反组装为 Claude 事件结构。
func renderSettingsJSON(settings []ir.SettingEntry, hooks []ir.Hook) ([]byte, error) {
	root := map[string]any{}
	for _, s := range settings {
		root[s.Key] = s.Value
	}
	if len(hooks) > 0 {
		root["hooks"] = renderHooks(hooks)
	}
	return json.MarshalIndent(root, "", "  ")
}

// renderHooks IR Hook 列表 → Claude settings.json 的 hooks 键结构。
// {event: [{matcher, hooks: [handler]}]}，按 event+matcher 聚合。
func renderHooks(hooks []ir.Hook) map[string]any {
	type key struct{ event, matcher string }
	groups := map[key][]ir.HookHandler{}
	var order []key
	for _, h := range hooks {
		ev := claudeEventName(h.Event)
		k := key{event: ev, matcher: h.Matcher.Tool}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], h.Handler)
	}
	out := map[string]any{}
	// 按 event 聚合 matchers
	byEvent := map[string][]any{}
	var eventOrder []string
	for _, k := range order {
		handlers := []any{}
		for _, hh := range groups[k] {
			handlers = append(handlers, renderHookHandler(hh))
		}
		matcherObj := map[string]any{"hooks": handlers}
		if k.matcher != "" {
			matcherObj["matcher"] = k.matcher
		}
		if _, ok := byEvent[k.event]; !ok {
			eventOrder = append(eventOrder, k.event)
		}
		byEvent[k.event] = append(byEvent[k.event], matcherObj)
	}
	for _, ev := range eventOrder {
		out[ev] = byEvent[ev]
	}
	return out
}

// renderHookHandler 单个 handler → Claude 格式。
func renderHookHandler(h ir.HookHandler) map[string]any {
	m := map[string]any{"type": h.Type}
	if h.Command != "" {
		m["command"] = h.Command
	}
	if h.URL != "" {
		m["url"] = h.URL
	}
	if h.Prompt != "" {
		m["prompt"] = h.Prompt
	}
	if h.TimeoutSec > 0 {
		m["timeout"] = h.TimeoutSec
	}
	return m
}

// claudeEventName IR 标准事件 → Claude 事件名（PascalCase）。
func claudeEventName(e ir.HookEvent) string {
	switch e {
	case ir.HookSessionStart:
		return "SessionStart"
	case ir.HookSessionEnd:
		return "SessionEnd"
	case ir.HookPreToolUse:
		return "PreToolUse"
	case ir.HookPostToolUse:
		return "PostToolUse"
	case ir.HookNotification:
		return "Notification"
	case ir.HookStop:
		return "Stop"
	case ir.HookUserPromptSubmit:
		return "UserPromptSubmit"
	case ir.HookPreCompact:
		return "PreCompact"
	default:
		return string(e)
	}
}

// renderMCPJSON MCP 列表 → .mcp.json（{"mcpServers": {name: conf}}），name 取原始键名。
func renderMCPJSON(servers []ir.MCPServer) ([]byte, error) {
	mcp := map[string]any{}
	for _, s := range servers {
		conf := map[string]any{}
		if s.Transport != "" && s.Transport != "stdio" {
			conf["type"] = s.Transport
		}
		if s.Command != "" {
			conf["command"] = s.Command
		}
		if len(s.Args) > 0 {
			conf["args"] = s.Args
		}
		if len(s.Env) > 0 {
			conf["env"] = s.Env
		}
		if s.URL != "" {
			conf["url"] = s.URL
		}
		if len(s.Headers) > 0 {
			conf["headers"] = s.Headers
		}
		name := s.Name // 原始键名（导出键名唯一来源）
		if name == "" {
			name = ir.NameTail(s.ID)
		}
		mcp[name] = conf
	}
	root := map[string]any{"mcpServers": mcp}
	return json.MarshalIndent(root, "", "  ")
}

// renderPackMD PromptPack → frontmatter+正文 Markdown（agents/commands/skills）。
func renderPackMD(p ir.PromptPack) ([]byte, error) {
	fm := map[string]any{"name": p.Name}
	if p.Description != "" {
		fm["description"] = p.Description
	}
	if len(p.Tools) > 0 {
		if p.Kind == ir.KindSkill {
			fm["allowed-tools"] = p.Tools
		} else {
			fm["tools"] = p.Tools
		}
	}
	if p.Model != nil {
		fm["model"] = p.Model
	}
	// 手动构造 frontmatter（保证 name 在前）
	var sb strings.Builder
	sb.WriteString("---\n")
	order := []string{"name", "description", "tools", "allowed-tools", "model"}
	for _, k := range order {
		if v, ok := fm[k]; ok {
			sb.WriteString(fmt.Sprintf("%s: %s\n", k, formatYAMLScalar(v)))
		}
	}
	sb.WriteString("---\n")
	sb.WriteString(p.Body)
	return []byte(sb.String()), nil
}

// formatYAMLScalar 简单 YAML 标量/列表格式化（frontmatter 用）。
func formatYAMLScalar(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []string:
		return "[" + strings.Join(t, ", ") + "]"
	default:
		return fmt.Sprintf("%v", t)
	}
}
