package profile

import (
	"testing"

	"github.com/timywel/ai4config/internal/core/ir"
)

// 对抗用例回归（research/adversarial-cases.md）。

// AC-A4 mcp-env-single-key：浅字段级合并——object 整体覆盖、数组整体替换、未写键继承。
// 固化合并语义；同时记录"覆盖导致继承键丢失"应提示的改进点（见 TASKS 变更候选）。
func TestAdversarial_A4_MergeEnvSingleKey(t *testing.T) {
	global := layer(ir.ScopeGlobal, &ir.Bundle{MCPServers: []ir.MCPServer{
		func() ir.MCPServer {
			s := ir.MCPServer{
				Header:    ir.Header{ID: "mcp.filesystem", IRVersion: 1},
				Name:      "filesystem",
				Transport: "stdio",
				Command:   "npx",
				Args:      []string{"-y", "srv", "/data"},
				Env:       map[string]string{"ROOT": "/data", "DEBUG": "1"},
			}
			return s
		}(),
	}})
	project := layer(ir.ScopeProject, &ir.Bundle{MCPServers: []ir.MCPServer{
		{Header: ir.Header{ID: "mcp.filesystem", IRVersion: 1}, Env: map[string]string{"DEBUG": "0"}},
	}})

	merged := MergeBundles(global, project)
	got := findMCPForTest(merged.MCPServers, "mcp.filesystem")
	if got == nil {
		t.Fatal("merged 缺 mcp.filesystem")
	}
	// object 整体覆盖：env 只有 DEBUG，ROOT 被蒸发（浅合并语义正确）
	if len(got.Env) != 1 || got.Env["DEBUG"] != "0" {
		t.Errorf("env 应整体覆盖为 {DEBUG:0}: %v", got.Env)
	}
	// 数组整体继承（项目未写 args）
	if len(got.Args) != 3 {
		t.Errorf("args 应继承全局 3 元素: %v", got.Args)
	}
	// command 继承
	if got.Command != "npx" {
		t.Errorf("command 应继承: %q", got.Command)
	}
}

func findMCPForTest(list []ir.MCPServer, id string) *ir.MCPServer {
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}
