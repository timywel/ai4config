package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "快照管理（list/create）",
}

var snapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出快照",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		list, err := repo.ListSnapshots()
		if err != nil {
			return exitErr(1, "读取快照失败: %v", err)
		}
		if isJSON() {
			output(list)
			return nil
		}
		if len(list) == 0 {
			fmt.Println("（无快照）")
			return nil
		}
		w := table()
		fmt.Fprintln(w, "ID\tCREATED\tNOTE\tFILES")
		for _, s := range list {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", s.ID, s.CreatedAt.Format(time.DateTime), s.Note, len(s.Files))
		}
		w.Flush()
		return nil
	},
}

var snapshotCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建快照",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		note, _ := cmd.Flags().GetString("note")
		id, err := repo.CreateSnapshot(note)
		if err != nil {
			return exitErr(1, "创建快照失败: %v", err)
		}
		fmt.Println("已创建快照:", id)
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore <snapshot-id>",
	Short: "恢复快照（先对现状打反向快照）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			fmt.Printf("（dry-run）将恢复快照 %s\n", args[0])
			return nil
		}
		// 反向快照（可回滚本次 restore）
		back, err := repo.CreateSnapshot("before-restore-" + args[0])
		if err != nil {
			return exitErr(1, "反向快照失败: %v", err)
		}
		if err := repo.RestoreSnapshot(args[0]); err != nil {
			return exitErr(1, "恢复失败: %v", err)
		}
		fmt.Printf("已恢复快照 %s（现状已存为 %s）\n", args[0], back)
		return nil
	},
}

func init() {
	snapshotCreateCmd.Flags().String("note", "", "快照备注")
	restoreCmd.Flags().Bool("dry-run", false, "只预览")
	snapshotCmd.AddCommand(snapshotListCmd, snapshotCreateCmd)
	rootCmd.AddCommand(snapshotCmd, restoreCmd)
}
