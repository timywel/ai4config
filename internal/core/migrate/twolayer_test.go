package migrate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/timywel/ai4config/internal/adapters/codex"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/store"
)

// P0 验收项 3：双层继承 e2e——global + project profile 合并后导出，产物含两层内容。
func TestTwoLayerInheritance(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	repo, err := store.Open(root)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	when := time.Now()

	// 全局 profile：一条全局指令 + 一个共享 MCP
	gm := &profile.Manifest{IRVersion: 1, Profile: profile.Meta{Name: "global", Kind: "global", CreatedAt: when}}
	gb := &ir.Bundle{
		IRVersion: 1, Scope: ir.ScopeGlobal,
		Instructions: []ir.Instruction{
			{Header: ir.Header{ID: "instruction.global-rule", IRVersion: 1}, Priority: 100, Body: "全局规范\n"},
		},
		MCPServers: []ir.MCPServer{
			{Header: ir.Header{ID: "mcp.shared", IRVersion: 1}, Name: "shared", Transport: "stdio", Command: "npx"},
		},
	}
	if err := profile.Save(repo.Path(store.DirProfiles, "global"), gb, gm); err != nil {
		t.Fatalf("save global: %v", err)
	}

	// 项目 profile：一条项目指令（写到项目 slug 目录）
	projPath := filepath.Join(t.TempDir(), "myproj")
	slug := slugifyPath(projPath)
	pm := &profile.Manifest{IRVersion: 1, Profile: profile.Meta{Name: "myproj", Kind: "project", CreatedAt: when}}
	pb := &ir.Bundle{
		IRVersion: 1, Scope: ir.ScopeProject,
		Instructions: []ir.Instruction{
			{Header: ir.Header{ID: "instruction.proj-rule", IRVersion: 1}, Priority: 200, Body: "项目规范\n"},
		},
	}
	if err := profile.Save(repo.Path(store.DirProfiles, "projects", slug), pb, pm); err != nil {
		t.Fatalf("save project: %v", err)
	}

	// 导出（合并双层）到 codex
	codexHome := t.TempDir() // 项目导出时产物在 projPath 下，CODEX_HOME 仅占位
	os.Setenv("CODEX_HOME", codexHome)
	defer os.Unsetenv("CODEX_HOME")
	e := &Engine{Repo: repo}
	res, err := e.Export(context.Background(), ExportRequest{To: "codex", ProjectPath: projPath})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	// 验证 AGENTS.md 含两层指令（concat：全局在前，项目在后）
	data, err := os.ReadFile(filepath.Join(projPath, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md 未生成: %v", err)
	}
	content := string(data)
	gi := strings.Index(content, "全局规范")
	pi := strings.Index(content, "项目规范")
	if gi < 0 || pi < 0 {
		t.Fatalf("AGENTS.md 应含双层指令（全局+项目）:\n%s", content)
	}
	if gi > pi {
		t.Errorf("concat 顺序应为全局在前、项目在后: global@%d project@%d", gi, pi)
	}
	// 共享 MCP 也应继承导出
	if len(res.Written) == 0 {
		t.Error("应有写入文件")
	}
	_ = res
}
