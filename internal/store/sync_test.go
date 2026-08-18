package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	git "github.com/go-git/go-git/v5"
	"github.com/timywel/ai4config/internal/core/secrets"
)

func TestSyncInitGitignoreBaseline(t *testing.T) {
	r, _ := Open(filepath.Join(t.TempDir(), "repo"))
	if err := r.SyncInit(""); err != nil {
		t.Fatalf("SyncInit: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(r.Root, ".gitignore"))
	content := string(data)
	// 白名单放行
	if !strings.Contains(content, "!profiles") {
		t.Error(".gitignore 应放行 profiles")
	}
	// 默认全忽略
	if !strings.Contains(content, "*\n") {
		t.Error(".gitignore 应默认全忽略（白名单制）")
	}
}

func TestSyncPushPull(t *testing.T) {
	// bare 远端（初始化为 bare git 仓库）
	bare := t.TempDir()
	git.PlainInit(bare, true)
	r1, _ := Open(filepath.Join(t.TempDir(), "r1"))
	if err := r1.SyncInit(bare); err != nil {
		t.Fatalf("SyncInit: %v", err)
	}
	// 写内容到白名单
	os.MkdirAll(r1.Path(DirProfiles, "global"), 0o700)
	os.WriteFile(r1.Path(DirProfiles, "global", "manifest.yaml"), []byte("ir_version: 1\n"), 0o600)

	if err := r1.SyncPush(nil, nil); err != nil {
		t.Fatalf("SyncPush: %v", err)
	}
}

// 红队 T-05：含 secret 的文件在 preflight 被阻断
func TestSyncPreflightBlocksSecret(t *testing.T) {
	r, _ := Open(filepath.Join(t.TempDir(), "repo"))
	r.SyncInit("")
	// 白名单范围写入含 secret 的自由文本
	os.MkdirAll(r.Path(DirProfiles, "global", "instructions"), 0o700)
	secretBody := "使用 sk-" + strings.Repeat("a", 30) + " 调用"
	os.WriteFile(r.Path(DirProfiles, "global", "instructions", "x.md"), []byte(secretBody), 0o600)

	err := r.SyncPush(secrets.DefaultScanner(), nil) // confirm=nil → 命中即阻断
	if err == nil {
		t.Fatal("含 secret 的 push 应被 preflight 阻断")
	}
	if !strings.Contains(err.Error(), "阻断") {
		t.Errorf("错误应提示阻断: %v", err)
	}
}

// 干净内容 + confirm 确认后放行
func TestSyncPreflightConfirm(t *testing.T) {
	r, _ := Open(filepath.Join(t.TempDir(), "repo"))
	r.SyncInit("")
	os.MkdirAll(r.Path(DirProfiles, "global", "instructions"), 0o700)
	secretBody := "使用 sk-" + strings.Repeat("a", 30) + " 调用"
	os.WriteFile(r.Path(DirProfiles, "global", "instructions", "x.md"), []byte(secretBody), 0o600)

	confirmed := false
	err := r.SyncPush(secrets.DefaultScanner(), func(m []secrets.ScanMatch) bool {
		confirmed = true
		return true // 用户显式确认
	})
	if !confirmed {
		t.Error("命中时应调用 confirm 回调")
	}
	if err != nil && strings.Contains(err.Error(), "阻断") {
		t.Errorf("用户确认后不应阻断: %v", err)
	}
}
