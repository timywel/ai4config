package copilot

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/timywel/ai4config/internal/core/ir"
)

// IR → Copilot 文件格式渲染。

// renderCopilotInstructionsMD always 激活的多条 instruction 拼接为 copilot-instructions.md。
func renderCopilotInstructionsMD(instructions []ir.Instruction) string {
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

// renderInstructionFile 单条 glob instruction → .instructions.md（frontmatter 含 applyTo）。
func renderInstructionFile(inst ir.Instruction) []byte {
	var sb strings.Builder
	sb.WriteString("---\n")
	if inst.Name != "" {
		sb.WriteString("name: " + inst.Name + "\n")
	}
	if inst.Description != "" {
		sb.WriteString("description: " + inst.Description + "\n")
	}
	if len(inst.FilePatterns) > 0 {
		sb.WriteString("applyTo: \"" + strings.Join(inst.FilePatterns, ",") + "\"\n")
	}
	sb.WriteString("---\n")
	sb.WriteString(inst.Body)
	return []byte(sb.String())
}

// renderPackMD PromptPack → .prompt.md / .agent.md（frontmatter + 正文）。
func renderPackMD(p ir.PromptPack) []byte {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("name: " + p.Name + "\n")
	if p.Description != "" {
		sb.WriteString("description: " + p.Description + "\n")
	}
	if p.Model != nil {
		sb.WriteString(fmt.Sprintf("model: %v\n", p.Model))
	}
	if len(p.Tools) > 0 {
		sb.WriteString("tools: [" + strings.Join(p.Tools, ", ") + "]\n")
	}
	sb.WriteString("---\n")
	sb.WriteString(p.Body)
	return []byte(sb.String())
}

// renderMCPJSON MCP 列表 + 文件级扩展 → .vscode/mcp.json（servers 键）。
func renderMCPJSON(servers []ir.MCPServer, fileExt map[string]any) ([]byte, error) {
	svcs := map[string]any{}
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
		if s.EnvFile != "" {
			conf["envFile"] = s.EnvFile
		}
		if s.URL != "" {
			conf["url"] = s.URL
		}
		if len(s.Headers) > 0 {
			conf["headers"] = s.Headers
		}
		if s.Cwd != "" {
			conf["cwd"] = s.Cwd
		}
		name := s.Name
		if name == "" {
			name = ir.NameTail(s.ID)
		}
		svcs[name] = conf
	}
	root := map[string]any{"servers": svcs}
	// 文件级扩展（inputs/sandbox）回写
	if v, ok := fileExt["inputs"]; ok {
		root["inputs"] = v
	}
	if v, ok := fileExt["sandbox"]; ok {
		root["sandbox"] = v
	}
	return json.MarshalIndent(root, "", "  ")
}

// renderSettingsJSON Setting 条目 → settings.json（点号 key 重组对象）。
func renderSettingsJSON(settings []ir.SettingEntry) ([]byte, error) {
	root := map[string]any{}
	for _, s := range settings {
		root[s.Key] = s.Value
	}
	return json.MarshalIndent(root, "", "  ")
}
