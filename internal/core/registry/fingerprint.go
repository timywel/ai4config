// Package registry 项目注册表与指纹关联（ARCHITECTURE §4、CLI-SPEC §4）。
package registry

import (
	"os"
	"os/exec"

	"github.com/timywel/ai4config/internal/platform/hidecmd"
	"path/filepath"
	"strings"
)

// Fingerprint 项目指纹（用于目录迁移/改名后的重新关联）。
type Fingerprint struct {
	GitRemote   string `yaml:"git_remote,omitempty"`   // 规范化后的 git remote
	RootName    string `yaml:"root_name,omitempty"`    // 根目录名
	FirstCommit string `yaml:"first_commit,omitempty"` // 首次提交 hash
}

// NormalizeGitRemote 规范化 git remote URL（4 条规则：去协议/去 .git/host 小写/scp 转标准）。
// 使 git@github.com:u/r.git 与 https://github.com/u/r 归一为同一指纹 github.com/u/r。
func NormalizeGitRemote(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		// scp 风格：git@github.com:user/repo.git → github.com/user/repo.git
		if at := strings.Index(s, "@"); at >= 0 {
			s = s[at+1:]
		}
		s = strings.Replace(s, ":", "/", 1)
	} else {
		// 含协议：https://user@github.com/user/repo.git → github.com/user/repo.git
		if i := strings.Index(s, "://"); i >= 0 {
			s = s[i+3:]
		}
		if at := strings.Index(s, "@"); at >= 0 {
			s = s[at+1:]
		}
	}
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimSuffix(s, "/")
	// host 小写（路径保留原样）
	parts := strings.SplitN(s, "/", 2)
	host := strings.ToLower(parts[0])
	if len(parts) == 2 {
		return host + "/" + parts[1]
	}
	return host
}

// ComputeFingerprint 计算项目指纹（读取 .git；无 git 仓库降级为 root_name）。
func ComputeFingerprint(projectPath string) Fingerprint {
	fp := Fingerprint{RootName: filepath.Base(projectPath)}
	if !isDir(filepath.Join(projectPath, ".git")) {
		return fp
	}
	fp.GitRemote = NormalizeGitRemote(gitConfigGet(projectPath, "remote.origin.url"))
	fp.FirstCommit = gitFirstCommit(projectPath)
	return fp
}

// gitConfigGet 解析 .git/config（INI 文本）取 remote URL（避免外部命令依赖）。
func gitConfigGet(projectPath, key string) string {
	data, err := os.ReadFile(filepath.Join(projectPath, ".git", "config"))
	if err != nil {
		return ""
	}
	// key 形如 remote.origin.url → 找 [remote "origin"] 段的 url =
	section := ""
	if strings.HasPrefix(key, "remote.") {
		parts := strings.Split(key, ".")
		if len(parts) == 3 {
			section = `[remote "` + parts[1] + `"]`
			key = parts[2]
		}
	}
	inSection := section == ""
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") {
			inSection = t == section
			continue
		}
		if inSection && strings.HasPrefix(t, key) {
			if eq := strings.Index(t, "="); eq >= 0 {
				return strings.TrimSpace(t[eq+1:])
			}
		}
	}
	return ""
}

// gitFirstCommit 取首次提交 hash（git rev-list --max-parents=0 HEAD 的最早一条）。
func gitFirstCommit(projectPath string) string {
	cmd := exec.Command("git", "-C", projectPath, "rev-list", "--max-parents=0", "HEAD")
	hidecmd.Hide(cmd)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Fields(string(out))
	if len(lines) > 0 {
		return lines[0]
	}
	return ""
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
