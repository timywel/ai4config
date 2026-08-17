package claudecode

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/timywel/ai4config/internal/core/ir"
)

// 各文件格式解析器（Claude Code 格式 → IR）。

// importRe 提取 CLAUDE.md 的 @path 引用（IR-SCHEMA §3.1 imports）。
var importRe = regexp.MustCompile(`(?:^|\s)@([~./\w][\w./\\-]*)`)

// extractImports 从指令正文提取 @path 引用清单。
func extractImports(body string) []ir.Import {
	seen := map[string]bool{}
	var out []ir.Import
	for _, m := range importRe.FindAllStringSubmatch(body, -1) {
		p := m[1]
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, ir.Import{Path: p, Resolved: true})
	}
	return out
}

// parseInstructionMD 纯 Markdown 指令文件 → Instruction（CLAUDE.md 等）。
func parseInstructionMD(id string, body string, scope ir.Scope, originPath string, sub map[string]string) ir.Instruction {
	inst := ir.Instruction{
		Header: ir.Header{
			ID:         id,
			IRVersion:  1,
			Origin:     &ir.Origin{Tool: "claude-code", Path: originPath, Scope: scope},
			Extensions: nil,
		},
		Activation: ir.ActivationAlways,
		AppliesTo:  []string{"claude-code"},
		Body:       body,
	}
	inst.Priority = defaultPriority(scope)
	if imports := extractImports(body); len(imports) > 0 {
		inst.Imports = imports
	}
	return inst
}

// defaultPriority 按 scope 给指令默认优先级（IR-SCHEMA §3.1）。
func defaultPriority(scope ir.Scope) int {
	switch scope {
	case ir.ScopeProject, ir.ScopeLocal:
		return 200
	case ir.ScopeRemote:
		return 150
	default: // global
		return 100
	}
}

// parsePromptPackMD frontmatter+正文 Markdown → PromptPack（agents/commands/skills）。
func parsePromptPackMD(data []byte, kind ir.EntityKind, fallbackName string) (ir.PromptPack, error) {
	var p ir.PromptPack
	body, ext, err := ir.UnmarshalMarkdownDoc(data, &p)
	if err != nil {
		return p, err
	}
	p.Extensions = ext
	p.Body = body
	p.Kind = kind
	if p.Name == "" {
		p.Name = fallbackName
	}
	// frontmatter 的 name 映射到 id；allowed-tools → tools
	if p.ID == "" {
		p.ID = string(kind) + "." + sanitizeIDName(p.Name)
	}
	if p.Origin != nil {
		p.Origin.Tool = "claude-code"
	}
	return p, nil
}

// sanitizeIDName 把任意名称规范为 id 合法 name 段（小写、非法字符转 -）。
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

// ---------- settings.json ----------

// settingsFile settings.json 顶层结构（局部字段；未知键并入 x-claude-code）。
type settingsFile struct {
	Model       string                     `json:"model"`
	Permissions map[string]any             `json:"permissions"`
	Env         map[string]string          `json:"env"`
	Hooks       map[string]json.RawMessage `json:"hooks"`
	MCPServers  json.RawMessage            `json:"mcpServers"`
}

// parseSettingsJSON settings.json → Settings + Hooks（+ 可选 MCP）。
// 返回 settings 条目、hooks、mcpServers 原始 JSON（由调用方决定如何并入）。
func parseSettingsJSON(data []byte, scope ir.Scope, originPath string) ([]ir.SettingEntry, []ir.Hook, []ir.MCPServer, error) {
	var sf settingsFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, nil, nil, err
	}
	var settings []ir.SettingEntry
	mk := func(key string, value any) {
		settings = append(settings, ir.SettingEntry{
			Header:    ir.Header{ID: "setting.claude-code." + key, IRVersion: 1, Origin: &ir.Origin{Tool: "claude-code", Path: originPath, Scope: scope}},
			Key:       key,
			Value:     value,
			ToolScope: []string{"claude-code"},
		})
	}
	if sf.Model != "" {
		mk("model", sf.Model)
	}
	if sf.Permissions != nil {
		mk("permissions", sf.Permissions) // 不透明 value，不参与跨工具翻译
	}
	if len(sf.Env) > 0 {
		mk("env", sf.Env)
	}

	// hooks 键 → Hook 实体
	hooks := parseHooks(sf.Hooks, scope, originPath)

	// mcpServers（settings.json 内嵌，较少见；.mcp.json 才是主路径）
	var mcps []ir.MCPServer
	if len(sf.MCPServers) > 0 {
		mcps, _ = parseMCPServers(sf.MCPServers, scope, originPath)
	}
	return settings, hooks, mcps, nil
}

