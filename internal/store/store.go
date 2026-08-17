// Package store 是 SSOT 仓库的唯一入口：仓库初始化、写锁、快照、blob、导出清单。
// 权威规范：docs/ARCHITECTURE.md §7（存储布局）、§5.3（写入协议）、§9（安全）。
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// 仓库子目录（ARCHITECTURE §7）。
const (
	DirProfiles  = "profiles"
	DirExports   = "exports"
	DirSnapshots = "snapshots"
	DirBlobs     = "blobs"
	DirCache     = "cache"
	DirLogs      = "logs"

	FileConfig   = "config.yaml"
	FileRegistry = "registry.yaml"
	FileLock     = ".lock"
)

// Repo SSOT 仓库根。
type Repo struct {
	Root  string
	flock *flock.Flock
}

// Open 打开仓库根目录；不存在则按 0700 初始化目录骨架。
// root 通常由 platform/paths.DataHome() 提供（CFG4AI_HOME 覆盖）。
func Open(root string) (*Repo, error) {
	r := &Repo{Root: root}
	if err := r.ensureLayout(); err != nil {
		return nil, err
	}
	r.flock = flock.New(filepath.Join(root, FileLock))
	return r, nil
}

// ensureLayout 初始化目录骨架并收紧权限（目录 0700 / 已存在则校验修正）。
func (r *Repo) ensureLayout() error {
	dirs := []string{
		r.Root,
		filepath.Join(r.Root, DirProfiles),
		filepath.Join(r.Root, DirExports),
		filepath.Join(r.Root, DirSnapshots),
		filepath.Join(r.Root, DirBlobs),
		filepath.Join(r.Root, DirCache),
		filepath.Join(r.Root, DirLogs),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return fmt.Errorf("store: 创建目录 %s 失败: %w", d, err)
		}
		if err := os.Chmod(d, 0o700); err != nil { // Unix 语义；Windows no-op 见 chmod 实现
			return fmt.Errorf("store: 收紧目录权限 %s 失败: %w", d, err)
		}
	}
	return nil
}

// Lock 取仓库级写锁（跨平台 flock；W1[1] 等流程的并发防线）。
// 锁被占用时返回明确错误（调用方可提示"另一个 cfg4ai 进程正在写入"）。
func (r *Repo) Lock() error {
	ok, err := r.flock.TryLock()
	if err != nil {
		return fmt.Errorf("store: 获取写锁失败: %w", err)
	}
	if !ok {
		return fmt.Errorf("store: 仓库写锁被占用（另一 cfg4ai 进程可能正在写入）；若确认无进程运行，请用 doctor 清理 stale 锁")
	}
	return nil
}

// Unlock 释放写锁。
func (r *Repo) Unlock() error {
	if r.flock == nil {
		return nil
	}
	return r.flock.Unlock()
}

// Path 拼接仓库内路径。
func (r *Repo) Path(elem ...string) string {
	return filepath.Join(append([]string{r.Root}, elem...)...)
}

// SnapshotDir 返回某时间戳的快照目录路径（不创建）。
func (r *Repo) SnapshotDir(ts time.Time) string {
	return filepath.Join(r.Root, DirSnapshots, ts.UTC().Format("20060102-150405"))
}
