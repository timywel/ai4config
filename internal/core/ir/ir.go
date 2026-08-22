// Package ir 定义 cfg4ai 的统一中间表示（IR）数据模型。
// 权威规范：docs/IR-SCHEMA.md（v0.3）。本文件只承载类型与常量，
// 合并语义（merge-by-id/concat/墓碑遮蔽）在 core/profile 的物化逻辑中实现。
package ir

import "time"

// Scope 五层配置层级（IR-SCHEMA §1.2，D1）。
type Scope string

const (
	ScopeManaged Scope = "managed" // 企业下发；只读采集，不参与物化
	ScopeRemote  Scope = "remote"  // 组织远程订阅；参与合并，不写回
	ScopeLocal   Scope = "local"   // 项目内私人层；参与合并，sync 排除
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
	ScopeMerged  Scope = "merged" // 物化后的有效配置（仅 export 用，不回写）
)

// layerRank 返回层级优先级（值越小优先级越高），concat/遮蔽排序依据。
// 优先级：managed > local > project > remote > global（IR-SCHEMA §1.2，具体赢通用）。
func layerRank(s Scope) int {
	switch s {
	case ScopeManaged:
		return 0
	case ScopeLocal:
		return 1
	case ScopeProject:
		return 2
	case ScopeRemote:
		return 3
	case ScopeGlobal:
		return 4
	default:
		return 5
	}
}

// LayerRank 暴露层级比较，供 profile 包做 concat 排序（global→remote→project→local）。
func LayerRank(s Scope) int { return layerRank(s) }

// EntityKind 八类实体（IR-SCHEMA §1.3 词表；CLI --type 取值同构）。
type EntityKind string

const (
	KindInstruction EntityKind = "instruction"
	KindMCP         EntityKind = "mcp"
	KindSkill       EntityKind = "skill"
	KindAgent       EntityKind = "agent"
	KindCommand     EntityKind = "command"
	KindWorkflow    EntityKind = "workflow"
	KindHook        EntityKind = "hook"
	KindSetting     EntityKind = "setting"
)

// Activation 统一激活模型（IR-SCHEMA §3.1/§3.3，D4）。
type Activation string

const (
	ActivationAlways        Activation = "always"
	ActivationModelDecision Activation = "model-decision"
	ActivationGlob          Activation = "glob"
	ActivationManual        Activation = "manual"
	ActivationScene         Activation = "scene"
)

// Origin 来源追踪（IR-SCHEMA §1.1）。Path 一律记录 raw 变量形态
// （~/ 或 %APPDATA% 等），保证跨机可移植。
type Origin struct {
	Tool        string    `yaml:"tool"`
	Path        string    `yaml:"path"`
	Scope       Scope     `yaml:"scope"`
	CollectedAt time.Time `yaml:"collected_at"`
	RawHash     string    `yaml:"raw_hash"`           // 源文件原始字节 sha256（增量比对）
	StoredHash  string    `yaml:"stored_hash"`        // 脱敏后入库内容 sha256
	RawBlob     string    `yaml:"raw_blob,omitempty"` // 可选：源文件整体快照（overlay 兜底）
}

// Header 所有实体的公共头。Extensions 承载 x-<tool> 透传字段
// （只读规则见 IR-SCHEMA §1.1：异构采集不触碰他工具扩展位）。
type Header struct {
	ID         string         `yaml:"id"` // <type>.<name>，首个点号分隔 type
	IRVersion  int            `yaml:"ir_version"`
	Origin     *Origin        `yaml:"origin,omitempty"`
	Tombstone  bool           `yaml:"tombstone,omitempty"` // 墓碑（判定前提见 IR-SCHEMA §2.3）
	Extensions map[string]any `yaml:"-"`                   // x-<tool>；序列化时展开
}

// GetHeader 返回实体头（供 profile 合并泛型统一访问 ID/Tombstone/Extensions）。
func (h *Header) GetHeader() *Header { return h }

