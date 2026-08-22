// Package gui 提供 cfg4ai 的本地 Web 界面（标准库 net/http，零外部 GUI 依赖，
// 守住 CGO_ENABLED=0 静态分发纪律）。
// 形态：cfg4ai gui 起本地 HTTP 服务 + 自动打开浏览器。
package gui

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"

	"github.com/timywel/ai4config/internal/platform/hidecmd"
	"runtime"
	"time"
)

//go:embed index.html
var indexHTML []byte

// Handlers GUI 后端操作回调（由 cmd 注入，接引擎/profile/store）。
type Handlers struct {
	// Entities 实体列表（仪表盘+浏览）。
	Entities func() ([]Entity, error)
	// Overview 仪表盘概览。
	Overview func() (Overview, error)
	// Collect 触发采集（tool 为空=全部）。
	Collect func(tool string) (string, error)
	// Export 导出（dryRun=true 预览）。
	Export func(to string, dryRun bool) (string, []string, error)
	// Snapshots 快照列表。
	Snapshots func() ([]Snapshot, error)
	// SnapshotCreate 创建快照。
	SnapshotCreate func(note string) (string, error)
	// SnapshotRestore 恢复快照。
	SnapshotRestore func(id string) (string, error)
}

// Entity 展示实体。
type Entity struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Note string `json:"note"`
}

// Overview 仪表盘概览。
type Overview struct {
	Tools     int    `json:"tools"`
	Entities  int    `json:"entities"`
	Snapshots int    `json:"snapshots"`
	RepoRoot  string `json:"repo_root"`
}

// Snapshot 快照条目。
type Snapshot struct {
	ID    string `json:"id"`
	Note  string `json:"note"`
	Files int    `json:"files"`
}

// Server 本地 Web 界面服务。
type Server struct {
	repoRoot string
	handlers Handlers
	srv      *http.Server
	ln       net.Listener
}

// NewServer 创建服务（随机空闲端口）。
func NewServer(repoRoot string, h Handlers) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{repoRoot: repoRoot, handlers: h, ln: ln}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.serveIndex)
	mux.HandleFunc("/api/overview", s.serveOverview)
	mux.HandleFunc("/api/entities", s.serveEntities)
	mux.HandleFunc("/api/collect", s.serveCollect)
	mux.HandleFunc("/api/export", s.serveExport)
	mux.HandleFunc("/api/snapshots", s.serveSnapshots)
	mux.HandleFunc("/api/snapshot/create", s.serveSnapshotCreate)
	mux.HandleFunc("/api/snapshot/restore", s.serveSnapshotRestore)
	s.srv = &http.Server{Handler: mux}
	return s, nil
}

func (s *Server) Addr() string { return "http://" + s.ln.Addr().String() }

func (s *Server) Start() error {
	go s.srv.Serve(s.ln)
	time.Sleep(100 * time.Millisecond)
	openBrowser(s.Addr())
	return nil
}

func (s *Server) Stop(ctx context.Context) error { return s.srv.Shutdown(ctx) }

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (s *Server) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) serveOverview(w http.ResponseWriter, r *http.Request) {
	if s.handlers.Overview == nil {
		s.writeJSON(w, Overview{RepoRoot: s.repoRoot})
		return
	}
	ov, err := s.handlers.Overview()
	if err != nil {
		s.writeJSON(w, Overview{RepoRoot: s.repoRoot})
		return
	}
	s.writeJSON(w, ov)
}

func (s *Server) serveEntities(w http.ResponseWriter, r *http.Request) {
	if s.handlers.Entities == nil {
		s.writeJSON(w, []Entity{})
		return
	}
	items, err := s.handlers.Entities()
	if err != nil {
		s.writeJSON(w, []Entity{})
		return
	}
	s.writeJSON(w, items)
}

func (s *Server) serveCollect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Tool string `json:"tool"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if s.handlers.Collect == nil {
		s.writeJSON(w, map[string]string{"error": "未接入"})
		return
	}
	msg, err := s.handlers.Collect(req.Tool)
	if err != nil {
		s.writeJSON(w, map[string]string{"ok": "false", "error": err.Error()})
		return
	}
	s.writeJSON(w, map[string]string{"ok": "true", "message": msg})
}

func (s *Server) serveExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		To     string `json:"to"`
		DryRun bool   `json:"dryRun"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if s.handlers.Export == nil {
		s.writeJSON(w, map[string]string{"error": "未接入"})
		return
	}
	msg, files, err := s.handlers.Export(req.To, req.DryRun)
	if err != nil {
		s.writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.writeJSON(w, map[string]any{"ok": true, "message": msg, "files": files})
}

func (s *Server) serveSnapshots(w http.ResponseWriter, r *http.Request) {
	if s.handlers.Snapshots == nil {
		s.writeJSON(w, []Snapshot{})
		return
	}
	list, err := s.handlers.Snapshots()
	if err != nil {
		s.writeJSON(w, []Snapshot{})
		return
	}
	s.writeJSON(w, list)
}

func (s *Server) serveSnapshotCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Note string `json:"note"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if s.handlers.SnapshotCreate == nil {
		s.writeJSON(w, map[string]string{"error": "未接入"})
		return
	}
	id, err := s.handlers.SnapshotCreate(req.Note)
	if err != nil {
		s.writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.writeJSON(w, map[string]any{"ok": true, "id": id})
}

func (s *Server) serveSnapshotRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if s.handlers.SnapshotRestore == nil {
		s.writeJSON(w, map[string]string{"error": "未接入"})
		return
	}
	msg, err := s.handlers.SnapshotRestore(req.ID)
	if err != nil {
		s.writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	s.writeJSON(w, map[string]any{"ok": true, "message": msg})
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	hidecmd.Hide(cmd)
	_ = cmd.Start()
}

// Run 启动服务并阻塞。
func Run(repoRoot string, h Handlers) error {
	s, err := NewServer(repoRoot, h)
	if err != nil {
		return err
	}
	if err := s.Start(); err != nil {
		return err
	}
	fmt.Println("cfg4ai GUI 已启动:", s.Addr())
	fmt.Println("Ctrl+C 停止")
	select {}
}
