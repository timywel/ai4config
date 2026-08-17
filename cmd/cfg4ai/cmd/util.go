package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
	"github.com/timywel/ai4config/internal/store"
)

// openRepo 按 --home 或默认位置打开 SSOT 仓库。
func openRepo() (*store.Repo, error) {
	root := flagHome
	if root == "" {
		var err error
		root, err = paths.DataHome()
		if err != nil {
			return nil, err
		}
	}
	return store.Open(root)
}

// output 按 --format 输出结果（text=表格/人类可读，json=结构化）。
func output(v any) {
	if flagFormat == "json" {
		data, _ := json.MarshalIndent(v, "", "  ")
		fmt.Println(string(data))
	}
	// text 由各命令自行打印（output 只处理 json 快捷路径）
}

// isJSON 当前是否 json 输出。
func isJSON() bool { return flagFormat == "json" }

// table 创建 tabwriter（text 输出对齐表格）。
func table() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
}

// warnExit 有警告时返回退出码 5（部分成功，CLI-SPEC §0）。
func warnExit(warnings []ir.Warning) error {
	if len(warnings) == 0 {
		return nil
	}
	fmt.Fprintf(os.Stderr, "\n%d 条警告：\n", len(warnings))
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "  [%s] %s %s\n", w.Kind, w.Entity, w.Message)
	}
	return &ExitError{Code: 5, Err: fmt.Errorf("部分成功：%d 条警告", len(warnings))}
}
