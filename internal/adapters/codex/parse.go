package codex

import (
	"github.com/pelletier/go-toml/v2"

	"github.com/timywel/ai4config/internal/core/ir"
)

// config.toml 解析（Codex 格式 → IR）。
// 关键差异（ADAPTERS §2.10 差异表）：enabled 正极性取反、startup_timeout_sec→ms 换算、
// 未知顶层键作不透明 Setting（保真往返）。

// parseConfigTOML 解析 config.toml 为 settings + mcps + hooks。
func parseConfigTOML(data []byte, scope ir.Scope, originPath string) ([]ir.SettingEntry, []ir.MCPServer, []ir.Hook, error) {
	var root map[string]any
	if err := toml.Unmarshal(data, &root); err != nil {
		return nil, nil, nil, err
	}
	var settings []ir.SettingEntry
	var mcps []ir.MCPServer
	var hooks []ir.Hook

	for key, val := range root {
		switch key {
		case "mcp_servers":
			mcps = parseCodexMCP(val, scope, originPath)
		case "hooks":
			hooks = parseCodexHooks(val, scope, originPath)
		default:
			// 其余顶层键（model/approval_policy/sandbox_mode/profiles/features/...）→ 不透明 Setting
			settings = append(settings, ir.SettingEntry{
				Header:    ir.Header{ID: "setting.codex." + key, IRVersion: 1, Origin: &ir.Origin{Tool: "codex", Path: originPath, Scope: scope}},
				Key:       key,
				Value:     val,
				ToolScope: []string{"codex"},
			})
		}
	}
	return settings, mcps, hooks, nil
}

// parseCodexMCP 解析 [mcp_servers.<name>] 表（极性取反 + 单位换算）。
func parseCodexMCP(val any, scope ir.Scope, originPath string) []ir.MCPServer {
	serversMap, ok := val.(map[string]any)
	if !ok {
		return nil
	}
	var out []ir.MCPServer
	for name, confAny := range serversMap {
		conf, ok := confAny.(map[string]any)
		if !ok {
			continue
		}
		s := ir.MCPServer{
			Header: ir.Header{
				ID:        "mcp." + sanitizeIDName(name),
				IRVersion: 1,
				Origin:    &ir.Origin{Tool: "codex", Path: originPath, Scope: scope},
			},
			Name:          name, // 原始键名（导出还原）
			Transport:     getStr(conf, "type", "stdio"),
			Command:       getStr(conf, "command", ""),
			Args:          getStrSlice(conf, "args"),
			Env:           getStrMap(conf, "env"),
			URL:           getStr(conf, "url", ""),
			Cwd:           getStr(conf, "cwd", ""),
			EnabledTools:  getStrSlice(conf, "enabled_tools"),
			DisabledTools: getStrSlice(conf, "disabled_tools"),
		}
		// 极性取反：codex enabled（正）→ IR disabled
		if enabled, ok := getBool(conf, "enabled"); ok {
			s.Disabled = !enabled
		}
		// timeout：startup_timeout_sec（秒）→ startup_ms（毫秒）；tool_timeout_sec → tool_sec
		if st, ok := getInt(conf, "startup_timeout_sec"); ok {
			s.Timeout = &ir.Timeout{StartupMs: st * 1000}
		}
		if tt, ok := getInt(conf, "tool_timeout_sec"); ok {
			if s.Timeout == nil {
				s.Timeout = &ir.Timeout{}
			}
			s.Timeout.ToolSec = tt
		}
		out = append(out, s)
	}
	return out
}

// parseCodexHooks 解析 hooks 键：{Event: [{hooks: [handler]}]}（Codex 无 matcher 层）。
func parseCodexHooks(val any, scope ir.Scope, originPath string) []ir.Hook {
	eventsMap, ok := val.(map[string]any)
	if !ok {
		return nil
	}
	var out []ir.Hook
	for eventName, groupsAny := range eventsMap {
		groups, ok := groupsAny.([]any)
		if !ok {
			continue
		}
		event := normalizeCodexEvent(eventName)
		for _, gAny := range groups {
			g, ok := gAny.(map[string]any)
			if !ok {
				continue
			}
			handlers, _ := g["hooks"].([]any)
			for _, hAny := range handlers {
				h, ok := hAny.(map[string]any)
				if !ok {
					continue
				}
				hook := ir.Hook{
					Header: ir.Header{
						ID:        "hook." + sanitizeIDName(eventName),
						IRVersion: 1,
						Origin:    &ir.Origin{Tool: "codex", Path: originPath, Scope: scope},
					},
					Event: event,
					Handler: ir.HookHandler{
						Type:           getStr(h, "type", "command"),
						Command:        getStr(h, "command", ""),
						CommandWindows: getStr(h, "command_windows", ""), // TOML 别名
						TimeoutSec:     getIntDefault(h, "timeout", 600),
					},
				}
				out = append(out, hook)
			}
		}
	}
	return out
}

// normalizeCodexEvent Codex 事件名（PascalCase，与 Claude 同命名）→ IR 标准词表。
func normalizeCodexEvent(codexEvent string) ir.HookEvent {
	switch codexEvent {
	case "SessionStart":
		return ir.HookSessionStart
	case "SessionEnd":
		return ir.HookSessionEnd
	case "PreToolUse":
		return ir.HookPreToolUse
	case "PostToolUse":
		return ir.HookPostToolUse
	case "Notification":
		return ir.HookNotification
	case "Stop":
		return ir.HookStop
	case "UserPromptSubmit":
		return ir.HookUserPromptSubmit
	case "PreCompact":
		return ir.HookPreCompact
	default:
		return ir.HookEvent(codexEvent)
	}
}

// ---------- map 取值辅助 ----------

func getStr(m map[string]any, k, def string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return def
}

func getInt(m map[string]any, k string) (int, bool) {
	switch v := m[k].(type) {
	case int64:
		return int(v), true
	case int:
		return v, true
	case float64:
		return int(v), true
	}
	return 0, false
}

func getIntDefault(m map[string]any, k string, def int) int {
	if v, ok := getInt(m, k); ok {
		return v
	}
	return def
}

func getBool(m map[string]any, k string) (bool, bool) {
	v, ok := m[k].(bool)
	return v, ok
}

func getStrSlice(m map[string]any, k string) []string {
	raw, ok := m[k].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func getStrMap(m map[string]any, k string) map[string]string {
	raw, ok := m[k].(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}
