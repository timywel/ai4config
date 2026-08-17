package secrets

import "strings"

// 回采保护（IR-SCHEMA §1.1/§3.6；红队 T-03 修复）：
// 导出物中的 secretref 占位符或空值被再次采集时，永不覆盖已有 secretref——
// 否则换机后首次采集会用空占位符把 keyring 里的真实引用"冲掉"。

const refPrefix = "secretref://cfg4ai/"

// MakeRef 构造 secretref（IR-SCHEMA §3.6 格式：secretref://cfg4ai/<profile>/<entity-id>/<field>）。
func MakeRef(profile, entityID, field string) string {
	return refPrefix + profile + "/" + entityID + "/" + field
}

// IsSecretRef 判断是否为 secretref 占位符。
func IsSecretRef(v string) bool { return strings.HasPrefix(v, refPrefix) }

// IsPlaceholder 判断是否为"空值/占位"（不应覆盖真实引用的值）。
// 覆盖：空串、secretref 自身、常见占位文本。
func IsPlaceholder(v string) bool {
	if IsSecretRef(v) {
		return true
	}
	t := strings.TrimSpace(v)
	if t == "" {
		return true
	}
	switch strings.ToLower(t) {
	case "placeholder", "your-key-here", "your_api_key", "changeme", "todo", "xxx", "***", "<redacted>":
		return true
	}
	return false
}

// ShouldPreserveExisting 回采保护判定：
// 已有值是 secretref，而新采集值是空/占位 → 保留已有 secretref（不覆盖）。
// 其余情况（新值是真实明文、或已有值非 secretref）→ 正常采用新值。
func ShouldPreserveExisting(existing, incoming string) bool {
	if !IsSecretRef(existing) {
		return false // 已有值不是 secretref，正常覆盖
	}
	return IsPlaceholder(incoming) // 已有是 secretref：仅当新值是占位时保留
}
