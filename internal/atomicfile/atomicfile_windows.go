//go:build windows

package atomicfile

import "os"

// renameFile Windows：os.Rename。
// TODO(P0): 目标被占用（SHARING_VIOLATION/ACCESS_DENIED，杀软/索引/IDE 持锁）
// 时按指数退避重试，终失败报"文件被占用"及路径（ARCHITECTURE §5.3）。
func renameFile(oldpath, newpath string) error { return os.Rename(oldpath, newpath) }

// applyPerm Windows：权限位无 Unix 语义，no-op。
func applyPerm(path string, perm os.FileMode) error { return nil }

// syncDir Windows 无父目录 fsync 语义，no-op。
func syncDir(dir string) error { return nil }
