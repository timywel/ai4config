package aiassist

import (
	"regexp"

	"github.com/timywel/ai4config/internal/core/secrets"
)

// 出域脱敏（ARCHITECTURE §5.2）：发送给 AI 前剥离 secret 与内网地址。

// internalAddrRe 内网地址（私有 IP 段 + 常见内网域名模式）。
var internalAddrRe = regexp.MustCompile(
	`\b(10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3}|127\.\d{1,3}\.\d{1,3}\.\d{1,3})` +
		`(:[0-9]{2,5})?\b|\b[\w.-]+\.(internal|local|lan|corp|intranet)\b`)

// Sanitizer AI 出域脱敏器。
type Sanitizer struct {
	scanner  *secrets.Scanner
	customRe []*regexp.Regexp // 可配置正则（企业自定义脱敏）
}

// NewSanitizer 构造脱敏器（复用 secrets.Scanner 的敏感规则）。
func NewSanitizer(scanner *secrets.Scanner, customPatterns []string) (*Sanitizer, error) {
	s := &Sanitizer{scanner: scanner}
	for _, p := range customPatterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		s.customRe = append(s.customRe, re)
	}
	return s, nil
}

// Sanitize 脱敏文本：secret → 占位、内网地址 → 占位、自定义正则 → 占位。
func (s *Sanitizer) Sanitize(text string) string {
	out := text
	// 1. 敏感 token（规则级替换为占位）
	out = redactSecrets(out)
	// 2. 内网地址
	out = internalAddrRe.ReplaceAllString(out, "[internal-addr]")
	// 3. 自定义正则
	for _, re := range s.customRe {
		out = re.ReplaceAllString(out, "[redacted]")
	}
	return out
}

// redactSecrets 把命中的敏感值替换为占位（保留结构）。
func redactSecrets(text string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`),
		regexp.MustCompile(`sk-proj-[A-Za-z0-9_-]{20,}`),
		regexp.MustCompile(`ghp_[A-Za-z0-9]{30,}`),
		regexp.MustCompile(`gho_[A-Za-z0-9]{30,}`),
		regexp.MustCompile(`github_pat_[A-Za-z0-9_]{22,}`),
		regexp.MustCompile(`glpat-[A-Za-z0-9_-]{20,}`),
		regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),
		regexp.MustCompile(`AIza[0-9A-Za-z_-]{35}`),
		regexp.MustCompile(`hf_[A-Za-z0-9]{30,}`),
		regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`),
	}
	out := text
	for _, re := range patterns {
		out = re.ReplaceAllString(out, "[secret]")
	}
	return out
}
