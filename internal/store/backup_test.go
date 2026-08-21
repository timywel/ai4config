package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupRoundTrip(t *testing.T) {
	// 源仓库：写白名单内容
	src, _ := Open(filepath.Join(t.TempDir(), "src"))
	os.MkdirAll(src.Path(DirProfiles, "global"), 0o700)
	os.WriteFile(src.Path(DirProfiles, "global", "manifest.yaml"), []byte("ir_version: 1\n"), 0o600)

	dest := filepath.Join(t.TempDir(), "backup.cfg4aibak")
	pass := "test-passphrase"
	n, err := src.ExportBackup(dest, pass)
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}
	if n < 1 {
		t.Error("应至少导出 manifest")
	}

	// 内容清单
	names, err := src.BackupContents(dest, pass)
	if err != nil {
		t.Fatalf("BackupContents: %v", err)
	}
	found := false
	for _, name := range names {
		if name == "profiles/global/manifest.yaml" {
			found = true
		}
	}
	if !found {
		t.Errorf("清单应含 profiles/global/manifest.yaml: %v", names)
	}

	// 导入到新仓库
	dst, _ := Open(filepath.Join(t.TempDir(), "dst"))
	cnt, err := dst.ImportBackup(dest, pass, "overwrite")
	if err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}
	if cnt < 1 {
		t.Error("应导入文件")
	}
	if _, err := os.Stat(dst.Path(DirProfiles, "global", "manifest.yaml")); err != nil {
		t.Errorf("导入后应有 manifest.yaml: %v", err)
	}

	// 错误口令应失败
	if _, err := dst.ImportBackup(dest, "wrong-passphrase", "overwrite"); err == nil {
		t.Error("错误口令应解密失败")
	}
}

func TestExportBackupShortPassword(t *testing.T) {
	src, _ := Open(filepath.Join(t.TempDir(), "src"))
	_, err := src.ExportBackup(filepath.Join(t.TempDir(), "x.cfg4aibak"), "short")
	if err == nil {
		t.Error("口令<8位应拒绝")
	}
}
