package secrets

import (
	"math"
	"regexp"
	"strings"
)

// FieldKind 命中位置的字段类别（决定处置级别，IR-SCHEMA §5 规则 4）。
type FieldKind int

const (
	FieldStructured FieldKind = iota // 结构化字段（env/headers/args/value）→ 默认抽取（可否决）
	FieldFreeText                    // 自由文本（instruction/prompt 正文）→ 仅 Warning，绝不自动改写
)

// ScanMatch 一次命中。
type ScanMatch struct {
	Rule  string // 规则名
	Kind  FieldKind
	Found string // 命中的脱敏片段（前 6 后 2，中间打码）
}

// rule 内置规则（对齐 gitleaks 常见集，REVIEW-REPORT M16）。
type rule struct {
	name string
	re   *regexp.Regexp
}

var builtinRules = []rule{
	{"openai-api-key", regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`)},
	{"openai-project-key", regexp.MustCompile(`sk-proj-[A-Za-z0-9_-]{20,}`)},
	{"github-pat", regexp.MustCompile(`ghp_[A-Za-z0-9]{30,}`)},
	{"github-oauth", regexp.MustCompile(`gho_[A-Za-z0-9]{30,}`)},
	{"github-fine-grained-pat", regexp.MustCompile(`github_pat_[A-Za-z0-9_]{22,}`)},
	{"gitlab-pat", regexp.MustCompile(`glpat-[A-Za-z0-9_-]{20,}`)},
	{"slack-token", regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`)},
	{"google-api-key", regexp.MustCompile(`AIza[0-9A-Za-z_-]{35}`)},
	{"huggingface-token", regexp.MustCompile(`hf_[A-Za-z0-9]{30,}`)},
	{"aws-access-key-id", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"jwt", regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)},
	{"private-key", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`)},
	{"generic-secret-assignment", regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password|passwd|credentials)["']?\s*[:=]\s*["']?[A-Za-z0-9/+=_-]{16,}`)},
}

// Scanner 敏感扫描器（外置规则集 + 熵检测 + 豁免清单）。
type Scanner struct {
	rules     []rule
	allowlist []string
	entropy   bool // 是否启用高熵串兜底检测
}

// DefaultScanner 内置规则 + 启用熵检测。
func DefaultScanner() *Scanner {
	return &Scanner{rules: builtinRules, entropy: true}
}

// NewScanner 自定义规则 + 豁免清单。
func NewScanner(customRules map[string]string, allowlist []string, enableEntropy bool) (*Scanner, error) {
	s := &Scanner{allowlist: allowlist, entropy: enableEntropy}
	for name, pattern := range customRules {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		s.rules = append(s.rules, rule{name: name, re: re})
	}
	return s, nil
}

// SetAllowlist 设置豁免清单（命中这些子串不报——如教学示例 token）。
func (s *Scanner) SetAllowlist(items []string) { s.allowlist = items }

// Scan 扫描文本，返回全部命中（分级由调用方按 FieldKind 决定）。
func (s *Scanner) Scan(text string, kind FieldKind) []ScanMatch {
	var out []ScanMatch
	for _, rl := range s.rules {
		for _, m := range rl.re.FindAllString(text, -1) {
			if s.allowed(m) {
				continue
			}
			out = append(out, ScanMatch{Rule: rl.name, Kind: kind, Found: mask(m)})
		}
	}
	if s.entropy {
		for _, tok := range strings.FieldsFunc(text, splitNonToken) {
			if len(tok) >= 20 && shannon(tok) >= 4.5 && !s.allowed(tok) {
				out = append(out, ScanMatch{Rule: "high-entropy", Kind: kind, Found: mask(tok)})
			}
		}
	}
	return out
}

// IsSecret 便捷判定（任一命中即 true）。
func (s *Scanner) IsSecret(text string) bool {
	return len(s.Scan(text, FieldStructured)) > 0
}

// allowed 命中是否在豁免清单。
func (s *Scanner) allowed(match string) bool {
	for _, a := range s.allowlist {
		if a != "" && strings.Contains(match, a) {
			return true
		}
	}
	return false
}

// mask 脱敏片段：前 6 后 2，中间打码。
func mask(s string) string {
	if len(s) <= 10 {
		return s[:2] + "***"
	}
	return s[:6] + "***" + s[len(s)-2:]
}

// splitNonToken 分词（取疑似 token 的连续字符段）。
func splitNonToken(r rune) bool {
	return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
		r == '-' || r == '_' || r == '/' || r == '+' || r == '=' || r == '.')
}

// shannon 香农熵（bits/char）。
func shannon(s string) float64 {
	if s == "" {
		return 0
	}
	freq := map[rune]float64{}
	for _, r := range s {
		freq[r]++
	}
	var ent float64
	n := float64(len(s))
	for _, c := range freq {
		p := c / n
		ent -= p * math.Log2(p)
	}
	return ent
}
