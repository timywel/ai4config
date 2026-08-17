package profile

import (
	"testing"

	"github.com/timywel/ai4config/internal/core/ir"
)

// --- 测试构造助手 ---

func mcp(id, command string, env map[string]string) ir.MCPServer {
	return ir.MCPServer{
		Header:    ir.Header{ID: id, IRVersion: 1},
		Name:      ir.NameTail(id),
		Transport: "stdio",
		Command:   command,
		Env:       env,
	}
}

func inst(id string, priority int, path string) ir.Instruction {
	return ir.Instruction{
		Header:   ir.Header{ID: id, IRVersion: 1, Origin: &ir.Origin{Path: path}},
		Priority: priority,
		Body:     "body of " + id,
	}
}

func layer(scope ir.Scope, b *ir.Bundle) *ScopedBundle {
	return &ScopedBundle{Scope: scope, Bundle: b}
}

func findMCP(list []ir.MCPServer, id string) *ir.MCPServer {
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}

// --- merge-by-id：浅字段级合并（IR-SCHEMA §2.1 例子） ---

func TestMergeShallowFieldLevel(t *testing.T) {
	global := layer(ir.ScopeGlobal, &ir.Bundle{MCPServers: []ir.MCPServer{
		func() ir.MCPServer {
			s := mcp("mcp.fs", "npx", map[string]string{"ROOT": "/data", "DEBUG": "1"})
			s.Args = []string{"-y", "server"}
			return s
		}(),
	}})
	project := layer(ir.ScopeProject, &ir.Bundle{MCPServers: []ir.MCPServer{
		mcp("mcp.fs", "", map[string]string{"DEBUG": "0"}), // 只写 env 一个字段
	}})

	merged := MergeBundles(global, project)
	got := findMCP(merged.MCPServers, "mcp.fs")
	if got == nil {
		t.Fatal("merged 缺少 mcp.fs")
	}
	if got.Command != "npx" {
		t.Errorf("command 应继承全局 npx，实际 %q", got.Command)
	}
	if len(got.Args) != 2 { // 数组整体继承（项目未写）
		t.Errorf("args 应继承全局，实际 %v", got.Args)
	}
	// env 是 object 字段：整体覆盖（不是逐键合并）
	if len(got.Env) != 1 || got.Env["DEBUG"] != "0" {
		t.Errorf("env 应整体覆盖为 {DEBUG:0}，实际 %v", got.Env)
	}
	if _, existed := got.Env["ROOT"]; existed {
		t.Errorf("env 整体覆盖后不应残留全局的 ROOT 键: %v", got.Env)
	}
}

// --- merge-by-id：数组字段整体替换（不逐项合并） ---

func TestMergeArrayReplace(t *testing.T) {
	global := layer(ir.ScopeGlobal, &ir.Bundle{MCPServers: []ir.MCPServer{
		func() ir.MCPServer {
			s := mcp("mcp.fs", "npx", nil)
			s.Args = []string{"a", "b", "c"}
			return s
		}(),
	}})
	project := layer(ir.ScopeProject, &ir.Bundle{MCPServers: []ir.MCPServer{
		func() ir.MCPServer {
			s := mcp("mcp.fs", "", nil)
			s.Args = []string{"x"}
			return s
		}(),
	}})

	got := findMCP(MergeBundles(global, project).MCPServers, "mcp.fs")
	if len(got.Args) != 1 || got.Args[0] != "x" {
		t.Errorf("数组应整体替换为 [x]，实际 %v", got.Args)
	}
}

// --- 墓碑遮蔽（红队 T-01/T-07 防线） ---

