package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
)

// 探测目标（ADAPTERS §3.1）：
//   全局：~/.claude/（settings.json、CLAUDE.md、agents/、commands/、skills/、rules/）+ ~/.claude.json
//   项目：<proj>/CLAUDE.md、<proj>/.claude/、<proj>/.mcp.json（自 cwd 向上至 git root）
//   managed：只读（不物化）

// Detect 只读探测全部配置位置（不得创建/修改任何文件）。
func (a *adapter) Detect(ctx context.Context) ([]adapters.Location, error) {
	var locs []adapters.Location
	running := detectRunning()

	// 全局：~/.claude 目录存在即有全局配置
	if home, err := paths.Home(); err == nil {
		globalDir := filepath.Join(home, ".claude")
		if isDir(globalDir) || isFile(filepath.Join(home, ".claude.json")) {
			locs = append(locs, adapters.Location{
				Scope:   ir.ScopeGlobal,
				Root:    globalDir,
				Version: detectVersion(),
				Running: running,
			})
		}
	}

	// 项目：自当前目录向上找项目根（含 .claude/、CLAUDE.md 或 .mcp.json）
	if projRoot := findProjectRoot("."); projRoot != "" {
		locs = append(locs, adapters.Location{
			Scope:   ir.ScopeProject,
			Root:    projRoot,
			Version: detectVersion(),
			Running: running,
		})
	}

	// managed 层（只读，不参与物化）
	if md := managedDir(); md != "" && isDir(md) {
		locs = append(locs, adapters.Location{
			Scope:   ir.ScopeManaged,
			Root:    md,
			Version: detectVersion(),
			Running: running,
		})
	}

	return locs, nil
}

// isDir / isFile 只读探测。
func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func isFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// findProjectRoot 自 dir 向上查找项目根（含 .claude/、CLAUDE.md、.mcp.json 或 .git 者）。
func findProjectRoot(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		if isDir(filepath.Join(abs, ".claude")) ||
			isFile(filepath.Join(abs, "CLAUDE.md")) ||
			isFile(filepath.Join(abs, ".mcp.json")) ||
			isDir(filepath.Join(abs, ".git")) {
			return abs
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return ""
		}
		abs = parent
	}
}

// managedDir 返回企业管理层的配置目录（只读）。
func managedDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("ProgramFiles"), "ClaudeCode")
	case "darwin":
		return "/Library/Application Support/ClaudeCode"
	default:
		return "/etc/claude-code"
	}
}

// detectVersion best-effort 探测 claude CLI 版本（取不到返回空）。
func detectVersion() string {
	// 说明：不执行外部命令探测版本（避免 Detect 副作用）；版本护栏在 Import 时按文件结构判定。
	// 若未来需要真实版本，可在 e2e 层以 `claude --version` 探测。
	return ""
}