// Instruction 指令/记忆（IR-SCHEMA §3.1）。
type Instruction struct {
	Header          `yaml:",inline"`
	Name            string     `yaml:"name,omitempty"`
	Description     string     `yaml:"description,omitempty"`
	Activation      Activation `yaml:"activation,omitempty"`
	AppliesTo       []string   `yaml:"applies_to,omitempty"` // 缺省 = [origin.tool]
	FilePatterns    []string   `yaml:"file_patterns,omitempty"`
	Subtree         string     `yaml:"subtree,omitempty"`
	Priority        int        `yaml:"priority,omitempty"` // 默认 global=100/project=200/local=200/remote=150
	Language        string     `yaml:"language,omitempty"`
	Imports         []Import   `yaml:"imports,omitempty"`
	RoundtripPolicy string     `yaml:"roundtrip_policy,omitempty"` // preserve | inline
	Body            string     `yaml:"-"`                          // frontmatter 之后的 Markdown 正文
}

// Import 承载 CLAUDE.md 的 @path 引用（IR-SCHEMA §3.1，B6）。
type Import struct {
	Path     string `yaml:"path"`
	Blob     string `yaml:"blob,omitempty"`
	Resolved bool   `yaml:"resolved"`
}

// MCPServer（IR-SCHEMA §3.2，D7 字段扩充）。
type MCPServer struct {
	Header        `yaml:",inline"`
	Name          string            `yaml:"name"`      // 目标工具内原始键名（导出键名唯一来源）
	Transport     string            `yaml:"transport"` // stdio | sse | http | ws
	Command       string            `yaml:"command,omitempty"`
	Args          []string          `yaml:"args,omitempty"`
	Cwd           string            `yaml:"cwd,omitempty"`
	Env           map[string]string `yaml:"env,omitempty"` // 敏感值为 secretref://；间接寻址 "{env:VAR}"
	EnvFile       string            `yaml:"env_file,omitempty"`
	URL           string            `yaml:"url,omitempty"`
	Headers       map[string]string `yaml:"headers,omitempty"`
	HeadersHelper string            `yaml:"headers_helper,omitempty"`
	Timeout       *Timeout          `yaml:"timeout,omitempty"`
	EnabledTools  []string          `yaml:"enabled_tools,omitempty"`
	DisabledTools []string          `yaml:"disabled_tools,omitempty"`
	Trust         *bool             `yaml:"trust,omitempty"`
	AutoApprove   []string          `yaml:"auto_approve,omitempty"`
	OAuth         map[string]any    `yaml:"oauth,omitempty"`    // 密钥值一律 secretref
	Disabled      bool              `yaml:"disabled,omitempty"` // codex enabled 正极性由适配器取反
	PerMachine    bool              `yaml:"per_machine,omitempty"`
}

// Timeout MCP 超时（gemini 特有语义不硬映射，进 x-gemini）。
type Timeout struct {
	StartupMs int `yaml:"startup_ms,omitempty"`
	ToolSec   int `yaml:"tool_sec,omitempty"`
}

// PromptPack Skill/Agent/Command/Workflow 的统一承载（IR-SCHEMA §3.3，D5/D14）。
type PromptPack struct {
	Header        `yaml:",inline"`
	Kind          EntityKind  `yaml:"kind"`
	Name          string      `yaml:"name"`
	Description   string      `yaml:"description,omitempty"`
	Activation    Activation  `yaml:"activation,omitempty"`
	Invocation    string      `yaml:"invocation,omitempty"` // 调用名（slash-command 形态）
	FilePatterns  []string    `yaml:"file_patterns,omitempty"`
	Scene         string      `yaml:"scene,omitempty"`
	Model         any         `yaml:"model,omitempty"` // string 或 []string
	Tools         []string    `yaml:"tools,omitempty"`
	MCPServers    []string    `yaml:"mcp_servers,omitempty"` // 引用 mcp.* id
	UserInvocable *bool       `yaml:"user_invocable,omitempty"`
	ArgumentHint  string      `yaml:"argument_hint,omitempty"`
	Parameters    []Parameter `yaml:"parameters,omitempty"` // Workflow 专用（D14）
	Steps         []Step      `yaml:"steps,omitempty"`      // Workflow 专用（D14）
	Body          string      `yaml:"-"`                    // prompt.md 正文
}

