package paths

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExpandRawHome(t *testing.T) {
	got, err := ExpandRaw("~/CLAUDE.md")
	if err != nil {
		t.Fatalf("ExpandRaw: %v", err)
	}
	home, _ := Home()
	if !strings.HasPrefix(filepath.ToSlash(got), filepath.ToSlash(home)) {
		t.Errorf("~/ 应展开为主目录，实际 %q", got)
	}
	if !strings.HasSuffix(filepath.ToSlash(got), "CLAUDE.md") {
		t.Errorf("展开后应保留文件名，实际 %q", got)
	}
}

func TestExpandRawVariables(t *testing.T) {
	cases := []string{"%APPDATA%/Claude/c.json", "$XDG_CONFIG_HOME/Claude/c.json"}
	for _, c := range cases {
		got, err := ExpandRaw(c)
		if err != nil {
			t.Fatalf("ExpandRaw(%q): %v", c, err)
		}
		// 变量前缀应被替换为实际目录
		if strings.Contains(got, "%APPDATA%") || strings.Contains(got, "$XDG_CONFIG_HOME") {
			t.Errorf("ExpandRaw(%q) 未展开变量: %q", c, got)
		}
		if !strings.HasSuffix(filepath.ToSlash(got), "Claude/c.json") {
			t.Errorf("ExpandRaw(%q) 应保留子路径: %q", c, got)
		}
	}
}

func TestExpandRawNoPrefix(t *testing.T) {
	in := "some/relative/path.md"
	got, err := ExpandRaw(in)
	if err != nil || got != in {
		t.Errorf("无变量前缀应原样返回: got=%q err=%v", got, err)
	}
}

func TestExpandRawEmpty(t *testing.T) {
	if _, err := ExpandRaw(""); err == nil {
		t.Error("空路径应报错")
	}
}

func TestCollapseRawHome(t *testing.T) {
	home, _ := Home()
	abs := filepath.Join(home, "CLAUDE.md")
	got, err := CollapseRaw(abs)
	if err != nil {
		t.Fatalf("CollapseRaw: %v", err)
	}
	if got != "~/CLAUDE.md" {
		t.Errorf("主目录下文件应折叠为 ~/ 形态，实际 %q", got)
	}
}

func TestCollapseRawConfigDir(t *testing.T) {
	cfg, err := configBaseDir()
	if err != nil {
		t.Skipf("无 configBaseDir: %v", err)
	}
	abs := filepath.Join(cfg, "Claude", "c.json")
	got, err := CollapseRaw(abs)
	if err != nil {
		t.Fatalf("CollapseRaw: %v", err)
	}
	// 应折叠为当前平台的 config 变量形态（最长前缀优先于 ~）
	var wantPrefix string
	switch runtime.GOOS {
	case "windows":
		wantPrefix = "%APPDATA%/Claude/c.json"
	case "darwin":
		wantPrefix = "~/Library/Application Support/Claude/c.json"
	default:
		wantPrefix = "$XDG_CONFIG_HOME/Claude/c.json"
	}
	if got != wantPrefix {
		t.Errorf("config 目录应折叠为 %q，实际 %q", wantPrefix, got)
	}
}

// 往返一致：CollapseRaw(ExpandRaw(v)) 与 ExpandRaw(CollapseRaw(abs))
func TestRoundTrip(t *testing.T) {
	home, _ := Home()
	abs := filepath.Join(home, "x", "y.md")
	collapsed, err := CollapseRaw(abs)
	if err != nil {
		t.Fatalf("CollapseRaw: %v", err)
	}
	if collapsed != "~/x/y.md" {
		t.Fatalf("折叠结果异常: %q", collapsed)
	}
	expanded, err := ExpandRaw(collapsed)
	if err != nil {
		t.Fatalf("ExpandRaw: %v", err)
	}
	if filepath.Clean(expanded) != filepath.Clean(abs) {
		t.Errorf("往返不一致: %q → %q → %q", abs, collapsed, expanded)
	}
}
