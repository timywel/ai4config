//go:build windows

package atomicfile

import (
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// renameFile Windows：目标被占用（杀软/索引/IDE 持锁）时按指数退避重试（ARCHITECTURE §5.3）。
// 终失败报"文件被占用"及路径；非占用类错误立即返回。
func renameFile(oldpath, newpath string) error {
	const maxAttempts = 6
	delay := 50 * time.Millisecond
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = os.Rename(oldpath, newpath)
		if err == nil {
			return nil
		}
		if !isSharingViolation(err) {
			return err // 非占用类错误（如权限、路径非法）不重试
		}
		if attempt < maxAttempts {
			time.Sleep(delay)
			delay *= 2 // 指数退避：50ms→100→200→400→800
		}
	}
	return fmt.Errorf("atomicfile: 目标文件被占用（重试 %d 次仍失败，可能被杀软/索引/IDE 持锁）: %s: %w",
		maxAttempts, newpath, err)
}

// isSharingViolation 识别 Windows 文件占用类错误。
//   - ERROR_SHARING_VIOLATION(32)：另一进程以非共享模式打开
//   - ERROR_ACCESS_DENIED(5)：权限/只读/占用亦可表现为此（杀软扫描瞬时持锁常见）
func isSharingViolation(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_ACCESS_DENIED)
}

// applyPerm Windows：权限位无 Unix 语义，no-op。
func applyPerm(path string, perm os.FileMode) error { return nil }

// syncDir Windows 无父目录 fsync 语义，no-op。
func syncDir(dir string) error { return nil }
