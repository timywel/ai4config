package ir

import (
	"strings"
	"testing"
)

func baseOpts() ValidateOptions {
	return ValidateOptions{
		RegisteredTools:   []string{"claude-code", "codex", "copilot", "zhanlu"},
		ManifestIRVersion: 1,
		CurrentIRVersion:  1,
	}
}

func hasRule(issues []Issue, rule int, level Severity) bool {
	for _, i := range issues {
		if i.Rule == rule && i.Level == level {
			return true
		}
	}
	return false
}

func TestValidateCleanBundle(t *testing.T) {
	b := &Bundle{
		MCPServers: []MCPServer{{
			Header:    Header{ID: "mcp.fs", IRVersion: 1, Origin: &Origin{Tool: "codex"}},
			Name:      "fs",
			Transport: "stdio",
			Command:   "npx",
		}},
	}
	issues := Validate(b, baseOpts())
	if len(issues) != 0 {
		t.Fatalf("干净 Bundle 不应有问题: %v", issues)
	}
}

func TestValidateRule1UniqueAndFormat(t *testing.T) {
	b := &Bundle{
		Instructions: []Instruction{
			{Header: Header{ID: "instruction.a", IRVersion: 1}},
			{Header: Header{ID: "instruction.a", IRVersion: 1}}, // 重复
			{Header: Header{ID: "bad id", IRVersion: 1}},        // 格式错
		},
	}
	issues := Validate(b, baseOpts())
	if !hasRule(issues, 1, SeverityError) {
		t.Fatal("应命中规则 1（重复/格式）")
	}
}

func TestValidateRule1SettingToolRegistered(t *testing.T) {
	b := &Bundle{
		Settings: []SettingEntry{
			{Header: Header{ID: "setting.unknown-tool.model", IRVersion: 1}, Key: "model"},
		},
	}
	if !hasRule(Validate(b, baseOpts()), 1, SeverityError) {
		t.Fatal("未注册 tool 的 setting id 应命中规则 1")
	}
}

