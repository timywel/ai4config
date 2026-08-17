//go:build !windows

package atomicfile

import "os"

// renameFile Unix：os.Rename 原子替换（同卷前提由 WriteFile 的 temp 位置保证）。
func renameFile(oldpath, newpath string) error { return os.Rename(oldpath, newpath) }

// applyPerm Unix：恢复调用方指定权限（skill assets 的 0755 执行位、SSOT 的 0600 等）。
func applyPerm(path string, perm os.FileMode) error { return os.Chmod(path, perm) }

// syncDir fsync 父目录，保证 rename 结果落盘。
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
