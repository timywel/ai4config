// Package aiassist AI 辅助迁移：Provider 接口、consent 状态机、脱敏、语义转换。
// 权威规范：docs/ARCHITECTURE.md §5.2（规则优先 AI 兜底、consent、脱敏）。
package aiassist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message 对话消息。
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Provider AI 提供商接口（可插拔；任何 OpenAI 兼容端点均可接入）。
type Provider interface {
	Chat(ctx context.Context, messages []Message) (string, error)
	Name() string
}

// OpenAIProvider OpenAI 兼容端点（/v1/chat/completions）。
type OpenAIProvider struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// Name 返回提供商标识。
func (p *OpenAIProvider) Name() string { return "openai-compatible:" + p.BaseURL }

// Chat 调用 chat completions。
func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	if p.BaseURL == "" {
		return "", fmt.Errorf("aiassist: provider base_url 未配置")
	}
	model := p.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	url := strings.TrimSuffix(p.BaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	timeout := p.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("aiassist: 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("aiassist: provider 返回 %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("aiassist: 解析响应失败: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("aiassist: 响应无 choices")
	}
	return result.Choices[0].Message.Content, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
