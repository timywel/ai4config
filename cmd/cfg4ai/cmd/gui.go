package cmd

import (
	"github.com/spf13/cobra"

	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/gui"
	"github.com/timywel/ai4config/internal/store"
)

var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "本地 Web 界面（标准库 HTTP，零外部 GUI 依赖，浏览器打开）",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		provider := func() ([]gui.Entity, error) {
			sb, err := profile.Load(repo.Path(store.DirProfiles, "global"), ir.ScopeGlobal)
			if err != nil {
				return nil, err
			}
			var items []gui.Entity
			for _, x := range sb.Bundle.Instructions {
				items = append(items, gui.Entity{Kind: "instruction", ID: x.ID, Note: x.Description})
			}
			for _, x := range sb.Bundle.MCPServers {
				items = append(items, gui.Entity{Kind: "mcp", ID: x.ID, Note: x.Command + x.URL})
			}
			for _, x := range sb.Bundle.Skills {
				items = append(items, gui.Entity{Kind: "skill", ID: x.ID, Note: x.Description})
			}
			for _, x := range sb.Bundle.Settings {
				items = append(items, gui.Entity{Kind: "setting", ID: x.ID, Note: x.Key})
			}
			return items, nil
		}
		return gui.Run(repo.Root, provider)
	},
}

func init() {
	rootCmd.AddCommand(guiCmd)
}
