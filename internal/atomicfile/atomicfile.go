// Package atomicfile 是写入协议的唯一实现（ARCHITECTURE §5.3）。
// 全部适配器禁止手写写文件逻辑，必须经本包。
//
// 协议要点：
//  1. 单文件原子：temp（与目标同卷同目录）→ fsync → rename → 父目录 sync
//  2. 批量非原子，以快照补偿（由调用方/core/store 编排）
//  3. Windows 共享冲突指数退避重试
//  4. 写入前 EvalSymlinks 穿透（绝不替换链接本身）
package atomicfile

import (
	"os"
	"path/filepath"
)

// WriteFile 原子写入 data 到 path（遵循上述协议）。perm 仅 Unix 语义（Windows 忽略）。
func WriteFile(path string, data []byte, perm os.FileMode) error {
	// TODO(P0): Windows SHARING_VIOLATION/ACCESS_DENIED 指数退避重试（renameFile 平台实现）
	// 符号链接穿透：先解析父目录（覆盖新建文件场景——目标不存在时 EvalSymlinks(path) 会失败），
	// 再解析目标本身（覆盖已存在文件场景），保证"绝不替换链接本身"不依赖 OS 隐式行为。
	if realDir, err := filepath.EvalSymlinks(filepath.Dir(path)); err == nil {
		path = filepath.Join(realDir, filepath.Base(path))
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // 失败路径清理；rename 成功后为 no-op
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil { // fsync：掉电窗口保障
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := renameFile(tmpName, path); err != nil {
		return err
	}
	if err := applyPerm(path, perm); err != nil { // 应用调用方权限（Unix chmod；Windows no-op）
		return err
	}
	return syncDir(dir) // 父目录 sync（Unix；Windows 为 no-op）
}
