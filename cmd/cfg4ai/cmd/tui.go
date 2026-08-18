package cmd

import (
	"github.com/spf13/cobra"

	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/store"
	"github.com/timywel/ai4config/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "交互式终端界面（浏览已采集实体）",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		sb, err := profile.Load(repo.Path(store.DirProfiles, "global"), ir.ScopeGlobal)
		if err != nil {
			return exitErr(1, "加载 profile 失败（先 collect？）: %v", err)
		}
		var items []tui.Item
		b := sb.Bundle
		for _, x := range b.Instructions {
			items = append(items, tui.Item{Kind: "instruction", ID: x.ID, Note: x.Description})
		}
		for _, x := range b.MCPServers {
			items = append(items, tui.Item{Kind: "mcp", ID: x.ID, Note: x.Command + x.URL})
		}
		for _, x := range b.Skills {
			items = append(items, tui.Item{Kind: "skill", ID: x.ID, Note: x.Description})
		}
		for _, x := range b.Agents {
			items = append(items, tui.Item{Kind: "agent", ID: x.ID, Note: x.Description})
		}
		for _, x := range b.Hooks {
			items = append(items, tui.Item{Kind: "hook", ID: x.ID, Note: string(x.Event)})
		}
		for _, x := range b.Settings {
			items = append(items, tui.Item{Kind: "setting", ID: x.ID, Note: x.Key})
		}
		if len(items) == 0 {
			return exitErr(1, "SSOT 为空（先 cfg4ai collect 采集）")
		}
		return tui.RunBrowser("cfg4ai — 已采集实体（global profile）", items)
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
