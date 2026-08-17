//go:build !windows

package paths

import (
	"os"
	"path/filepath"
)

// configBaseDir Unix：$XDG_CONFIG_HOME（默认 ~/.config）；
// macOS 为 ~/Library/Application Support。
func configBaseDir() (string, error) {
	if isDarwin() {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support"), nil
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

// cacheBaseDir Unix：$XDG_CACHE_HOME（默认 ~/.cache）；macOS 为 ~/Library/Caches。
func cacheBaseDir() (string, error) {
	if isDarwin() {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Caches"), nil
	}
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache"), nil
}
