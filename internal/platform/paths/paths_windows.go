//go:build windows

package paths

import (
	"os"
	"path/filepath"
)

// configBaseDir Windows：%APPDATA%（Roaming，配置与元数据）。
// 环境变量缺失（病态环境）时回退 UserHomeDir，避免拼出 \cfg4ai 相对根路径。
func configBaseDir() (string, error) {
	if v := os.Getenv("APPDATA"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "AppData", "Roaming"), nil
}

// cacheBaseDir Windows：%LOCALAPPDATA%（快照/blobs/缓存不随域漫游），同样带回退。
func cacheBaseDir() (string, error) {
	if v := os.Getenv("LOCALAPPDATA"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "AppData", "Local"), nil
}
