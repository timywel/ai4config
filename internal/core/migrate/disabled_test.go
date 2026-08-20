package migrate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/timywel/ai4config/internal/adapters/codex"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/store"
)

// OPT-B3：annotations.yaml 禁用清单在导出时被过滤（IR-SCHEMA §4.5）。
func TestFilterDisabledOnExport(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	repo, _ := store.Open(root)
	when := time.Now()

	m := &profile.Manifest{IRVersion: 1, Profile: profile.Meta{Name: "global", Kind: "global", CreatedAt: when}}
	b := &ir.Bundle{
		IRVersion: 1, Scope: ir.ScopeGlobal,
		Instructions: []ir.Instruction{
			{Header: ir.Header{ID: "instruction.active", IRVersion: 1}, Body: "启用中\n"},
			{Header: ir.Header{ID: "instruction.legacy", IRVersion: 1}, Body: "已禁用\n"},
		},
	}
	profile.Save(repo.Path(store.DirProfiles, "global"), b, m)
	// 禁用 instruction.legacy
	ann := &profile.Annotations{Disabled: []string{"instruction.legacy"}}
	if err := profile.SaveAnnotations(repo.Path(store.DirProfiles, "global"), ann); err != nil {
		t.Fatalf("SaveAnnotations: %v", err)
	}

	codexHome := t.TempDir()
	os.Setenv("CODEX_HOME", codexHome)
	defer os.Unsetenv("CODEX_HOME")
	e := &Engine{Repo: repo}
	if _, err := e.Export(context.Background(), ExportRequest{To: "codex"}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(codexHome, "AGENTS.md"))
	content := string(data)
	if !containsStr2(content, "启用中") {
		t.Error("启用条目应导出")
	}
	if containsStr2(content, "已禁用") {
		t.Error("禁用条目不应导出（annotations 过滤）")
	}
}

func containsStr2(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
