package atomicfile

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteFileBasic(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a.txt")
	if err := WriteFile(target, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "hello" {
		t.Fatalf("内容不一致: %v %q", err, got)
	}
	// 无残留 temp
	matches, _ := filepath.Glob(filepath.Join(dir, ".a.txt.tmp-*"))
	if len(matches) != 0 {
		t.Errorf("不应残留 temp 文件: %v", matches)
	}
}

func TestWriteFileOverwrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a.txt")
	WriteFile(target, []byte("v1"), 0o600)
	if err := WriteFile(target, []byte("v2"), 0o600); err != nil {
		t.Fatalf("覆盖写: %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "v2" {
		t.Errorf("覆盖后内容错误: %q", got)
	}
}

// symlink 穿透：目标是已存在的链接 → 写入应落到链接真实目标，链接本身保留
func TestWriteFileSymlinkTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 创建 symlink 需特权，跳过")
	}
	dir := t.TempDir()
	real := filepath.Join(dir, "real.txt")
	os.WriteFile(real, []byte("old"), 0o600)
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(real, link); err != nil {
		t.Fatalf("创建链接: %v", err)
	}
	if err := WriteFile(link, []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteFile 穿透链接: %v", err)
	}
	// 链接本身仍在（未被替换为普通文件）
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Error("链接本身被替换为普通文件（穿透失败）")
	}
	got, _ := os.ReadFile(real)
	if string(got) != "new" {
		t.Errorf("真实目标内容应为 new: %q", got)
	}
}

// 父目录是链接 + 新建文件：穿透到真实目录创建
func TestWriteFileSymlinkParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 创建 symlink 需特权，跳过")
	}
	dir := t.TempDir()
	realDir := filepath.Join(dir, "realdir")
	os.MkdirAll(realDir, 0o755)
	linkDir := filepath.Join(dir, "linkdir")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("创建目录链接: %v", err)
	}
	target := filepath.Join(linkDir, "new.txt")
	if err := WriteFile(target, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile 父目录穿透: %v", err)
	}
	if _, err := os.Stat(filepath.Join(realDir, "new.txt")); err != nil {
		t.Errorf("文件应创建在真实目录: %v", err)
	}
}

func TestWriteFilePermUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 无 Unix 权限语义")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "exec.sh")
	if err := WriteFile(target, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm() != 0o755 {
		t.Errorf("权限应为 0755（执行位），实际 %o", info.Mode().Perm())
	}
}
