// Package paths 跨平台路径抽象（ARCHITECTURE §8）。
// 统一封装 XDG / %APPDATA% / ~/Library/Application Support 差异，
// 适配器只声明相对路径，平台分支集中在本包（_windows.go/_unix.go 实现）。
package paths

import (
	"os"
	"path/filepath"
)

// Home 返回用户主目录。
func Home() (string, error) {
	return os.UserHomeDir()
}

// DataHome 返回 cfg4ai 仓库（SSOT）根目录：
// CFG4AI_HOME 环境变量优先，否则平台默认（见 ARCHITECTURE §7）。
// Windows：配置/元数据 %APPDATA%，大体积内容（快照/blobs/缓存）%LOCALAPPDATA%（见 CacheHome）。
func DataHome() (string, error) {
	if v := os.Getenv("CFG4AI_HOME"); v != "" {
		return v, nil
	}
	base, err := configBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cfg4ai"), nil
}

// CacheHome 返回大体积内容根目录（Windows 为 %LOCALAPPDATA%\cfg4ai）。
func CacheHome() (string, error) {
	base, err := cacheBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cfg4ai"), nil
}

// ExpandRaw / CollapseRaw 的路径变量双向展开见 paths_expand.go。
