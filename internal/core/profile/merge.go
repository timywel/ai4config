package profile

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/timywel/ai4config/internal/core/ir"
)

// ScopedBundle 一个 scope 层的配置集合（物化输入）。
type ScopedBundle struct {
	Scope    ir.Scope
	Bundle   *ir.Bundle
	Manifest *Manifest // 该层的 manifest（含 merge_policy、ir_version）
}

// MergeBundles 把多个 scope 的 Bundle 物化为一个 merged Bundle（IR-SCHEMA §2/§4）。
//
// 语义：
//   - 层序：按优先级从低到高处理（global→remote→project→local）；managed 层不物化（自动剔除）。
//   - merge-by-id 实体（mcp/skill/agent/command/workflow/hook/setting）：浅字段级合并——
//     同 id 时高层键覆盖低层（数组整体替换、未出现键继承），x- 扩展同键覆盖。
//   - instructions：concat 两段式——层级（低优先级在前）→ priority 升序 → origin.path 字典序。
//   - 墓碑遮蔽：高层墓碑遮蔽低层同 id 条目；merged 结果不含墓碑（规则 10）。
//   - merged 条目的 origin 取胜出层（对 merge-by-id）；concat 条目保留各自 origin。
func MergeBundles(layers ...*ScopedBundle) *ir.Bundle {
	// 1. 剔除 managed 层，按 rank 从高到低... 不，物化从低优先级（rank 大）到高优先级（rank 小）
	//    排序：rank 降序（global=4 在前 → remote=3 → project=2 → local=1）
	usable := make([]*ScopedBundle, 0, len(layers))
	for _, l := range layers {
		if l == nil || l.Bundle == nil {
			continue
		}
		if l.Scope == ir.ScopeManaged { // managed 不物化
			continue
		}
		usable = append(usable, l)
	}
	sort.SliceStable(usable, func(i, j int) bool {
		return ir.LayerRank(usable[i].Scope) > ir.LayerRank(usable[j].Scope) // rank 大（低优先级）在前
	})

	out := &ir.Bundle{Scope: ir.ScopeMerged, IRVersion: currentIRVersion()}

	// 2. merge-by-id 各类实体
	out.MCPServers = mergeByID(collect(usable, func(b *ir.Bundle) []ir.MCPServer { return b.MCPServers }))
	out.Skills = mergeByID(collect(usable, func(b *ir.Bundle) []ir.PromptPack { return b.Skills }))
	out.Agents = mergeByID(collect(usable, func(b *ir.Bundle) []ir.PromptPack { return b.Agents }))
	out.Commands = mergeByID(collect(usable, func(b *ir.Bundle) []ir.PromptPack { return b.Commands }))
	out.Workflows = mergeByID(collect(usable, func(b *ir.Bundle) []ir.PromptPack { return b.Workflows }))
	out.Hooks = mergeByID(collect(usable, func(b *ir.Bundle) []ir.Hook { return b.Hooks }))
	out.Settings = mergeByID(collect(usable, func(b *ir.Bundle) []ir.SettingEntry { return b.Settings }))

	// 3. instructions：concat 两段式
	out.Instructions = concatInstructions(collect(usable, func(b *ir.Bundle) []ir.Instruction { return b.Instructions }))

	// 4. 合并过程不产生 Warnings（降级在 export 阶段判定）；MCPFileExtensions 取最高优先级非空层
	for i := len(usable) - 1; i >= 0; i-- { // usable 末尾是最高优先级层
		if len(usable[i].Bundle.MCPFileExtensions) > 0 {
			out.MCPFileExtensions = usable[i].Bundle.MCPFileExtensions
			break
		}
	}

	return out
}

// ranked 携带层优先级的实体集合。
type ranked[T any] struct {
	rank  int
	items []T
}

// collect 把各层中某类实体连同其层 rank 收集出来（usable 已按低优先级在前排序）。
func collect[T any](usable []*ScopedBundle, pick func(*ir.Bundle) []T) []ranked[T] {
	out := make([]ranked[T], 0, len(usable))
	for _, l := range usable {
		out = append(out, ranked[T]{rank: ir.LayerRank(l.Scope), items: pick(l.Bundle)})
	}
	return out
}

// headerCarrier 泛型约束：指向含 Header 的实体。
type headerCarrier[T any] interface {
	*T
	GetHeader() *ir.Header
}

// tombstoneRanks 收集各墓碑 id 的最高优先级（最小 rank）。
func tombstoneRanks[T any, PT headerCarrier[T]](layers []ranked[T]) map[string]int {
	set := map[string]int{}
	for _, l := range layers {
		for i := range l.items {
			h := PT(&l.items[i]).GetHeader()
			if h.Tombstone {
				if r, ok := set[h.ID]; !ok || l.rank < r {
					set[h.ID] = l.rank
				}
			}
		}
	}
	return set
}

