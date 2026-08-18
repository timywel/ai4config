package aiassist

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// mock OpenAI 兼容端点。
func mockServer(t *testing.T, reply string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []any{map[string]any{"message": map[string]string{"content": reply}}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestProviderChat(t *testing.T) {
	srv := mockServer(t, "改写结果")
	defer srv.Close()
	p := &OpenAIProvider{BaseURL: srv.URL, Model: "test"}
	out, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if out != "改写结果" {
		t.Errorf("响应错误: %q", out)
	}
}

func TestConsentStateMachine(t *testing.T) {
	root := t.TempDir()
	cfg := AIConfig{Provider: "openai", BaseURL: "http://localhost:1234", Model: "m1"}

	// 首次 → NeedFirst
	if s := CheckConsent(root, cfg); s != ConsentNeedFirst {
		t.Errorf("首次应 NeedFirst，实际 %v", s)
	}
	// 同意后 → OK
	if err := RecordConsent(root, cfg); err != nil {
		t.Fatalf("RecordConsent: %v", err)
	}
	if s := CheckConsent(root, cfg); s != ConsentOK {
		t.Errorf("同意后应 OK，实际 %v", s)
	}
	// 配置段变更（换端点）→ NeedReconfirm（红队 T-09）
	changed := AIConfig{Provider: "openai", BaseURL: "http://evil.example.com", Model: "m1"}
	if s := CheckConsent(root, changed); s != ConsentNeedReconfirm {
		t.Errorf("端点变更应 NeedReconfirm，实际 %v", s)
	}
	// 仅 model 变更也应重确认
	changed2 := AIConfig{Provider: "openai", BaseURL: "http://localhost:1234", Model: "m2"}
	if s := CheckConsent(root, changed2); s != ConsentNeedReconfirm {
		t.Errorf("model 变更应 NeedReconfirm，实际 %v", s)
	}
}

func TestSanitize(t *testing.T) {
	san, _ := NewSanitizer(nil, nil)
	cases := []struct{ in, wantNot string }{
		{"使用 key sk-abcdefghij1234567890abcdef 调用", "sk-abcdefghij1234567890abcdef"},
		{"内网 http://192.168.1.10:8080/api 服务", "192.168.1.10"},
		{"数据库 10.0.0.5 连接", "10.0.0.5"},
		{"内网域名 api.corp 内部", "api.corp"},
	}
	for _, tc := range cases {
		out := san.Sanitize(tc.in)
		if strings.Contains(out, tc.wantNot) {
			t.Errorf("脱敏后仍含 %q: %q", tc.wantNot, out)
		}
	}
}

func TestClientRewriteAndLog(t *testing.T) {
	srv := mockServer(t, "改写后内容")
	defer srv.Close()
	p := &OpenAIProvider{BaseURL: srv.URL, Model: "test"}
	root := t.TempDir()
	c, err := NewClient(p, root)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	out, err := c.Rewrite(context.Background(), "原始 skill 内容 sk-abcdefghij1234567890abcdef", "copilot", "skill")
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	if out != "改写后内容" {
		t.Errorf("Rewrite 响应错误: %q", out)
	}
	// 决策日志已写
	if _, err := os.Stat(c.logPath); err != nil {
		t.Errorf("决策日志应生成: %v", err)
	}
	// 日志不含原文（只记元数据）
	logData, _ := os.ReadFile(c.logPath)
	if strings.Contains(string(logData), "原始 skill 内容") {
		t.Error("决策日志不应含 prompt 原文")
	}
}
