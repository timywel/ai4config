package gui

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestServerServesPagesAndAPI(t *testing.T) {
	s, err := NewServer(t.TempDir(), Handlers{
		Entities: func() ([]Entity, error) {
			return []Entity{{Kind: "mcp", ID: "mcp.fs", Note: "npx"}}, nil
		},
		Overview: func() (Overview, error) {
			return Overview{Tools: 13, Entities: 5, Snapshots: 2, RepoRoot: "/x"}, nil
		},
		Snapshots: func() ([]Snapshot, error) {
			return []Snapshot{{ID: "s1", Note: "n", Files: 3}}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	go s.srv.Serve(s.ln)
	defer s.Stop(context.Background())
	base := "http://" + s.ln.Addr().String()

	// 首页（多区块）
	resp, _ := http.Get(base + "/")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "仪表盘") || !strings.Contains(string(body), "导出") {
		t.Error("首页应含多区块导航")
	}
	// overview
	resp2, _ := http.Get(base + "/api/overview")
	var ov Overview
	json.NewDecoder(resp2.Body).Decode(&ov)
	resp2.Body.Close()
	if ov.Tools != 13 {
		t.Errorf("overview tools 错误: %d", ov.Tools)
	}
	// entities
	resp3, _ := http.Get(base + "/api/entities")
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	if !strings.Contains(string(body3), "mcp.fs") {
		t.Error("entities API 应返回实体")
	}
	// snapshots
	resp4, _ := http.Get(base + "/api/snapshots")
	body4, _ := io.ReadAll(resp4.Body)
	resp4.Body.Close()
	if !strings.Contains(string(body4), "s1") {
		t.Error("snapshots API 应返回快照")
	}
}
