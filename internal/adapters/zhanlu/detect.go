package zhanlu

import (
	"context"
	"os"
	"path/filepath"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
)

// 探测目标（ADAPTERS §3.4）：
//   全局：~/.config/zhanlu/（zhanlu.json）+ ~/.agents/skills/
//   项目：<proj>/AGENTS.md、.zhanlu/、.kilo/（agent/command）、kilo.json

// Detect 只读探测全部配置位置（防御式：任一缺失不阻塞）。
func (a *adapter) Detect(ctx context.Context) ([]adapters.Location, error) {
	var locs []adapters.Location

	// 全局：zhanlu 配置目录
	if gd := globalDir(); gd != "" && (isDir(gd) || isFile(filepath.Join(gd, "zhanlu.json")) || isFile(filepath.Join(gd, "zhanlu.jsonc"))) {
		locs = append(locs, adapters.Location{Scope: ir.ScopeGlobal, Root: gd, Running: detectRunning()})
	}

	// 项目：向上找含 AGENTS.md/.kilo/.zhanlu/.git 的根
	if projRoot := findProjectRoot("."); projRoot != "" {
		locs = append(locs, adapters.Location{Scope: ir.ScopeProject, Root: projRoot, Running: detectRunning()})
	}

	return locs, nil
}

// globalDir 返回 zhanlu 全局配置目录（~/.config/zhanlu）。
func globalDir() string {
	home, err := paths.Home()
	if err != nil {
		return ""
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "zhanlu")
	}
	return filepath.Join(home, ".config", "zhanlu")
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
		if isFile(filepath.Join(abs, "AGENTS.md")) ||
			isDir(filepath.Join(abs, ".kilo")) ||
			isDir(filepath.Join(abs, ".zhanlu")) ||
			isFile(filepath.Join(abs, "kilo.json")) ||
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
