package secrets

import (
	"strings"
	"testing"
)

// 对抗用例回归（research/adversarial-cases.md F 类 secret 对抗）。

// AC-F5 placeholder-recollect：none 后端导出占位符 → 回采不得覆盖已有 secretref（红队 T-03 防线）
func TestAdversarial_F5_PlaceholderRecollect(t *testing.T) {
	existing := MakeRef("global", "mcp.fs", "env.TOKEN") // A 机 keyring 真值的引用
	// B 机 none 后端导出，env 值为空（"留空人工填"）
	incomingEmpty := ""
	if !ShouldPreserveExisting(existing, incomingEmpty) {
		t.Error("空值回采必须保留已有 secretref（否则引用断链→prune 级联清 keyring）")
	}
	// B 机导出占位形态
	if !ShouldPreserveExisting(existing, existing) {
		t.Error("占位符回采必须保留已有 secretref")
	}
	// 用户在 B 机手工填了真实值 → 应采用（更新 secret）
	if ShouldPreserveExisting(existing, tk("sk-", strings.Repeat("r", 24))) {
		t.Error("真实明文应采用（允许更新 secret）")
	}
}

// AC-F1 apikey-in-instruction-body：自由文本含真实 key → 仅 Warning 绝不自动改写正文
func TestAdversarial_F1_FreeTextOnlyWarns(t *testing.T) {
	s := DefaultScanner()
	body := tk("调用内部服务统一使用 `sk-", strings.Repeat("l", 32), "`（这是生产 key）")
	// 自由文本：命中但分级为 FieldFreeText
	matches := s.Scan(body, FieldFreeText)
	if len(matches) == 0 {
		t.Fatal("自由文本 key 应被扫描命中")
	}
	for _, m := range matches {
		if m.Kind != FieldFreeText {
			t.Errorf("正文命中应分级为 FieldFreeText（仅 Warning），实际 %v", m.Kind)
		}
		// 关键：脱敏片段不得含完整 key（防 Warning 日志二次泄漏）
		if strings.Contains(m.Found, strings.Repeat("l", 32)) {
			t.Error("Warning 的脱敏片段泄露了完整 key")
		}
	}
	// 对比：结构化字段命中分级为 FieldStructured（默认抽取）
	structMatches := s.Scan(tk("sk-", strings.Repeat("l", 32)), FieldStructured)
	for _, m := range structMatches {
		if m.Kind != FieldStructured {
			t.Error("结构化字段应分级为 FieldStructured（默认抽取）")
		}
	}
}
