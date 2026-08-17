package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func openRepo(t *testing.T) *Repo {
	t.Helper()
	r, err := Open(filepath.Join(t.TempDir(), "repo"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return r
}

func TestOpenInitializesLayout(t *testing.T) {
	r := openRepo(t)
	for _, d := range []string{DirProfiles, DirExports, DirSnapshots, DirBlobs, DirCache, DirLogs} {
		if info, err := os.Stat(r.Path(d)); err != nil || !info.IsDir() {
			t.Errorf("应创建目录 %s: %v", d, err)
		}
	}
}

func TestBlobPutGetDedup(t *testing.T) {
	r := openRepo(t)
	data := []byte("hello blob")
	h1, err := r.PutBlob(data)
	if err != nil {
		t.Fatalf("PutBlob: %v", err)
	}
	h2, _ := r.PutBlob(data) // 同内容
	if h1 != h2 {
		t.Errorf("同内容应同 hash（去重）: %s vs %s", h1, h2)
	}
	if BlobHash(data) != h1 {
		t.Errorf("BlobHash 不一致")
	}
	got, err := r.GetBlob(h1)
	if err != nil || string(got) != "hello blob" {
		t.Fatalf("GetBlob: %v got=%q", err, got)
	}
	if !r.HasBlob(h1) {
		t.Error("HasBlob 应为 true")
	}
}

func TestSnapshotCreateRestore(t *testing.T) {
	r := openRepo(t)
	// 造一个 profile 文件
	pf := r.Path(DirProfiles, "global", "manifest.yaml")
	os.MkdirAll(filepath.Dir(pf), 0o700)
	os.WriteFile(pf, []byte("ir_version: 1\n"), 0o600)

	id, err := r.CreateSnapshot("test")
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}
	if id == "" {
		t.Fatal("快照 id 为空")
	}

	// 修改文件后恢复
	os.WriteFile(pf, []byte("ir_version: 2\n"), 0o600)
	if err := r.RestoreSnapshot(id); err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}
	got, _ := os.ReadFile(pf)
	if string(got) != "ir_version: 1\n" {
		t.Errorf("恢复内容错误: %q", got)
	}
}

func TestSnapshotPrune(t *testing.T) {
	r := openRepo(t)
	os.MkdirAll(r.Path(DirProfiles), 0o700)
	os.WriteFile(r.Path(DirProfiles, "a.txt"), []byte("x"), 0o600)

	var ids []string
	for i := 0; i < 3; i++ {
		id, err := r.CreateSnapshot("")
		if err != nil {
			t.Fatalf("CreateSnapshot %d: %v", i, err)
		}
		ids = append(ids, id)
		time.Sleep(time.Millisecond * 1100) // 保证时间戳不同（秒级）
	}
	list, _ := r.ListSnapshots()
	if len(list) != 3 {
		t.Fatalf("应有 3 个快照，实际 %d", len(list))
	}
	removed, err := r.PruneSnapshots(1)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if removed != 2 {
		t.Errorf("应回收 2 个，实际 %d", removed)
	}
	list2, _ := r.ListSnapshots()
	if len(list2) != 1 {
		t.Errorf("回收后应剩 1 个，实际 %d", len(list2))
	}
}

func TestExportsClassify(t *testing.T) {
	m := &ExportManifest{Tool: "codex", Scope: "global"}
	content := []byte("line1\nline2\n")
	m.Record("/x/config.toml", content)

	// 一致 → ours
	if got := m.Classify("/x/config.toml", content); got != StatusOurs {
		t.Errorf("一致应为 ours，实际 %s", got)
	}
	// CRLF/BOM 差异 → 仍 ours（规范化）
	crlf := []byte("line1\r\nline2\r\n")
	if got := m.Classify("/x/config.toml", crlf); got != StatusOurs {
		t.Errorf("CRLF 差异应规范化为 ours，实际 %s", got)
	}
	bom := append([]byte{0xEF, 0xBB, 0xBF}, content...)
	if got := m.Classify("/x/config.toml", bom); got != StatusOurs {
		t.Errorf("BOM 差异应规范化为 ours，实际 %s", got)
	}
	// 内容变化 → modified
	if got := m.Classify("/x/config.toml", []byte("changed\n")); got != StatusModified {
		t.Errorf("内容变化应为 modified，实际 %s", got)
	}
	// 不在清单 → foreign
	if got := m.Classify("/other/file", content); got != StatusForeign {
		t.Errorf("不在清单应为 foreign，实际 %s", got)
	}
}

func TestExportsSaveLoad(t *testing.T) {
	r := openRepo(t)
	m := &ExportManifest{Tool: "codex", Scope: "global"}
	m.Record("/x/config.toml", []byte("data\n"))
	if err := r.SaveExportManifest(m); err != nil {
		t.Fatalf("Save: %v", err)
	}
	back, err := r.LoadExportManifest("codex", "global")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(back.Files) != 1 || back.Files[0].Path != "/x/config.toml" {
		t.Errorf("往返不一致: %+v", back.Files)
	}
	// 不存在的清单返回空
	empty, err := r.LoadExportManifest("ghost", "global")
	if err != nil || len(empty.Files) != 0 {
		t.Errorf("不存在清单应返回空: %v %+v", err, empty.Files)
	}
}

func TestExportsRebase(t *testing.T) {
	m := &ExportManifest{Tool: "codex", Scope: "global"}
	m.Record(`D:\proj\.mcp.json`, []byte("a"))
	m.Record(`D:\proj\config.toml`, []byte("b"))
	// 换机：D:\proj → F:\proj
	m.Rebase(func(old string) (string, bool) {
		if len(old) > 3 && old[:3] == `D:\` {
			return `F:\` + old[3:], true
		}
		return old, false
	})
	if m.Files[0].Path != `F:\proj\.mcp.json` {
		t.Errorf("rebase 路径错误: %s", m.Files[0].Path)
	}
}

func TestRepoLockContention(t *testing.T) {
	r := openRepo(t)
	if err := r.Lock(); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	// 同进程再开一个 Repo 指向同目录，应获取锁失败
	r2, err := Open(r.Root)
	if err != nil {
		t.Fatalf("Open2: %v", err)
	}
	if err := r2.Lock(); err == nil {
		t.Error("锁被占用时第二次 Lock 应失败")
	}
	if err := r.Unlock(); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	// 释放后可获取
	if err := r2.Lock(); err != nil {
		t.Errorf("释放后应可获取锁: %v", err)
	}
	r2.Unlock()
}
