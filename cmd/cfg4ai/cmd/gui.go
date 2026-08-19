package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/migrate"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/core/secrets"
	"github.com/timywel/ai4config/internal/gui"
	"github.com/timywel/ai4config/internal/store"
)

var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "本地 Web 界面（标准库 HTTP，零外部依赖，浏览器打开）",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}

		entities := func() ([]gui.Entity, error) {
			sb, err := profile.Load(repo.Path(store.DirProfiles, "global"), ir.ScopeGlobal)
			if err != nil {
				return nil, err
			}
			var items []gui.Entity
			b := sb.Bundle
			for _, x := range b.Instructions {
				items = append(items, gui.Entity{Kind: "instruction", ID: x.ID, Note: x.Description})
			}
			for _, x := range b.MCPServers {
				items = append(items, gui.Entity{Kind: "mcp", ID: x.ID, Note: x.Command + x.URL})
			}
			for _, x := range b.Skills {
				items = append(items, gui.Entity{Kind: "skill", ID: x.ID, Note: x.Description})
			}
			for _, x := range b.Agents {
				items = append(items, gui.Entity{Kind: "agent", ID: x.ID, Note: x.Description})
			}
			for _, x := range b.Hooks {
				items = append(items, gui.Entity{Kind: "hook", ID: x.ID, Note: string(x.Event)})
			}
			for _, x := range b.Settings {
				items = append(items, gui.Entity{Kind: "setting", ID: x.ID, Note: x.Key})
			}
			return items, nil
		}

		handlers := gui.Handlers{
			Entities: entities,
			Overview: func() (gui.Overview, error) {
				items, _ := entities()
				snaps, _ := repo.ListSnapshots()
				return gui.Overview{
					Tools:     len(adapters.List()),
					Entities:  len(items),
					Snapshots: len(snaps),
					RepoRoot:  repo.Root,
				}, nil
			},
			Collect: func(tool string) (string, error) {
				scanner := secrets.DefaultScanner()
				backend, _ := resolveSecretsBackend(repo)
				totalNew := 0
				for _, a := range adapters.List() {
					if tool != "" && string(a.Meta().ID) != tool {
						continue
					}
					locs, err := a.Detect(cmd.Context())
					if err != nil {
						continue
					}
					for _, loc := range locs {
						b, err := a.Import(cmd.Context(), loc)
						if err != nil {
							continue
						}
						sanitizeBundle(b, scanner, backend)
						n, _, _, err := reconcileInto(repo, loc.Scope, b)
						if err != nil {
							continue
						}
						totalNew += n
					}
				}
				return fmt.Sprintf("采集完成，新增 %d 条", totalNew), nil
			},
			Export: func(to string, dryRun bool) (string, []string, error) {
				e := &migrate.Engine{Repo: repo}
				res, err := e.Export(cmd.Context(), migrate.ExportRequest{To: adapters.ToolID(to), DryRun: dryRun, Force: false})
				if err != nil {
					return "", nil, err
				}
				var paths []string
				for _, f := range res.Written {
					paths = append(paths, f.Path)
				}
				verb := "已写入"
				if dryRun {
					verb = "预览（未落盘）"
				}
				return fmt.Sprintf("%s %d 个文件", verb, len(paths)), paths, nil
			},
			Snapshots: func() ([]gui.Snapshot, error) {
				list, err := repo.ListSnapshots()
				if err != nil {
					return nil, err
				}
				var out []gui.Snapshot
				for _, s := range list {
					out = append(out, gui.Snapshot{ID: s.ID, Note: s.Note, Files: len(s.Files)})
				}
				return out, nil
			},
			SnapshotCreate: func(note string) (string, error) {
				return repo.CreateSnapshot(note)
			},
			SnapshotRestore: func(id string) (string, error) {
				if _, err := repo.CreateSnapshot("before-restore-" + id); err != nil {
					return "", err
				}
				if err := repo.RestoreSnapshot(id); err != nil {
					return "", err
				}
				return "已恢复快照 " + id, nil
			},
		}

		return gui.Run(repo.Root, handlers)
	},
}

func init() {
	rootCmd.AddCommand(guiCmd)
}
