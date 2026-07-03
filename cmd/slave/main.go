package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/trusted-technologies/cuttlefish/internal/shared"
	"github.com/trusted-technologies/cuttlefish/internal/slave"
)

func main() {
	var cfg slave.Config
	flag.StringVar(&cfg.ID, "id", shared.EnvDefault("SLAVE_ID", ""), "unique slave id")
	flag.StringVar(&cfg.Name, "name", shared.EnvDefault("SLAVE_NAME", ""), "human readable name")
	flag.StringVar(&cfg.PublicURL, "public-url", shared.EnvDefault("SLAVE_PUBLIC_URL", ""), "publicly reachable URL of this slave")
	flag.StringVar(&cfg.MasterURL, "master-url", shared.EnvDefault("MASTER_URL", ""), "master registration URL")
	flag.StringVar(&cfg.Token, "token", shared.EnvDefault("SLAVE_TOKEN", ""), "shared secret with master")
	flag.StringVar(&cfg.Location, "location", shared.EnvDefault("SLAVE_LOCATION", ""), "location label")
	flag.StringVar(&cfg.HTTPAddr, "addr", shared.EnvDefault("SLAVE_ADDR", ":8080"), "HTTP listen address")
	flag.StringVar(&cfg.FilesDir, "files-dir", shared.EnvDefault("FILES_DIR", "/data/files"), "directory for test files")
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
