package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/trusted-technologies/cuttlefish/internal/master"
	"github.com/trusted-technologies/cuttlefish/internal/shared"
)

func main() {
	var cfg master.Config
	flag.StringVar(&cfg.HTTPAddr, "addr", shared.EnvDefault("MASTER_ADDR", ":8080"), "HTTP listen address")
	flag.StringVar(&cfg.Token, "token", shared.EnvDefault("MASTER_TOKEN", ""), "shared secret required from slaves")
	flag.Parse()

	templates, err := master.LoadTemplates()
	if err != nil {
		slog.Error("failed to load templates", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	m := master.New(cfg, templates)
	if err := m.Run(ctx); err != nil {
		slog.Error("master failed", "error", err)
		os.Exit(1)
	}
}
