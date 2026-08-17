// Package profile 实现 profile 的读写、五层合并物化与 ir_version 迁移。
// 权威规范：docs/IR-SCHEMA.md §2（合并语义）、§2.2（manifest）、ARCHITECTURE §7（存储布局）。
package profile

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Manifest 对应 profile 根清单 manifest.yaml（IR-SCHEMA §2.2）。
type Manifest struct {
	IRVersion   int               `yaml:"ir_version"`
	Profile     Meta              `yaml:"profile"`
	MergePolicy map[string]string `yaml:"merge_policy,omitempty"`
}

// Meta profile 元数据。
type Meta struct {
	Name      string    `yaml:"name"`
	Kind      string    `yaml:"kind"` // global | project
	CreatedAt time.Time `yaml:"created_at"`
}

// DefaultMergePolicy 默认合并策略（IR-SCHEMA §2.2）。
func DefaultMergePolicy() map[string]string {
	return map[string]string{
		"instructions": "concat",
		"mcp_servers":  "merge-by-id",
		"skills":       "merge-by-id",
		"agents":       "merge-by-id",
		"commands":     "merge-by-id",
		"workflows":    "merge-by-id",
		"hooks":        "merge-by-id",
		"settings":     "merge-by-id",
	}
}

// PolicyFor 返回某实体类型的合并策略（manifest 覆盖优先，否则默认）。
func (m *Manifest) PolicyFor(kindKey string) string {
	if m.MergePolicy != nil {
		if v, ok := m.MergePolicy[kindKey]; ok {
			return v
		}
	}
	return DefaultMergePolicy()[kindKey]
}

// LoadManifest 读取并校验 manifest.yaml（含 merge_policy 规则 8 校验与 ir_version 迁移）。
func LoadManifest(dir string) (*Manifest, error) {
	data, err := os.ReadFile(manifestPath(dir))
	if err != nil {
		return nil, fmt.Errorf("profile: 读取 manifest 失败: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("profile: 解析 manifest 失败: %w", err)
	}
	if m.IRVersion == 0 {
		return nil, fmt.Errorf("profile: manifest 缺少 ir_version")
	}
	// 规则 8：merge_policy 键值合法性
	if issues := validateMergePolicy(m.MergePolicy); len(issues) > 0 {
		return nil, fmt.Errorf("profile: merge_policy 非法: %v", issues[0])
	}
	// ir_version 链式迁移（高于实现版本拒绝）
	if err := MigrateManifest(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// SaveManifest 写入 manifest.yaml（0600，走 atomicfile 由调用方保证——此处直接写，
// 实际落盘统一经 store.Save 编排）。
func SaveManifest(dir string, m *Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("profile: 编码 manifest 失败: %w", err)
	}
	return writeProfileFile(manifestPath(dir), data)
}

// validateMergePolicy 包装 ir 层规则 8 校验（避免循环依赖，直接引用）。
func validateMergePolicy(policy map[string]string) []string {
	if len(policy) == 0 {
		return nil
	}
	var errs []string
	kinds := DefaultMergePolicy()
	for k, v := range policy {
		if _, ok := kinds[k]; !ok {
			errs = append(errs, fmt.Sprintf("merge_policy 键 %q 不是已知实体类型", k))
			continue
		}
		if k == "instructions" {
			if v != "concat" && v != "project-only" && v != "global-only" {
				errs = append(errs, fmt.Sprintf("merge_policy.instructions 值 %q 非法", v))
			}
		} else if v != "merge-by-id" {
			errs = append(errs, fmt.Sprintf("merge_policy.%s 值 %q 非法（仅允许 merge-by-id）", k, v))
		}
	}
	return errs
}
