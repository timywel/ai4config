package codex

import (
	"github.com/pelletier/go-toml/v2"

	"github.com/timywel/ai4config/internal/core/ir"
)

// IR → Codex TOML 渲染。关键逆向（ADAPTERS §2.10）：
// IR disabled → codex enabled（取反）；timeout startup_ms → startup_timeout_sec（/1000）。

// machineOnlyKeys 机器级键（项目级 config.toml 不可覆盖，ADAPTERS §3.2 trusted-gate 规则）。
var machineOnlyKeys = map[string]bool{
	"model_provider":  true,
	"model_providers": true,
	"notify":          true,
	"profiles":        true,
	"otel":            true,
	"otel_exporter":   true,
	"auth":            true,
	"cli_auth":        true,
	"history":         true,
	"projects":        true,
	"notice":          true,
}

// renderConfigTOML 渲染 config.toml。project=true 时跳过机器级键并记 Warning。
// TOML 不保注释 → 整块重写（IR-SCHEMA §1.3 免责），快照兜底由引擎/store 层负责。
func renderConfigTOML(settings []ir.SettingEntry, mcps []ir.MCPServer, hooks []ir.Hook, project bool) ([]byte, []ir.Warning, error) {
	root := map[string]any{}
	var warnings []ir.Warning

	// 不透明 setting 键原样写回
	for _, s := range settings {
		if project && machineOnlyKeys[s.Key] {
			warnings = append(warnings, ir.Warning{
				Kind:    "skip",
				Entity:  s.ID,
				Message: "机器级键 " + s.Key + " 项目级不可覆盖，已跳过",
			})
			continue
		}
		root[s.Key] = s.Value
	}

	// mcp_servers
	if len(mcps) > 0 {
		mcpServers := map[string]any{}
		for _, s := range mcps {
			conf := map[string]any{}
			conf["enabled"] = !s.Disabled // 极性取反回 codex
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
			if s.Cwd != "" {
				conf["cwd"] = s.Cwd
			}
			if len(s.EnabledTools) > 0 {
				conf["enabled_tools"] = s.EnabledTools
			}
			if len(s.DisabledTools) > 0 {
				conf["disabled_tools"] = s.DisabledTools
			}
			if s.Timeout != nil {
				if s.Timeout.StartupMs > 0 {
					conf["startup_timeout_sec"] = s.Timeout.StartupMs / 1000 // ms→s
				}
				if s.Timeout.ToolSec > 0 {
					conf["tool_timeout_sec"] = s.Timeout.ToolSec
				}
			}
			name := s.Name
			if name == "" {
				name = ir.NameTail(s.ID)
			}
			mcpServers[name] = conf
		}
		root["mcp_servers"] = mcpServers
	}

	// hooks
	if len(hooks) > 0 {
		root["hooks"] = renderCodexHooks(hooks)
	}

	data, err := toml.Marshal(root)
	return data, warnings, err
}

// renderCodexHooks IR Hook → Codex hooks 结构 {Event: [{hooks: [handler]}]}（无 matcher 层）。
func renderCodexHooks(hooks []ir.Hook) map[string]any {
	byEvent := map[string][]any{}
	var order []string
	for _, h := range hooks {
		ev := codexEventName(h.Event)
		if _, ok := byEvent[ev]; !ok {
			order = append(order, ev)
		}
		handler := map[string]any{"type": h.Handler.Type}
		if h.Handler.Command != "" {
			handler["command"] = h.Handler.Command
		}
		if h.Handler.CommandWindows != "" {
			handler["command_windows"] = h.Handler.CommandWindows
		}
		if h.Handler.TimeoutSec > 0 {
			handler["timeout"] = h.Handler.TimeoutSec
		}
		byEvent[ev] = append(byEvent[ev], map[string]any{"hooks": []any{handler}})
	}
	out := map[string]any{}
	for _, ev := range order {
		out[ev] = byEvent[ev]
	}
	return out
}

// codexEventName IR 标准事件 → Codex 事件名（PascalCase）。
func codexEventName(e ir.HookEvent) string {
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
