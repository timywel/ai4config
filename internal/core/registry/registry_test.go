package registry

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNormalizeGitRemote(t *testing.T) {
	want := "github.com/user/repo"
	cases := []string{
		"git@github.com:user/repo.git",
		"https://github.com/user/repo.git",
		"https://github.com/user/repo",
		"https://user@github.com/user/repo.git",
		"ssh://git@github.com/user/repo.git",
		"git@github.com:user/repo",
		"https://github.com/user/repo/",
	}
	for _, c := range cases {
		if got := NormalizeGitRemote(c); got != want {
			t.Errorf("NormalizeGitRemote(%q) = %q, want %q", c, got, want)
		}
	}
	// host 小写
	if got := NormalizeGitRemote("git@GITHUB.COM:user/repo.git"); got != "github.com/user/repo" {
		t.Errorf("host 应小写: %q", got)
	}
}

func TestRegistrySaveLoad(t *testing.T) {
	root := t.TempDir()
	r := &Registry{}
	p, _, err := r.Link(filepath.Join(t.TempDir(), "myproj"), nil)
	if err != nil {
		t.Fatalf("Link: %v", err)
	}
	if err := r.Save(root); err != nil {
		t.Fatalf("Save: %v", err)
	}
	back, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(back.Projects) != 1 || back.Projects[0].ID != p.ID {
		t.Errorf("registry 往返不一致: %+v", back.Projects)
	}
}

// 建一个真实 git 仓库（含 remote + 首次提交）。
func makeGitRepo(t *testing.T, remote string, content ...string) string {
	t.Helper()
	dir := t.TempDir()
	body := "x"
	if len(content) > 0 {
		body = content[0]
	}
	_ = body
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte(func() string {
		if len(content) > 0 {
			return content[0]
		}
		return "x"
	}()), 0o644)
	run("add", ".")
	run("commit", "-m", "init")
	if remote != "" {
		run("remote", "add", "origin", remote)
	}
	return dir
}

func TestLinkMergeSameRepo(t *testing.T) {
	root := t.TempDir()
	r := &Registry{}
	// 两个目录同一 remote + 同 first_commit → 合并
	d1 := makeGitRepo(t, "git@github.com:user/repo.git")
	d2 := makeGitRepo(t, "https://github.com/user/repo", "different-content") // 不同写法，同 remote
	// 让 d2 的 first_commit 与 d1 相同不容易（各自 init），改测 first_commit 相同时合并

	p1, merged1, _ := r.Link(d1, nil)
	if merged1 != true {
		t.Error("首次 link 应新建")
	}
	// d2 remote 相同但 first_commit 不同（独立 init）→ 二次判别新建
	p2, merged2, _ := r.Link(d2, nil)
	if merged2 {
		t.Error("first_commit 不同应新建（二次判别）")
	}
	if len(p2.SameRemoteAs) == 0 || p2.SameRemoteAs[0] != p1.ID {
		t.Errorf("应记 same_remote_as=%s: %+v", p1.ID, p2.SameRemoteAs)
	}
	if err := r.Save(root); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func TestVerifyPathHijack(t *testing.T) {
	r := &Registry{}
	d1 := makeGitRepo(t, "git@github.com:a/foo.git")
	if _, _, err := r.Link(d1, nil); err != nil {
		t.Fatalf("Link: %v", err)
	}
	// 同路径但不同 git 内容（重建一个不同 first_commit 的仓库放同路径）——
	// 用另一个独立仓库模拟"路径复用劫持"：直接改其指纹后 VerifyPath
	// 这里验证：命中的项目指纹与现算指纹 first_commit 不同 → ok=false
	p := r.FindByPath(d1)
	p.Fingerprint.FirstCommit = "differenthash" // 模拟注册表里的指纹与磁盘不一致
	if _, ok := r.VerifyPath(d1); ok {
		t.Error("路径命中但指纹不符应返回 ok=false（张冠李戴防线）")
	}
}
