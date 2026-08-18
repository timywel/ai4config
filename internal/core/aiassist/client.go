package aiassist

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/timywel/ai4config/internal/atomicfile"
)

// Client AI 辅助客户端（引擎层 Assist 步骤调用）。
// 职责：consent 校验 + 脱敏 + 调用 provider + 决策日志。
type Client struct {
	provider  Provider
	sanitizer *Sanitizer
	repoRoot  string
	logPath   string
}

// NewClient 构造客户端。repoRoot 用于 consent 与决策日志。
func NewClient(provider Provider, repoRoot string) (*Client, error) {
	san, err := NewSanitizer(nil, nil)
	if err != nil {
		return nil, err
	}
	return &Client{
		provider:  provider,
		sanitizer: san,
		repoRoot:  repoRoot,
		logPath:   filepath.Join(repoRoot, "logs", "ai-decisions.yaml"),
	}, nil
}

// SetCustomPatterns 设置自定义脱敏正则。
func (c *Client) SetCustomPatterns(patterns []string) error {
	san, err := NewSanitizer(nil, patterns)
	if err != nil {
		return err
	}
	c.sanitizer = san
	return nil
}

// Rewrite 语义改写（如 skill 描述 → 目标工具 prompt 风格）。
func (c *Client) Rewrite(ctx context.Context, content, targetTool, entityKind string) (string, error) {
	prompt := fmt.Sprintf("你是 AI 编码工具配置迁移助手。把下面这段 %s 配置改写为适合 %s 的风格与惯例。保持语义不变，只调整表达风格与结构。直接输出改写后的内容，不要解释。\n\n---\n%s", entityKind, targetTool, c.sanitizer.Sanitize(content))
	return c.chat(ctx, "rewrite", prompt)
}

// SuggestMerge 冲突合并建议。
func (c *Client) SuggestMerge(ctx context.Context, existing, incoming string) (string, error) {
	prompt := fmt.Sprintf("合并以下两段 AI 工具配置，消除冲突，保留双方有效信息。直接输出合并结果：\n\n[已有]\n%s\n\n[新]\n%s", c.sanitizer.Sanitize(existing), c.sanitizer.Sanitize(incoming))
	return c.chat(ctx, "merge", prompt)
}

// Translate 语言适配（中英文指令互译）。
func (c *Client) Translate(ctx context.Context, content, targetLang string) (string, error) {
	prompt := fmt.Sprintf("把下面的 AI 工具指令翻译为%s，保持所有技术术语、命令、代码、路径原文不动：\n\n%s", targetLang, c.sanitizer.Sanitize(content))
	return c.chat(ctx, "translate", prompt)
}

// chat 统一调用 + 决策日志。
func (c *Client) chat(ctx context.Context, op, userPrompt string) (string, error) {
	messages := []Message{{Role: "user", Content: userPrompt}}
	out, err := c.provider.Chat(ctx, messages)
	c.logDecision(op, len(userPrompt), err)
	if err != nil {
		return "", err
	}
	return out, nil
}

// decisionEntry 决策日志条目（默认只记元数据，不记原文——ARCHITECTURE §5.2）。
type decisionEntry struct {
	Time      time.Time `yaml:"time"`
	Op        string    `yaml:"op"`
	Provider  string    `yaml:"provider"`
	PromptLen int       `yaml:"prompt_len"`
	Error     string    `yaml:"error,omitempty"`
}

// logDecision 追加决策日志（仅元数据）。
func (c *Client) logDecision(op string, promptLen int, err error) {
	entry := decisionEntry{
		Time:      time.Now().UTC(),
		Op:        op,
		Provider:  c.provider.Name(),
		PromptLen: promptLen,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	// 追加写（读取既有 + 追加条目）
	var all []decisionEntry
	if existing, rErr := os.ReadFile(c.logPath); rErr == nil {
		_ = yaml.Unmarshal(existing, &all)
	}
	all = append(all, entry)
	out, mErr := yaml.Marshal(all)
	if mErr != nil {
		return
	}
	os.MkdirAll(filepath.Dir(c.logPath), 0o700)
	_ = atomicfile.WriteFile(c.logPath, out, 0o600)
}
