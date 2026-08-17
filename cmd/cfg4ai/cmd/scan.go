package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timywel/ai4config/internal/adapters"
	_ "github.com/timywel/ai4config/internal/adapters/all" // 注册全部适配器
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/store"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "发现本机已安装的 AI 工具及其配置位置（只读）",
	RunE: func(cmd *cobra.Command, args []string) error {
		type row struct {
			Tool   string `json:"tool"`
			Scope  string `json:"scope"`
			Path   string `json:"path"`
			Status string `json:"status"`
		}
		var rows []row

		for _, a := range adapters.List() {
			locs, err := a.Detect(cmd.Context())
			if err != nil {
				continue
			}
			for _, loc := range locs {
				rows = append(rows, row{
					Tool:   string(a.Meta().ID),
					Scope:  string(loc.Scope),
					Path:   loc.Root,
					Status: collectStatus(string(a.Meta().ID), loc.Root),
				})
			}
		}

		if isJSON() {
			output(rows)
			return nil
		}
		if len(rows) == 0 {
			fmt.Println("未发现已接入工具的任意配置位置")
			return nil
		}
		w := table()
		fmt.Fprintln(w, "TOOL\tSCOPE\tPATH\tSTATUS")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Tool, r.Scope, r.Path, r.Status)
		}
		w.Flush()
		return nil
	},
}

// collectStatus 判断该位置是否已采集（查 global profile 有无该 tool 来源条目）。
func collectStatus(tool, root string) string {
	repo, err := openRepo()
	if err != nil {
		return "new"
	}
	sb, err := profile.Load(repo.Path(store.DirProfiles, "global"), ir.ScopeGlobal)
	if err != nil {
		return "new"
	}
	for _, s := range sb.Bundle.Settings {
		if s.Origin != nil && s.Origin.Tool == tool {
			return "collected"
		}
	}
	for _, m := range sb.Bundle.MCPServers {
		if m.Origin != nil && m.Origin.Tool == tool {
			return "collected"
		}
	}
	return "new"
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
