package copilot

import (
	"encoding/json"
	"strings"

	"github.com/timywel/ai4config/internal/core/ir"
)

// Copilot 各文件格式 → IR（ADAPTERS §3.3）。

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

// parsePlainInstruction 纯 Markdown 指令（copilot-instructions.md，always 激活）。
func parsePlainInstruction(id, body string, scope ir.Scope, originPath string) ir.Instruction {
	return ir.Instruction{
		Header: ir.Header{
			ID:        id,
			IRVersion: 1,
			Origin:    &ir.Origin{Tool: "copilot", Path: originPath, Scope: scope},
		},
		Activation: ir.ActivationAlways,
		AppliesTo:  []string{"copilot"},
		Priority:   defaultPriority(scope),
		Body:       body,
	}
}

// parseInstructionFile .instructions.md → Instruction（applyTo → file_patterns，glob 激活）。
func parseInstructionFile(data []byte, fallbackName string, scope ir.Scope, originPath string) ir.Instruction {
	var tmp struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		ApplyTo     string `yaml:"applyTo"`
	}
	body, ext, _ := ir.UnmarshalMarkdownDoc(data, &tmp)
	name := tmp.Name
	if name == "" {
		name = fallbackName
	}
	inst := ir.Instruction{
		Header: ir.Header{
			ID:         "instruction." + sanitizeIDName(name),
			IRVersion:  1,
			Origin:     &ir.Origin{Tool: "copilot", Path: originPath, Scope: scope},
			Extensions: ext,
		},
		Name:        tmp.Name,
		Description: tmp.Description,
		Activation:  ir.ActivationGlob,
		AppliesTo:   []string{"copilot"},
		Priority:    defaultPriority(scope),
		Body:        body,
	}
	if tmp.ApplyTo != "" {
		inst.FilePatterns = []string{tmp.ApplyTo}
	}
	return inst
}

// parsePackMD .prompt.md / .agent.md → PromptPack（frontmatter + 正文）。
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
	p.Origin = &ir.Origin{Tool: "copilot", Path: originPath, Scope: scope}
	return p
}

// ---------- mcp.json（VS Code：servers 键 + 文件级 inputs/sandbox） ----------

type mcpServerConf struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	EnvFile string            `json:"envFile"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Cwd     string            `json:"cwd"`
}

// parseMCPJSON 解析 .vscode/mcp.json：servers → MCPServer；inputs/sandbox → fileExtensions。
func parseMCPJSON(data []byte, scope ir.Scope, originPath string) ([]ir.MCPServer, map[string]any, error) {
	var f struct {
		Servers map[string]mcpServerConf `json:"servers"`
		Inputs  []any                    `json:"inputs"`
		Sandbox map[string]any           `json:"sandbox"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, nil, err
	}
	var out []ir.MCPServer
	for name, conf := range f.Servers {
		transport := conf.Type
		if transport == "" {
			if conf.URL != "" {
				transport = "http"
			} else {
				transport = "stdio"
			}
		}
		s := ir.MCPServer{
			Header: ir.Header{
				ID:        "mcp." + sanitizeIDName(name),
				IRVersion: 1,
				Origin:    &ir.Origin{Tool: "copilot", Path: originPath, Scope: scope},
			},
			Name:      name,
			Transport: transport,
			Command:   conf.Command,
			Args:      conf.Args,
			Env:       conf.Env,
			EnvFile:   conf.EnvFile,
			URL:       conf.URL,
			Headers:   conf.Headers,
			Cwd:       conf.Cwd,
		}
		out = append(out, s)
	}
	// 文件级扩展位（IR-SCHEMA §3.2 file_extensions）
	var fileExt map[string]any
	if len(f.Inputs) > 0 || len(f.Sandbox) > 0 {
		fileExt = map[string]any{}
		if len(f.Inputs) > 0 {
			fileExt["inputs"] = f.Inputs
		}
		if len(f.Sandbox) > 0 {
			fileExt["sandbox"] = f.Sandbox
		}
	}
	return out, fileExt, nil
}

// parseSettingsJSON .vscode/settings.json → Setting 条目（点号 key 路径）。
func parseSettingsJSON(data []byte, scope ir.Scope, originPath string) []ir.SettingEntry {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}
	var out []ir.SettingEntry
	for key, val := range root {
		out = append(out, ir.SettingEntry{
			Header:    ir.Header{ID: "setting.copilot." + key, IRVersion: 1, Origin: &ir.Origin{Tool: "copilot", Path: originPath, Scope: scope}},
			Key:       key,
			Value:     val,
			ToolScope: []string{"copilot"},
		})
	}
	return out
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
