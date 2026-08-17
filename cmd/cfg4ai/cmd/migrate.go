package cmd

import (
	"github.com/spf13/cobra"
)

var (
	migrateFrom    string
	migrateTo      string
	migrateProject string
	migrateAI      bool
	migrateDryRun  bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "一步到位迁移：collect(from) + export(to --include-foreign)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if migrateFrom == "" || migrateTo == "" {
			return exitErr(2, "缺少 --from 或 --to")
		}
		// 1. 采集源工具（W1）
		collectTools = []string{migrateFrom}
		collectScope = "all"
		if err := collectCmd.RunE(cmd, args); err != nil {
			return err
		}
		// 2. 导出到目标（W2，纳入异构条目）
		exportTo = migrateTo
		exportProject = migrateProject
		exportDryRun = migrateDryRun
		exportIncludeForeign = true
		exportAI = migrateAI
		return exportCmd.RunE(cmd, args)
	},
}

func init() {
	f := migrateCmd.Flags()
	f.StringVar(&migrateFrom, "from", "", "源工具 id（必需）")
	f.StringVar(&migrateTo, "to", "", "目标工具 id（必需）")
	f.StringVar(&migrateProject, "project", "", "项目路径")
	f.BoolVar(&migrateAI, "ai", false, "启用 AI 语义转换")
	f.BoolVar(&migrateDryRun, "dry-run", false, "只预览不落盘")
	rootCmd.AddCommand(migrateCmd)
}
