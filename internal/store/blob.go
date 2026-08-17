package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/timywel/ai4config/internal/atomicfile"
)

// blob 内容寻址存储：blobs/<sha256 前2位>/<sha256 完整>。
// 职责边界（ARCHITECTURE §9）：blob 只负责存取；**脱敏是写入方的义务**——
// 本包假设传入内容已脱敏（先扫描替换→落盘→零命中校验由 secrets 管线保证）。

// BlobHash 计算内容 sha256（hex），即 blob 的寻址键。
func BlobHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// blobPath 返回某 hash 的存储路径。
func (r *Repo) blobPath(hash string) string {
	return filepath.Join(r.Root, DirBlobs, hash[:2], hash)
}

// PutBlob 写入内容并返回其 hash；同内容已存在则直接返回（天然去重）。
func (r *Repo) PutBlob(data []byte) (string, error) {
	hash := BlobHash(data)
	p := r.blobPath(hash)
	if _, err := os.Stat(p); err == nil {
		return hash, nil // 去重：已存在
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return "", err
	}
	if err := atomicfile.WriteFile(p, data, 0o600); err != nil {
		return "", fmt.Errorf("store: 写 blob 失败: %w", err)
	}
	return hash, nil
}

// GetBlob 按 hash 读取内容。
func (r *Repo) GetBlob(hash string) ([]byte, error) {
	data, err := os.ReadFile(r.blobPath(hash))
	if err != nil {
		return nil, fmt.Errorf("store: 读 blob %s 失败: %w", hash, err)
	}
	return data, nil
}

// HasBlob 判断 hash 是否存在。
func (r *Repo) HasBlob(hash string) bool {
	_, err := os.Stat(r.blobPath(hash))
	return err == nil
}

// ListBlobs 列出全部 blob hash（GC 标记-清除用）。
func (r *Repo) ListBlobs() ([]string, error) {
	var out []string
	root := filepath.Join(r.Root, DirBlobs)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		out = append(out, filepath.Base(path))
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	return out, err
}
