package claudedesktop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

// configFilePath 返回 claude_desktop_config.json 跨平台路径。
func configFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Claude", "claude_desktop_config.json")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	default:
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "Claude", "claude_desktop_config.json")
		}
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	}
}

// Detect 探测 claude_desktop_config.json（仅全局）。
func (a *adapter) Detect(ctx context.Context) ([]adapters.Location, error) {
	var locs []adapters.Location
	p := configFilePath()
	if p != "" {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			locs = append(locs, adapters.Location{Scope: ir.ScopeGlobal, Root: filepath.Dir(p)})
		}
	}
	return locs, nil
}

// mcpServerConf Claude Desktop 的 mcpServers 条目（stdio 为主）。
type mcpServerConf struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

func (a *adapter) importLocation(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	b := &ir.Bundle{Scope: loc.Scope, IRVersion: 1}
	data, err := os.ReadFile(filepath.Join(loc.Root, "claude_desktop_config.json"))
	if err != nil {
		return b, nil
	}
	var f struct {
		MCPServers map[string]mcpServerConf `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return b, nil
	}
	for name, conf := range f.MCPServers {
		transport := conf.Type
		if transport == "" {
			transport = "stdio"
		}
		s := ir.MCPServer{
			Header: ir.Header{
				ID:        "mcp." + sanitize(name),
				IRVersion: 1,
				Origin:    &ir.Origin{Tool: "claude-desktop", Path: "claude_desktop_config.json", Scope: loc.Scope},
			},
			Name:      name,
			Transport: transport,
			Command:   conf.Command,
			Args:      conf.Args,
			Env:       conf.Env,
			URL:       conf.URL,
			Headers:   conf.Headers,
		}
		b.MCPServers = append(b.MCPServers, s)
		b.Add(ir.KindMCP, s.ID)
	}
	return b, nil
}

// exportBundle 物化：局部 patch claude_desktop_config.json（只改 mcpServers，保留其他键）。
func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	if len(b.MCPServers) == 0 {
		return nil, nil
	}
	path := configFilePath()
	root := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &root) // 保留既有键
	}
	servers := map[string]any{}
	for _, s := range b.MCPServers {
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
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	return []adapters.WrittenFile{{Path: path, Hash: hex.EncodeToString(sum[:]), Content: data}}, nil
}

func sanitize(s string) string {
	var b []rune
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			b = append(b, r)
		} else {
			b = append(b, '-')
		}
	}
	return string(b)
}
