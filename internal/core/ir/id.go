package ir

import (
	"fmt"
	"regexp"
	"strings"
)

// nameRe 实体 id name 段字符集（IR-SCHEMA §5 规则 1；D2 放行点号与大写）。
var nameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// ParseID 解析实体 id：首个点号分隔 type 与 name（D2）。
// 规则（IR-SCHEMA §5 规则 1）：
//   - 必须含 type 与 name 两段；
//   - type 必须是已知 EntityKind；
//   - name 匹配 [a-zA-Z0-9][a-zA-Z0-9._-]*；
//   - setting 类必须 setting.<tool>.<key> 三段式（tool 注册性由校验层规则 1 检查）。
func ParseID(id string) (kind EntityKind, name string, err error) {
	i := strings.Index(id, ".")
	if i <= 0 || i == len(id)-1 {
		return "", "", fmt.Errorf("ir: id %q 缺少 type 或 name 段", id)
	}
	kind = EntityKind(id[:i])
	name = id[i+1:]
	switch kind {
	case KindInstruction, KindMCP, KindSkill, KindAgent,
		KindCommand, KindWorkflow, KindHook, KindSetting:
	default:
		return "", "", fmt.Errorf("ir: id %q 的 type 段 %q 不是已知实体类别", id, kind)
	}
	if !nameRe.MatchString(name) {
		return "", "", fmt.Errorf("ir: id %q 的 name 段 %q 含非法字符", id, name)
	}
	if kind == KindSetting {
		rest := name[len(""):] // name = <tool>.<key>
		j := strings.Index(rest, ".")
		if j <= 0 || j == len(rest)-1 {
			return "", "", fmt.Errorf("ir: setting id %q 必须为 setting.<tool>.<key> 三段式", id)
		}
	}
	return kind, name, nil
}

// ParseSettingID 解析 setting 三段式 id，返回 tool 与 key。
func ParseSettingID(id string) (tool, key string, err error) {
	kind, name, err := ParseID(id)
	if err != nil {
		return "", "", err
	}
	if kind != KindSetting {
		return "", "", fmt.Errorf("ir: id %q 不是 setting 类", id)
	}
	j := strings.Index(name, ".")
	return name[:j], name[j+1:], nil
}

// NameTail 返回 id 的 name 段末段（用于规则 7：与目录/文件名一致性比对）。
func NameTail(id string) string {
	if i := strings.LastIndex(id, "."); i >= 0 && i < len(id)-1 {
		return id[i+1:]
	}
	return id
}
