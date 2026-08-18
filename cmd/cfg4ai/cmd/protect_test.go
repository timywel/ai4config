package cmd

import (
	"testing"

	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/secrets"
)

// T19.1 回采保护：占位符不覆盖已有 secretref（红队 T-03）。
func TestProtectSecrets(t *testing.T) {
	existing := &ir.Bundle{MCPServers: []ir.MCPServer{{
		Header: ir.Header{ID: "mcp.fs", IRVersion: 1}, Name: "fs",
		Env: map[string]string{"TOKEN": secrets.MakeRef("global", "mcp.fs", "env.TOKEN")},
	}}}
	result := &ir.Bundle{MCPServers: []ir.MCPServer{{
		Header: ir.Header{ID: "mcp.fs", IRVersion: 1}, Name: "fs",
		Env: map[string]string{"TOKEN": ""}, // 回采到空值（none 后端导出占位）
	}}}

	protectSecrets(result, existing)
	if result.MCPServers[0].Env["TOKEN"] != secrets.MakeRef("global", "mcp.fs", "env.TOKEN") {
		t.Errorf("空值回采应恢复 secretref，实际 %q", result.MCPServers[0].Env["TOKEN"])
	}

	// 真实明文应正常采用（不恢复）
	result2 := &ir.Bundle{MCPServers: []ir.MCPServer{{
		Header: ir.Header{ID: "mcp.fs", IRVersion: 1}, Name: "fs",
		Env: map[string]string{"TOKEN": "real-token-value"},
	}}}
	protectSecrets(result2, existing)
	if result2.MCPServers[0].Env["TOKEN"] != "real-token-value" {
		t.Errorf("真实明文应采用: %q", result2.MCPServers[0].Env["TOKEN"])
	}
}
