package master

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/trusted-technologies/cuttlefish/internal/shared"
)

// Config holds master configuration.
type Config struct {
	HTTPAddr string
	Token    string
}

// Master is the looking-glass master server.
type Master struct {
	cfg       Config
	slaves    map[string]*shared.SlaveInfo
	mu        sync.RWMutex
	templates *template.Template
}

// New creates a new master server.
func New(cfg Config, templates *template.Template) *Master {
	return &Master{
		cfg:       cfg,
		slaves:    make(map[string]*shared.SlaveInfo),
		templates: templates,
	}
}

// Run starts the HTTP server.
func (m *Master) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", m.handleIndex)
	mux.HandleFunc("/slave/", m.handleSlavePage)
	mux.HandleFunc("/api/myip", m.handleMyIP)
	mux.HandleFunc("/api/slaves", m.handleListSlaves)
	mux.HandleFunc("/api/slaves/", m.handleSlaveAPI)
	mux.HandleFunc("/internal/register", m.handleRegister)
	mux.HandleFunc("/internal/heartbeat", m.handleHeartbeat)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	srv := &http.Server{
		Addr:    m.cfg.HTTPAddr,
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("master listening", "addr", m.cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

func (m *Master) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data := m.slaveListData()
	if err := m.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		slog.Error("template error", "error", err)
		return
	}
}

func (m *Master) handleSlavePage(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/slave/")
	m.mu.RLock()
	s, ok := m.slaves[id]
	m.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	files := shared.TestFiles(s.PublicURL)
	data := map[string]any{
		"Slave": s,
		"Files": files,
	}
	if err := m.templates.ExecuteTemplate(w, "slave.html", data); err != nil {
		slog.Error("template error", "error", err)
		return
	}
}

func (m *Master) handleMyIP(w http.ResponseWriter, r *http.Request) {
	ip := shared.RemoteIP(r)
	info := shared.MyIPInfo{}
	if shared.IsIPv6(ip) {
		info.IPv6 = ip
	} else {
		info.IPv4 = ip
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

func (m *Master) handleListSlaves(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*shared.SlaveInfo, 0, len(m.slaves))
	for _, s := range m.slaves {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (m *Master) handleSlaveAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/slaves/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	id, action := parts[0], parts[1]

	m.mu.RLock()
	s, ok := m.slaves[id]
	m.mu.RUnlock()
	if !ok {
		http.Error(w, "slave not found", http.StatusNotFound)
		return
	}

	switch action {
	case "ping", "mtr", "traceroute", "iperf":
		m.proxyCommand(w, r, s, action)
	case "files":
		m.proxyFiles(w, r, s)
	default:
		http.NotFound(w, r)
	}
}

func (m *Master) proxyCommand(w http.ResponseWriter, r *http.Request, s *shared.SlaveInfo, tool string) {
	url := fmt.Sprintf("%s/exec/%s", s.PublicURL, tool)
	if r.Method == http.MethodGet {
		url += "?" + r.URL.RawQuery
	}
	var body io.Reader
	if r.Method == http.MethodPost {
		body = r.Body
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, url, body)
	if err != nil {
		http.Error(w, "failed to build request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "failed to reach slave: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (m *Master) proxyFiles(w http.ResponseWriter, r *http.Request, s *shared.SlaveInfo) {
	files := shared.TestFiles(s.PublicURL)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(files)
}

func (m *Master) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req shared.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if m.cfg.Token != "" && req.Token != m.cfg.Token {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	m.mu.Lock()
	iperfPort := req.IperfPort
	if iperfPort == "" {
		iperfPort = "5201"
	}
	m.slaves[req.ID] = &shared.SlaveInfo{
		ID:        req.ID,
		Name:      req.Name,
		PublicURL: strings.TrimSuffix(req.PublicURL, "/"),
		IPv4:      req.IPv4,
		IPv6:      req.IPv6,
		Location:  req.Location,
		IperfPort: iperfPort,
		LastSeen:  time.Now(),
	}
	m.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (m *Master) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req shared.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if m.cfg.Token != "" && req.Token != m.cfg.Token {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	m.mu.Lock()
	if s, ok := m.slaves[req.ID]; ok {
		s.IPv4 = req.IPv4
		s.IPv6 = req.IPv6
		s.LastSeen = time.Now()
	}
	m.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (m *Master) slaveListData() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*shared.SlaveInfo, 0, len(m.slaves))
	for _, s := range m.slaves {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return map[string]any{"Slaves": list}
}

// staticFS placeholder, replaced in templates.go.
var staticFS fs.FS
