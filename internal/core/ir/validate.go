// Package ir 校验：IR-SCHEMA §5 的 12 条校验规则实现。
// 规则 7（id 末段=文件名）与规则 9（imports 跨文件环检测）需要文件系统上下文，
// 由 core/profile 在采集/读写层调用本包提供的独立函数完成。
package ir

import (
	"fmt"
	"regexp"
	"strings"
)

// Severity 校验问题级别。
type Severity int

const (
	SeverityError   Severity = iota // 校验失败（CLI 退出码 3）
	SeverityWarning                 // 警告（进 Warnings，退出码 5）
)

// Issue 单条校验问题。
type Issue struct {
	Rule    int    // IR-SCHEMA §5 规则编号（1–12）
	Entity  string // 实体 id（可为空）
	Message string
	Level   Severity
}

func (i Issue) String() string {
	lv := "ERROR"
	if i.Level == SeverityWarning {
		lv = "WARN"
	}
	return fmt.Sprintf("[规则%d][%s] %s: %s", i.Rule, lv, i.Entity, i.Message)
}

// ValidateOptions 校验上下文注入。
type ValidateOptions struct {
	RegisteredTools   []string              // 已注册适配器 id（规则 1/3）
	ManifestIRVersion int                   // profile manifest 版本（规则 11）
	CurrentIRVersion  int                   // 实现版本（规则 11）
	IsMerged          bool                  // 是否为 merged Bundle（规则 10）
	SecretResolver    func(ref string) bool // 规则 6：secretref 可解析性；nil 跳过
}

var (
	secretRefRe = regexp.MustCompile(`^secretref://cfg4ai/[a-zA-Z0-9./_-]+$`)

	transports   = map[string]bool{"stdio": true, "sse": true, "http": true, "ws": true}
	activations  = map[string]bool{"always": true, "model-decision": true, "glob": true, "manual": true, "scene": true}
	handlerTypes = map[string]bool{"command": true, "http": true, "prompt": true, "mcp_tool": true, "agent": true}
	hookEventSet = map[HookEvent]bool{
		HookSessionStart: true, HookSessionEnd: true, HookPreToolUse: true, HookPostToolUse: true,
		HookNotification: true, HookStop: true, HookUserPromptSubmit: true, HookPreCompact: true,
	}
)

