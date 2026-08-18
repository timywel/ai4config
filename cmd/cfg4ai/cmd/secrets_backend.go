package cmd

import (
	"os"

	"github.com/timywel/ai4config/internal/core/secrets"
	"github.com/timywel/ai4config/internal/store"
)

// resolveSecretsBackend 解析三级 secret 后端降级链（ARCHITECTURE §9）。
// 口令来源（file 后端）：环境变量 CFG4AI_SECRETS_PASSPHRASE 优先；交互场景留待 TUI。
func resolveSecretsBackend(repo *store.Repo) (secrets.Backend, string) {
	prompt := secrets.PassphrasePrompt(func() (string, bool) {
		if v := os.Getenv("CFG4AI_SECRETS_PASSPHRASE"); v != "" {
			return v, true
		}
		return "", false
	})
	b, err := secrets.ResolveBackend(flagSecretsBackend, repo.Root, prompt)
	if err != nil {
		// 降级链兜底：none（占位符）
		return secrets.NoneBackend{}, "none"
	}
	return b, string(b.Type())
}
