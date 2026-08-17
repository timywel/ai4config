package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/timywel/ai4config/internal/atomicfile"
	"github.com/timywel/ai4config/internal/core/ir"
)

// profile 目录布局（ARCHITECTURE §7）：
//   <dir>/manifest.yaml
//   <dir>/instructions/<name>.md        frontmatter + 正文
//   <dir>/mcp.yaml                      servers: [...] + file_extensions
//   <dir>/<kind>s/<name>/<kind>.yaml    元数据（含 x- 扩展）
//   <dir>/<kind>s/<name>/prompt.md      正文
//   <dir>/<kind>s/<name>/assets/        附带资产
//   <dir>/hooks.yaml                    hooks: [...]
//   <dir>/settings.yaml                 entries: [...]

func manifestPath(dir string) string { return filepath.Join(dir, "manifest.yaml") }

// Load 读取一个 profile 目录为 ScopedBundle。
func Load(dir string, scope ir.Scope) (*ScopedBundle, error) {
	m, err := LoadManifest(dir)
	if err != nil {
		return nil, err
	}
	b := &ir.Bundle{IRVersion: m.IRVersion, Scope: scope}

	if b.Instructions, err = loadInstructions(dir); err != nil {
		return nil, err
	}
	if b.MCPServers, b.MCPFileExtensions, err = loadMCP(dir); err != nil {
		return nil, err
	}
	if b.Skills, err = loadPacks(dir, "skills", ir.KindSkill); err != nil {
		return nil, err
	}
	if b.Agents, err = loadPacks(dir, "agents", ir.KindAgent); err != nil {
		return nil, err
	}
	if b.Commands, err = loadPacks(dir, "commands", ir.KindCommand); err != nil {
		return nil, err
	}
	if b.Workflows, err = loadPacks(dir, "workflows", ir.KindWorkflow); err != nil {
		return nil, err
	}
	if b.Hooks, err = loadHooks(dir); err != nil {
		return nil, err
	}
	if b.Settings, err = loadSettings(dir); err != nil {
		return nil, err
	}

	return &ScopedBundle{Scope: scope, Bundle: b, Manifest: m}, nil
}

// Save 把 Bundle + Manifest 写入 profile 目录（0600，经 atomicfile 原子写）。
func Save(dir string, b *ir.Bundle, m *Manifest) error {
	if err := SaveManifest(dir, m); err != nil {
		return err
	}
	if err := saveInstructions(dir, b.Instructions); err != nil {
		return err
	}
	if err := saveMCP(dir, b.MCPServers, b.MCPFileExtensions); err != nil {
		return err
	}
	if err := savePacks(dir, "skills", ir.KindSkill, b.Skills); err != nil {
		return err
	}
	if err := savePacks(dir, "agents", ir.KindAgent, b.Agents); err != nil {
		return err
	}
	if err := savePacks(dir, "commands", ir.KindCommand, b.Commands); err != nil {
		return err
	}
	if err := savePacks(dir, "workflows", ir.KindWorkflow, b.Workflows); err != nil {
		return err
	}
	if err := saveHooks(dir, b.Hooks); err != nil {
		return err
	}
	return saveSettings(dir, b.Settings)
}

// ---------- instructions ----------

func loadInstructions(dir string) ([]ir.Instruction, error) {
	files, err := filepath.Glob(filepath.Join(dir, "instructions", "*.md"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	var out []ir.Instruction
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("profile: 读取 %s 失败: %w", f, err)
		}
		var inst ir.Instruction
		body, ext, err := ir.UnmarshalMarkdownDoc(data, &inst)
		if err != nil {
			return nil, fmt.Errorf("profile: 解析 %s 失败: %w", f, err)
		}
		inst.Body = body
		inst.Extensions = ext
		out = append(out, inst)
	}
	return out, nil
}

func saveInstructions(dir string, items []ir.Instruction) error {
	for _, inst := range items {
		name := ir.NameTail(inst.ID)
		if name == "" {
			return fmt.Errorf("profile: instruction 缺 id: %+v", inst)
		}
		data, err := ir.MarshalMarkdownDoc(&inst, inst.Extensions, inst.Body)
		if err != nil {
			return err
		}
		if err := writeProfileFile(filepath.Join(dir, "instructions", name+".md"), data); err != nil {
			return err
		}
	}
	return nil
}

// ---------- mcp.yaml ----------

func loadMCP(dir string) ([]ir.MCPServer, map[string]any, error) {
	data, err := os.ReadFile(filepath.Join(dir, "mcp.yaml"))
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	items, topExt, err := decodeListFile[ir.MCPServer](data, "servers")
	return items, topExt, err
}

func saveMCP(dir string, items []ir.MCPServer, fileExt map[string]any) error {
	if len(items) == 0 && len(fileExt) == 0 {
		return nil
	}
	return saveListFile(filepath.Join(dir, "mcp.yaml"), "servers", items, fileExt)
}

// ---------- hooks.yaml / settings.yaml ----------

