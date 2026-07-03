package slave

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"github.com/trusted-technologies/cuttlefish/internal/shared"
)

// HandleExec runs a network tool and streams output via SSE.
func HandleExec(w http.ResponseWriter, r *http.Request, tool string) {
	req, err := parseCommandRequest(r)
	if err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := shared.ValidateTarget(req.Target); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	count := shared.Clamp(req.Count, 1, 30)
	timeout := time.Duration(shared.Clamp(req.Timeout, 5, 120)) * time.Second

	sse, ok := shared.NewSSEWriter(w)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	cmd := buildCommand(ctx, tool, req.Target, req.IPv6, count)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = sse.Event("result", shared.CommandResult{Error: err.Error(), Done: true})
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = sse.Event("result", shared.CommandResult{Error: err.Error(), Done: true})
		return
	}
	if err := cmd.Start(); err != nil {
		_ = sse.Event("result", shared.CommandResult{Error: err.Error(), Done: true})
		return
	}

	done := make(chan struct{})
	go streamPipe(sse, bufio.NewReader(stdout), done)
	go streamPipe(sse, bufio.NewReader(stderr), done)

	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		_ = cmd.Process.Kill()
	}
	_ = sse.Event("result", shared.CommandResult{Done: true})
}

func parseCommandRequest(r *http.Request) (shared.CommandRequest, error) {
	var req shared.CommandRequest
	if r.Method == http.MethodPost {
		if err := jsonDecode(r.Body, &req); err != nil {
			return req, err
		}
		return req, nil
	}
	req.Target = r.URL.Query().Get("target")
	req.IPv6 = r.URL.Query().Get("ipv6") == "true"
	if c := r.URL.Query().Get("count"); c != "" {
		if v, err := strconv.Atoi(c); err == nil {
			req.Count = v
		}
	}
	if t := r.URL.Query().Get("timeout"); t != "" {
		if v, err := strconv.Atoi(t); err == nil {
			req.Timeout = v
		}
	}
	if req.Count == 0 {
		req.Count = 4
	}
	if req.Timeout == 0 {
		req.Timeout = 30
	}
	return req, nil
}

func buildCommand(ctx context.Context, tool, target string, ipv6 bool, count int) *exec.Cmd {
	switch tool {
	case "ping":
		if ipv6 {
			return exec.CommandContext(ctx, "ping6", "-c", fmt.Sprintf("%d", count), target)
		}
		return exec.CommandContext(ctx, "ping", "-c", fmt.Sprintf("%d", count), target)
	case "mtr":
		return exec.CommandContext(ctx, "mtr", "--report", "--report-cycles", fmt.Sprintf("%d", count), target)
	case "traceroute":
		if ipv6 {
			return exec.CommandContext(ctx, "traceroute6", target)
		}
		return exec.CommandContext(ctx, "traceroute", target)
	case "iperf":
		// iperf3 client mode; runs for up to ~10 seconds by default.
		return exec.CommandContext(ctx, "iperf3", "-c", target, "-t", "10", "-J")
	default:
		return exec.CommandContext(ctx, "false")
	}
}

func streamPipe(sse *shared.SSEWriter, r *bufio.Reader, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
		}
		line, err := r.ReadString('\n')
		if line != "" {
			_ = sse.Event("result", shared.CommandResult{Line: line})
		}
		if err != nil {
			return
		}
	}
}
