package migrate

import (
	"context"
	"fmt"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/aiassist"
	"github.com/timywel/ai4config/internal/core/ir"
)

// assist AI 语义转换（ARCHITECTURE §5.1 管线 Assist 步骤，引擎层主导）。
// 职责：consent 校验（含配置变更重确认，红队 T-09）→ 对文本条目做语义改写。
// 适配器完全不感知 AI（保持纯粹可测）。
func (e *Engine) assist(ctx context.Context, b *ir.Bundle, target adapters.ToolID, req ExportRequest) (*ir.Bundle, error) {
	if e.AI == nil {
		return b, fmt.Errorf("migrate: 未配置 AI provider（config set ai.base_url / ai.provider）")
	}
	// consent 检查：首次使用或配置变更需确认；--ai-approve 为无人值守确认
	status := aiassist.CheckConsent(e.Repo.Root, e.AIConfig)
	if status != aiassist.ConsentOK {
		if !req.AIApprove {
			return nil, fmt.Errorf("migrate: AI 出域需确认（consent=%v）；确认请用 --ai-approve", status)
		}
		if err := aiassist.RecordConsent(e.Repo.Root, e.AIConfig); err != nil {
			return nil, err
		}
	}
	return b, nil
}