// hookMatcherConf 单个 hook 配置（事件 → matcher 数组 → handler 数组）。
type hookMatcherConf struct {
	Matcher string `json:"matcher"`
	Hooks   []struct {
		Type    string `json:"type"`
		Command string `json:"command"`
		URL     string `json:"url"`
		Prompt  string `json:"prompt"`
		Timeout int    `json:"timeout"`
	} `json:"hooks"`
}

// parseHooks 把 settings.json 的 hooks 键（{event: [{matcher, hooks:[handler]}]}）解析为 Hook 实体。
func parseHooks(raw map[string]json.RawMessage, scope ir.Scope, originPath string) []ir.Hook {
	var out []ir.Hook
	for event, rawVal := range raw {
		var matchers []hookMatcherConf
		if err := json.Unmarshal(rawVal, &matchers); err != nil {
			continue
		}
		eventName := normalizeHookEvent(event)
		for _, mc := range matchers {
			for _, h := range mc.Hooks {
				hook := ir.Hook{
					Header: ir.Header{
						ID:        "hook." + sanitizeIDName(string(eventName)+"-"+mc.Matcher),
						IRVersion: 1,
						Origin:    &ir.Origin{Tool: "claude-code", Path: originPath, Scope: scope},
					},
					Event:   eventName,
					Matcher: ir.HookMatcher{Tool: mc.Matcher},
					Handler: ir.HookHandler{
						Type:       h.Type,
						Command:    h.Command,
						URL:        h.URL,
						Prompt:     h.Prompt,
						TimeoutSec: h.Timeout,
					},
				}
				out = append(out, hook)
			}
		}
	}
	return out
}

// normalizeHookEvent 把 Claude 事件名（PreToolUse 等）映射为 IR 标准词表（kebab-case）。
// 工具特有事件保留原名（进不了标准词表的由校验层记 Warning）。
func normalizeHookEvent(claudeEvent string) ir.HookEvent {
	switch claudeEvent {
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
		return ir.HookEvent(claudeEvent)
	}
}

// ---------- .mcp.json / mcpServers ----------

// mcpServerConf 单个 MCP server 配置（Claude 格式）。
type mcpServerConf struct {
	Type    string            `json:"type"` // stdio | sse | http
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// parseMCPServers 解析 mcpServers 对象（{name: conf}）为 MCPServer 列表。
func parseMCPServers(raw json.RawMessage, scope ir.Scope, originPath string) ([]ir.MCPServer, error) {
	var servers map[string]mcpServerConf
	if err := json.Unmarshal(raw, &servers); err != nil {
		return nil, err
	}
	out := make([]ir.MCPServer, 0, len(servers))
	for name, conf := range servers {
		transport := conf.Type
		if transport == "" {
			transport = "stdio" // Claude 默认 stdio
		}
		s := ir.MCPServer{
			Header: ir.Header{
				ID:        "mcp." + sanitizeIDName(name),
				IRVersion: 1,
				Origin:    &ir.Origin{Tool: "claude-code", Path: originPath, Scope: scope},
			},
			Name:      name, // 原始键名（导出键名唯一来源）
			Transport: transport,
			Command:   conf.Command,
			Args:      conf.Args,
			Env:       conf.Env,
			URL:       conf.URL,
			Headers:   conf.Headers,
		}
		out = append(out, s)
	}
	return out, nil
}

// parseMCPJSON 解析 .mcp.json 文件（{"mcpServers": {...}}）。
func parseMCPJSON(data []byte, scope ir.Scope, originPath string) ([]ir.MCPServer, error) {
	var f struct {
		MCPServers json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if len(f.MCPServers) == 0 {
		return nil, nil
	}
	return parseMCPServers(f.MCPServers, scope, originPath)
}
