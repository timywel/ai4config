package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/timywel/ai4config/internal/adapters"
	_ "github.com/timywel/ai4config/internal/adapters/all"
	"github.com/timywel/ai4config/internal/core/aiassist"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/migrate"
	"github.com/timywel/ai4config/internal/store"
)

var (
	exportTo             string
	exportProject        string
	exportDryRun         bool
	exportForce          bool
	exportOnly           []string
	exportIncludeForeign bool
	exportAI             bool
	exportAIApprove      bool
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "把 SSOT 配置交付到目标工具",
	RunE: func(cmd *cobra.Command, args []string) error {
		if exportTo == "" {
			return exitErr(2, "缺少 --to <tool>（目标工具 id）")
		}
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}

		e := &migrate.Engine{Repo: repo}
		// AI 接线：--ai 且未 --no-ai 时，从 config 构造 provider + consent
		if exportAI && !flagNoAI {
			cfg := loadAppConfig(repo)
			if cfg.AI.BaseURL == "" {
				return exitErr(1, "--ai 需要先配置：cfg4ai config set ai.base_url <url>")
			}
			provider := &aiassist.OpenAIProvider{BaseURL: cfg.AI.BaseURL, Model: cfg.AI.Model}
			if client, cerr := aiassist.NewClient(provider, repo.Root); cerr == nil {
				e.AI = client
				e.AIConfig = aiassist.AIConfig{Provider: cfg.AI.Provider, BaseURL: cfg.AI.BaseURL, Model: cfg.AI.Model}
			}
		}
		// 外来内容确认回调（交互四选项；--yes 时 foreign/modified 默认 skip，安全）
		if !flagYes && !exportForce {
			e.Hooks.ConfirmForeign = confirmForeignInteractive
		}

		res, err := e.Export(cmd.Context(), migrate.ExportRequest{
			To:             adapters.ToolID(exportTo),
			ProjectPath:    exportProject,
			DryRun:         exportDryRun,
			Force:          exportForce,
			IncludeForeign: exportIncludeForeign,
			Only:           parseKinds(exportOnly),
			AI:             exportAI && !flagNoAI,
			AIApprove:      exportAIApprove,
		})
		if err != nil {
			return exitErr(1, "%v", err)
		}

		// 输出
		verb := "已写入"
		if exportDryRun {
			verb = "计划写入（dry-run 未落盘）"
		}
		fmt.Printf("导出到 %s：%s %d 个文件\n", exportTo, verb, len(res.Written))
		if flagVerbose || exportDryRun {
			for _, f := range res.Written {
				fmt.Printf("  %s\n", f.Path)
			}
		}
		if res.SnapshotID != "" && flagVerbose {
			fmt.Printf("快照：%s\n", res.SnapshotID)
		}
		return warnExit(res.Warnings)
	},
}

// confirmForeignInteractive 外来内容四选项交互（W2[7]）。
func confirmForeignInteractive(path string, status store.ForeignStatus) (string, error) {
	fmt.Printf("目标文件 %s（%s）：[o]覆盖 / [s]跳过 / [d]看差异 / [b]备份覆盖 ? ", path, status)
	var in string
	fmt.Scanln(&in)
	switch strings.ToLower(strings.TrimSpace(in)) {
	case "o", "overwrite":
		return "overwrite", nil
	case "b", "backup":
		return "backup-overwrite", nil
	case "d", "diff":
		return "view-diff", nil
	default:
		return "skip", nil
	}
}

// parseKinds 解析 --only 实体类型。
func parseKinds(list []string) []ir.EntityKind {
	var out []ir.EntityKind
	for _, s := range list {
		for _, part := range strings.Split(s, ",") {
			out = append(out, ir.EntityKind(strings.TrimSpace(part)))
		}
	}
	return out
}

func init() {
	f := exportCmd.Flags()
	f.StringVar(&exportTo, "to", "", "目标工具 id（必需）")
	f.StringVar(&exportProject, "project", "", "项目路径（合并项目 profile）")
	f.BoolVar(&exportDryRun, "dry-run", false, "只预览不落盘")
	f.BoolVar(&exportForce, "force", false, "全部按备份覆盖")
	f.StringSliceVar(&exportOnly, "only", nil, "只导出指定实体类型（instruction,mcp,...）")
	f.BoolVar(&exportIncludeForeign, "include-foreign", false, "纳入异构来源条目")
	f.BoolVar(&exportAI, "ai", false, "启用 AI 语义转换（需确认）")
	f.BoolVar(&exportAIApprove, "ai-approve", false, "无人值守确认 AI 转换（记决策日志）")
	rootCmd.AddCommand(exportCmd)
}
