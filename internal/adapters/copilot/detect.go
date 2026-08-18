package copilot

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

// 探测目标（ADAPTERS §3.3）：
//   全局（user profile）：<Code User>/mcp.json、settings.json、instructions/prompts 目录
//   项目：<proj>/.github/copilot-instructions.md、instructions/、prompts/、agents/、.vscode/mcp.json、settings.json

// Detect 只读探测全部配置位置。
func (a *adapter) Detect(ctx context.Context) ([]adapters.Location, error) {
	var locs []adapters.Location

	// 全局：VS Code user profile 目录
	if up := userProfileDir(); up != "" && isDir(up) {
		locs = append(locs, adapters.Location{
			Scope:   ir.ScopeGlobal,
			Root:    up,
			Running: detectRunning(),
		})
	}

	// 项目：向上找含 .github/ 或 .vscode/ 的项目根
	if projRoot := findProjectRoot("."); projRoot != "" {
		locs = append(locs, adapters.Location{
			Scope:   ir.ScopeProject,
			Root:    projRoot,
			Running: detectRunning(),
		})
	}

	return locs, nil
}

// userProfileDir 返回 VS Code user profile 目录（跨平台）。
func userProfileDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Code", "User")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Code", "User")
	default:
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "Code", "User")
		}
		return filepath.Join(home, ".config", "Code", "User")
	}
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func isFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// findProjectRoot 自 dir 向上查找项目根（含 .github/、.vscode/ 或 .git）。
func findProjectRoot(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		if isDir(filepath.Join(abs, ".github")) ||
			isDir(filepath.Join(abs, ".vscode")) ||
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
