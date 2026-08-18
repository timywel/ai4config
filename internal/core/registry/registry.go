package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/timywel/ai4config/internal/atomicfile"
)

// Project 注册的项目条目（ARCHITECTURE §4）。
type Project struct {
	ID           string      `yaml:"id"`
	Name         string      `yaml:"name"`
	Paths        []string    `yaml:"paths"` // 历史路径列表，重定位追加
	Fingerprint  Fingerprint `yaml:"fingerprint"`
	Profile      string      `yaml:"profile"` // profiles/projects/<pid>
	SameRemoteAs []string    `yaml:"same_remote_as,omitempty"`
	LinkedTools  []string    `yaml:"linked_tools,omitempty"`
	UpdatedAt    time.Time   `yaml:"updated_at"`
}

// Registry 项目注册表（registry.yaml）。
type Registry struct {
	Projects []Project `yaml:"projects"`
}

func registryPath(repoRoot string) string { return filepath.Join(repoRoot, "registry.yaml") }

// Load 读取 registry.yaml（不存在返回空注册表）。
func Load(repoRoot string) (*Registry, error) {
	data, err := os.ReadFile(registryPath(repoRoot))
	if os.IsNotExist(err) {
		return &Registry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var r Registry
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("registry: 解析 registry.yaml 失败: %w", err)
	}
	return &r, nil
}

// Save 写入 registry.yaml（atomicfile + 0600）。
func (r *Registry) Save(repoRoot string) error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	return atomicfile.WriteFile(registryPath(repoRoot), data, 0o600)
}

// FindByFingerprint 按指纹优先级查找：git_remote > root_name+first_commit > root_name。
func (r *Registry) FindByFingerprint(fp Fingerprint) *Project {
	// 1. git_remote
	if fp.GitRemote != "" {
		for i := range r.Projects {
			if r.Projects[i].Fingerprint.GitRemote == fp.GitRemote {
				return &r.Projects[i]
			}
		}
	}
	// 2. root_name + first_commit
	if fp.FirstCommit != "" {
		for i := range r.Projects {
			p := &r.Projects[i]
			if p.Fingerprint.RootName == fp.RootName && p.Fingerprint.FirstCommit == fp.FirstCommit {
				return p
			}
		}
	}
	// 3. root_name
	for i := range r.Projects {
		if r.Projects[i].Fingerprint.RootName == fp.RootName && fp.RootName != "" {
			return &r.Projects[i]
		}
	}
	return nil
}

// FindByPath 路径直接命中。
func (r *Registry) FindByPath(path string) *Project {
	for i := range r.Projects {
		for _, p := range r.Projects[i].Paths {
			if p == path {
				return &r.Projects[i]
			}
		}
	}
	return nil
}

// Link 关联项目路径到 profile（含二次判别，D10）。
// confirm 为二次判别确认回调（nil 视为确认）。返回 (项目, 是否合并到已有, 错误)。
func (r *Registry) Link(path string, confirm func(prompt string) bool) (*Project, bool, error) {
	fp := ComputeFingerprint(path)
	existing := r.FindByFingerprint(fp)

	if existing != nil {
		// 二次判别：first_commit 冲突 → 新建 pid（记 same_remote_as）
		if fp.FirstCommit != "" && existing.Fingerprint.FirstCommit != "" &&
			fp.FirstCommit != existing.Fingerprint.FirstCommit {
			np := r.createProject(path, fp, existing.ID)
			return np, false, nil
		}
		// 用户确认
		if confirm != nil && !confirm(fmt.Sprintf("路径 %s 命中已注册项目 %s，合并关联？", path, existing.ID)) {
			np := r.createProject(path, fp, existing.ID)
			return np, false, nil
		}
		// 合并：追加路径
		if !containsStr(existing.Paths, path) {
			existing.Paths = append(existing.Paths, path)
		}
		existing.UpdatedAt = time.Now().UTC()
		return existing, true, nil
	}

	np := r.createProject(path, fp, "")
	return np, true, nil
}

// Relink 凭指纹重定位已注册项目（目录迁移/改名后认亲）。
func (r *Registry) Relink(path string) (*Project, error) {
	fp := ComputeFingerprint(path)
	existing := r.FindByFingerprint(fp)
	if existing == nil {
		return nil, fmt.Errorf("registry: 无指纹匹配的项目（root=%s remote=%s）", fp.RootName, fp.GitRemote)
	}
	if !containsStr(existing.Paths, path) {
		existing.Paths = append(existing.Paths, path)
	}
	existing.UpdatedAt = time.Now().UTC()
	return existing, nil
}

// VerifyPath 路径命中时的指纹复核（D10，红队 T-02 防线）：
// 路径命中注册项但 first_commit 不匹配 → 返回 false（疑似路径复用劫持）。
func (r *Registry) VerifyPath(path string) (matched *Project, ok bool) {
	p := r.FindByPath(path)
	if p == nil {
		return nil, true // 未命中不算异常
	}
	fp := ComputeFingerprint(path)
	if p.Fingerprint.FirstCommit != "" && fp.FirstCommit != "" &&
		p.Fingerprint.FirstCommit != fp.FirstCommit {
		return p, false // 路径命中但指纹不符（张冠李戴风险）
	}
	return p, true
}

// createProject 新建项目条目。
func (r *Registry) createProject(path string, fp Fingerprint, sameRemoteAs string) *Project {
	id := newProjectID()
	p := Project{
		ID:          id,
		Name:        fp.RootName,
		Paths:       []string{path},
		Fingerprint: fp,
		Profile:     filepath.Join("profiles", "projects", id),
		UpdatedAt:   time.Now().UTC(),
	}
	if sameRemoteAs != "" {
		p.SameRemoteAs = []string{sameRemoteAs}
	}
	r.Projects = append(r.Projects, p)
	return &r.Projects[len(r.Projects)-1]
}

// newProjectID 生成项目 id（prj_ + 时间戳 base36 + 随机后缀）。
func newProjectID() string {
	now := time.Now().UTC()
	return fmt.Sprintf("prj_%d%02d%02d%02d%02d%02d", now.Year()%100, now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
}

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
