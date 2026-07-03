package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/trusted-technologies/cuttlefish/internal/shared"
	"github.com/trusted-technologies/cuttlefish/internal/slave"
)

func main() {
	var cfg slave.Config
	flag.StringVar(&cfg.ID, "id", shared.EnvDefault("SLAVE_ID", ""), "unique slave id")
	flag.StringVar(&cfg.Name, "name", shared.EnvDefault("SLAVE_NAME", ""), "human readable name")
	flag.StringVar(&cfg.PublicURL, "public-url", shared.EnvDefault("SLAVE_PUBLIC_URL", ""), "publicly reachable URL of this slave")
	flag.StringVar(&cfg.IPv4, "ipv4", shared.EnvDefault("SLAVE_IPV4", ""), "public IPv4 address (overrides auto-detection)")
	flag.StringVar(&cfg.IPv6, "ipv6", shared.EnvDefault("SLAVE_IPV6", ""), "public IPv6 address (overrides auto-detection)")
	flag.StringVar(&cfg.MasterURL, "master-url", shared.EnvDefault("MASTER_URL", ""), "master registration URL")
	flag.StringVar(&cfg.Token, "token", shared.EnvDefault("SLAVE_TOKEN", ""), "shared secret with master")
	flag.StringVar(&cfg.Location, "location", shared.EnvDefault("SLAVE_LOCATION", ""), "location label")
	flag.StringVar(&cfg.HTTPAddr, "addr", shared.EnvDefault("SLAVE_ADDR", ":8080"), "HTTP listen address")
	flag.StringVar(&cfg.IperfPort, "iperf-port", shared.EnvDefault("IPERF_PORT", "5201"), "iperf3 server port")
	flag.StringVar(&cfg.FilesDir, "files-dir", shared.EnvDefault("FILES_DIR", "/data/files"), "directory for test files")
	flag.Func("stats-interfaces", "comma-separated network interfaces to monitor (e.g. eth0,ens18)", func(s string) error {
		cfg.StatsInterfaces = splitTrim(s)
		return nil
	})
	if s := shared.EnvDefault("STATS_INTERFACES", ""); s != "" {
		cfg.StatsInterfaces = splitTrim(s)
	}

	flag.Func("files-sizes", "comma-separated test file sizes (e.g. 1M,10M,100M,1G)", func(s string) error {
		var err error
		cfg.FileSizes, err = shared.ParseFileSizesList(s)
		return err
	})
	if s := shared.EnvDefault("FILES_SIZES", ""); s != "" {
		sizes, err := shared.ParseFileSizesList(s)
		if err != nil {
			slog.Error("invalid FILES_SIZES", "error", err)
			os.Exit(1)
		}
		cfg.FileSizes = sizes
	}
	flag.Parse()

	if cfg.ID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			slog.Error("failed to get hostname", "error", err)
			os.Exit(1)
		}
		cfg.ID = hostname
	}
	if cfg.Name == "" {
		cfg.Name = cfg.ID
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	s := slave.New(cfg)
	if err := s.Run(ctx); err != nil {
		slog.Error("slave failed", "error", err)
		os.Exit(1)
	}
}

func splitTrim(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
