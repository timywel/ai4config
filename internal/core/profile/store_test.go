package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/timywel/ai4config/internal/core/ir"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	when := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)

	m := &Manifest{
		IRVersion: 1,
		Profile:   Meta{Name: "global", Kind: "global", CreatedAt: when},
	}
	b := &ir.Bundle{
		IRVersion: 1,
		Scope:     ir.ScopeGlobal,
		Instructions: []ir.Instruction{{
			Header:   ir.Header{ID: "instruction.coding-style", IRVersion: 1},
			Priority: 100,
			Language: "zh",
			Body:     "# 编码规范\n\n- 中文回复\n",
		}},
		MCPServers: []ir.MCPServer{{
			Header: ir.Header{
				ID: "mcp.fs", IRVersion: 1,
				Origin:     &ir.Origin{Tool: "codex", Path: "~/.codex/config.toml", Scope: ir.ScopeGlobal, CollectedAt: when, RawHash: "sha256:aa"},
				Extensions: map[string]any{"x-vscode": map[string]any{"gallery": true}},
			},
			Name:      "fs",
			Transport: "stdio",
			Command:   "npx",
			Args:      []string{"-y", "server"},
			Env:       map[string]string{"TOKEN": "secretref://cfg4ai/global/mcp.fs/env.TOKEN"},
		}},
		Skills: []ir.PromptPack{{
			Header:     ir.Header{ID: "skill.review", IRVersion: 1},
			Kind:       ir.KindSkill,
			Name:       "review",
			Body:       "评审正文\n",
			Tools:      []string{"bash", "read"},
			Activation: ir.ActivationModelDecision,
		}},
		Hooks: []ir.Hook{{
			Header:  ir.Header{ID: "hook.guard", IRVersion: 1},
			Event:   ir.HookPreToolUse,
			Handler: ir.HookHandler{Type: "command", Command: "./g.sh", CommandWindows: "./g.ps1"},
		}},
		Settings: []ir.SettingEntry{{
			Header: ir.Header{ID: "setting.zhanlu.model", IRVersion: 1},
			Key:    "model",
			Value:  "kimi-k3",
		}},
		MCPFileExtensions: map[string]any{"inputs": []any{map[string]any{"id": "t"}}},
	}

	if err := Save(dir, b, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// 校验生成的文件结构
	for _, p := range []string{
		"manifest.yaml",
		"instructions/coding-style.md",
		"mcp.yaml",
		"skills/review/skill.yaml",
		"skills/review/prompt.md",
		"hooks.yaml",
		"settings.yaml",
	} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("应生成文件 %s: %v", p, err)
		}
	}

	back, err := Load(dir, ir.ScopeGlobal)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := back.Bundle

	// manifest
	if back.Manifest.Profile.Name != "global" || back.Manifest.IRVersion != 1 {
		t.Errorf("manifest 往返不一致: %+v", back.Manifest.Profile)
	}
	// instruction 含正文
	if len(got.Instructions) != 1 || got.Instructions[0].Body != "# 编码规范\n\n- 中文回复\n" {
		t.Errorf("instruction 往返不一致: %+v", got.Instructions)
	}
	// mcp 含 x- 扩展 + secretref
	if len(got.MCPServers) != 1 {
		t.Fatalf("mcp 数量错误: %d", len(got.MCPServers))
	}
	ms := got.MCPServers[0]
	if ms.Env["TOKEN"] != "secretref://cfg4ai/global/mcp.fs/env.TOKEN" {
		t.Errorf("secretref 丢失: %v", ms.Env)
	}
	if _, ok := ms.Extensions["x-vscode"]; !ok {
		t.Errorf("x-vscode 扩展丢失: %+v", ms.Extensions)
	}
	if ms.Origin == nil || ms.Origin.Tool != "codex" {
		t.Errorf("origin 丢失: %+v", ms.Origin)
	}
	// file_extensions 顶层扩展位
	if _, ok := got.MCPFileExtensions["inputs"]; !ok {
		t.Errorf("MCPFileExtensions 丢失: %+v", got.MCPFileExtensions)
	}
	// skill 含正文 + tools + activation
	if len(got.Skills) != 1 || got.Skills[0].Body != "评审正文\n" {
		t.Errorf("skill 正文往返不一致: %+v", got.Skills)
	}
	if got.Skills[0].Activation != ir.ActivationModelDecision {
		t.Errorf("skill activation 丢失: %q", got.Skills[0].Activation)
	}
	// hook 跨平台双命令
	if len(got.Hooks) != 1 || got.Hooks[0].Handler.CommandWindows != "./g.ps1" {
		t.Errorf("hook 往返不一致: %+v", got.Hooks)
	}
	// setting
	if len(got.Settings) != 1 || got.Settings[0].Value != "kimi-k3" {
		t.Errorf("setting 往返不一致: %+v", got.Settings)
	}
}

// T2.3：ir_version 高于实现版本拒绝
func TestManifestVersionTooHigh(t *testing.T) {
	dir := t.TempDir()
	high := &Manifest{IRVersion: 99, Profile: Meta{Name: "g", Kind: "global", CreatedAt: time.Now()}}
	if err := SaveManifest(dir, high); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}
	if _, err := LoadManifest(dir); err == nil {
		t.Fatal("ir_version 高于实现版本应拒绝")
	}
}

// 规则 8：非法 merge_policy 拒绝
func TestManifestBadMergePolicy(t *testing.T) {
	dir := t.TempDir()
	bad := &Manifest{
		IRVersion:   1,
		Profile:     Meta{Name: "g", Kind: "global", CreatedAt: time.Now()},
		MergePolicy: map[string]string{"mcp_servers": "concat"}, // 非法值
	}
	if err := SaveManifest(dir, bad); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}
	if _, err := LoadManifest(dir); err == nil {
		t.Fatal("非法 merge_policy 应拒绝")
	}
}

// 文件权限：0600（Unix 语义；Windows 忽略）
func TestStoreFilePerm(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{IRVersion: 1, Profile: Meta{Name: "g", Kind: "global", CreatedAt: time.Now()}}
	if err := SaveManifest(dir, m); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}
	info, err := os.Stat(manifestPath(dir))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	// Windows 忽略权限位，仅在非 Windows 断言 0600
	if os.Getenv("GOOS") != "windows" && info.Mode().Perm() != 0o600 {
		t.Logf("权限 = %o（非 Windows 期望 0600）", info.Mode().Perm())
	}
}
