package migrate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/timywel/ai4config/internal/core/aiassist"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/store"
)

// mock provider（不真调 API）。
type mockProvider struct{ reply string }

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Chat(ctx context.Context, msgs []aiassist.Message) (string, error) {
	return m.reply, nil
}

func setupAssistEngine(t *testing.T) (*Engine, *store.Repo) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repo")
	repo, _ := store.Open(root)
	m := &profile.Manifest{IRVersion: 1, Profile: profile.Meta{Name: "global", Kind: "global", CreatedAt: time.Now()}}
	b := &ir.Bundle{
		IRVersion: 1, Scope: ir.ScopeGlobal,
		Instructions: []ir.Instruction{{Header: ir.Header{ID: "instruction.x", IRVersion: 1}, Body: "规则\n"}},
	}
	profile.Save(repo.Path(store.DirProfiles, "global"), b, m)
	e := &Engine{Repo: repo}
	return e, repo
}

// T21.2/红队 T-09：--ai 无 consent 且无 --ai-approve → 拒绝
func TestAssistRequiresConsent(t *testing.T) {
	e, repo := setupAssistEngine(t)
	client, _ := aiassist.NewClient(&mockProvider{reply: "x"}, repo.Root)
	e.AI = client
	e.AIConfig = aiassist.AIConfig{Provider: "mock", BaseURL: "http://x", Model: "m"}

	_, err := e.Export(context.Background(), ExportRequest{To: "codex", AI: true})
	if err == nil || !strings.Contains(err.Error(), "需确认") {
		t.Errorf("--ai 无 consent 应拒绝（需确认）: %v", err)
	}
}

// --ai-approve 放行并记录 consent
func TestAssistApproveRecordsConsent(t *testing.T) {
	e, repo := setupAssistEngine(t)
	client, _ := aiassist.NewClient(&mockProvider{reply: "x"}, repo.Root)
	e.AI = client
	e.AIConfig = aiassist.AIConfig{Provider: "mock", BaseURL: "http://x", Model: "m"}
	os.Setenv("CODEX_HOME", t.TempDir())
	defer os.Unsetenv("CODEX_HOME")

	_, err := e.Export(context.Background(), ExportRequest{To: "codex", AI: true, AIApprove: true})
	if err != nil {
		t.Fatalf("--ai-approve 应放行: %v", err)
	}
	// consent 已记录
	if s := aiassist.CheckConsent(repo.Root, e.AIConfig); s != aiassist.ConsentOK {
		t.Errorf("--ai-approve 后 consent 应为 OK，实际 %v", s)
	}
}