func TestValidateRule2Transport(t *testing.T) {
	b := &Bundle{
		MCPServers: []MCPServer{
			{Header: Header{ID: "mcp.a", IRVersion: 1}, Name: "a", Transport: "stdio"},                 // 缺 command
			{Header: Header{ID: "mcp.b", IRVersion: 1}, Name: "b", Transport: "http"},                  // 缺 url
			{Header: Header{ID: "mcp.c", IRVersion: 1}, Name: "c", Transport: "carrier-pigeon"},        // 非法
			{Header: Header{ID: "mcp.d", IRVersion: 1}, Name: "d", Transport: "sse", URL: "https://x"}, // 合法
		},
	}
	issues := Validate(b, baseOpts())
	count := 0
	for _, i := range issues {
		if i.Rule == 2 && i.Level == SeverityError {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("规则 2 应命中 3 次，实际 %d: %v", count, issues)
	}
}

func TestValidateRule3UnregisteredWarning(t *testing.T) {
	b := &Bundle{
		Instructions: []Instruction{{
			Header:    Header{ID: "instruction.x", IRVersion: 1, Origin: &Origin{Tool: "ghost"}, Extensions: map[string]any{"x-ghost": 1}},
			AppliesTo: []string{"ghost"},
		}},
	}
	issues := Validate(b, baseOpts())
	for _, rule := range []int{3} {
		if !hasRule(issues, rule, SeverityWarning) {
			t.Fatalf("应命中规则 %d Warning: %v", rule, issues)
		}
	}
}

func TestValidateRule4And6SecretRef(t *testing.T) {
	b := &Bundle{
		MCPServers: []MCPServer{{
			Header:    Header{ID: "mcp.fs", IRVersion: 1},
			Name:      "fs",
			Transport: "stdio",
			Command:   "npx",
			Env: map[string]string{
				"BAD":  "secretref://cfg4ai/Bad Key!",               // 格式非法
				"GOOD": "secretref://cfg4ai/global/mcp.fs/env.GOOD", // 合法但不可解析
			},
		}},
	}
	opts := baseOpts()
	opts.SecretResolver = func(ref string) bool { return false }
	issues := Validate(b, opts)
	if !hasRule(issues, 4, SeverityError) {
		t.Fatal("非法 secretref 应命中规则 4")
	}
	if !hasRule(issues, 6, SeverityWarning) {
		t.Fatal("dangling secretref 应命中规则 6 Warning")
	}
}

func TestValidateRule5Required(t *testing.T) {
	b := &Bundle{
		Skills: []PromptPack{{Header: Header{IRVersion: 1}, Kind: KindSkill, Name: "x"}}, // 缺 id
	}
	if !hasRule(Validate(b, baseOpts()), 5, SeverityError) {
		t.Fatal("缺 id 应命中规则 5")
	}
}

func TestValidateRule10TombstoneInMerged(t *testing.T) {
	b := &Bundle{
		Skills: []PromptPack{{Header: Header{ID: "skill.x", IRVersion: 1, Tombstone: true}, Kind: KindSkill, Name: "x"}},
	}
	opts := baseOpts()
	opts.IsMerged = true
	if !hasRule(Validate(b, opts), 10, SeverityError) {
		t.Fatal("merged 含墓碑应命中规则 10")
	}
	if hasRule(Validate(b, baseOpts()), 10, SeverityError) {
		t.Fatal("非 merged 含墓碑不应命中规则 10")
	}
}

func TestValidateRule11Version(t *testing.T) {
	b := &Bundle{
		Skills: []PromptPack{{Header: Header{ID: "skill.x", IRVersion: 9}, Kind: KindSkill, Name: "x"}},
	}
	if !hasRule(Validate(b, baseOpts()), 11, SeverityError) {
		t.Fatal("ir_version 超限应命中规则 11")
	}
}

func TestValidateRule12Enums(t *testing.T) {
	b := &Bundle{
		Skills: []PromptPack{
			{Header: Header{ID: "skill.a", IRVersion: 1}, Kind: KindSkill, Name: "a", Activation: "weird"},
			{Header: Header{ID: "skill.b", IRVersion: 1}, Kind: KindSkill, Name: "b", Activation: ActivationManual, Scene: "git_message"},
		},
		Hooks: []Hook{
			{Header: Header{ID: "hook.a", IRVersion: 1}, Event: HookPreToolUse, Handler: HookHandler{Type: "quantum"}},
		},
	}
	issues := Validate(b, baseOpts())
	if !hasRule(issues, 12, SeverityError) {
		t.Fatal("非法 activation/handler.type 应命中规则 12 Error")
	}
	// scene 仅 scene 激活有意义：skill.b 命中
	found := false
	for _, i := range issues {
		if i.Rule == 12 && strings.Contains(i.Entity, "skill.b") {
			found = true
		}
	}
	if !found {
		t.Fatal("scene+非 scene 激活应命中规则 12")
	}
}

func TestValidateRule9DuplicateImports(t *testing.T) {
	b := &Bundle{
		Instructions: []Instruction{{
			Header:  Header{ID: "instruction.x", IRVersion: 1},
			Imports: []Import{{Path: "./a.md", Resolved: true}, {Path: "./a.md", Resolved: true}},
		}},
	}
	if !hasRule(Validate(b, baseOpts()), 9, SeverityWarning) {
		t.Fatal("重复 imports 应命中规则 9 Warning")
	}
}

func TestValidateMergePolicy(t *testing.T) {
	bad := map[string]string{
		"instructions": "concat",      // 合法
		"unicorns":     "merge-by-id", // 非法键
		"mcp_servers":  "concat",      // 非法值
	}
	issues := ValidateMergePolicy(bad)
	if len(issues) != 2 {
		t.Fatalf("应命中 2 条，实际 %d: %v", len(issues), issues)
	}
}

func TestValidateNameMatch(t *testing.T) {
	if err := ValidateNameMatch("skill.code-review", "code-review"); err != nil {
		t.Fatalf("一致不应报错: %v", err)
	}
	if err := ValidateNameMatch("skill.code-review", "other"); err == nil {
		t.Fatal("不一致应报错")
	}
}
