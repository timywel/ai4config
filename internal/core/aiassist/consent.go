package aiassist

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/timywel/ai4config/internal/atomicfile"
)

// ConsentStatus consent 检查结果。
type ConsentStatus int

const (
	ConsentOK            ConsentStatus = iota // 已同意且配置未变
	ConsentNeedFirst                          // 首次使用，需同意
	ConsentNeedReconfirm                      // AI 配置段变更，需重新同意（红队 T-09 防线）
)

// ConsentRecord consent 记录（consent.yaml，本机文件，不入 sync）。
type ConsentRecord struct {
	AgreedAt   time.Time `yaml:"agreed_at"`
	Endpoint   string    `yaml:"endpoint"`
	ConfigHash string    `yaml:"config_hash"` // AI 配置段（provider+base_url+model）的 hash
}

// AIConfig AI 配置段（consent 绑定对象）。
type AIConfig struct {
	Provider string
	BaseURL  string
	Model    string
}

// configHash 计算 AI 配置段 hash（变更检测依据）。
func (c AIConfig) configHash() string {
	h := sha256.Sum256([]byte(c.Provider + "|" + c.BaseURL + "|" + c.Model))
	return hex.EncodeToString(h[:])
}

func consentPath(repoRoot string) string { return filepath.Join(repoRoot, "consent.yaml") }

// CheckConsent 检查 consent 状态（红队 T-09：配置变更强制重确认）。
func CheckConsent(repoRoot string, cfg AIConfig) ConsentStatus {
	data, err := os.ReadFile(consentPath(repoRoot))
	if err != nil {
		return ConsentNeedFirst
	}
	var rec ConsentRecord
	if err := yaml.Unmarshal(data, &rec); err != nil {
		return ConsentNeedFirst
	}
	if rec.ConfigHash != cfg.configHash() {
		return ConsentNeedReconfirm
	}
	return ConsentOK
}

// RecordConsent 记录 consent（同意后调用）。
func RecordConsent(repoRoot string, cfg AIConfig) error {
	rec := ConsentRecord{
		AgreedAt:   time.Now().UTC(),
		Endpoint:   cfg.BaseURL,
		ConfigHash: cfg.configHash(),
	}
	data, err := yaml.Marshal(&rec)
	if err != nil {
		return err
	}
	if err := atomicfile.WriteFile(consentPath(repoRoot), data, 0o600); err != nil {
		return fmt.Errorf("aiassist: 写 consent 记录失败: %w", err)
	}
	return nil
}
