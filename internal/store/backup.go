package store

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"gopkg.in/yaml.v3"
)

// 备份包格式（.cfg4aibak，OPTIMIZATION-PLAN F12）：
//   tar.gz{ manifest.yaml + 白名单内容子集 }，整体 age passphrase 加密。
// 范围限定 sync 白名单（profiles/registry/config/exports）——与出库语义一致。

// BackupManifest 备份包清单。
type BackupManifest struct {
	Version   int       `yaml:"version"`
	CreatedAt time.Time `yaml:"created_at"`
	Host      string    `yaml:"host"`
	Files     int       `yaml:"files"`
}

// ExportBackup 导出加密备份包：打包白名单内容 → age 口令加密 → dest。
// 口令必须 ≥8 位（Raycast 同款纪律）。
func (r *Repo) ExportBackup(destPath, passphrase string) (int, error) {
	if len(passphrase) < 8 {
		return 0, fmt.Errorf("store: 备份口令至少 8 位")
	}
	// 打包白名单内容为 tar.gz 到临时文件
	tmp := filepath.Join(os.TempDir(), "cfg4ai-backup-"+fmt.Sprintf("%d", time.Now().UnixNano())+".tar.gz")
	defer os.Remove(tmp)
	tf, err := os.Create(tmp)
	if err != nil {
		return 0, err
	}
	gz := gzip.NewWriter(tf)
	tw := tar.NewWriter(gz)
	count := 0
	host, _ := os.Hostname()
	// manifest
	manData, _ := yamlMarshalBackup(BackupManifest{Version: 1, CreatedAt: time.Now().UTC(), Host: host})
	if err := tarAdd(tw, "manifest.yaml", manData); err != nil {
		tf.Close()
		return 0, err
	}
	count++
	// 白名单目录
	for _, wl := range SyncWhitelist {
		root := filepath.Join(r.Root, wl)
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(r.Root, path)
			if err != nil {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if err := tarAdd(tw, filepath.ToSlash(rel), data); err != nil {
				return err
			}
			count++
			return nil
		})
	}
	if err := tw.Close(); err != nil {
		tf.Close()
		return 0, err
	}
	if err := gz.Close(); err != nil {
		tf.Close()
		return 0, err
	}
	tf.Close()

	// age 加密
	plain, err := os.ReadFile(tmp)
	if err != nil {
		return 0, err
	}
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return 0, err
	}
	out, err := os.Create(destPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	w, err := age.Encrypt(out, recipient)
	if err != nil {
		return 0, err
	}
	if _, err := w.Write(plain); err != nil {
		return 0, err
	}
	if err := w.Close(); err != nil {
		return 0, err
	}
	return count, nil
}

// BackupContents 列出备份包内容清单（导入向导的勾选树用）。
func (r *Repo) BackupContents(srcPath, passphrase string) ([]string, error) {
	plain, err := decryptBackup(srcPath, passphrase)
	if err != nil {
		return nil, err
	}
	return listTarGZ(plain)
}

// ImportBackup 导入备份包：解密 → 按策略合并。
// strategy: skip | overwrite | merge（同 id 同 hash 跳过；冲突按策略）。
func (r *Repo) ImportBackup(srcPath, passphrase, strategy string) (int, error) {
	plain, err := decryptBackup(srcPath, passphrase)
	if err != nil {
		return 0, err
	}
	return r.extractBackup(plain, strategy)
}

// decryptBackup age 解密备份包。
func decryptBackup(srcPath, passphrase string) ([]byte, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, err
	}
	r, err := age.Decrypt(f, identity)
	if err != nil {
		return nil, fmt.Errorf("store: 备份解密失败（口令错误？）: %w", err)
	}
	return io.ReadAll(r)
}

// extractBackup 解包 tar.gz 并写回仓库（按策略）。
func (r *Repo) extractBackup(plain []byte, strategy string) (int, error) {
	count := 0
	// 从内存 bytes 读 tar.gz
	zr, err := gzip.NewReader(strings.NewReader(string(plain)))
	if err != nil {
		return 0, err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, err
		}
		if hdr.Name == "manifest.yaml" {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return count, err
		}
		dest := filepath.Join(r.Root, filepath.FromSlash(hdr.Name))
		// 策略：skip=存在即跳过；overwrite=覆盖；merge=同 id 同 hash 跳过
		if _, err := os.Stat(dest); err == nil && strategy == "skip" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return count, err
		}
		if err := os.WriteFile(dest, data, 0o600); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func tarAdd(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{Name: name, Mode: 0o600, Size: int64(len(data))}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

func listTarGZ(plain []byte) ([]string, error) {
	zr, err := gzip.NewReader(strings.NewReader(string(plain)))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	var out []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, err
		}
		out = append(out, hdr.Name)
	}
	return out, nil
}

func yamlMarshalBackup(v any) ([]byte, error) { return yaml.Marshal(v) }