// Validate 对 Bundle 执行 12 条校验，返回全部问题（error 与 warning 混合，调用方按 Level 分流）。
func Validate(b *Bundle, opts ValidateOptions) []Issue {
	var issues []Issue
	add := func(rule int, entity string, level Severity, format string, args ...any) {
		issues = append(issues, Issue{Rule: rule, Entity: entity, Level: level, Message: fmt.Sprintf(format, args...)})
	}

	registered := map[string]bool{"all": true}
	for _, t := range opts.RegisteredTools {
		registered[t] = true
	}

	seen := map[string]bool{}
	refs := map[string][]string{} // 规则 6 汇总：ref -> 所属实体

	checkHeader := func(id string, h *Header) {
		// 规则 5：frontmatter 必填
		if id == "" {
			add(5, "", SeverityError, "缺少必填字段 id")
			return
		}
		if h.IRVersion == 0 {
			add(5, id, SeverityError, "缺少必填字段 ir_version")
		}
		// 规则 1：id 格式与唯一性
		kind, _, err := ParseID(id)
		if err != nil {
			add(1, id, SeverityError, "%v", err)
		} else if kind == KindSetting {
			if tool, _, err := ParseSettingID(id); err != nil {
				add(1, id, SeverityError, "%v", err)
			} else if !registered[tool] {
				add(1, id, SeverityError, "setting id 的 tool 段 %q 未注册", tool)
			}
		}
		if seen[id] {
			add(1, id, SeverityError, "id 重复")
		}
		seen[id] = true
		// 规则 10：merged 不得含墓碑
		if opts.IsMerged && h.Tombstone {
			add(10, id, SeverityError, "merged Bundle 中出现墓碑条目（引擎应已遮蔽/剔除）")
		}
		// 规则 11：版本链
		if opts.ManifestIRVersion > 0 && h.IRVersion > opts.ManifestIRVersion {
			add(11, id, SeverityError, "实体 ir_version %d 高于 manifest 版本 %d", h.IRVersion, opts.ManifestIRVersion)
		}
		if opts.CurrentIRVersion > 0 && h.IRVersion > opts.CurrentIRVersion {
			add(11, id, SeverityError, "实体 ir_version %d 高于实现版本 %d（需升级 cfg4ai）", h.IRVersion, opts.CurrentIRVersion)
		}
		// 规则 3：origin.tool 与 x-<tool>
		if h.Origin != nil && !registered[h.Origin.Tool] {
			add(3, id, SeverityWarning, "origin.tool %q 不是已注册适配器 id", h.Origin.Tool)
		}
		for k := range h.Extensions {
			tool := strings.TrimPrefix(k, "x-")
			if !registered[tool] {
				add(3, id, SeverityWarning, "扩展键 %q 对应工具未注册（保留透传）", k)
			}
		}
	}

	checkAppliesTo := func(id string, tools []string) {
		for _, t := range tools {
			if !registered[t] {
				add(3, id, SeverityWarning, "applies_to 条目 %q 未注册", t)
			}
		}
	}

	for _, e := range b.Instructions {
		checkHeader(e.ID, &e.Header)
		checkAppliesTo(e.ID, e.AppliesTo)
		// 规则 12：activation 词表 + scene 语义
		if e.Activation != "" && !activations[string(e.Activation)] {
			add(12, e.ID, SeverityError, "activation %q 不在词表", e.Activation)
		}
		// 规则 9：imports 重复引用（跨文件环检测由采集层 DFS 完成）
		seenPath := map[string]bool{}
		for _, imp := range e.Imports {
			if seenPath[imp.Path] {
				add(9, e.ID, SeverityWarning, "imports 重复引用 %q", imp.Path)
			}
			seenPath[imp.Path] = true
		}
	}

	for _, e := range b.MCPServers {
		checkHeader(e.ID, &e.Header)
		// 规则 2：transport 与必填字段
		if !transports[e.Transport] {
			add(2, e.ID, SeverityError, "transport %q 非法（stdio|sse|http|ws）", e.Transport)
		} else if e.Transport == "stdio" && e.Command == "" {
			add(2, e.ID, SeverityError, "transport=stdio 必须有 command")
		} else if e.Transport != "stdio" && e.URL == "" {
			add(2, e.ID, SeverityError, "transport=%s 必须有 url", e.Transport)
		}
		collectRefsMap(refs, e.ID, e.Env)
		collectRefsMap(refs, e.ID, e.Headers)
		collectRefsList(refs, e.ID, e.Args)
	}

	for _, group := range [][]PromptPack{b.Skills, b.Agents, b.Commands, b.Workflows} {
		for _, e := range group {
			checkHeader(e.ID, &e.Header)
			if e.Activation != "" && !activations[string(e.Activation)] {
				add(12, e.ID, SeverityError, "activation %q 不在词表", e.Activation)
			}
			if e.Scene != "" && e.Activation != ActivationScene {
				add(12, e.ID, SeverityError, "scene=%q 仅当 activation=scene 时有意义", e.Scene)
			}
		}
	}

	for _, e := range b.Hooks {
		checkHeader(e.ID, &e.Header)
		if !hookEventSet[e.Event] {
			add(12, e.ID, SeverityWarning, "event %q 不在标准词表（应为工具特有事件，移入 x-）", e.Event)
		}
		if !handlerTypes[e.Handler.Type] {
			add(12, e.ID, SeverityError, "handler.type %q 非法", e.Handler.Type)
		}
	}

	for _, e := range b.Settings {
		checkHeader(e.ID, &e.Header)
		for _, t := range e.ToolScope {
			if !registered[t] {
				add(3, e.ID, SeverityWarning, "tool_scope 条目 %q 未注册", t)
			}
		}
		collectRefsAny(refs, e.ID, e.Value)
	}

	// 规则 4/6：secretref 格式与可解析性
	for ref, owners := range refs {
		if !secretRefRe.MatchString(ref) {
			for _, o := range owners {
				add(4, o, SeverityError, "secretref 格式非法：%q", ref)
			}
			continue
		}
		if opts.SecretResolver != nil && !opts.SecretResolver(ref) {
			for _, o := range owners {
				add(6, o, SeverityWarning, "secretref 不可解析（dangling）：%q", ref)
			}
		}
	}

	return issues
}

