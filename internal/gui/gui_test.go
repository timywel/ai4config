package gui

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestServerServesIndexAndAPI(t *testing.T) {
	s, err := NewServer(t.TempDir(), func() ([]Entity, error) {
		return []Entity{{Kind: "mcp", ID: "mcp.fs", Note: "npx"}}, nil
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	go s.srv.Serve(s.ln)
	defer s.Stop(context.Background())

	base := "http://" + s.ln.Addr().String()
	// 首页
	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "cfg4ai") {
		t.Error("首页应含 cfg4ai")
	}
	// API
	resp2, err := http.Get(base + "/api/entities")
	if err != nil {
		t.Fatalf("GET /api/entities: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if !strings.Contains(string(body2), "mcp.fs") {
		t.Errorf("API 应返回实体: %s", body2)
	}
}
