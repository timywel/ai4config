// Command cfg4ai 是 AI 编码工具配置的采集、治理与迁移系统。
// 命令面规范见 docs/CLI-SPEC.md（v0.3）。
package main

import (
	"os"

	"github.com/timywel/ai4config/cmd/cfg4ai/cmd"
)

// version 由发布流程注入（goreleaser -X main.version=...）。
var version = "0.0.1-dev"

func main() {
	cmd.SetVersion(version)
	os.Exit(cmd.Execute())
}
