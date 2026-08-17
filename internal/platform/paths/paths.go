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

// ExpandRaw 把 IR 中记录的 raw 变量形态路径（~/...、%APPDATA%/...）
// 展开为当前平台绝对路径（导出/回写时调用）。
func ExpandRaw(raw string) (string, error) {
	// TODO(P0): 实现 ~/、%APPDATA%、$XDG_CONFIG_HOME 变量展开
	return raw, nil
}

// CollapseRaw 是 ExpandRaw 的逆操作：绝对路径 → raw 变量形态（采集入 origin.path 时调用）。
func CollapseRaw(abs string) (string, error) {
	// TODO(P0): 实现逆展开（origin.path 以变量形态记录，跨机可移植）
	return abs, nil
}
