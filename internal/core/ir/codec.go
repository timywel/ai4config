package ir

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SplitFrontmatter 拆分 Markdown 文件的 YAML frontmatter 与正文。
// 仅当文件以 "---" 行开头时识别 frontmatter；否则整体视为正文。
// 返回的 frontmatter 不含 --- 分隔线；body 保留原始内容（含换行风格）。
func SplitFrontmatter(data []byte) (fm, body []byte, err error) {
	nl := []byte("\n")
	if bytes.HasPrefix(data, []byte("---\r\n")) {
		nl = []byte("\r\n")
	}
	open := append([]byte("---"), nl...)
	if !bytes.HasPrefix(data, open) {
		return nil, data, nil
	}
	rest := data[len(open):]
	close := append(nl, []byte("---")...)
	close = append(close, nl...)
	idx := bytes.Index(rest, close)
	if idx < 0 {
		// 文件尾紧邻 ---（无尾换行）
		tail := append(nl, []byte("---")...)
		if bytes.HasSuffix(rest, tail) {
			return rest[:len(rest)-len(tail)], nil, nil
		}
		return nil, nil, fmt.Errorf("ir: frontmatter 未闭合（缺少结尾 ---）")
	}
	fm = rest[:idx]
	body = rest[idx+len(close):]
	return fm, body, nil
}

// MarshalEntity 序列化实体：struct 字段 + Header.Extensions 展开为 x-<tool> 键。
func MarshalEntity(structPtr any, ext map[string]any) ([]byte, error) {
	var node yaml.Node
	if err := node.Encode(structPtr); err != nil {
		return nil, fmt.Errorf("ir: 编码实体失败: %w", err)
	}
	if len(ext) == 0 {
		return yaml.Marshal(&node)
	}
	m, err := docMapping(&node)
	if err != nil {
		return nil, err
	}
	for k, v := range ext {
		if !strings.HasPrefix(k, "x-") {
			return nil, fmt.Errorf("ir: 扩展键 %q 必须以 x- 前缀", k)
		}
		var kn, vn yaml.Node
		kn.SetString(k)
		if err := vn.Encode(v); err != nil {
			return nil, fmt.Errorf("ir: 编码扩展键 %q 失败: %w", k, err)
		}
		m.Content = append(m.Content, &kn, &vn)
	}
	return yaml.Marshal(&node)
}

// UnmarshalEntity 反序列化实体：x-<tool> 键收拢进返回值，其余键解码进 structPtr。
func UnmarshalEntity(data []byte, structPtr any) (ext map[string]any, err error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("ir: 解析 YAML 失败: %w", err)
	}
	m, err := docMapping(&doc)
	if err != nil {
		return nil, err
	}
	ext = map[string]any{}
	filtered := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for i := 0; i+1 < len(m.Content); i += 2 {
		k, v := m.Content[i], m.Content[i+1]
		if strings.HasPrefix(k.Value, "x-") {
			var anyv any
			if err := v.Decode(&anyv); err != nil {
				return nil, fmt.Errorf("ir: 解码扩展键 %q 失败: %w", k.Value, err)
			}
			ext[k.Value] = anyv
			continue
		}
		filtered.Content = append(filtered.Content, k, v)
	}
	if err := filtered.Decode(structPtr); err != nil {
		return nil, fmt.Errorf("ir: 解码实体失败: %w", err)
	}
	if len(ext) == 0 {
		ext = nil
	}
	return ext, nil
}

// MarshalMarkdownDoc 组合 frontmatter 与正文为完整 Markdown 文件（--- 包裹）。
func MarshalMarkdownDoc(fmStructPtr any, ext map[string]any, body string) ([]byte, error) {
	fm, err := MarshalEntity(fmStructPtr, ext)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fm)
	buf.WriteString("---\n")
	buf.WriteString(body)
	return buf.Bytes(), nil
}

// UnmarshalMarkdownDoc 拆分并解码 frontmatter；正文原文返回（保留换行风格）。
func UnmarshalMarkdownDoc(data []byte, fmStructPtr any) (body string, ext map[string]any, err error) {
	fm, rawBody, err := SplitFrontmatter(data)
	if err != nil {
		return "", nil, err
	}
	if fm != nil {
		ext, err = UnmarshalEntity(fm, fmStructPtr)
		if err != nil {
			return "", nil, err
		}
	}
	return string(rawBody), ext, nil
}

// docMapping 取出 Document 根部的 mapping node。
func docMapping(doc *yaml.Node) (*yaml.Node, error) {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		doc = doc.Content[0]
	}
	if doc.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("ir: 期望 YAML mapping，实际 kind=%v", doc.Kind)
	}
	return doc, nil
}
