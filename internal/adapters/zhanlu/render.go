package zhanlu

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/timywel/ai4config/internal/core/ir"
)

// renderAgentsMD 多条 instruction 拼接为 AGENTS.md（边界注释）。
func renderAgentsMD(instructions []ir.Instruction) string {
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

// renderPackMD PromptPack → SKILL.md / agent|command .md。
func renderPackMD(p ir.PromptPack) []byte {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("name: " + p.Name + "\n")
	if p.Description != "" {
		sb.WriteString("description: " + p.Description + "\n")
	}
	if p.Activation != "" {
		sb.WriteString("activation: " + string(p.Activation) + "\n")
	}
	sb.WriteString("---\n")
	sb.WriteString(p.Body)
	return []byte(sb.String())
}

// renderZhanluJSON settings + mcp → zhanlu.json（防御式：mcp 段用 mcpServers 键）。
func renderZhanluJSON(settings []ir.SettingEntry, mcps []ir.MCPServer) ([]byte, error) {
	root := map[string]any{}
	for _, s := range settings {
		root[s.Key] = s.Value
	}
	if len(mcps) > 0 {
		servers := map[string]any{}
		for _, s := range mcps {
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
			name := s.Name
			if name == "" {
				name = ir.NameTail(s.ID)
			}
			servers[name] = conf
		}
		root["mcpServers"] = servers
	}
	return json.MarshalIndent(root, "", "  ")
}
