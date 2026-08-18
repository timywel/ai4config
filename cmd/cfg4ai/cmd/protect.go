package cmd

import (
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/secrets"
)

// protectSecrets 回采保护（IR-SCHEMA §3.6、红队 T-03）：
// reconcile 后，若某 MCP 字段被新采集的空值/占位符覆盖，而既有同 id 条目对应值是
// secretref，则恢复 secretref——防止 none 后端导出占位回采后把真实引用冲掉。
func protectSecrets(result, existing *ir.Bundle) {
	byID := map[string]*ir.MCPServer{}
	for i := range existing.MCPServers {
		byID[existing.MCPServers[i].ID] = &existing.MCPServers[i]
	}
	for i := range result.MCPServers {
		ex, ok := byID[result.MCPServers[i].ID]
		if !ok {
			continue
		}
		result.MCPServers[i].Env = protectMap(result.MCPServers[i].Env, ex.Env)
		result.MCPServers[i].Headers = protectMap(result.MCPServers[i].Headers, ex.Headers)
	}
}

// protectMap 字段级保护：new 中占位/空值，若 old 对应键是 secretref 则恢复。
func protectMap(new, old map[string]string) map[string]string {
	if len(new) == 0 || len(old) == 0 {
		return new
	}
	for k, nv := range new {
		if ov, ok := old[k]; ok && secrets.IsSecretRef(ov) && secrets.IsPlaceholder(nv) {
			new[k] = ov
		}
	}
	return new
}
