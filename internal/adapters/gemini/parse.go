package gemini

import (
	"encoding/json"
	"strings"

	"github.com/timywel/ai4config/internal/core/ir"
)

// parseSettingsJSON 解析 settings.json：mcpServers → MCPServer；其余顶层键 → 不透明 Setting。
func parseSettingsJSON(data []byte, scope ir.Scope, originPath string) ([]ir.SettingEntry, []ir.MCPServer, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, nil, err
	}
	var settings []ir.SettingEntry
	var mcps []ir.MCPServer
	for key, val := range root {
		if key == "mcpServers" {
			mcps = parseMCPServers(val, scope, originPath)
			continue
		}
		settings = append(settings, ir.SettingEntry{
			Header:    ir.Header{ID: "setting.gemini." + key, IRVersion: 1, Origin: &ir.Origin{Tool: "gemini", Path: originPath, Scope: scope}},
			Key:       key,
			Value:     val, // 嵌套对象作为不透明 value
			ToolScope: []string{"gemini"},
		})
	}
	return settings, mcps, nil
}

// parseMCPServers 解析顶级 mcpServers（键名同 Claude）。
func parseMCPServers(val any, scope ir.Scope, originPath string) []ir.MCPServer {
	m, ok := val.(map[string]any)
	if !ok {
		return nil
	}
	var out []ir.MCPServer
	for name, confAny := range m {
		conf, ok := confAny.(map[string]any)
		if !ok {
			continue
		}
		s := ir.MCPServer{
			Header: ir.Header{
				ID:        "mcp." + sanitizeIDName(name),
				IRVersion: 1,
				Origin:    &ir.Origin{Tool: "gemini", Path: originPath, Scope: scope},
			},
			Name:      name,
			Transport: getStr(conf, "type", "stdio"),
			Command:   getStr(conf, "command", ""),
			Args:      getStrSlice(conf, "args"),
			Env:       getStrMap(conf, "env"),
			URL:       getStr(conf, "url", ""),
			Headers:   getStrMap(conf, "headers"),
			Cwd:       getStr(conf, "cwd", ""),
		}
		// gemini 特有：trust（绕过确认）、includeTools/excludeTools、timeout（毫秒）
		if trust, ok := conf["trust"].(bool); ok {
			s.Trust = &trust
		}
		s.EnabledTools = getStrSlice(conf, "includeTools")
		s.DisabledTools = getStrSlice(conf, "excludeTools")
		if t, ok := conf["timeout"].(float64); ok {
			s.Timeout = &ir.Timeout{StartupMs: int(t)} // gemini timeout 为毫秒
		}
		out = append(out, s)
	}
	return out
}

// parseGeminiMD GEMINI.md → instruction。
func parseGeminiMD(body string, scope ir.Scope, originPath string) ir.Instruction {
	return ir.Instruction{
		Header: ir.Header{
			ID:        "instruction.gemini-md",
			IRVersion: 1,
			Origin:    &ir.Origin{Tool: "gemini", Path: originPath, Scope: scope},
		},
		Activation: ir.ActivationAlways,
		AppliesTo:  []string{"gemini"},
		Priority:   defaultPriority(scope),
		Body:       body,
	}
}

func defaultPriority(scope ir.Scope) int {
	switch scope {
	case ir.ScopeProject, ir.ScopeLocal:
		return 200
	case ir.ScopeRemote:
		return 150
	default:
		return 100
	}
}

func sanitizeIDName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func getStr(m map[string]any, k, def string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return def
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
