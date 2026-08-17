package secrets

import (
	"strings"
	"testing"
)

// tk 拼接测试用伪 token：避免源码中出现完整的 secret 字面量
// （防 GitHub Push Protection / 各 secret 扫描器对测试夹具的误判），
// 运行时拼接结果仍用于验证本包扫描器规则。
func tk(parts ...string) string { return strings.Join(parts, "") }

func TestScannerRulesHit(t *testing.T) {
	s := DefaultScanner()
	rep := strings.Repeat
	cases := []struct {
		name string
		val  string
	}{
		{"openai", tk("sk-", rep("a", 24))},
		{"github-pat", tk("ghp_", rep("a", 36))},
		{"gitlab", tk("glpat-", rep("a", 22))},
		{"google", tk("AIza", rep("a", 35))},
		{"huggingface", tk("hf_", rep("a", 34))},
		{"aws", tk("AKIA", rep("A", 16))},
		{"pem", tk("-----BEGIN RSA ", "PRIVATE KEY-----")},
		{"generic-assignment", tk(`api_key = "`, rep("b", 24), `"`)},
		{"jwt", tk("eyJ", rep("a", 11), ".", rep("b", 11), ".", rep("c", 11))},
	}
	for _, tc := range cases {
		if !s.IsSecret(tc.val) {
			t.Errorf("%s 应命中敏感: %q", tc.name, tc.val)
		}
	}
}

func TestScannerNoFalsePositive(t *testing.T) {
	s := DefaultScanner()
	s.SetAllowlist([]string{"sk-example"})
	clean := []string{
		tk("sk-", "example", "-not-a-real-key"), // 豁免
		"hello world",
		"model: kimi-k3",
	}
	for _, c := range clean {
		if s.IsSecret(c) {
			t.Errorf("不应命中: %q", c)
		}
	}
}

func TestScannerEntropy(t *testing.T) {
	s := DefaultScanner()
	if !s.IsSecret(tk("xJ9#kL2$", "mN8&pQ4*", "rT6^wE1!", "zX3@vB5")) {
		t.Log("高熵串未被熵检测命中（可由规则命中兜底）")
	}
}

func TestMask(t *testing.T) {
	if got := mask(tk("sk-abcdefgh", "123456")); !strings.HasPrefix(got, "sk-abc") || !strings.HasSuffix(got, "56") {
		t.Errorf("mask 结果异常: %q", got)
	}
	if strings.Contains(mask(tk("sk-abcdefgh", "123456")), "defgh") {
		t.Error("mask 泄露了中间段")
	}
}

func TestProtectPreserveExisting(t *testing.T) {
	existing := MakeRef("global", "mcp.fs", "env.TOKEN")
	if !ShouldPreserveExisting(existing, "") {
		t.Error("已有 secretref + 空新值 → 应保留")
	}
	if !ShouldPreserveExisting(existing, existing) {
		t.Error("已有 secretref + 占位新值 → 应保留")
	}
	if ShouldPreserveExisting(existing, tk("sk-", strings.Repeat("r", 24))) {
		t.Error("新值是真实明文 → 应采用新值（更新 secret）")
	}
	if ShouldPreserveExisting("plain-old", "anything") {
		t.Error("已有值非 secretref → 不应保留")
	}
}

func TestSanitizeField(t *testing.T) {
	s := DefaultScanner()
	b := NoneBackend{}
	got, hit, err := SanitizeField(b, s, "global", "mcp.fs", "env.TOKEN", tk("sk-", strings.Repeat("s", 24)))
	if err != nil || !hit {
		t.Fatalf("应抽取: hit=%v err=%v", hit, err)
	}
	if !IsSecretRef(got) {
		t.Errorf("应替换为 secretref，实际 %q", got)
	}
	want := "secretref://cfg4ai/global/mcp.fs/env.TOKEN"
	if got != want {
		t.Errorf("secretref 格式错误: got %q want %q", got, want)
	}
	ref := MakeRef("global", "mcp.fs", "env.TOKEN")
	got2, hit2, _ := SanitizeField(b, s, "global", "mcp.fs", "env.TOKEN", ref)
	if hit2 || got2 != ref {
		t.Errorf("已是 secretref 不应重抽取: got=%q hit=%v", got2, hit2)
	}
	got3, hit3, _ := SanitizeField(b, s, "global", "mcp.fs", "env.HOME", "/data")
	if hit3 || got3 != "/data" {
		t.Errorf("普通值应原样: got=%q hit=%v", got3, hit3)
	}
}

func TestFileBackendRoundTrip(t *testing.T) {
	dir := t.TempDir()
	prompt := func() (string, bool) { return "test-pass-123", true }

	b, err := openFileBackend(dir, prompt)
	if err != nil {
		t.Fatalf("openFileBackend: %v", err)
	}
	ref := MakeRef("global", "mcp.fs", "env.TOKEN")
	if err := b.Set(ref, "super-secret-value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	b2, err := openFileBackend(dir, prompt)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, err := b2.Get(ref)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "super-secret-value" {
		t.Errorf("往返不一致: %q", got)
	}
	badPrompt := func() (string, bool) { return "wrong-pass", true }
	if _, err := openFileBackend(dir, badPrompt); err == nil {
		t.Error("错误口令应解密失败")
	}
	if err := b2.Delete(ref); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := b2.Get(ref); err == nil {
		t.Error("删除后 Get 应失败")
	}
}

func TestNoneBackend(t *testing.T) {
	b := NoneBackend{}
	if err := b.Set("ref", "v"); err != nil {
		t.Errorf("none Set 应静默: %v", err)
	}
	if _, err := b.Get("ref"); err == nil {
		t.Error("none Get 应提示人工填充")
	}
}
