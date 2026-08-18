package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timywel/ai4config/internal/core/secrets"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "跨机同步（白名单 git 远端 + preflight 敏感扫描）",
}

var syncInitCmd = &cobra.Command{
	Use:   "init <git-remote-url>",
	Short: "初始化同步远端",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		if err := repo.SyncInit(args[0]); err != nil {
			return exitErr(1, "%v", err)
		}
		fmt.Println("已初始化 sync 远端:", args[0])
		return nil
	},
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "推送（preflight 全仓敏感扫描，命中阻断）",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		if err := repo.Lock(); err != nil {
			return exitErr(1, "%v", err)
		}
		defer repo.Unlock()
		// preflight 命中确认回调（--yes 不豁免，必须显式确认）
		confirm := func(matches []secrets.ScanMatch) bool {
			if flagYes {
				return false // --yes 不豁免 preflight（CLI-SPEC §8）
			}
			fmt.Printf("preflight 命中 %d 处疑似敏感内容，仍要 push？[y/N] ", len(matches))
			var in string
			fmt.Scanln(&in)
			return in == "y" || in == "Y"
		}
		if err := repo.SyncPush(secrets.DefaultScanner(), confirm); err != nil {
			return exitErr(1, "%v", err)
		}
		fmt.Println("已推送")
		return nil
	},
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "拉取远端",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		if err := repo.Lock(); err != nil {
			return exitErr(1, "%v", err)
		}
		defer repo.Unlock()
		conflict, err := repo.SyncPull()
		if err != nil {
			if conflict {
				return exitErr(1, "%v", err)
			}
			return exitErr(1, "%v", err)
		}
		fmt.Println("已拉取")
		return nil
	},
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "同步状态",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		status, err := repo.SyncStatus()
		if err != nil {
			return exitErr(1, "%v", err)
		}
		fmt.Println(status)
		return nil
	},
}

func init() {
	syncCmd.AddCommand(syncInitCmd, syncPushCmd, syncPullCmd, syncStatusCmd)
	rootCmd.AddCommand(syncCmd)
}