func loadHooks(dir string) ([]ir.Hook, error) {
	data, err := os.ReadFile(filepath.Join(dir, "hooks.yaml"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items, _, err := decodeListFile[ir.Hook](data, "hooks")
	return items, err
}

func saveHooks(dir string, items []ir.Hook) error {
	if len(items) == 0 {
		return nil
	}
	return saveListFile(filepath.Join(dir, "hooks.yaml"), "hooks", items, nil)
}

func loadSettings(dir string) ([]ir.SettingEntry, error) {
	data, err := os.ReadFile(filepath.Join(dir, "settings.yaml"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items, _, err := decodeListFile[ir.SettingEntry](data, "entries")
	return items, err
}

func saveSettings(dir string, items []ir.SettingEntry) error {
	if len(items) == 0 {
		return nil
	}
	return saveListFile(filepath.Join(dir, "settings.yaml"), "entries", items, nil)
}

// ---------- PromptPack 目录（skills/agents/commands/workflows）----------

func loadPacks(root, sub string, kind ir.EntityKind) ([]ir.PromptPack, error) {
	base := filepath.Join(root, sub)
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []ir.PromptPack
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		metaFile := filepath.Join(base, name, string(kind)+".yaml")
		data, err := os.ReadFile(metaFile)
		if os.IsNotExist(err) {
			continue // 无元数据文件的目录跳过（如纯 assets 目录）
		}
		if err != nil {
			return nil, err
		}
		var p ir.PromptPack
		ext, err := ir.UnmarshalEntity(data, &p)
		if err != nil {
			return nil, fmt.Errorf("profile: 解析 %s 失败: %w", metaFile, err)
		}
		p.Extensions = ext
		if p.Kind == "" {
			p.Kind = kind
		}
		if body, err := os.ReadFile(filepath.Join(base, name, "prompt.md")); err == nil {
			p.Body = string(body)
		}
		out = append(out, p)
	}
	return out, nil
}

func savePacks(root, sub string, kind ir.EntityKind, items []ir.PromptPack) error {
	for _, p := range items {
		name := ir.NameTail(p.ID)
		if name == "" {
			return fmt.Errorf("profile: %s 缺 id", kind)
		}
		dir := filepath.Join(root, sub, name)
		meta, err := ir.MarshalEntity(&p, p.Extensions)
		if err != nil {
			return err
		}
		if err := writeProfileFile(filepath.Join(dir, string(kind)+".yaml"), meta); err != nil {
			return err
		}
		if p.Body != "" {
			if err := writeProfileFile(filepath.Join(dir, "prompt.md"), []byte(p.Body)); err != nil {
				return err
			}
		}
	}
	return nil
}

// ---------- 列表型 YAML 文件的通用编解码（保留 x- 扩展与顶层扩展位） ----------

// decodeListFile 解析 <listKey>: [ ... ] 结构的文件，逐条保留 x- 扩展；
// 其余顶层键（如 file_extensions）作为 topExt 返回。
func decodeListFile[T any, PT headerCarrier[T]](data []byte, listKey string) (items []T, topExt map[string]any, err error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("profile: 解析 %s 列表失败: %w", listKey, err)
	}
	if len(doc.Content) == 0 {
		return nil, nil, nil
	}
	m := doc.Content[0]
	if m.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("profile: %s 文件根应为 mapping", listKey)
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		k, v := m.Content[i], m.Content[i+1]
		if k.Value == listKey {
			if v.Kind != yaml.SequenceNode {
				return nil, nil, fmt.Errorf("profile: %s 应为数组", listKey)
			}
			for _, itemNode := range v.Content {
				b, err := yaml.Marshal(itemNode)
				if err != nil {
					return nil, nil, err
				}
				var item T
				p := PT(&item)
				ext, err := ir.UnmarshalEntity(b, p)
				if err != nil {
					return nil, nil, err
				}
				p.GetHeader().Extensions = ext
				items = append(items, item)
			}
		} else {
			var anyv any
			if err := v.Decode(&anyv); err != nil {
				return nil, nil, err
			}
			if topExt == nil {
				topExt = map[string]any{}
			}
			topExt[k.Value] = anyv
		}
	}
	return items, topExt, nil
}

// saveListFile 写入 <listKey>: [ ... ] 结构；topExt（如 file_extensions）并入顶层。
func saveListFile[T any, PT headerCarrier[T]](path, listKey string, items []T, topExt map[string]any) error {
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	// 顶层扩展位（file_extensions 等）在前
	for k, v := range topExt {
		var kn, vn yaml.Node
		kn.SetString(k)
		if err := vn.Encode(v); err != nil {
			return err
		}
		root.Content = append(root.Content, &kn, &vn)
	}

	// 列表键
	var keyNode yaml.Node
	keyNode.SetString(listKey)
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for i := range items {
		b, err := ir.MarshalEntity(PT(&items[i]), PT(&items[i]).GetHeader().Extensions)
		if err != nil {
			return err
		}
		var itemNode yaml.Node
		if err := yaml.Unmarshal(b, &itemNode); err != nil {
			return err
		}
		if len(itemNode.Content) > 0 {
			seq.Content = append(seq.Content, itemNode.Content[0])
		}
	}
	root.Content = append(root.Content, &keyNode, seq)

	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return writeProfileFile(path, data)
}

// ---------- 落盘助手 ----------

// writeProfileFile 经 atomicfile 原子写，目录 0700 / 文件 0600（ARCHITECTURE §7）。
func writeProfileFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return atomicfile.WriteFile(path, data, 0o600)
}