func TestMergeTombstoneShadowing(t *testing.T) {
	global := layer(ir.ScopeGlobal, &ir.Bundle{MCPServers: []ir.MCPServer{
		mcp("mcp.a", "npx", nil),
		mcp("mcp.b", "npx", nil),
	}})
	project := layer(ir.ScopeProject, &ir.Bundle{MCPServers: []ir.MCPServer{
		func() ir.MCPServer {
			s := mcp("mcp.a", "", nil)
			s.Tombstone = true // 项目层删除 mcp.a
			return s
		}(),
	}})

	merged := MergeBundles(global, project)
	if findMCP(merged.MCPServers, "mcp.a") != nil {
		t.Error("mcp.a 被项目层墓碑遮蔽，不应出现在 merged（防删除复活）")
	}
	if findMCP(merged.MCPServers, "mcp.b") == nil {
		t.Error("mcp.b 未被遮蔽，应保留")
	}
}

// --- 墓碑不进 merged（IR-SCHEMA §5 规则 10） ---

func TestMergeTombstoneNotInMerged(t *testing.T) {
	project := layer(ir.ScopeProject, &ir.Bundle{Skills: []ir.PromptPack{
		{Header: ir.Header{ID: "skill.x", IRVersion: 1, Tombstone: true}, Kind: ir.KindSkill, Name: "x"},
	}})
	merged := MergeBundles(project)
	for _, s := range merged.Skills {
		if s.Tombstone {
			t.Error("merged 不应包含墓碑条目")
		}
	}
	if len(merged.Skills) != 0 {
		t.Errorf("merged.Skills 应为空，实际 %d", len(merged.Skills))
	}
}

// --- 更高优先级层可在墓碑之上"复活"（local > project 墓碑） ---

func TestMergeHigherLayerOverridesTombstone(t *testing.T) {
	project := layer(ir.ScopeProject, &ir.Bundle{MCPServers: []ir.MCPServer{
		func() ir.MCPServer { s := mcp("mcp.a", "", nil); s.Tombstone = true; return s }(),
	}})
	local := layer(ir.ScopeLocal, &ir.Bundle{MCPServers: []ir.MCPServer{
		mcp("mcp.a", "local-cmd", nil), // local 优先级(1) > project(2) 墓碑
	}})
	merged := MergeBundles(project, local)
	got := findMCP(merged.MCPServers, "mcp.a")
	if got == nil {
		t.Fatal("local 层优先级高于 project 墓碑，mcp.a 应复活")
	}
	if got.Command != "local-cmd" {
		t.Errorf("复活条目应取 local 值，实际 %q", got.Command)
	}
}

// --- managed 层不物化 ---

func TestMergeManagedExcluded(t *testing.T) {
	managed := layer(ir.ScopeManaged, &ir.Bundle{MCPServers: []ir.MCPServer{mcp("mcp.corp", "corp", nil)}})
	global := layer(ir.ScopeGlobal, &ir.Bundle{MCPServers: []ir.MCPServer{mcp("mcp.user", "user", nil)}})
	merged := MergeBundles(managed, global)
	if findMCP(merged.MCPServers, "mcp.corp") != nil {
		t.Error("managed 层不应物化")
	}
	if findMCP(merged.MCPServers, "mcp.user") == nil {
		t.Error("global 层应物化")
	}
	if merged.Scope != ir.ScopeMerged {
		t.Errorf("merged.Scope 应为 merged，实际 %s", merged.Scope)
	}
}

// --- concat：两段式排序（层级 → priority → path） ---

func TestConcatInstructionsOrdering(t *testing.T) {
	global := layer(ir.ScopeGlobal, &ir.Bundle{Instructions: []ir.Instruction{
		inst("instruction.a", 100, "~/CLAUDE.md"), // global priority 100
		inst("instruction.b", 50, "~/rules.md"),   // global priority 50（小，靠前）
	}})
	project := layer(ir.ScopeProject, &ir.Bundle{Instructions: []ir.Instruction{
		inst("instruction.c", 200, "CLAUDE.md"), // project priority 200
	}})

	merged := MergeBundles(global, project)
	if len(merged.Instructions) != 3 {
		t.Fatalf("应有 3 条，实际 %d", len(merged.Instructions))
	}
	want := []string{"instruction.b", "instruction.a", "instruction.c"}
	for i, id := range want {
		if merged.Instructions[i].ID != id {
			t.Errorf("位置 %d 应为 %s，实际 %s", i, id, merged.Instructions[i].ID)
		}
	}
}

