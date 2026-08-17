package adapters

import (
	"fmt"
	"sort"
	"sync"
)

// 适配器注册表。各适配器包在 init() 中调用 Register 注册；
// internal/adapters/all 聚合包集中 blank import，由 cmd/cfg4ai 引用。

var (
	mu       sync.RWMutex
	registry = map[ToolID]Adapter{}
)

// Register 注册适配器；重复注册同一 id 直接 panic（编程错误，启动即暴露）。
func Register(a Adapter) {
	mu.Lock()
	defer mu.Unlock()
	id := a.Meta().ID
	if _, dup := registry[id]; dup {
		panic(fmt.Sprintf("adapters: duplicate registration %q", id))
	}
	registry[id] = a
}

// Get 按 id 取适配器。
func Get(id ToolID) (Adapter, bool) {
	mu.RLock()
	defer mu.RUnlock()
	a, ok := registry[id]
	return a, ok
}

// List 返回全部已注册适配器（按 id 排序，输出稳定）。
func List() []Adapter {
	mu.RLock()
	defer mu.RUnlock()
	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	out := make([]Adapter, 0, len(ids))
	for _, id := range ids {
		out = append(out, registry[ToolID(id)])
	}
	return out
}
