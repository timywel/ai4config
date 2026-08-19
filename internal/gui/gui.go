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
	"runtime"
	"time"
)

//go:embed index.html
var indexHTML []byte

// Server 本地 Web 界面服务。
type Server struct {
	repoRoot string
	srv      *http.Server
	ln       net.Listener
}

// EntitiesProvider 实体数据源（由 cmd 注入，读 profile）。
type EntitiesProvider func() ([]Entity, error)

// Entity 一个展示实体。
type Entity struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Note string `json:"note"`
}

var provider EntitiesProvider

// NewServer 创建服务（随机空闲端口）。
func NewServer(repoRoot string, p EntitiesProvider) (*Server, error) {
	provider = p
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex)
	mux.HandleFunc("/api/entities", serveEntities)
	s := &Server{repoRoot: repoRoot, srv: &http.Server{Handler: mux}, ln: ln}
	return s, nil
}

// Addr 返回监听地址。
func (s *Server) Addr() string { return "http://" + s.ln.Addr().String() }

// Start 启动服务（非阻塞）+ 打开浏览器。
func (s *Server) Start() error {
	go s.srv.Serve(s.ln)
	time.Sleep(100 * time.Millisecond)
	openBrowser(s.Addr())
	return nil
}

// Stop 关闭。
func (s *Server) Stop(ctx context.Context) error { return s.srv.Shutdown(ctx) }

func serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func serveEntities(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if provider == nil {
		json.NewEncoder(w).Encode([]Entity{})
		return
	}
	items, err := provider()
	if err != nil {
		json.NewEncoder(w).Encode([]Entity{})
		return
	}
	json.NewEncoder(w).Encode(items)
}

// openBrowser 跨平台打开浏览器。
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
	_ = cmd.Start()
}

// Run 启动服务并阻塞（直到中断）。
func Run(repoRoot string, p EntitiesProvider) error {
	s, err := NewServer(repoRoot, p)
	if err != nil {
		return err
	}
	if err := s.Start(); err != nil {
		return err
	}
	fmt.Println("cfg4ai GUI 已启动:", s.Addr())
	fmt.Println("Ctrl+C 停止")
	select {} // 阻塞
}
