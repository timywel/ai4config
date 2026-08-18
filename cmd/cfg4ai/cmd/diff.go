package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timywel/ai4config/internal/adapters"
	_ "github.com/timywel/ai4config/internal/adapters/all"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/store"
)

var diffTool string

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "对比 SSOT 与磁盘现状（--tool）",
	RunE: func(cmd *cobra.Command, args []string) error {
		if diffTool == "" {
			return exitErr(2, "缺少 --tool <id>（profile 间对比待 P2）")
		}
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		sb, err := profile.Load(repo.Path(store.DirProfiles, "global"), ir.ScopeGlobal)
		if err != nil {
			return exitErr(1, "加载 global profile 失败（先 collect？）: %v", err)
		}

		// SSOT 中该工具的条目
		ssot := map[string]ir.MCPServer{}
		for _, m := range sb.Bundle.MCPServers {
			if m.Origin != nil && m.Origin.Tool == diffTool {
				ssot[m.Name] = m
			}
		}

		// 磁盘现状（重新 Import）
		a, ok := adapters.Get(adapters.ToolID(diffTool))
		if !ok {
			return exitErr(1, "未注册适配器 %q", diffTool)
		}
		locs, _ := a.Detect(cmd.Context())
		disk := map[string]ir.MCPServer{}
		for _, loc := range locs {
			b, err := a.Import(cmd.Context(), loc)
			if err != nil {
				continue
			}
			for _, m := range b.MCPServers {
				disk[m.Name] = m
			}
		}

		// 对比
		w := table()
		fmt.Fprintln(w, "STATUS\tMCP\tDETAIL")
		diff := 0
		for name, s := range ssot {
			d, ok := disk[name]
			if !ok {
				fmt.Fprintf(w, "仅SSOT\t%s\t磁盘无此 server\n", name)
				diff++
				continue
			}
			if s.Command != d.Command || s.URL != d.URL || s.Transport != d.Transport {
				fmt.Fprintf(w, "已变更\t%s\tSSOT(cmd=%s url=%s) vs 磁盘(cmd=%s url=%s)\n", name, s.Command, s.URL, d.Command, d.URL)
				diff++
			}
		}
		for name := range disk {
			if _, ok := ssot[name]; !ok {
				fmt.Fprintf(w, "仅磁盘\t%s\tSSOT 未采集\n", name)
				diff++
			}
		}
		w.Flush()
		if diff == 0 {
			fmt.Println("无差异")
		}
		return nil
	},
}

func init() {
	diffCmd.Flags().StringVar(&diffTool, "tool", "", "对比指定工具")
	rootCmd.AddCommand(diffCmd)
}