// mergeByID 浅字段级合并（IR-SCHEMA §2.1 field-level-shallow）。
// layers 传入时已是"低优先级在前"，遍历时高层覆盖低层。
func mergeByID[T any, PT headerCarrier[T]](layers []ranked[T]) []T {
	tomb := tombstoneRanks[T, PT](layers)
	merged := map[string]T{}
	var order []string

	for _, l := range layers { // 低优先级在前
		for i := range l.items {
			p := PT(&l.items[i])
			h := p.GetHeader()
			if h.ID == "" {
				continue
			}
			if h.Tombstone {
				continue // 墓碑不进 merged
			}
			if tr, ok := tomb[h.ID]; ok && l.rank > tr {
				continue // 被更高优先级层的墓碑遮蔽（防"删除复活"）
			}
			if existing, ok := merged[h.ID]; ok {
				merged[h.ID] = shallowMerge[T, PT](existing, l.items[i]) // 高层覆盖低层
			} else {
				merged[h.ID] = l.items[i]
				order = append(order, h.ID)
			}
		}
	}

	out := make([]T, 0, len(order))
	for _, id := range order {
		out = append(out, merged[id])
	}
	return out
}

// concatInstructions 指令两段式拼接（IR-SCHEMA §2.1 layer-ordered）：
// 层级（低优先级在前）→ priority 升序 → origin.path 字典序；不跨层混排。
func concatInstructions(layers []ranked[ir.Instruction]) []ir.Instruction {
	tomb := tombstoneRanks[ir.Instruction](layers)

	type item struct {
		rank int
		inst ir.Instruction
	}
	var picked []item
	for _, l := range layers {
		for _, inst := range l.items {
			h := &inst.Header
			if h.ID == "" || h.Tombstone {
				continue
			}
			if tr, ok := tomb[h.ID]; ok && l.rank > tr {
				continue // 遮蔽
			}
			picked = append(picked, item{rank: l.rank, inst: inst})
		}
	}

	sort.SliceStable(picked, func(i, j int) bool {
		a, b := picked[i], picked[j]
		if a.rank != b.rank {
			return a.rank > b.rank // rank 大（低优先级）在前
		}
		if a.inst.Priority != b.inst.Priority {
			return a.inst.Priority < b.inst.Priority // priority 小在前
		}
		return originPath(&a.inst) < originPath(&b.inst) // path 字典序
	})

	out := make([]ir.Instruction, 0, len(picked))
	for _, p := range picked {
		out = append(out, p.inst)
	}
	return out
}

// shallowMerge 浅字段级合并：高层（higher）键覆盖低层（lower）。
// 标量/object/数组键均整体替换；未出现键继承低层；x- 扩展同键覆盖、异键合并。
func shallowMerge[T any, PT headerCarrier[T]](lower, higher T) T {
	lm := toStringMap(lower)
	hm := toStringMap(higher)
	for k, v := range hm {
		lm[k] = v // 键级覆盖（数组/object 整体替换，不递归）
	}
	var out T
	fromStringMap(lm, &out)

	// x- 扩展（yaml:"-" 不参与 map 中转，单独合并）
	lh := PT(&lower).GetHeader()
	hh := PT(&higher).GetHeader()
	oh := PT(&out).GetHeader()
	oh.Extensions = mergeExtensions(lh.Extensions, hh.Extensions)
	// origin 取胜出层（higher，高层）；若 higher 无 origin 则继承 lower
	if oh.Origin == nil {
		oh.Origin = lh.Origin
	}
	return out
}

// mergeExtensions 合并 x- 扩展：同键高层覆盖，异键保留。
func mergeExtensions(lower, higher map[string]any) map[string]any {
	if len(lower) == 0 && len(higher) == 0 {
		return nil
	}
	out := map[string]any{}
	for k, v := range lower {
		out[k] = v
	}
	for k, v := range higher {
		out[k] = v
	}
	return out
}

// toStringMap / fromStringMap：实体经 YAML 中转成 map 以做键级浅合并。
func toStringMap[T any](v T) map[string]any {
	b, err := yaml.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	if err := yaml.Unmarshal(b, &m); err != nil {
		return map[string]any{}
	}
	if m == nil {
		m = map[string]any{}
	}
	return m
}

func fromStringMap[T any](m map[string]any, out *T) error {
	b, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("profile: 浅合并重编码失败: %w", err)
	}
	if err := yaml.Unmarshal(b, out); err != nil {
		return fmt.Errorf("profile: 浅合并解码失败: %w", err)
	}
	return nil
}

func originPath(i *ir.Instruction) string {
	if i.Origin == nil {
		return ""
	}
	return i.Origin.Path
}

func currentIRVersion() int { return CurrentIRVersion }
