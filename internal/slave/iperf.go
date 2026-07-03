package slave

import (
	"context"
	"log/slog"
	"os/exec"
	"time"
)

// startIperfServer keeps an iperf3 server running until ctx is cancelled.
// The server listens on all interfaces at the given port.
func startIperfServer(ctx context.Context, port string) {
	if port == "" {
		port = "5201"
	}
	for {
		slog.Info("starting iperf3 server", "port", port)
		cmd := exec.CommandContext(ctx, "iperf3", "-s", "-p", port)
		if err := cmd.Run(); err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			slog.Warn("iperf3 server exited", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}
