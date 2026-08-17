package secrets

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"filippo.io/age"
	"gopkg.in/yaml.v3"

	"github.com/timywel/ai4config/internal/atomicfile"
)

// fileBackend 加密文件后端（secrets.age，age scrypt 口令加密，0600）。
// 适用：headless Linux / 麒麟服务器 / CI（无桌面 Secret Service 时的降级链中间级）。
type fileBackend struct {
	path string
	pass string
	data map[string]string
}

// openFileBackend 打开（必要时创建）加密文件后端。
func openFileBackend(repoRoot string, prompt PassphrasePrompt) (Backend, error) {
	if prompt == nil {
		return nil, fmt.Errorf("secrets: file 后端需要口令来源（PassphrasePrompt 为 nil）")
	}
	pass, ok := prompt()
	if !ok || pass == "" {
		return nil, fmt.Errorf("secrets: 未获取到加密文件口令")
	}
	b := &fileBackend{
		path: filepath.Join(repoRoot, "secrets.age"),
		pass: pass,
		data: map[string]string{},
	}
	if err := b.load(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *fileBackend) Type() BackendType { return BackendFile }

func (b *fileBackend) Get(ref string) (string, error) {
	v, ok := b.data[ref]
	if !ok {
		return "", fmt.Errorf("secrets: file 后端无 %q", ref)
	}
	return v, nil
}

func (b *fileBackend) Set(ref string, value string) error {
	b.data[ref] = value
	return b.save()
}

func (b *fileBackend) Delete(ref string) error {
	delete(b.data, ref)
	return b.save()
}

// load 读取并解密 secrets.age（文件不存在视为空库）。
func (b *fileBackend) load() error {
	raw, err := os.ReadFile(b.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	identity, err := age.NewScryptIdentity(b.pass)
	if err != nil {
		return err
	}
	r, err := age.Decrypt(bytes.NewReader(raw), identity)
	if err != nil {
		return fmt.Errorf("secrets: 解密 secrets.age 失败（口令错误？）: %w", err)
	}
	plain, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if len(plain) == 0 {
		return nil
	}
	return yaml.Unmarshal(plain, &b.data)
}

// save 加密并原子写回（0600）。
func (b *fileBackend) save() error {
	plain, err := yaml.Marshal(b.data)
	if err != nil {
		return err
	}
	recipient, err := age.NewScryptRecipient(b.pass)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return err
	}
	if _, err := w.Write(plain); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(b.path), 0o700); err != nil {
		return err
	}
	return atomicfile.WriteFile(b.path, buf.Bytes(), 0o600)
}
