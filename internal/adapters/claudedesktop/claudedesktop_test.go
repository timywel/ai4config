package claudedesktop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

func TestImportExportMCP(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "claude_desktop_config.json"), []byte(`{
  "mcpServers": {"fs": {"command": "npx", "args": ["-y","fs"], "env": {"TOKEN": "x"}}},
  "globalShortcut": "Ctrl+Space"
}`), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeGlobal, Root: dir})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(b.MCPServers) != 1 || b.MCPServers[0].Name != "fs" {
		t.Fatalf("mcpServers 解析错误: %+v", b.MCPServers)
	}
	if b.MCPServers[0].Env["TOKEN"] != "x" {
		t.Errorf("env 未解析: %+v", b.MCPServers[0].Env)
	}

	// Export：局部 patch 保留 globalShortcut
	// 模拟 configFilePath 指向临时文件
	os.Setenv("APPDATA", dir)
	defer os.Unsetenv("APPDATA")
	target := filepath.Join(dir, "Claude", "claude_desktop_config.json")
	os.MkdirAll(filepath.Dir(target), 0o755)
	os.WriteFile(target, []byte(`{"globalShortcut":"Ctrl+Space","mcpServers":{}}`), 0o644)

	files, err := a.Export(context.Background(), b, adapters.ExportOpts{})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("应产生 1 个计划文件，实际 %d", len(files))
	}
	content := string(files[0].Content)
	if !strings.Contains(content, "globalShortcut") {
		t.Error("局部 patch 应保留既有键 globalShortcut")
	}
	if !strings.Contains(content, "mcpServers") || !strings.Contains(content, "fs") {
		t.Errorf("应写入 mcpServers.fs:\n%s", content)
	}
}
