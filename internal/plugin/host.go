package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/hashicorp/go-plugin"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

// hostAdapter 把外置插件进程的 RPC 调用包装为本地 adapters.Adapter。
type hostAdapter struct {
	client *plugin.Client
	rpc    AdapterRPC
	meta   adapters.ToolMeta
}

// LoadPlugin 启动一个插件进程并包装为 adapters.Adapter。
// 返回 (适配器, 清理函数 kill, 错误)。
func LoadPlugin(path string) (adapters.Adapter, func(), error) {
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  Handshake,
		Plugins:          map[string]plugin.Plugin{"adapter": &AdapterPlugin{}},
		Cmd:              exec.Command(path),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
	})
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, nil, fmt.Errorf("plugin: 连接插件进程失败: %w", err)
	}
	raw, err := rpcClient.Dispense("adapter")
	if err != nil {
		client.Kill()
		return nil, nil, fmt.Errorf("plugin: 获取适配器失败: %w", err)
	}
	rpc, ok := raw.(AdapterRPC)
	if !ok {
		client.Kill()
		return nil, nil, fmt.Errorf("plugin: 插件未实现 AdapterRPC")
	}
	// 取 meta
	var metaJSON []byte
	if err := rpc.Meta(struct{}{}, &metaJSON); err != nil {
		client.Kill()
		return nil, nil, fmt.Errorf("plugin: Meta 调用失败: %w", err)
	}
	var meta adapters.ToolMeta
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		client.Kill()
		return nil, nil, err
	}
	ha := &hostAdapter{client: client, rpc: rpc, meta: meta}
	return ha, func() { client.Kill() }, nil
}

func (h *hostAdapter) Meta() adapters.ToolMeta { return h.meta }

func (h *hostAdapter) Detect(ctx context.Context) ([]adapters.Location, error) {
	var out []byte
	if err := h.rpc.Detect(struct{}{}, &out); err != nil {
		return nil, err
	}
	var locs []adapters.Location
	if err := json.Unmarshal(out, &locs); err != nil {
		return nil, err
	}
	return locs, nil
}

func (h *hostAdapter) Import(ctx context.Context, loc adapters.Location) (*ir.Bundle, error) {
	locJSON, err := json.Marshal(loc)
	if err != nil {
		return nil, err
	}
	var out []byte
	if err := h.rpc.Import(locJSON, &out); err != nil {
		return nil, err
	}
	var b ir.Bundle
	if err := json.Unmarshal(out, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func (h *hostAdapter) Export(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	argsJSON, err := json.Marshal(ExportArgs{Bundle: b, ProjectRoot: opts.ProjectRoot, DryRun: opts.DryRun, Force: opts.Force})
	if err != nil {
		return nil, err
	}
	var out []byte
	if err := h.rpc.Export(argsJSON, &out); err != nil {
		return nil, err
	}
	var files []adapters.WrittenFile
	if err := json.Unmarshal(out, &files); err != nil {
		return nil, err
	}
	return files, nil
}
