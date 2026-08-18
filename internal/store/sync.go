package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/timywel/ai4config/internal/core/secrets"
)

// Sync 白名单制（ARCHITECTURE §9、CLI-SPEC §8）：
// 仅以下路径入 git 远端；其余强制 gitignore（防 secret/快照/锁外泄）。

// SyncWhitelist 允许同步的路径。
var SyncWhitelist = []string{DirProfiles, FileRegistry, FileConfig, DirExports}

// syncGitignoreBaseline sync init 自动写入的排除基线。
var syncGitignoreBaseline = []string{
	DirSnapshots + "/",
	DirBlobs + "/",
	DirLogs + "/",
	DirCache + "/",
	"secrets.age",
	FileLock,
	"consent.yaml",
	".gitignore",
}

// SyncInit 初始化 sync：git init + remote + .gitignore 白名单基线。
func (r *Repo) SyncInit(remoteURL string) error {
	// .gitignore 基线（白名单制：默认全部忽略，仅放行白名单）
	gi := filepath.Join(r.Root, ".gitignore")
	var sb strings.Builder
	sb.WriteString("# cfg4ai sync 白名单制：默认全忽略，仅放行 profiles/registry/config/exports\n")
	sb.WriteString("*\n")
	for _, w := range SyncWhitelist {
		sb.WriteString("!" + w + "\n!" + w + "/**\n")
	}
	if err := os.WriteFile(gi, []byte(sb.String()), 0o600); err != nil {
		return err
	}

	repo, err := git.PlainInit(r.Root, false)
	if err != nil && err != git.ErrRepositoryAlreadyExists {
		return fmt.Errorf("store: git init 失败: %w", err)
	}
	if repo == nil {
		repo, err = git.PlainOpen(r.Root)
		if err != nil {
			return err
		}
	}
	if remoteURL != "" {
		_ = repo.DeleteRemote("origin")
		if _, err := repo.CreateRemote(&gitconfig.RemoteConfig{Name: "origin", URLs: []string{remoteURL}}); err != nil {
			return fmt.Errorf("store: 配置 remote 失败: %w", err)
		}
	}
	return nil
}

// SyncPush preflight 全仓敏感扫描 → 提交白名单 → push。
// preflight 命中即阻断（红队 T-05：自由文本 secret 不得上远端）。
func (r *Repo) SyncPush(scanner *secrets.Scanner, confirm func(matches []secrets.ScanMatch) bool) error {
	// preflight：全仓扫描白名单范围内容（含自由文本）
	matches := r.preflightScan(scanner)
	if len(matches) > 0 {
		if confirm == nil || !confirm(matches) {
			return fmt.Errorf("store: sync preflight 命中 %d 处疑似敏感内容，已阻断 push", len(matches))
		}
	}

	repo, err := git.PlainOpen(r.Root)
	if err != nil {
		return fmt.Errorf("store: 仓库未初始化 sync（先 sync init）: %w", err)
	}
	w, err := repo.Worktree()
	if err != nil {
		return err
	}
	for _, path := range SyncWhitelist {
		if _, err := os.Stat(filepath.Join(r.Root, path)); err == nil {
			if _, err := w.Add(path); err != nil {
				return fmt.Errorf("store: add %s 失败: %w", path, err)
			}
		}
	}
	_, err = w.Commit("cfg4ai sync "+time.Now().UTC().Format(time.DateTime), &git.CommitOptions{
		Author: &object.Signature{Name: "cfg4ai", Email: "cfg4ai@localhost", When: time.Now()},
	})
	if err != nil && err != git.ErrEmptyCommit {
		return fmt.Errorf("store: commit 失败: %w", err)
	}
	if err := repo.Push(&git.PushOptions{}); err != nil {
		if err == git.NoErrAlreadyUpToDate {
			return nil
		}
		return fmt.Errorf("store: push 失败: %w", err)
	}
	return nil
}

// preflightScan 扫描白名单范围内全部文件内容（含自由文本）。
func (r *Repo) preflightScan(scanner *secrets.Scanner) []secrets.ScanMatch {
	if scanner == nil {
		scanner = secrets.DefaultScanner()
	}
	var matches []secrets.ScanMatch
	for _, wl := range SyncWhitelist {
		root := filepath.Join(r.Root, wl)
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			matches = append(matches, scanner.Scan(string(data), secrets.FieldFreeText)...)
			return nil
		})
	}
	return matches
}

// SyncPull 拉取远端（冲突检测：返回是否冲突）。
func (r *Repo) SyncPull() (bool, error) {
	repo, err := git.PlainOpen(r.Root)
	if err != nil {
		return false, fmt.Errorf("store: 仓库未初始化 sync: %w", err)
	}
	w, err := repo.Worktree()
	if err != nil {
		return false, err
	}
	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err == git.NoErrAlreadyUpToDate {
		return false, nil
	}
	if err != nil {
		if strings.Contains(err.Error(), "conflict") || strings.Contains(err.Error(), "non-fast-forward") {
			return true, fmt.Errorf("store: pull 冲突（请按标准 git 流程处理后重试）: %w", err)
		}
		return false, err
	}
	return false, nil
}

// SyncStatus 返回 sync 状态（是否初始化、远端 URL）。
func (r *Repo) SyncStatus() (string, error) {
	repo, err := git.PlainOpen(r.Root)
	if err != nil {
		return "未初始化（sync init <remote>）", nil
	}
	remote, err := repo.Remote("origin")
	if err != nil {
		return "已初始化（无 origin）", nil
	}
	return "origin: " + strings.Join(remote.Config().URLs, ","), nil
}
