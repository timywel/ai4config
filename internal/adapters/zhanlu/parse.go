package zhanlu

import (
	"encoding/json"
	"strings"

	"github.com/timywel/ai4config/internal/core/ir"
)

// Zhanlu 各文件格式 → IR。防御式解析：未知/缺失键不报错，进不透明 setting 或 x-zhanlu。

// parseZhanluJSON 解析 zhanlu.json 主配置（防御式）。
// 已知：mcp/mcpServers 段、providers（api_key 为敏感值）；其余顶层键 → 不透明 setting。
func parseZhanluJSON(data []byte, scope ir.Scope, originPath string) ([]ir.SettingEntry, []ir.MCPServer, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, nil, err
	}
	var settings []ir.SettingEntry
	var mcps []ir.MCPServer

	for key, val := range root {
		switch key {
		case "mcp", "mcpServers", "mcp_servers":
			mcps = parseMCPSegment(val, scope, originPath)
		default:
			// providers/permission/model 等 → 不透明 setting（providers.api_key 由脱敏管线处理）
			settings = append(settings, ir.SettingEntry{
				Header:    ir.Header{ID: "setting.zhanlu." + key, IRVersion: 1, Origin: &ir.Origin{Tool: "zhanlu", Path: originPath, Scope: scope}},
				Key:       key,
				Value:     val,
				ToolScope: []string{"zhanlu"},
			})
		}
	}
	return settings, mcps, nil
}

// parseMCPSegment 解析 mcp 段（{name: conf}，结构待校准，防御式）。
func parseMCPSegment(val any, scope ir.Scope, originPath string) []ir.MCPServer {
	m, ok := val.(map[string]any)
	if !ok {
		return nil
	}
	// 兼容 {servers: {...}} 或直接 {name: conf}
	if inner, ok := m["servers"].(map[string]any); ok {
		m = inner
	} else if inner, ok := m["mcpServers"].(map[string]any); ok {
		m = inner
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
				Origin:    &ir.Origin{Tool: "zhanlu", Path: originPath, Scope: scope},
			},
			Name:      name,
			Transport: getStr(conf, "type", "stdio"),
			Command:   getStr(conf, "command", ""),
			Args:      getStrSlice(conf, "args"),
			Env:       getStrMap(conf, "env"),
			URL:       getStr(conf, "url", ""),
			Headers:   getStrMap(conf, "headers"),
		}
		out = append(out, s)
	}
	return out
}

// parseAgentsMD AGENTS.md → instruction（always）。
func parseAgentsMD(body string, scope ir.Scope, originPath string, subtree string) ir.Instruction {
	id := "instruction.agents"
	if subtree != "" {
		id = "instruction.agents." + sanitizeIDName(strings.ReplaceAll(subtree, "/", "-"))
	}
	return ir.Instruction{
		Header: ir.Header{
			ID:        id,
			IRVersion: 1,
			Origin:    &ir.Origin{Tool: "zhanlu", Path: originPath, Scope: scope},
		},
		Activation: ir.ActivationAlways,
		AppliesTo:  []string{"zhanlu"},
		Priority:   defaultPriority(scope),
		Subtree:    subtree,
		Body:       body,
	}
}

// parsePackMD SKILL.md / agent/*.md / command/*.md → PromptPack（frontmatter + 正文）。
// zhanlu skill 的 description 语义路由 → activation: model-decision。
func parsePackMD(data []byte, kind ir.EntityKind, fallbackName string, scope ir.Scope, originPath string) ir.PromptPack {
	var p ir.PromptPack
	body, ext, err := ir.UnmarshalMarkdownDoc(data, &p)
	if err != nil {
		return ir.PromptPack{}
	}
	p.Extensions = ext
	p.Body = body
	p.Kind = kind
	if p.Name == "" {
		p.Name = fallbackName
	}
	if p.ID == "" {
		p.ID = string(kind) + "." + sanitizeIDName(p.Name)
	}
	if p.Activation == "" {
		if kind == ir.KindSkill {
			p.Activation = ir.ActivationModelDecision // zhanlu skill 语义路由
		} else if kind == ir.KindCommand {
			p.Activation = ir.ActivationManual
		}
	}
	p.Origin = &ir.Origin{Tool: "zhanlu", Path: originPath, Scope: scope}
	return p
}

// defaultPriority 按 scope 给指令默认优先级。
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

// sanitizeIDName 名称规范为 id 合法 name 段。
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
