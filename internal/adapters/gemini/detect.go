package gemini

import (
	"context"
	"os"
	"path/filepath"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
)

// Detect 探测 ~/.gemini（全局）与项目 .gemini/GEMINI.md。
func (a *adapter) Detect(ctx context.Context) ([]adapters.Location, error) {
	var locs []adapters.Location
	if home, err := paths.Home(); err == nil {
		gd := filepath.Join(home, ".gemini")
		if isDir(gd) {
			locs = append(locs, adapters.Location{Scope: ir.ScopeGlobal, Root: gd, Running: detectRunning()})
		}
	}
	if projRoot := findProjectRoot("."); projRoot != "" {
		locs = append(locs, adapters.Location{Scope: ir.ScopeProject, Root: projRoot, Running: detectRunning()})
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

func findProjectRoot(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		if isFile(filepath.Join(abs, "GEMINI.md")) || isDir(filepath.Join(abs, ".gemini")) || isDir(filepath.Join(abs, ".git")) {
			return abs
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return ""
		}
		abs = parent
	}
}
