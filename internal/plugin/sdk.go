package plugin

import (
	"github.com/hashicorp/go-plugin"
)

// Serve 插件进程入口：第三方适配器实现 AdapterRPC 后调用 Serve 启动服务。
// 用法（插件的 main.go）：
//
//	func main() { plugin.Serve(&myAdapter{}) }
func Serve(impl AdapterRPC) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins:         map[string]plugin.Plugin{"adapter": &AdapterPlugin{Impl: impl}},
	})
}