// --- concat：层级优先于 priority（不跨层混排） ---

func TestConcatLayerBeforePriority(t *testing.T) {
	global := layer(ir.ScopeGlobal, &ir.Bundle{Instructions: []ir.Instruction{
		inst("instruction.g", 500, "~/g.md"), // global 但 priority 很大
	}})
	project := layer(ir.ScopeProject, &ir.Bundle{Instructions: []ir.Instruction{
		inst("instruction.p", 10, "p.md"), // project priority 很小
	}})
	merged := MergeBundles(global, project)
	// global 层（低优先级）整体在前，即使其 priority 数值更大
	if merged.Instructions[0].ID != "instruction.g" {
		t.Errorf("global 层应整体在前（两段式），实际首位 %s", merged.Instructions[0].ID)
	}
}

// --- concat：同层同 priority 按 origin.path 字典序 ---

func TestConcatSamePriorityByPath(t *testing.T) {
	global := layer(ir.ScopeGlobal, &ir.Bundle{Instructions: []ir.Instruction{
		inst("instruction.z", 100, "~/z.md"),
		inst("instruction.y", 100, "~/a.md"),
	}})
	merged := MergeBundles(global)
	if merged.Instructions[0].ID != "instruction.y" {
		t.Errorf("同 priority 应按 path 字典序（a.md 在前），实际 %s", merged.Instructions[0].ID)
	}
}

// --- x- 扩展合并：同键覆盖、异键保留 ---

func TestMergeExtensions(t *testing.T) {
	global := layer(ir.ScopeGlobal, &ir.Bundle{Skills: []ir.PromptPack{
		{Header: ir.Header{ID: "skill.x", IRVersion: 1, Extensions: map[string]any{
			"x-claude-code": map[string]any{"model": "opus"},
			"x-codex":       map[string]any{"sandbox": true},
		}}, Kind: ir.KindSkill, Name: "x"},
	}})
	project := layer(ir.ScopeProject, &ir.Bundle{Skills: []ir.PromptPack{
		{Header: ir.Header{ID: "skill.x", IRVersion: 1, Extensions: map[string]any{
			"x-claude-code": map[string]any{"model": "sonnet"}, // 覆盖同键
		}}, Kind: ir.KindSkill, Name: "x"},
	}})

	merged := MergeBundles(global, project)
	ext := merged.Skills[0].Extensions
	cc := ext["x-claude-code"].(map[string]any)
	if cc["model"] != "sonnet" {
		t.Errorf("同键应被高层覆盖为 sonnet，实际 %v", cc["model"])
	}
	if _, ok := ext["x-codex"]; !ok {
		t.Error("异键 x-codex 应保留（异构采集不触碰他工具扩展位）")
	}
}

// --- origin 取胜出层 ---

func TestMergeOriginFromWinningLayer(t *testing.T) {
	global := layer(ir.ScopeGlobal, &ir.Bundle{MCPServers: []ir.MCPServer{
		func() ir.MCPServer {
			s := mcp("mcp.a", "g", nil)
			s.Origin = &ir.Origin{Tool: "codex", Path: "~/.codex/config.toml", Scope: ir.ScopeGlobal}
			return s
		}(),
	}})
	project := layer(ir.ScopeProject, &ir.Bundle{MCPServers: []ir.MCPServer{
		func() ir.MCPServer {
			s := mcp("mcp.a", "p", nil)
			s.Origin = &ir.Origin{Tool: "codex", Path: ".codex/config.toml", Scope: ir.ScopeProject}
			return s
		}(),
	}})
	got := findMCP(MergeBundles(global, project).MCPServers, "mcp.a")
	if got.Origin == nil || got.Origin.Path != ".codex/config.toml" {
		t.Errorf("origin 应取胜出层（project），实际 %+v", got.Origin)
	}
}
