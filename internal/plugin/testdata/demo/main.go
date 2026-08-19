// demo 是一个示例适配器插件进程（验证 host↔plugin 进程互通）。
package main

import (
	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/plugin"
)

type demoAdapter struct{}

func (d *demoAdapter) Meta(_ struct{}, resp *[]byte) error {
	return plugin.EncodeInto(adapters.ToolMeta{ID: "demo", DisplayName: "Demo Tool"}, resp)
}

func (d *demoAdapter) Detect(_ struct{}, resp *[]byte) error {
	return plugin.EncodeInto([]adapters.Location{{Scope: ir.ScopeGlobal, Root: "/tmp/demo"}}, resp)
}

func (d *demoAdapter) Import(locJSON []byte, resp *[]byte) error {
	b := &ir.Bundle{Scope: ir.ScopeGlobal, IRVersion: 1}
	b.MCPServers = append(b.MCPServers, ir.MCPServer{Header: ir.Header{ID: "mcp.demo", IRVersion: 1}, Name: "demo", Transport: "stdio", Command: "demo"})
	return plugin.EncodeInto(b, resp)
}

func (d *demoAdapter) Export(argsJSON []byte, resp *[]byte) error {
	return plugin.EncodeInto([]adapters.WrittenFile{{Path: "/tmp/demo/out.json"}}, resp)
}

func main() { plugin.Serve(&demoAdapter{}) }
