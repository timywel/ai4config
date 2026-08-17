package paths

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// 路径变量形态的双向转换（ARCHITECTURE §8、IR-SCHEMA §1.1 origin.path raw 形态）。
//
// 设计意图：origin.path 以"变量形态"记录（如 %APPDATA%/Claude/...），
// 使 sync 到异平台机器后仍可还原为当地等价路径。
//   - ExpandRaw：变量形态 → 当前平台绝对路径（识别全平台变量写法）
//   - CollapseRaw：绝对路径 → 变量形态（当前平台优先，最长前缀匹配）

// variable 一个可识别的路径前缀变量。
type variable struct {
	token   string                 // 变量形态（如 %APPDATA%、$XDG_CONFIG_HOME、~）
	resolve func() (string, error) // 展开为当前平台绝对路径
}

// knownVariables 所有可识别的变量形态（跨平台来源都要能展开）。
func knownVariables() []variable {
	return []variable{
		{"%APPDATA%", configBaseDir},
		{"%LOCALAPPDATA%", cacheBaseDir},
		{"$XDG_CONFIG_HOME", configBaseDir},
		{"$XDG_CACHE_HOME", cacheBaseDir},
		{"$HOME", Home},
		{"~", Home},
	}
}

// ExpandRaw 把变量形态路径展开为当前平台绝对路径。
// 无已知变量前缀时原样返回（视为已绝对/相对路径）。
func ExpandRaw(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("paths: 空路径")
	}
	norm := filepath.ToSlash(raw)
	for _, v := range knownVariables() {
		if norm == v.token {
			return v.resolve()
		}
		if strings.HasPrefix(norm, v.token+"/") {
			base, err := v.resolve()
			if err != nil {
				return "", err
			}
			return filepath.Join(base, strings.TrimPrefix(norm, v.token+"/")), nil
		}
	}
	return raw, nil
}

// CollapseRaw 把绝对路径折叠为变量形态（当前平台规范写法）。
// 无法折叠时返回斜杠规范化的原路径。
func CollapseRaw(abs string) (string, error) {
	norm := filepath.ToSlash(abs)
	type cand struct{ prefix, token string }
	var cands []cand
	if cfg, err := configBaseDir(); err == nil {
		cands = append(cands, cand{filepath.ToSlash(cfg), configToken()})
	}
	if ch, err := cacheBaseDir(); err == nil {
		cands = append(cands, cand{filepath.ToSlash(ch), cacheToken()})
	}
	if home, err := Home(); err == nil {
		cands = append(cands, cand{filepath.ToSlash(home), "~"})
	}
	// 最长前缀优先（config/cache 是 home 子目录，须先匹配）
	sort.Slice(cands, func(i, j int) bool { return len(cands[i].prefix) > len(cands[j].prefix) })
	for _, c := range cands {
		if norm == c.prefix {
			return c.token, nil
		}
		if strings.HasPrefix(norm, c.prefix+"/") {
			return c.token + "/" + strings.TrimPrefix(norm, c.prefix+"/"), nil
		}
	}
	return norm, nil
}

// configToken 当前平台记录 config 目录用的规范变量形态。
func configToken() string {
	switch runtime.GOOS {
	case "windows":
		return "%APPDATA%"
	case "darwin":
		return "~/Library/Application Support"
	default:
		return "$XDG_CONFIG_HOME"
	}
}

// cacheToken 当前平台记录 cache 目录用的规范变量形态。
func cacheToken() string {
	switch runtime.GOOS {
	case "windows":
		return "%LOCALAPPDATA%"
	case "darwin":
		return "~/Library/Caches"
	default:
		return "$XDG_CACHE_HOME"
	}
}
