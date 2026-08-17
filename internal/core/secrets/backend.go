// Package secrets 实现 secret 的三级后端降级链、敏感扫描与回采保护。
// 权威规范：docs/ARCHITECTURE.md §9（安全）、IR-SCHEMA §3.6（secretref）。
package secrets

import (
	"fmt"

	"github.com/99designs/keyring"
)

// BackendType secret 后端类型（IR 记录 secret_backend 用）。
type BackendType string

const (
	BackendKeyring BackendType = "keyring" // 系统钥匙串（首选）
	BackendFile    BackendType = "file"    // 加密文件（headless/CI 出路）
	BackendNone    BackendType = "none"    // 占位符导出，人工填
)

// Backend secret 存取接口。
type Backend interface {
	Type() BackendType
	Get(ref string) (string, error)
	Set(ref string, value string) error
	Delete(ref string) error
}

// 三级降级链（ARCHITECTURE §9）：
//
//	① 系统 keyring（99designs/keyring，自带 file 后端与多后端自动探测）
//	② 加密文件（secrets.age，CI/headless）
//	③ none（占位符）
//
// ResolveBackend 按 backend 配置解析可用后端。
// prefer: "auto" | "keyring" | "file" | "none"（CLI-SPEC §10 secrets.backend）。
func ResolveBackend(prefer string, repoRoot string, prompt PassphrasePrompt) (Backend, error) {
	switch prefer {
	case "", "auto":
		// 降级链：keyring → file → none
		if b, err := openKeyring(); err == nil {
			return b, nil
		}
		if b, err := openFileBackend(repoRoot, prompt); err == nil {
			return b, nil
		}
		return NoneBackend{}, nil
	case "keyring":
		return openKeyring()
	case "file":
		return openFileBackend(repoRoot, prompt)
	case "none":
		return NoneBackend{}, nil
	default:
		return nil, fmt.Errorf("secrets: 未知后端 %q（keyring|file|none|auto）", prefer)
	}
}

// PassphrasePrompt 加密文件后端的口令获取回调（交互录入或环境变量注入）。
// 返回空口令且 ok=false 表示无法获取（降级到下一级）。
type PassphrasePrompt func() (passphrase string, ok bool)

// openKeyring 打开系统钥匙串（99designs/keyring 多后端自动探测）。
func openKeyring() (Backend, error) {
	kr, err := keyring.Open(keyring.Config{
		ServiceName: "cfg4ai",
		// 无桌面环境/无 libsecret（麒麟服务器、headless、CI）时 Open 返回错误 → 调用方降级
	})
	if err != nil {
		return nil, fmt.Errorf("secrets: keyring 不可用: %w", err)
	}
	return &keyringBackend{kr: kr}, nil
}

// ---------- keyring 后端 ----------

type keyringBackend struct {
	kr keyring.Keyring
}

func (b *keyringBackend) Type() BackendType { return BackendKeyring }

func (b *keyringBackend) Get(ref string) (string, error) {
	item, err := b.kr.Get(ref)
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

func (b *keyringBackend) Set(ref string, value string) error {
	return b.kr.Set(keyring.Item{Key: ref, Data: []byte(value)})
}

func (b *keyringBackend) Delete(ref string) error { return b.kr.Remove(ref) }

// ---------- none 后端（占位符，不存值） ----------

// NoneBackend 不存任何值：Get 永远报"需人工填充"，Set 静默忽略（导出物留占位符）。
type NoneBackend struct{}

func (NoneBackend) Type() BackendType { return BackendNone }
func (NoneBackend) Get(ref string) (string, error) {
	return "", fmt.Errorf("secrets: none 后端不存储值，%q 需人工填充", ref)
}
func (NoneBackend) Set(ref string, value string) error { return nil }
func (NoneBackend) Delete(ref string) error            { return nil }
