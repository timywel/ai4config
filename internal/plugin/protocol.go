// Package plugin 实现外置进程适配器插件（go-plugin，net/rpc over stdio）。
// 权威规范：docs/ARCHITECTURE.md §12 P3（外置进程插件，第三方任意语言）。
package plugin

import (
	"encoding/json"
	"net/rpc"

	"github.com/hashicorp/go-plugin"

	"github.com/timywel/ai4config/internal/core/ir"
)

// Handshake 插件握手配置（防误加载非本插件进程）。
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "CFG4AI_PLUGIN",
	MagicCookieValue: "cfg4ai-adapter",
}

// 数据经 JSON 序列化传输（规避 gob 对 any/接口类型的限制）。

// ExportArgs Export RPC 入参。
type ExportArgs struct {
	Bundle      *ir.Bundle `json:"bundle"`
	ProjectRoot string     `json:"project_root"`
	DryRun      bool       `json:"dry_run"`
	Force       bool       `json:"force"`
}

// AdapterRPC 适配器插件的 RPC 接口（net/rpc 可调用；全部用 JSON []byte 传输）。
type AdapterRPC interface {
	// Meta 返回 ToolMeta JSON。
	Meta(_ struct{}, resp *[]byte) error
	// Detect 返回 []adapters.Location JSON。
	Detect(_ struct{}, resp *[]byte) error
	// Import 入参 Location JSON，返回 Bundle JSON。
	Import(locJSON []byte, resp *[]byte) error
	// Export 入参 ExportArgs JSON，返回 []WrittenFile JSON。
	Export(argsJSON []byte, resp *[]byte) error
}

// ---------- 序列化助手 ----------

// Encode JSON 序列化。
func Encode(v any) ([]byte, error) { return json.Marshal(v) }

// Decode JSON 反序列化。
func Decode(data []byte, v any) error { return json.Unmarshal(data, v) }

// ---------- net/rpc 插件两端 ----------

// AdapterPlugin 实现 go-plugin 的 Plugin 接口（net/rpc 模式）。
type AdapterPlugin struct {
	// Impl 由插件侧提供实现；host 侧为 nil（走 RPC client）。
	Impl AdapterRPC
}

func (p *AdapterPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &rpcServer{impl: p.Impl}, nil
}

func (p *AdapterPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &rpcClient{client: c}, nil
}

// rpcServer 插件进程内：把 RPC 调用转发给 Impl。
type rpcServer struct{ impl AdapterRPC }

func (s *rpcServer) Meta(_ struct{}, resp *[]byte) error {
	meta := s.implMeta()
	return EncodeInto(meta, resp)
}

func (s *rpcServer) Detect(_ struct{}, resp *[]byte) error {
	var out []byte
	err := s.impl.Detect(struct{}{}, &out)
	*resp = out
	return err
}

func (s *rpcServer) Import(locJSON []byte, resp *[]byte) error {
	var out []byte
	err := s.impl.Import(locJSON, &out)
	*resp = out
	return err
}

func (s *rpcServer) Export(argsJSON []byte, resp *[]byte) error {
	var out []byte
	err := s.impl.Export(argsJSON, &out)
	*resp = out
	return err
}

// implMeta 取 Impl.Meta 结果（host 侧 nil 防护由调用方处理）。
func (s *rpcServer) implMeta() any {
	if s.impl == nil {
		return nil
	}
	var out []byte
	_ = s.impl.Meta(struct{}{}, &out)
	var v any
	_ = json.Unmarshal(out, &v)
	return v
}

// EncodeInto 编码到 resp。
func EncodeInto(v any, resp *[]byte) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	*resp = data
	return nil
}

// rpcClient host 进程内：把调用转发为 RPC。
type rpcClient struct{ client *rpc.Client }

func (c *rpcClient) Meta(_ struct{}, resp *[]byte) error {
	return c.client.Call("Plugin.Meta", struct{}{}, resp)
}
func (c *rpcClient) Detect(_ struct{}, resp *[]byte) error {
	return c.client.Call("Plugin.Detect", struct{}{}, resp)
}
func (c *rpcClient) Import(locJSON []byte, resp *[]byte) error {
	return c.client.Call("Plugin.Import", locJSON, resp)
}
func (c *rpcClient) Export(argsJSON []byte, resp *[]byte) error {
	return c.client.Call("Plugin.Export", argsJSON, resp)
}
