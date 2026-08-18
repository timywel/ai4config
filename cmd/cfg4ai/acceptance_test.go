package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runCLIWithEnv 带环境变量驱动编译产物。
func runCLIWithEnv(t *testing.T, env []string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	}
	return string(out), code
}

// P0 验收主用例：Claude Code → Codex 迁移闭环（ARCHITECTURE §12 P0 验收项 1）。
func TestP0Acceptance_ClaudeToCodex(t *testing.T) {
	fakeHome := t.TempDir()
	repoHome := t.TempDir()
	codexHome := t.TempDir()

	claudeDir := filepath.Join(fakeHome, ".claude")
	os.MkdirAll(filepath.Join(claudeDir, "agents"), 0o755)
	os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# 团队规范\n\n- 所有回复使用中文\n"), 0o644)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{"model":"claude-opus"}`), 0o644)
	os.WriteFile(filepath.Join(claudeDir, "agents", "reviewer.md"), []byte("---\nname: reviewer\ndescription: 评审\n---\n评审正文\n"), 0o644)
	os.WriteFile(filepath.Join(fakeHome, ".claude.json"), []byte(`{"mcpServers":{"fs":{"command":"npx","args":["-y","fs"]}}}`), 0o644)

	env := []string{"USERPROFILE=" + fakeHome, "HOME=" + fakeHome, "CODEX_HOME=" + codexHome}

	// migrate = collect(claude-code) + export(codex --include-foreign)
	out, code := runCLIWithEnv(t, env, "--home", repoHome, "migrate", "--from", "claude-code", "--to", "codex")
	t.Logf("migrate: code=%d\n%s", code, out)
	if code != 0 && code != 5 {
		t.Fatalf("migrate 退出码异常: %d\n%s", code, out)
	}

	// 验收 1a：MCP 互转——codex config.toml 含 mcp_servers.fs
	configData, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		t.Fatalf("codex config.toml 未生成: %v", err)
	}
	if !strings.Contains(string(configData), "mcp_servers") || !strings.Contains(string(configData), "fs") {
		t.Errorf("config.toml 应含 mcp_servers.fs:\n%s", configData)
	}

	// 验收 1b：指令互转——codex AGENTS.md 含迁移的指令内容
	agentsData, err := os.ReadFile(filepath.Join(codexHome, "AGENTS.md"))
	if err != nil {
		t.Fatalf("codex AGENTS.md 未生成: %v", err)
	}
	if !strings.Contains(string(agentsData), "中文") {
		t.Errorf("AGENTS.md 应含迁移的指令内容:\n%s", agentsData)
	}

	// 验收 1c：agent 物化
	if _, err := os.Stat(filepath.Join(codexHome, "agents", "reviewer.md")); err != nil {
		t.Errorf("codex agents/reviewer.md 未生成: %v", err)
	}
}
