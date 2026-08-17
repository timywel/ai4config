package codex

import (
	"context"
	"os"
	"path/filepath"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
)

// 探测目标（ADAPTERS §3.2）：
//   全局：~/.codex/（config.toml、AGENTS.md、AGENTS.override.md、skills/、auth.json）
//   项目：<proj>/AGENTS.md（逐目录）、<proj>/.codex/config.toml（trusted-gate）

// Detect 只读探测全部配置位置。
func (a *adapter) Detect(ctx context.Context) ([]adapters.Location, error) {
	var locs []adapters.Location

	// 全局：~/.codex 目录（CODEX_HOME 可覆盖）
	globalDir := os.Getenv("CODEX_HOME")
	if globalDir == "" {
		if home, err := paths.Home(); err == nil {
			globalDir = filepath.Join(home, ".codex")
		}
	}
	if globalDir != "" && isDir(globalDir) {
		locs = append(locs, adapters.Location{
			Scope:   ir.ScopeGlobal,
			Root:    globalDir,
			Running: detectRunning(),
		})
	}

	// 项目：向上找项目根（含 AGENTS.md / AGENTS.override.md / .codex/ / .git）
	if projRoot := findProjectRoot("."); projRoot != "" {
		locs = append(locs, adapters.Location{
			Scope:   ir.ScopeProject,
			Root:    projRoot,
			Running: detectRunning(),
		})
	}

	return locs, nil
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func isFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// findProjectRoot 自 dir 向上查找项目根（含 AGENTS.md/AGENTS.override.md/.codex/.git 者）。
func findProjectRoot(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		if isFile(filepath.Join(abs, "AGENTS.override.md")) ||
			isFile(filepath.Join(abs, "AGENTS.md")) ||
			isDir(filepath.Join(abs, ".codex")) ||
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
