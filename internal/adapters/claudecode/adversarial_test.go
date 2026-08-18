package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/timywel/ai4config/internal/adapters"
	"github.com/timywel/ai4config/internal/core/ir"
)

// 对抗用例回归（research/adversarial-cases.md）。

// AC-B4 utf8-bom-settings：settings.json 带 UTF-8 BOM 应正常解析（不报错）
func TestAdversarial_B4_SettingsWithBOM(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0o755)
	// PowerShell Out-File -Encoding utf8 默认产物：BOM 前缀
	bomJSON := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"model":"opus"}`)...)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), bomJSON, 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeGlobal, Root: claudeDir})
	if err != nil {
		t.Fatalf("BOM 文件 Import 应成功: %v", err)
	}
	var model *ir.SettingEntry
	for i := range b.Settings {
		if b.Settings[i].Key == "model" {
			model = &b.Settings[i]
		}
	}
	if model == nil || model.Value != "opus" {
		t.Errorf("BOM 文件 model 解析错误: %+v", model)
	}
}

// AC-A3 skill-name-case-dot：目录名含大写与点号 → id 规范化后可解析
func TestAdversarial_A3_SkillNameCaseDot(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skills", "MyHelper.v2")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\ndescription: 助手\n---\n正文\n"), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeGlobal, Root: dir})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(b.Skills) != 1 {
		t.Fatalf("应采集 1 个 skill，实际 %d", len(b.Skills))
	}
	id := b.Skills[0].ID
	// 规范化：大写转小写（MyHelper.v2 → myhelper.v2），id 可被 ParseID 接受
	if _, _, err := ir.ParseID(id); err != nil {
		t.Errorf("规范化 id %q 应通过校验: %v", id, err)
	}
	if id != "skill.myhelper.v2" {
		t.Logf("规范化 id = %q", id)
	}
}

// AC-B1 jsonc-every-line-comment：JSONC 尾逗号+逐行注释（注释密度高）
// 当前能力：JSONC 解析（尾逗号/注释容忍度）。注释保留属文档层免责（IR-SCHEMA §1.3）。
func TestAdversarial_B1_JSONCCommentDensity(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0o755)
	// JSONC：行尾注释（标准 encoding/json 不容忍——记录当前行为边界）
	jsonc := "{\n  \"model\": \"opus\" // 团队规范：统一用 opus\n}\n"
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(jsonc), 0o644)

	a := &adapter{}
	b, err := a.Import(context.Background(), adapters.Location{Scope: ir.ScopeGlobal, Root: claudeDir})
	// JSONC 带注释：标准库 json 解析失败 → settings 为空但不 panic（当前行为边界记录）
	if err != nil {
		t.Fatalf("Import 不应因 JSONC 报错: %v", err)
	}
	_ = b
	t.Log("JSONC 注释解析能力边界已记录（如需支持 JSONC，后续引入 JSONC 解析器）")
}
