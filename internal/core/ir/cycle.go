package ir

// DetectImportCycle 对 imports 引用图做 DFS 三色检环（IR-SCHEMA §5 规则 9）。
//
// refs：节点（文件路径）→ 它引用的路径列表。图论域为**全局引用图**
// （对抗用例 AC-A1 直接环、AC-E3 跨工具 CLAUDE.md↔AGENTS.md 互相引用环）。
// 返回环上的路径序列（无环返回 nil）。
func DetectImportCycle(refs map[string][]string) []string {
	const (
		white = 0 // 未访问
		gray  = 1 // 在当前 DFS 栈上
		black = 2 // 已完成
	)
	color := map[string]int{}
	var stack []string
	var cycle []string

	var visit func(p string) bool
	visit = func(p string) bool {
		color[p] = gray
		stack = append(stack, p)
		for _, next := range refs[p] {
			switch color[next] {
			case gray: // 回边 → 环
				idx := -1
				for i, s := range stack {
					if s == next {
						idx = i
						break
					}
				}
				if idx >= 0 {
					cycle = append([]string{}, stack[idx:]...)
					cycle = append(cycle, next) // 闭合
				}
				return true
			case white:
				if visit(next) {
					return true
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[p] = black
		return false
	}

	for p := range refs {
		if color[p] == white {
			if visit(p) {
				return cycle
			}
		}
	}
	return nil
}

// InstructionImportRefs 从一组 Instruction 构建全局引用图（path → imports）。
// 节点用 origin.path（缺失时用 id）；imports 的相对 path 归一化为引用目标。
func InstructionImportRefs(instructions []Instruction) map[string][]string {
	refs := map[string][]string{}
	for _, inst := range instructions {
		node := inst.ID
		if inst.Origin != nil && inst.Origin.Path != "" {
			node = inst.Origin.Path
		}
		var targets []string
		for _, imp := range inst.Imports {
			targets = append(targets, imp.Path)
		}
		refs[node] = targets
	}
	return refs
}
