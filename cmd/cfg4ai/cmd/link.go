package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/timywel/ai4config/internal/core/registry"
)

var linkCmd = &cobra.Command{
	Use:   "link <path>",
	Short: "把目录关联到项目 profile（无则创建）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		if err := repo.Lock(); err != nil {
			return exitErr(1, "%v", err)
		}
		defer repo.Unlock()

		reg, err := registry.Load(repo.Root)
		if err != nil {
			return exitErr(1, "%v", err)
		}
		var confirm func(string) bool
		if flagYes {
			confirm = nil // --yes 自动确认合并
		} else {
			confirm = func(prompt string) bool {
				fmt.Printf("%s [y/N] ", prompt)
				var in string
				fmt.Scanln(&in)
				return in == "y" || in == "Y"
			}
		}
		before := len(reg.Projects)
		p, _, err := reg.Link(args[0], confirm)
		if err != nil {
			return exitErr(1, "link 失败: %v", err)
		}
		merged := len(reg.Projects) == before // 项目数未增 = 合并到已有
		if err := reg.Save(repo.Root); err != nil {
			return exitErr(1, "写入注册表失败: %v", err)
		}
		if merged {
			fmt.Printf("已合并关联到已有项目 %s（%s）\n", p.ID, p.Name)
		} else {
			fmt.Printf("已新建项目 %s（%s），关联 %s\n", p.ID, p.Name, args[0])
		}
		return nil
	},
}

var relinkCmd = &cobra.Command{
	Use:   "relink [path]",
	Short: "凭指纹重新定位已注册项目（目录迁移/改名后）",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		abs, _ := filepathAbs(path)
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		if err := repo.Lock(); err != nil {
			return exitErr(1, "%v", err)
		}
		defer repo.Unlock()
		reg, err := registry.Load(repo.Root)
		if err != nil {
			return exitErr(1, "%v", err)
		}
		p, err := reg.Relink(abs)
		if err != nil {
			return exitErr(1, "%v", err)
		}
		if err := reg.Save(repo.Root); err != nil {
			return exitErr(1, "写入注册表失败: %v", err)
		}
		fmt.Printf("已重新关联项目 %s（%s）→ %s\n", p.ID, p.Name, abs)
		return nil
	},
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "列出项目注册表",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		reg, err := registry.Load(repo.Root)
		if err != nil {
			return exitErr(1, "%v", err)
		}
		if isJSON() {
			output(reg.Projects)
			return nil
		}
		if len(reg.Projects) == 0 {
			fmt.Println("（无注册项目）")
			return nil
		}
		w := table()
		fmt.Fprintln(w, "ID\tNAME\tPATHS\tREMOTE")
		for _, p := range reg.Projects {
			paths := ""
			if len(p.Paths) > 0 {
				paths = p.Paths[len(p.Paths)-1]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, p.Name, paths, p.Fingerprint.GitRemote)
		}
		w.Flush()
		return nil
	},
}

func filepathAbs(p string) (string, error) {
	return filepath.Abs(p)
}

func init() {
	rootCmd.AddCommand(linkCmd, relinkCmd, projectsCmd)
}
