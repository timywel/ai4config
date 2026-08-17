package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/store"
)

var (
	listProfile string
	listType    string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出 SSOT 中的配置实体",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		dir := repo.Path(store.DirProfiles, "global")
		if listProfile != "" && listProfile != "global" {
			dir = repo.Path(store.DirProfiles, "projects", listProfile)
		}
		sb, err := profile.Load(dir, ir.ScopeGlobal)
		if err != nil {
			return exitErr(1, "加载 profile 失败（先 collect？）: %v", err)
		}
		b := sb.Bundle

		type row struct{ Type, ID string }
		var rows []row
		add := func(t string, ids []string) {
			for _, id := range ids {
				rows = append(rows, row{t, id})
			}
		}
		if listType == "" || listType == "instruction" {
			add("instruction", idsOfInstructions(b.Instructions))
		}
		if listType == "" || listType == "mcp" {
			add("mcp", idsOfMCP(b.MCPServers))
		}
		if listType == "" || listType == "skill" {
			add("skill", idsOfPacks(b.Skills))
		}
		if listType == "" || listType == "agent" {
			add("agent", idsOfPacks(b.Agents))
		}
		if listType == "" || listType == "command" {
			add("command", idsOfPacks(b.Commands))
		}
		if listType == "" || listType == "workflow" {
			add("workflow", idsOfPacks(b.Workflows))
		}
		if listType == "" || listType == "hook" {
			add("hook", idsOfHooks(b.Hooks))
		}
		if listType == "" || listType == "setting" {
			add("setting", idsOfSettings(b.Settings))
		}

		if isJSON() {
			output(rows)
			return nil
		}
		if len(rows) == 0 {
			fmt.Println("（空）")
			return nil
		}
		w := table()
		fmt.Fprintln(w, "TYPE\tID")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\n", r.Type, r.ID)
		}
		w.Flush()
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show <entity-id>",
	Short: "显示单个实体详情",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		sb, err := profile.Load(repo.Path(store.DirProfiles, "global"), ir.ScopeGlobal)
		if err != nil {
			return exitErr(1, "加载 profile 失败: %v", err)
		}
		id := args[0]
		if e := findEntity(sb.Bundle, id); e != nil {
			output(e) // json
			if !isJSON() {
				fmt.Printf("%+v\n", e)
			}
			return nil
		}
		return exitErr(1, "未找到实体 %s", id)
	},
}

// findEntity 按 id 查实体。
func findEntity(b *ir.Bundle, id string) any {
	for _, x := range b.Instructions {
		if x.ID == id {
			return x
		}
	}
	for _, x := range b.MCPServers {
		if x.ID == id {
			return x
		}
	}
	for _, x := range b.Skills {
		if x.ID == id {
			return x
		}
	}
	for _, x := range b.Settings {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func idsOfInstructions(l []ir.Instruction) []string {
	var o []string
	for _, x := range l {
		o = append(o, x.ID)
	}
	return o
}
func idsOfMCP(l []ir.MCPServer) []string {
	var o []string
	for _, x := range l {
		o = append(o, x.ID)
	}
	return o
}
func idsOfPacks(l []ir.PromptPack) []string {
	var o []string
	for _, x := range l {
		o = append(o, x.ID)
	}
	return o
}
func idsOfHooks(l []ir.Hook) []string {
	var o []string
	for _, x := range l {
		o = append(o, x.ID)
	}
	return o
}
func idsOfSettings(l []ir.SettingEntry) []string {
	var o []string
	for _, x := range l {
		o = append(o, x.ID)
	}
	return o
}

func init() {
	listCmd.Flags().StringVar(&listProfile, "profile", "global", "profile 名（global 或项目 id）")
	listCmd.Flags().StringVar(&listType, "type", "", "限定实体类型")
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
}
