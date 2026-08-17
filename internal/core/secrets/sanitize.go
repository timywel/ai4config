package secrets

// 脱敏抽取管线（ARCHITECTURE §9：先扫描替换→后落盘→零命中校验）。
// 采集时把结构化字段中的真实敏感值抽取到 secret 后端，原位替换为 secretref 占位符。

// SanitizeField 处理单个结构化字段值：
//   - 已是 secretref 或占位 → 原样返回（不重抽取）；
//   - 扫描命中敏感 → 存入后端并返回 secretref；
//   - 未命中 → 原样返回。
//
// 返回 (处理后值, 是否发生抽取, 错误)。profile/entityID/field 用于构造 secretref。
func SanitizeField(b Backend, s *Scanner, profile, entityID, field, value string) (string, bool, error) {
	if IsSecretRef(value) || IsPlaceholder(value) {
		return value, false, nil // 已是占位，不动
	}
	if !s.IsSecret(value) {
		return value, false, nil // 未命中敏感，原样
	}
	ref := MakeRef(profile, entityID, field)
	if b != nil && b.Type() != BackendNone {
		if err := b.Set(ref, value); err != nil {
			return "", false, err
		}
	}
	return ref, true, nil // none 后端也返回 ref（导出物留占位，人工填）
}

// SanitizeMap 批量处理 map 字段（env/headers），entityID 为宿主实体，fieldPrefix 为字段前缀。
func SanitizeMap(b Backend, s *Scanner, profile, entityID, fieldPrefix string, m map[string]string) (map[string]string, int, error) {
	if len(m) == 0 {
		return m, 0, nil
	}
	out := make(map[string]string, len(m))
	extracted := 0
	for k, v := range m {
		nv, hit, err := SanitizeField(b, s, profile, entityID, fieldPrefix+"."+k, v)
		if err != nil {
			return nil, extracted, err
		}
		if hit {
			extracted++
		}
		out[k] = nv
	}
	return out, extracted, nil
}
