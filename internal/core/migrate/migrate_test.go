package migrate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/timywel/ai4config/internal/adapters/codex" // 注册 codex
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/store"
)

// 构造一个含 global profile 的测试仓库。
func setupRepo(t *testing.T) *store.Repo {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repo")
	repo, err := store.Open(root)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	m := &profile.Manifest{
		IRVersion: 1,
		Profile:   profile.Meta{Name: "global", Kind: "global", CreatedAt: time.Now()},
	}
	b := &ir.Bundle{
		IRVersion: 1,
		Scope:     ir.ScopeGlobal,
		Instructions: []ir.Instruction{
			{Header: ir.Header{ID: "instruction.style", IRVersion: 1}, Body: "用中文回复\n"},
		},
		MCPServers: []ir.MCPServer{
			{Header: ir.Header{ID: "mcp.fs", IRVersion: 1}, Name: "fs", Transport: "stdio", Command: "npx"},
		},
		Settings: []ir.SettingEntry{
			{Header: ir.Header{ID: "setting.codex.model", IRVersion: 1}, Key: "model", Value: "gpt-5"},
		},
	}
	if err := profile.Save(repo.Path(store.DirProfiles, "global"), b, m); err != nil {
		t.Fatalf("profile.Save: %v", err)
	}
	return repo
}

func TestExportToCodex(t *testing.T) {
	repo := setupRepo(t)
	e := &Engine{Repo: repo}
	target := t.TempDir()

	os.Setenv("CODEX_HOME", target) // 重定向 codex 全局目录到临时位置
	defer os.Unsetenv("CODEX_HOME")

	res, err := e.Export(context.Background(), ExportRequest{To: "codex"})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(res.Written) == 0 {
		t.Fatal("应有写入文件")
	}
	// config.toml 应生成且含 mcp_servers
	data, err := os.ReadFile(filepath.Join(target, "config.toml"))
	if err != nil {
		t.Fatalf("config.toml 未生成: %v", err)
	}
	if !contains(string(data), "mcp_servers") || !contains(string(data), "fs") {
		t.Errorf("config.toml 应含 mcp_servers.fs:\n%s", data)
	}
	// AGENTS.md 应生成
	if _, err := os.Stat(filepath.Join(target, "AGENTS.md")); err != nil {
		t.Errorf("AGENTS.md 未生成: %v", err)
	}
	// 快照应已创建
	if res.SnapshotID == "" {
		t.Error("导出前应创建快照")
	}
	// exports 清单应更新
	em, _ := repo.LoadExportManifest("codex", "global")
	if len(em.Files) == 0 {
		t.Error("导出清单应有记录")
	}
}

func TestExportEmptyProtection(t *testing.T) {
	// 空仓库（无 profile）
	root := filepath.Join(t.TempDir(), "repo")
	repo, _ := store.Open(root)
	e := &Engine{Repo: repo}

	_, err := e.Export(context.Background(), ExportRequest{To: "codex"})
	if err == nil {
		t.Fatal("无 profile 应报错（提示先 collect）")
	}
}

func TestExportDegradeWorkflow(t *testing.T) {
	repo := setupRepo(t)
	// 加一个 workflow（codex 不支持 workflow → 降级）
	repoGlobalDir := repo.Path(store.DirProfiles, "global")
	sb, _ := profile.Load(repoGlobalDir, ir.ScopeGlobal)
	sb.Bundle.Workflows = []ir.PromptPack{
		{Header: ir.Header{ID: "workflow.deploy", IRVersion: 1}, Kind: ir.KindWorkflow, Name: "deploy", Body: "部署步骤\n"},
	}
	profile.Save(repoGlobalDir, sb.Bundle, sb.Manifest)

	e := &Engine{Repo: repo}
	target := t.TempDir()
	os.Setenv("CODEX_HOME", target)
	defer os.Unsetenv("CODEX_HOME")

	res, err := e.Export(context.Background(), ExportRequest{To: "codex"})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	// workflow 应降级（codex workflow/command 均 SupportNone → instruction 附录）
	found := false
	for _, w := range res.Warnings {
		if w.Kind == "degrade" {
			found = true
		}
	}
	if !found {
		t.Error("workflow 应产生降级 Warning")
	}
}

func TestExportForeignProtection(t *testing.T) {
	repo := setupRepo(t)
	e := &Engine{Repo: repo}
	target := t.TempDir()
	os.Setenv("CODEX_HOME", target)
	defer os.Unsetenv("CODEX_HOME")

	// 第一次导出（建立清单）
	if _, err := e.Export(context.Background(), ExportRequest{To: "codex"}); err != nil {
		t.Fatalf("首次 Export: %v", err)
	}

	// 模拟外部修改目标文件
	configPath := filepath.Join(target, "config.toml")
	os.WriteFile(configPath, []byte("# 被外部修改\n"), 0o644)

	// 第二次导出（无 force、无确认回调）→ 外部修改的文件应被跳过（安全默认）
	called := false
	e.Hooks.ConfirmForeign = func(path string, status store.ForeignStatus) (string, error) {
		called = true
		if status == store.StatusModified {
			return "skip", nil
		}
		return "skip", nil
	}
	if _, err := e.Export(context.Background(), ExportRequest{To: "codex"}); err != nil {
		t.Fatalf("二次 Export: %v", err)
	}
	// 外部修改的内容应保留（skip）
	data, _ := os.ReadFile(configPath)
	if !contains(string(data), "被外部修改") {
		t.Error("外部修改的文件应被跳过（不被覆盖）")
	}
	_ = called
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
