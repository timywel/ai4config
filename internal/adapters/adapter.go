// Package adapters 定义工具适配器接口与注册表。
// 权威规范：docs/ADAPTERS.md（v0.3）。适配器保持纯粹——只做格式双向转换，
// 不接触 AI（语义转换由 core/migrate 引擎层主导）与写文件通道（统一 internal/atomicfile）。
package adapters

import (
	"context"

	"github.com/timywel/ai4config/internal/core/ir"
)

// ToolID 已注册工具标识（如 "claude-code" | "codex" | "copilot" | "zhanlu" | "gemini" | ...）。
type ToolID string

// SupportLevel 能力支持程度（ADAPTERS §1）。
type SupportLevel int

const (
	SupportNone    SupportLevel = iota
	SupportPartial              // 配 Note 说明边界
	SupportFull
)

// Capability 单实体能力声明。
type Capability struct {
	Level SupportLevel
	Note  string
}

// CapabilitySet 能力矩阵：导出降级的判定依据。
type CapabilitySet map[ir.EntityKind]Capability

// ToolMeta 工具元数据（版本护栏：超出 [Min,Max] 告警继续）。
type ToolMeta struct {
	ID           ToolID
	DisplayName  string
	MinVersion   string
	MaxVersion   string
	Capabilities CapabilitySet
}

// Location 探测到的配置位置（Detect 只读，不得创建/修改任何文件）。
type Location struct {
	Scope   ir.Scope // 五层之一
	Root    string   // 配置根目录（绝对路径）
	Version string   // 探测到的工具版本（可为空）
	Running bool     // 目标进程运行中（best-effort，热重载提示依据）
}

// WrittenFile 导出写入清单条目（供引擎 Verify 与导出清单使用）。
type WrittenFile struct {
	Path string
	Hash string // 写出内容 sha256
}

// ExportOpts 导出选项。AIAssist 已移除（AI 由引擎层主导，见 ARCHITECTURE §5.1）。
type ExportOpts struct {
	ProjectRoot string // scope=project 时的目标项目根
	DryRun      bool
	Force       bool
}

// Adapter 工具适配器：每个工具一对 Import/Export（ADAPTERS §1/§2 十条实现规范）。
type Adapter interface {
	Meta() ToolMeta
	Detect(ctx context.Context) ([]Location, error)
	Import(ctx context.Context, loc Location) (*ir.Bundle, error)
	Export(ctx context.Context, b *ir.Bundle, opts ExportOpts) ([]WrittenFile, error)
}
