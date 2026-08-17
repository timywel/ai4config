package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/timywel/ai4config/internal/atomicfile"
)

// SnapshotFile 快照中单个文件的引用（内容入 blob，按 hash 去重）。
type SnapshotFile struct {
	Path string `yaml:"path"` // 相对仓库根
	Hash string `yaml:"hash"` // blob sha256
	Mode int64  `yaml:"mode"` // 权限位（Unix 语义）
}

// SnapshotMeta 快照清单（snapshots/<id>/manifest.yaml）。
type SnapshotMeta struct {
	ID        string         `yaml:"id"`
	CreatedAt time.Time      `yaml:"created_at"`
	Note      string         `yaml:"note,omitempty"`
	Files     []SnapshotFile `yaml:"files"`
}

// snapshotIncludes 快照覆盖的仓库内路径（SSOT 全量只读副本；blobs/snapshots/cache/logs 不入）。
var snapshotIncludes = []string{DirProfiles, DirExports, FileRegistry, FileConfig}

// CreateSnapshot 对 SSOT 全量打快照，返回快照 id（时间戳）。
// 文件内容入 blob（天然去重，快照间共享同内容块）。
func (r *Repo) CreateSnapshot(note string) (string, error) {
	ts := time.Now().UTC()
	id := ts.Format("20060102-150405")
	meta := SnapshotMeta{ID: id, CreatedAt: ts, Note: note}

	for _, inc := range snapshotIncludes {
		root := filepath.Join(r.Root, inc)
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			// 文件项（registry.yaml/config.yaml）可能不存在，跳过缺失项
			if err == nil { // 是文件
				if err := r.addFileToSnapshot(&meta, inc); err != nil {
					return "", err
				}
			}
			continue
		}
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(r.Root, path)
			if err != nil {
				return err
			}
			return r.addFileToSnapshot(&meta, rel)
		})
		if err != nil {
			return "", err
		}
	}

	sort.Slice(meta.Files, func(i, j int) bool { return meta.Files[i].Path < meta.Files[j].Path })

	data, err := yaml.Marshal(&meta)
	if err != nil {
		return "", err
	}
	dir := r.SnapshotDir(ts)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := atomicfile.WriteFile(filepath.Join(dir, "manifest.yaml"), data, 0o600); err != nil {
		return "", fmt.Errorf("store: 写快照 manifest 失败: %w", err)
	}
	return id, nil
}

// addFileToSnapshot 把单个文件内容入 blob 并登记到快照清单。
func (r *Repo) addFileToSnapshot(meta *SnapshotMeta, rel string) error {
	abs := filepath.Join(r.Root, rel)
	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	hash, err := r.PutBlob(data)
	if err != nil {
		return err
	}
	var mode int64 = 0o600
	if info, err := os.Stat(abs); err == nil {
		mode = int64(info.Mode().Perm())
	}
	meta.Files = append(meta.Files, SnapshotFile{Path: rel, Hash: hash, Mode: mode})
	return nil
}

// RestoreSnapshot 恢复快照（覆盖写回各文件路径；调用方负责先打反向快照）。
func (r *Repo) RestoreSnapshot(id string) error {
	meta, err := r.readSnapshot(id)
	if err != nil {
		return err
	}
	for _, f := range meta.Files {
		data, err := r.GetBlob(f.Hash)
		if err != nil {
			return fmt.Errorf("store: 快照 %s 的 blob %s 缺失: %w", id, f.Hash, err)
		}
		abs := filepath.Join(r.Root, f.Path)
		if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
			return err
		}
		perm := os.FileMode(f.Mode)
		if perm == 0 {
			perm = 0o600
		}
		if err := atomicfile.WriteFile(abs, data, perm); err != nil {
			return err
		}
	}
	return nil
}

// ListSnapshots 列出全部快照（按时间升序）。
func (r *Repo) ListSnapshots() ([]SnapshotMeta, error) {
	root := filepath.Join(r.Root, DirSnapshots)
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []SnapshotMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := r.readSnapshot(e.Name())
		if err != nil {
			continue // 跳过损坏快照
		}
		out = append(out, *meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

// readSnapshot 读取某快照 manifest。
func (r *Repo) readSnapshot(id string) (*SnapshotMeta, error) {
	data, err := os.ReadFile(filepath.Join(r.Root, DirSnapshots, id, "manifest.yaml"))
	if err != nil {
		return nil, fmt.Errorf("store: 读快照 %s 失败: %w", id, err)
	}
	var meta SnapshotMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// PruneSnapshots 按保留策略回收快照（默认保留最近 keep 份）。
func (r *Repo) PruneSnapshots(keep int) (int, error) {
	list, err := r.ListSnapshots()
	if err != nil {
		return 0, err
	}
	if len(list) <= keep {
		return 0, nil
	}
	removed := 0
	for _, s := range list[:len(list)-keep] { // 最旧的在前
		if err := os.RemoveAll(filepath.Join(r.Root, DirSnapshots, s.ID)); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
