package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timywel/ai4config/internal/adapters"
	_ "github.com/timywel/ai4config/internal/adapters/all"
	"github.com/timywel/ai4config/internal/core/secrets"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "自检：仓库结构、适配器、secret 后端、AI 连通性",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}

		type check struct {
			Item   string `json:"item"`
			Status string `json:"status"`
			Detail string `json:"detail"`
		}
		var checks []check
		ok := func(item, detail string) { checks = append(checks, check{item, "OK", detail}) }
		warn := func(item, detail string) { checks = append(checks, check{item, "WARN", detail}) }

		// 仓库结构
		ok("仓库结构", repo.Root)

		// 适配器探测
		for _, a := range adapters.List() {
			locs, err := a.Detect(cmd.Context())
			if err != nil {
				warn("适配器 "+string(a.Meta().ID), "探测失败: "+err.Error())
				continue
			}
			ok("适配器 "+string(a.Meta().ID), fmt.Sprintf("探测到 %d 个配置位置", len(locs)))
		}

		// secret 后端
		b, err := secrets.ResolveBackend(flagSecretsBackend, repo.Root, nil)
		if err != nil {
			warn("secret 后端", err.Error())
		} else {
			ok("secret 后端", string(b.Type()))
		}

		// AI provider（P0 未配置，提示）
		warn("AI provider", "未配置（P2 功能；export --ai 需先 config 配置）")

		if isJSON() {
			output(checks)
			return nil
		}
		w := table()
		fmt.Fprintln(w, "ITEM\tSTATUS\tDETAIL")
		for _, c := range checks {
			fmt.Fprintf(w, "%s\t%s\t%s\n", c.Item, c.Status, c.Detail)
		}
		w.Flush()
		// 有 FAIL → 退出码 3
		for _, c := range checks {
			if c.Status == "FAIL" {
				return exitErr(3, "doctor 检出失败项")
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
