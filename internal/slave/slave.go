package slave

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/trusted-technologies/cuttlefish/internal/shared"
)

// Config holds slave configuration.
type Config struct {
	ID        string
	Name      string
	PublicURL string
	IPv4      string
	IPv6      string
	MasterURL string
	Token     string
	Location  string
	HTTPAddr  string
	IperfPort string
	FileSizes []string
	FilesDir  string
}

// Slave represents a running slave agent.
type Slave struct {
	cfg Config
}

// New creates a new slave instance.
func New(cfg Config) *Slave {
	return &Slave{cfg: cfg}
}

// Run starts the slave: registers with master, starts heartbeat, and serves HTTP.
func (s *Slave) Run(ctx context.Context) error {
	if err := EnsureTestFiles(s.cfg.FilesDir, shared.FilterFileSizes(s.cfg.FileSizes)); err != nil {
		return fmt.Errorf("ensure test files: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/info", s.handleInfo)
	mux.HandleFunc("/exec/ping", func(w http.ResponseWriter, r *http.Request) { HandleExec(w, r, "ping") })
	mux.HandleFunc("/exec/mtr", func(w http.ResponseWriter, r *http.Request) { HandleExec(w, r, "mtr") })
	mux.HandleFunc("/exec/traceroute", func(w http.ResponseWriter, r *http.Request) { HandleExec(w, r, "traceroute") })
	mux.HandleFunc("/exec/iperf", func(w http.ResponseWriter, r *http.Request) { HandleExec(w, r, "iperf") })
	mux.HandleFunc("/files/", s.handleFiles)

	srv := &http.Server{
		Addr:    s.cfg.HTTPAddr,
		Handler: mux,
	}

	if s.cfg.MasterURL != "" {
		if err := s.register(ctx); err != nil {
			slog.Warn("failed to register with master", "error", err)
		}
		go s.heartbeatLoop(ctx)
	}
	go startIperfServer(ctx, s.cfg.IperfPort)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("slave listening", "addr", s.cfg.HTTPAddr)
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

func (s *Slave) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Slave) handleInfo(w http.ResponseWriter, r *http.Request) {
	ipv4, ipv6 := s.getIPs()
	info := struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		PublicURL string `json:"public_url"`
		IPv4     string `json:"ipv4"`
		IPv6     string `json:"ipv6"`
		Location string `json:"location"`
	}{
		ID:        s.cfg.ID,
		Name:      s.cfg.Name,
		PublicURL: s.cfg.PublicURL,
		IPv4:      ipv4,
		IPv6:      ipv6,
		Location:  s.cfg.Location,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

func (s *Slave) handleFiles(w http.ResponseWriter, r *http.Request) {
	sizeName := r.URL.Path[len("/files/"):]
	ServeFile(w, r, sizeName, s.cfg.FilesDir, shared.FilterFileSizes(s.cfg.FileSizes))
}

// getIPs returns the configured public IPs or auto-detected ones.
func (s *Slave) getIPs() (string, string) {
	if s.cfg.IPv4 != "" || s.cfg.IPv6 != "" {
		return s.cfg.IPv4, s.cfg.IPv6
	}
	ipv4, ipv6, err := shared.DetectIPs()
	if err != nil {
		slog.Warn("failed to detect IPs", "error", err)
	}
	return ipv4, ipv6
}

func (s *Slave) register(ctx context.Context) error {
	ipv4, ipv6 := s.getIPs()
	reqBody := shared.RegisterRequest{
		ID:        s.cfg.ID,
		Name:      s.cfg.Name,
		PublicURL: s.cfg.PublicURL,
		Token:     s.cfg.Token,
		IPv4:      ipv4,
		IPv6:      ipv6,
		Location:  s.cfg.Location,
		IperfPort: s.cfg.IperfPort,
		FileSizes: s.cfg.FileSizes,
	}
	return s.postJSON(ctx, s.cfg.MasterURL+"/internal/register", reqBody)
}

func (s *Slave) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ipv4, ipv6 := s.getIPs()
			req := shared.HeartbeatRequest{
				ID:    s.cfg.ID,
				Token: s.cfg.Token,
				IPv4:  ipv4,
				IPv6:  ipv6,
			}
			if err := s.postJSON(ctx, s.cfg.MasterURL+"/internal/heartbeat", req); err != nil {
				slog.Warn("heartbeat failed, attempting to re-register", "error", err)
				if regErr := s.register(ctx); regErr != nil {
					slog.Warn("re-registration failed", "error", regErr)
				} else {
					slog.Info("re-registered with master")
				}
			}
		}
	}
}

func (s *Slave) postJSON(ctx context.Context, url string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return nil
}

