// Package cmd 是 cfg4ai 的 cobra 命令树（CLI-SPEC v0.3）。
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	version = "dev"

	flagHome           string
	flagFormat         string
	flagYes            bool
	flagVerbose        bool
	flagQuiet          bool
	flagNoAI           bool
	flagSecretsBackend string
)

// SetVersion 由 main 注入版本（goreleaser -X）。
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

// ExitError 携带退出码的错误（CLI-SPEC §0：0/1/2/3/4/5）。
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }

// exitErr 构造带退出码的错误。
func exitErr(code int, format string, args ...any) *ExitError {
	return &ExitError{Code: code, Err: fmt.Errorf(format, args...)}
}

var rootCmd = &cobra.Command{
	Use:     "cfg4ai",
	Version: version, // cobra 自动提供 --version
	Short:   "AI 编码工具配置的采集、治理与迁移系统",
	Long: `cfg4ai 采集各 AI 编码工具（Claude Code、Codex、Copilot、Zhanlu、Gemini…）的
指令/MCP/skills/hooks 等配置入统一 SSOT 仓库，经中间表示（IR）无损互转并交付到任意已接入工具。

设计文档见 docs/（ARCHITECTURE / IR-SCHEMA / CLI-SPEC / ADAPTERS）。`,
	SilenceUsage:  true, // 退出码语义自定义（CLI-SPEC §0）
	SilenceErrors: true,
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&flagHome, "home", "", "覆盖 SSOT 仓库位置（等价环境变量 CFG4AI_HOME）")
	pf.StringVar(&flagFormat, "format", "text", "输出格式 text|json")
	pf.BoolVar(&flagYes, "yes", false, "跳过常规交互确认（不豁免 AI 确认/空集保护/sync 预检）")
	pf.BoolVar(&flagVerbose, "verbose", false, "详细日志")
	pf.BoolVar(&flagQuiet, "quiet", false, "静默")
	pf.BoolVar(&flagNoAI, "no-ai", false, "本次运行禁用 AI 辅助")
	pf.StringVar(&flagSecretsBackend, "secrets-backend", "", "覆盖 secret 后端 keyring|file|none")

	rootCmd.AddCommand(versionCmd)
}

// Execute 运行命令树，返回进程退出码（CLI-SPEC §0）。
func Execute() int {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	err := rootCmd.Execute()
	if err == nil {
		return 0
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		if ee.Code != 0 && ee.Code != 5 {
			fmt.Fprintln(os.Stderr, "错误:", ee.Err)
		} else if ee.Code == 5 {
			fmt.Fprintln(os.Stderr, ee.Err)
		}
		return ee.Code
	}
	// cobra 用法错误（未知命令/flag）→ 退出码 2（CLI-SPEC §0）
	if strings.Contains(err.Error(), "unknown command") || strings.Contains(err.Error(), "unknown flag") ||
		strings.Contains(err.Error(), "unknown shorthand") {
		fmt.Fprintln(os.Stderr, "错误:", err)
		return 2
	}
	fmt.Fprintln(os.Stderr, "错误:", err)
	return 1
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "打印版本",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cfg4ai", version)
	},
}