// ValidateNameMatch 规则 7：实体 id 末段必须等于所在目录/文件名（规范化等价）。
// 由 core/profile 在读写文件时调用（需要文件名上下文）。
func ValidateNameMatch(id, fileOrDirName string) error {
	if NameTail(id) != fileOrDirName {
		return fmt.Errorf("ir: 规则7：id %q 末段与文件名 %q 不一致", id, fileOrDirName)
	}
	return nil
}

// mergePolicyKinds merge_policy 允许的键。
var mergePolicyKinds = map[string]bool{
	"instructions": true, "mcp_servers": true, "skills": true, "agents": true,
	"commands": true, "workflows": true, "hooks": true, "settings": true,
}

// mergePolicyValues 各键允许的策略值。
var mergePolicyValues = map[string]map[string]bool{
	"instructions": {"concat": true, "project-only": true, "global-only": true},
}

// ValidateMergePolicy 规则 8：merge_policy 键必须是已知实体类型，值合法。
func ValidateMergePolicy(policy map[string]string) []Issue {
	var issues []Issue
	for k, v := range policy {
		if !mergePolicyKinds[k] {
			issues = append(issues, Issue{Rule: 8, Level: SeverityError,
				Message: fmt.Sprintf("merge_policy 键 %q 不是已知实体类型", k)})
			continue
		}
		if allowed, ok := mergePolicyValues[k]; ok && !allowed[v] {
			issues = append(issues, Issue{Rule: 8, Level: SeverityError,
				Message: fmt.Sprintf("merge_policy.%s 值 %q 非法", k, v)})
		} else if !ok && v != "merge-by-id" {
			issues = append(issues, Issue{Rule: 8, Level: SeverityError,
				Message: fmt.Sprintf("merge_policy.%s 值 %q 非法（仅允许 merge-by-id）", k, v)})
		}
	}
	return issues
}

// collectRefStr 收集单个字符串值中的 secretref。
func collectRefStr(out map[string][]string, owner, v string) {
	if strings.HasPrefix(v, "secretref://") {
		out[v] = append(out[v], owner)
	}
}

// collectRefsMap 从 map[string]string 字段（env/headers）收集 secretref。
func collectRefsMap(out map[string][]string, owner string, m map[string]string) {
	for _, v := range m {
		collectRefStr(out, owner, v)
	}
}

// collectRefsList 从 []string 字段（args）收集 secretref。
func collectRefsList(out map[string][]string, owner string, l []string) {
	for _, v := range l {
		collectRefStr(out, owner, v)
	}
}

// collectRefsAny 从任意值（Setting.Value / OAuth map）递归收集 secretref。
func collectRefsAny(out map[string][]string, owner string, v any) {
	switch t := v.(type) {
	case string:
		if strings.HasPrefix(t, "secretref://") {
			out[t] = append(out[t], owner)
		}
	case map[string]any:
		for _, vv := range t {
			collectRefsAny(out, owner, vv)
		}
	case []any:
		for _, vv := range t {
			collectRefsAny(out, owner, vv)
		}
	}
}
