// Command cfg4ai 是 AI 编码工具配置的采集、治理与迁移系统。
// 命令面规范见 docs/CLI-SPEC.md（v0.3）。
package main

import (
	"fmt"
	"io"
	"os"
)

// version 由发布流程注入（goreleaser -X main.version=...）。
var version = "0.0.1-dev"

func main() { os.Exit(run(os.Args[1:], os.Stdout)) }

// run 为可测试的命令入口（返回进程退出码，语义见 CLI-SPEC §0）。
// TODO(P0/T10): 接入 cobra 命令树后替换为 cobra 的 Execute 封装。
func run(args []string, stdout io.Writer) int {
	if len(args) > 0 && (args[0] == "--version" || args[0] == "-v") {
		fmt.Fprintln(stdout, "cfg4ai", version)
		return 0
	}
	fmt.Fprintln(stdout, "cfg4ai — AI 编码工具配置的采集、治理与迁移系统（P0 开发骨架）")
	fmt.Fprintln(stdout, "命令面见 docs/CLI-SPEC.md；--version 查看版本")
	return 2 // 未实现命令按规范返回用法错误
}
