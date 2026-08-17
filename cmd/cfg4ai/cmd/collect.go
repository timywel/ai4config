package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/timywel/ai4config/internal/adapters"
	_ "github.com/timywel/ai4config/internal/adapters/all"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/core/profile"
	"github.com/timywel/ai4config/internal/core/secrets"
	"github.com/timywel/ai4config/internal/store"
)

var (
	collectTools []string
	collectScope string
)

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "采集各工具配置进 SSOT 仓库",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		if err := repo.Lock(); err != nil { // W1[1] 写锁
			return exitErr(1, "%v", err)
		}
		defer repo.Unlock()

		scanner := secrets.DefaultScanner()
		var backend secrets.Backend // P0：敏感值仅存占位符（后端接线见 T4 后续/e2e）
		var allWarnings []ir.Warning
		totalNew, totalUpdated, totalTombstone := 0, 0, 0

		for _, a := range adapters.List() {
			if len(collectTools) > 0 && !containsStr(collectTools, string(a.Meta().ID)) {
				continue
			}
			locs, err := a.Detect(cmd.Context())
			if err != nil {
				continue
			}
			for _, loc := range locs {
				if collectScope != "" && collectScope != "all" && string(loc.Scope) != collectScope {
					continue
				}
				// 防误判（红队 T-01）：源目录不存在/不可读 → 中止该 Location，绝不标墓碑
				if info, err := os.Stat(loc.Root); err != nil || !info.IsDir() {
					allWarnings = append(allWarnings, ir.Warning{
						Kind:    "source-missing",
						Message: fmt.Sprintf("%s 源目录不可读，已中止该位置采集（不标墓碑）: %s", a.Meta().ID, loc.Root),
					})
					continue
				}
				b, err := a.Import(cmd.Context(), loc)
				if err != nil {
					allWarnings = append(allWarnings, ir.Warning{Kind: "import", Message: string(a.Meta().ID) + " 导入失败: " + err.Error()})
					continue
				}
				// 脱敏（结构化字段抽取为 secretref）
				sanitizeBundle(b, scanner, backend)

				n, u, t, err := reconcileInto(repo, loc.Scope, b)
				if err != nil {
					return exitErr(1, "写入 profile 失败: %v", err)
				}
				totalNew += n
				totalUpdated += u
				totalTombstone += t
			}
		}

		fmt.Printf("采集完成：新增 %d，更新 %d，墓碑 %d\n", totalNew, totalUpdated, totalTombstone)
		return warnExit(allWarnings)
	},
}

// sanitizeBundle 对 Bundle 的结构化字段做脱敏抽取（IR-SCHEMA §5 规则 4）。
func sanitizeBundle(b *ir.Bundle, scanner *secrets.Scanner, backend secrets.Backend) {
	profileName := "global"
	if b.Scope == ir.ScopeProject {
		profileName = "project"
	}
	for i := range b.MCPServers {
		s := &b.MCPServers[i]
		s.Env, _, _ = secrets.SanitizeMap(backend, scanner, profileName, s.ID, "env", s.Env)
		s.Headers, _, _ = secrets.SanitizeMap(backend, scanner, profileName, s.ID, "headers", s.Headers)
	}
}

// reconcileInto 把新采集的 Bundle reconcile 进对应 profile（同 id 覆盖 + 消失标墓碑）。
// 返回 (新增数, 更新数, 墓碑数)。
func reconcileInto(repo *store.Repo, scope ir.Scope, fresh *ir.Bundle) (int, int, int, error) {
	dir := profileDirFor(repo, scope, fresh)

	// 加载既有
	var existing *ir.Bundle
	var man *profile.Manifest
	if sb, err := profile.Load(dir, scope); err == nil {
		existing = sb.Bundle
		man = sb.Manifest
	} else {
		existing = &ir.Bundle{IRVersion: profile.CurrentIRVersion, Scope: scope}
		man = &profile.Manifest{
			IRVersion: profile.CurrentIRVersion,
			Profile:   profile.Meta{Name: filepath2Base(dir), Kind: kindOf(scope), CreatedAt: time.Now()},
		}
	}

	merged, n, u, t := reconcileBundles(existing, fresh)
	if err := profile.Save(dir, merged, man); err != nil {
		return 0, 0, 0, err
	}
	return n, u, t, nil
}

// reconcileBundles 采集再合并（IR-SCHEMA §2.1 reconcile + §2.3 墓碑）。
func reconcileBundles(existing, fresh *ir.Bundle) (out *ir.Bundle, added, updated, tombstoned int) {
	result := *existing
	a, u, t := 0, 0, 0

	result.Instructions, added, updated, tombstoned = reconcileList[ir.Instruction](existing.Instructions, fresh.Instructions)
	result.MCPServers, a, u, t = reconcileList[ir.MCPServer](existing.MCPServers, fresh.MCPServers)
	added, updated, tombstoned = added+a, updated+u, tombstoned+t
	result.Skills, a, u, t = reconcileList[ir.PromptPack](existing.Skills, fresh.Skills)
	added, updated, tombstoned = added+a, updated+u, tombstoned+t
	result.Agents, a, u, t = reconcileList[ir.PromptPack](existing.Agents, fresh.Agents)
	added, updated, tombstoned = added+a, updated+u, tombstoned+t
	result.Commands, a, u, t = reconcileList[ir.PromptPack](existing.Commands, fresh.Commands)
	added, updated, tombstoned = added+a, updated+u, tombstoned+t
	result.Workflows, a, u, t = reconcileList[ir.PromptPack](existing.Workflows, fresh.Workflows)
	added, updated, tombstoned = added+a, updated+u, tombstoned+t
	result.Hooks, a, u, t = reconcileList[ir.Hook](existing.Hooks, fresh.Hooks)
	added, updated, tombstoned = added+a, updated+u, tombstoned+t
	result.Settings, a, u, t = reconcileList[ir.SettingEntry](existing.Settings, fresh.Settings)
	added, updated, tombstoned = added+a, updated+u, tombstoned+t

	return &result, added, updated, tombstoned
}

// profileDirFor 返回 scope 对应的 profile 目录。
func profileDirFor(repo *store.Repo, scope ir.Scope, b *ir.Bundle) string {
	if scope == ir.ScopeGlobal {
		return repo.Path(store.DirProfiles, "global")
	}
	// project/local：按首个条目的 origin.path 推导项目目录（骨架）
	return repo.Path(store.DirProfiles, "projects", projectSlug(b))
}

func kindOf(scope ir.Scope) string {
	if scope == ir.ScopeGlobal {
		return "global"
	}
	return "project"
}

func projectSlug(b *ir.Bundle) string {
	for _, inst := range b.Instructions {
		if inst.Origin != nil && inst.Origin.Path != "" {
			return slugify(inst.Origin.Path)
		}
	}
	return "default"
}

func slugify(s string) string {
	var b []rune
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b = append(b, r)
		} else {
			b = append(b, '-')
		}
	}
	return string(b)
}

func filepath2Base(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[i+1:]
		}
	}
	return p
}

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func init() {
	collectCmd.Flags().StringSliceVar(&collectTools, "tool", nil, "只采集指定工具")
	collectCmd.Flags().StringVar(&collectScope, "scope", "", "限定 global|project|local|all")
	rootCmd.AddCommand(collectCmd)
}
