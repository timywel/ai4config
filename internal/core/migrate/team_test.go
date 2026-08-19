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

// T28 团队 profile 共享：remote 层（profiles/team/*）参与合并导出。
func TestTeamLayerMerges(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	repo, _ := store.Open(root)
	when := time.Now()

	// global
	gm := &profile.Manifest{IRVersion: 1, Profile: profile.Meta{Name: "global", Kind: "global", CreatedAt: when}}
	profile.Save(repo.Path(store.DirProfiles, "global"), &ir.Bundle{
		IRVersion: 1, Scope: ir.ScopeGlobal,
		Instructions: []ir.Instruction{{Header: ir.Header{ID: "instruction.g", IRVersion: 1}, Priority: 100, Body: "全局规范\n"}},
	}, gm)
	// team（remote 层）
	tm := &profile.Manifest{IRVersion: 1, Profile: profile.Meta{Name: "team", Kind: "remote", CreatedAt: when}}
	profile.Save(repo.Path(store.DirProfiles, "team", "myteam"), &ir.Bundle{
		IRVersion: 1, Scope: ir.ScopeRemote,
		Instructions: []ir.Instruction{{Header: ir.Header{ID: "instruction.t", IRVersion: 1}, Priority: 150, Body: "团队规范\n"}},
	}, tm)

	codexHome := t.TempDir()
	os.Setenv("CODEX_HOME", codexHome)
	defer os.Unsetenv("CODEX_HOME")
	e := &Engine{Repo: repo}
	if _, err := e.Export(context.Background(), ExportRequest{To: "codex"}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(codexHome, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md 未生成: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "全局规范") || !strings.Contains(content, "团队规范") {
		t.Errorf("AGENTS.md 应含全局+团队两层内容:\n%s", content)
	}
	// remote 在 global 之后（低优先级在前：global → remote）
	if strings.Index(content, "全局规范") > strings.Index(content, "团队规范") {
		t.Error("concat 顺序应为 global 在前、remote（team）在后")
	}
}