// Parameter Workflow 参数（Goose recipes 公共面）。
type Parameter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty"`
	Default     string `yaml:"default,omitempty"`
}

// Step Workflow 有序步骤；复杂编排（子配方/重试）进 x-。
type Step struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run,omitempty"`
}

// HookEvent 标准 hook 事件词表（IR-SCHEMA §3.4 标准交集 + §5 校验规则 12）。
// 工具特有事件不进此词表，保留于 x-<tool>。
type HookEvent string

const (
	HookSessionStart     HookEvent = "session-start"
	HookSessionEnd       HookEvent = "session-end"
	HookPreToolUse       HookEvent = "pre-tool-use"
	HookPostToolUse      HookEvent = "post-tool-use"
	HookNotification     HookEvent = "notification"
	HookStop             HookEvent = "stop"
	HookUserPromptSubmit HookEvent = "user-prompt-submit"
	HookPreCompact       HookEvent = "pre-compact"
)

// Hook 事件钩子（IR-SCHEMA §3.4，D3）。
type Hook struct {
	Header  `yaml:",inline"`
	Event   HookEvent   `yaml:"event"` // 标准词表内取值；工具特有事件进 x-
	Matcher HookMatcher `yaml:"matcher,omitempty"`
	Handler HookHandler `yaml:"handler"`
}

// HookMatcher 事件匹配器。
type HookMatcher struct {
	Tool    string `yaml:"tool,omitempty"`
	Pattern string `yaml:"pattern,omitempty"`
}

// HookHandler 处理器；CommandWindows 跨平台双命令（cfg4ai 核心场景）。
type HookHandler struct {
	Type           string `yaml:"type"` // command | http | prompt | mcp_tool | agent
	Command        string `yaml:"command,omitempty"`
	CommandWindows string `yaml:"command_windows,omitempty"`
	URL            string `yaml:"url,omitempty"`
	Prompt         string `yaml:"prompt,omitempty"`
	TimeoutSec     int    `yaml:"timeout_sec,omitempty"`
}

// SettingEntry 工具自身设置条目（IR-SCHEMA §3.5，三段式 id：setting.<tool>.<key>）。
type SettingEntry struct {
	Header    `yaml:",inline"`
	Key       string   `yaml:"key"` // 目标文件内点号路径
	Value     any      `yaml:"value"`
	ToolScope []string `yaml:"tool_scope,omitempty"`
}

// Warning 管线告警（非空 → CLI 退出码 5，CLI-SPEC §0）。
type Warning struct {
	Kind    string `yaml:"kind"` // degrade | skip | secret | drift | ...
	Entity  string `yaml:"entity,omitempty"`
	Message string `yaml:"message"`
}

// CurrentVersion 当前 IR 结构版本（权威定义；采集器实体必填，IR-SCHEMA §4.1）。
const CurrentVersion = 1

// Bundle 迁移管线的内存模型（IR-SCHEMA §4）。merged 仅用于 export，不回写。
type Bundle struct {
	IRVersion    int
	Scope        Scope
	Instructions []Instruction
	MCPServers   []MCPServer
	Skills       []PromptPack
	Agents       []PromptPack
	Commands     []PromptPack
	Workflows    []PromptPack
	Hooks        []Hook
	Settings     []SettingEntry
	Warnings     []Warning

	// MCPFileExtensions 承载 mcp.yaml 顶层 file_extensions 键（IR-SCHEMA §3.2：
	// 文件级数据如 VS Code inputs[]/sandbox{}，不挂到单个 server 的 x- 下）。
	MCPFileExtensions map[string]any

	index map[string]EntityKind // merge-by-id 热路径索引（IR-SCHEMA §4）
}

// Add 将实体登记进 id 索引（采集/合并时调用）。
func (b *Bundle) Add(kind EntityKind, id string) {
	if b.index == nil {
		b.index = make(map[string]EntityKind)
	}
	b.index[id] = kind
}

// Lookup 按 id 查询实体类别；ok=false 表示未登记。
func (b *Bundle) Lookup(id string) (EntityKind, bool) {
	k, ok := b.index[id]
	return k, ok
}
