package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var backupPassphrase string
var backupStrategy string

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "加密备份包导出/导入（.cfg4aibak，age 口令加密）",
}

var backupExportCmd = &cobra.Command{
	Use:   "export <文件>",
	Short: "导出加密备份包",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		pass := backupPassphrase
		if pass == "" {
			pass = os.Getenv("CFG4AI_BACKUP_PASSPHRASE")
		}
		if pass == "" {
			return exitErr(1, "缺少口令：--passphrase 或环境变量 CFG4AI_BACKUP_PASSPHRASE（至少 8 位）")
		}
		n, err := repo.ExportBackup(args[0], pass)
		if err != nil {
			return exitErr(1, "%v", err)
		}
		fmt.Printf("已导出备份包 %s（%d 文件，age 加密）\n", args[0], n)
		return nil
	},
}

var backupImportCmd = &cobra.Command{
	Use:   "import <文件>",
	Short: "导入加密备份包",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		pass := backupPassphrase
		if pass == "" {
			pass = os.Getenv("CFG4AI_BACKUP_PASSPHRASE")
		}
		if pass == "" {
			return exitErr(1, "缺少口令：--passphrase 或环境变量 CFG4AI_BACKUP_PASSPHRASE")
		}
		n, err := repo.ImportBackup(args[0], pass, backupStrategy)
		if err != nil {
			return exitErr(1, "%v", err)
		}
		fmt.Printf("已导入 %s（%d 文件，策略 %s）\n", args[0], n, backupStrategy)
		return nil
	},
}

func init() {
	backupExportCmd.Flags().StringVar(&backupPassphrase, "passphrase", "", "备份口令（≥8 位）")
	backupImportCmd.Flags().StringVar(&backupPassphrase, "passphrase", "", "备份口令")
	backupImportCmd.Flags().StringVar(&backupStrategy, "strategy", "merge", "冲突策略 skip|overwrite|merge")
	backupCmd.AddCommand(backupExportCmd, backupImportCmd)
	rootCmd.AddCommand(backupCmd)
}
