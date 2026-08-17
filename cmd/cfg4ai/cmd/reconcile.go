package cmd

import (
	"github.com/timywel/ai4config/internal/core/ir"
)

// entityPtr 泛型约束：指向含 Header 的实体。
type entityPtr[T any] interface {
	*T
	GetHeader() *ir.Header
}

// reconcileList 采集再合并（IR-SCHEMA §2.1 reconcile + §2.3 墓碑遮蔽）：
//   - 同 id：整体用新采集值更新（reconcile 粒度为整条，IR-SCHEMA §2.1）；
//   - 既有有、新采集无、且来源（origin.tool+path）本次确实采集了 → 标墓碑；
//   - 既有有、来源本次未采集 → 原样保留（不标墓碑，防误判）。
//
// 返回 (merged, 新增, 更新, 墓碑)。
func reconcileList[T any, PT entityPtr[T]](existing, fresh []T) ([]T, int, int, int) {
	freshByID := map[string]int{} // id -> fresh 索引
	freshSources := map[string]bool{}
	for i := range fresh {
		h := PT(&fresh[i]).GetHeader()
		freshByID[h.ID] = i
		if h.Origin != nil {
			freshSources[h.Origin.Tool+"|"+h.Origin.Path] = true
		}
	}

	added, updated, tombstoned := 0, 0, 0
	out := make([]T, 0, len(existing)+len(fresh))
	inOut := map[string]bool{}

	// 处理既有条目
	for i := range existing {
		h := PT(&existing[i]).GetHeader()
		if idx, ok := freshByID[h.ID]; ok {
			out = append(out, fresh[idx]) // 同 id 整体更新
			inOut[h.ID] = true
			updated++
			continue
		}
		// fresh 无此 id：判定来源是否本次采集
		if h.Origin != nil && freshSources[h.Origin.Tool+"|"+h.Origin.Path] {
			if !h.Tombstone { // 该来源本次采了但此条目消失 → 墓碑
				e := existing[i]
				PT(&e).GetHeader().Tombstone = true
				out = append(out, e)
				inOut[h.ID] = true
				tombstoned++
				continue
			}
		}
		out = append(out, existing[i]) // 保留（含既有墓碑、未采集来源条目）
		inOut[h.ID] = true
	}

	// fresh 新增
	for i := range fresh {
		h := PT(&fresh[i]).GetHeader()
		if !inOut[h.ID] {
			out = append(out, fresh[i])
			added++
		}
	}

	return out, added, updated, tombstoned
}
