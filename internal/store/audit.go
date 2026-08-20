package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// AuditEntry 审计日志条目（logs/audit.jsonl，append-only；logs/ 不入 sync）。
// 依据 OPTIMIZATION-PLAN F15。
type AuditEntry struct {
	Ts       time.Time `json:"ts"`
	Op       string    `json:"op"`    // collect/export/edit/delete/restore/sync/ai-consent/snapshot/...
	Actor    string    `json:"actor"` // user | ai
	Profile  string    `json:"profile"`
	Targets  []string  `json:"targets,omitempty"`
	Detail   string    `json:"detail,omitempty"`
	Warnings int       `json:"warnings,omitempty"`
	Result   string    `json:"result"` // ok | error
}

// Audit 追加一条审计日志（logs/audit.jsonl，目录 0700/文件 0600）。
func (r *Repo) Audit(op, actor, profile, detail, result string, targets []string, warnings int) {
	entry := AuditEntry{
		Ts:       time.Now().UTC(),
		Op:       op,
		Actor:    actor,
		Profile:  profile,
		Targets:  targets,
		Detail:   detail,
		Warnings: warnings,
		Result:   result,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	dir := filepath.Join(r.Root, DirLogs)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "audit.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(data, '\n'))
}

// ReadAudit 读取审计日志（倒序由调用方处理）。limit<0 表示全部。
func (r *Repo) ReadAudit(limit int) ([]AuditEntry, error) {
	data, err := os.ReadFile(filepath.Join(r.Root, DirLogs, "audit.jsonl"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []AuditEntry
	for _, line := range splitLines(string(data)) {
		if line == "" {
			continue
		}
		var e AuditEntry
		if json.Unmarshal([]byte(line), &e) == nil {
			out = append(out, e)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
