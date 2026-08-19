package plugin

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/timywel/ai4config/internal/adapters"
)

// T26.2 验收：编译示例插件进程，host 端 LoadPlugin 启动并调用 Meta/Detect/Import。
func TestPluginHostRoundTrip(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "demo-plugin")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, "./testdata/demo")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("编译示例插件失败: %v\n%s", err, out)
	}

	a, kill, err := LoadPlugin(bin)
	if err != nil {
		t.Fatalf("LoadPlugin: %v", err)
	}
	defer kill()

	// Meta
	meta := a.Meta()
	if meta.ID != "demo" || meta.DisplayName != "Demo Tool" {
		t.Errorf("Meta 错误: %+v", meta)
	}
	// Detect
	locs, err := a.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(locs) != 1 || locs[0].Root != "/tmp/demo" {
		t.Errorf("Detect 错误: %+v", locs)
	}
	// Import
	b, err := a.Import(context.Background(), locs[0])
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(b.MCPServers) != 1 || b.MCPServers[0].Name != "demo" {
		t.Errorf("Import 错误: %+v", b.MCPServers)
	}
	// Export
	files, err := a.Export(context.Background(), b, adapters.ExportOpts{})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Export 错误: %+v", files)
	}
}

// adapters_ExportOpts 占位避免 import（用插件包内类型）。
type adapters_ExportOpts = struct {
	ProjectRoot string
	DryRun      bool
	Force       bool
}
