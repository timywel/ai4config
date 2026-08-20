package profile

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/timywel/ai4config/internal/atomicfile"
)

// Annotations 治理元数据侧车（IR-SCHEMA §4.5，profiles/<p>/annotations.yaml）。
// 承载不进实体的治理维度：禁用/标签/收藏/钉住。
type Annotations struct {
	Disabled []string            `yaml:"disabled,omitempty"` // 泛化禁用清单（merge 后、Map 前过滤）
	Labels   map[string][]string `yaml:"labels,omitempty"`   // 自定义标签
	Favorite []string            `yaml:"favorite,omitempty"` // 收藏
	Pinned   []string            `yaml:"pinned,omitempty"`   // 钉住：collect 回流不更新
}

func annotationsPath(dir string) string { return filepath.Join(dir, "annotations.yaml") }

// LoadAnnotations 读取侧车（不存在返回空）。
func LoadAnnotations(dir string) (*Annotations, error) {
	data, err := os.ReadFile(annotationsPath(dir))
	if os.IsNotExist(err) {
		return &Annotations{}, nil
	}
	if err != nil {
		return nil, err
	}
	var a Annotations
	if err := yaml.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// SaveAnnotations 写入侧车（atomicfile + 0600）。
func SaveAnnotations(dir string, a *Annotations) error {
	data, err := yaml.Marshal(a)
	if err != nil {
		return err
	}
	return atomicfile.WriteFile(annotationsPath(dir), data, 0o600)
}

// IsDisabled 条目是否在禁用清单。
func (a *Annotations) IsDisabled(id string) bool {
	if a == nil {
		return false
	}
	for _, d := range a.Disabled {
		if d == id {
			return true
		}
	}
	return false
}

// IsPinned 条目是否钉住（collect 回流不更新）。
func (a *Annotations) IsPinned(id string) bool {
	if a == nil {
		return false
	}
	for _, p := range a.Pinned {
		if p == id {
			return true
		}
	}
	return false
}

// ToggleDisabled 切换禁用态。
func (a *Annotations) ToggleDisabled(id string) {
	if a.Disabled == nil {
		a.Disabled = []string{}
	}
	if a.IsDisabled(id) {
		out := a.Disabled[:0]
		for _, d := range a.Disabled {
			if d != id {
				out = append(out, d)
			}
		}
		a.Disabled = out
		return
	}
	a.Disabled = append(a.Disabled, id)
}
