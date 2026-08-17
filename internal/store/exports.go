package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/timywel/ai4config/internal/atomicfile"
)

// ExportFile 导出清单单文件条目。
type ExportFile struct {
	Path      string    `yaml:"path"` // 目标工具内的文件绝对路径
	Hash      string    `yaml:"hash"` // 规范化后内容 sha256（CRLF→LF、去 BOM）
	WrittenAt time.Time `yaml:"written_at"`
}

// ExportManifest 导出清单（exports/<tool>/<scope>/manifest.yaml）。
// 用途（ARCHITECTURE §5.3）：识别"非本工具生成"的目标文件。
type ExportManifest struct {
	Tool      string       `yaml:"tool"`
	Scope     string       `yaml:"scope"`
	UpdatedAt time.Time    `yaml:"updated_at"`
	Files     []ExportFile `yaml:"files"`
}

// ForeignStatus 外来内容三态判定。
type ForeignStatus int

const (
	StatusOurs     ForeignStatus = iota // 本工具产物（清单内且 hash 一致）→ 直接覆盖
	StatusModified                      // 清单内但 hash 变 → 被外部修改（需确认）
	StatusForeign                       // 不在清单 → 外来内容（需确认）
)

func (s ForeignStatus) String() string {
	switch s {
	case StatusOurs:
		return "ours"
	case StatusModified:
		return "modified"
	case StatusForeign:
		return "foreign"
	}
	return "unknown"
}

// NormalizeForHash 字节级规范化（CRLF→LF、去 BOM）——防 IDE 自重写格式造成确认疲劳。
func NormalizeForHash(data []byte) []byte {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	return data
}

// exportManifestPath 清单文件路径。
func (r *Repo) exportManifestPath(tool, scope string) string {
	return filepath.Join(r.Root, DirExports, tool, scope, "manifest.yaml")
}

// LoadExportManifest 读取导出清单；不存在返回空清单（首次导出前）。
func (r *Repo) LoadExportManifest(tool, scope string) (*ExportManifest, error) {
	data, err := os.ReadFile(r.exportManifestPath(tool, scope))
	if os.IsNotExist(err) {
		return &ExportManifest{Tool: tool, Scope: scope}, nil
	}
	if err != nil {
		return nil, err
	}
	var m ExportManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("store: 解析导出清单失败: %w", err)
	}
	return &m, nil
}

// SaveExportManifest 写入导出清单。
func (r *Repo) SaveExportManifest(m *ExportManifest) error {
	m.UpdatedAt = time.Now().UTC()
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	p := r.exportManifestPath(m.Tool, m.Scope)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return atomicfile.WriteFile(p, data, 0o600)
}

// Classify 三态判定：path 的当前内容 vs 清单记录。
func (m *ExportManifest) Classify(path string, currentContent []byte) ForeignStatus {
	for _, f := range m.Files {
		if f.Path == path {
			if f.Hash == BlobHash(NormalizeForHash(currentContent)) {
				return StatusOurs
			}
			return StatusModified
		}
	}
	return StatusForeign
}

// Record 登记/更新一个写出文件（导出成功后调用）。
func (m *ExportManifest) Record(path string, content []byte) {
	hash := BlobHash(NormalizeForHash(content))
	now := time.Now().UTC()
	for i := range m.Files {
		if m.Files[i].Path == path {
			m.Files[i].Hash = hash
			m.Files[i].WrittenAt = now
			return
		}
	}
	m.Files = append(m.Files, ExportFile{Path: path, Hash: hash, WrittenAt: now})
}

// Rebase 换机/重定位后重写清单内路径（D9）。pathMapper 返回新路径与是否保留。
func (m *ExportManifest) Rebase(pathMapper func(old string) (newPath string, keep bool)) {
	out := m.Files[:0]
	for _, f := range m.Files {
		np, keep := pathMapper(f.Path)
		if keep {
			f.Path = np
			out = append(out, f)
		}
	}
	m.Files = out
}
