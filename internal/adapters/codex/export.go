package codex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
	"github.com/timywel/ai4config/internal/platform/paths"
)

// exportBundle materializes merged Bundle into Codex config (ADAPTERS 3.2).
func (a *adapter) exportBundle(ctx context.Context, b *ir.Bundle, opts adapters.ExportOpts) ([]adapters.WrittenFile, error) {
	project := opts.ProjectRoot != ""
	var codexDir, agentsBase string
	if project {
		codexDir = filepath.Join(opts.ProjectRoot, ".codex")
		agentsBase = opts.ProjectRoot
	} else {
		dir := os.Getenv("CODEX_HOME")
		if dir == "" {
			home, err := paths.Home()
			if err != nil {
				return nil, err
			}
			dir = filepath.Join(home, ".codex")
		}
		codexDir = dir
		agentsBase = dir
	}

	var files []adapters.WrittenFile
	var warnings []ir.Warning
	add := func(path string, content []byte) error {
		wf, err := writeOne(path, content, opts.DryRun)
		if err != nil {
			return err
		}
		files = append(files, wf)
		return nil
	}

	if len(b.Settings) > 0 || len(b.MCPServers) > 0 || len(b.Hooks) > 0 {
		data, w, err := renderConfigTOML(b.Settings, b.MCPServers, b.Hooks, project)
		if err != nil {
			return nil, err
		}
		warnings = append(warnings, w...)
		if err := add(filepath.Join(codexDir, "config.toml"), data); err != nil {
			return nil, err
		}
	}

	// instructions: group by subtree, join bodies per AGENTS.md (multi-layer concat).
	bySubtree := map[string][]string{}
	var stOrder []string
	for _, inst := range b.Instructions {
		st := inst.Subtree
		if _, ok := bySubtree[st]; !ok {
			stOrder = append(stOrder, st)
		}
		bySubtree[st] = append(bySubtree[st], inst.Body)
	}
	for _, st := range stOrder {
		target := filepath.Join(agentsBase, "AGENTS.md")
		if st != "" && st != "." {
			target = filepath.Join(agentsBase, filepath.FromSlash(st), "AGENTS.md")
		}
		if err := add(target, []byte(strings.Join(bySubtree[st], "\n"))); err != nil {
			return nil, err
		}
	}

	for _, p := range b.Skills {
		data := renderSkillMD(p)
		if err := add(filepath.Join(codexDir, "skills", p.Name, "SKILL.md"), data); err != nil {
			return nil, err
		}
	}

	for _, p := range b.Agents {
		data := renderSkillMD(p)
		if err := add(filepath.Join(codexDir, "agents", p.Name+".md"), data); err != nil {
			return nil, err
		}
	}

	_ = warnings
	return files, nil
}

// renderSkillMD renders a PromptPack as frontmatter+body markdown.
func renderSkillMD(p ir.PromptPack) []byte {
	var sb strings.Builder
	sb.WriteString("---\nname: " + p.Name + "\n")
	if p.Description != "" {
		sb.WriteString("description: " + p.Description + "\n")
	}
	sb.WriteString("---\n")
	sb.WriteString(p.Body)
	return []byte(sb.String())
}

// writeOne builds a render plan entry (no disk write; engine writes via atomicfile).
func writeOne(path string, content []byte, dryRun bool) (adapters.WrittenFile, error) {
	_ = dryRun
	sum := sha256.Sum256(content)
	return adapters.WrittenFile{Path: path, Hash: hex.EncodeToString(sum[:]), Content: content}, nil
}
