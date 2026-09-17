package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/app"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/config"
)

func main() {

	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, nil),
	)
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := app.Run(ctx, cfg); err != nil {
		slog.Error("application stopped with error", "error", err)
		os.Exit(1)
	}
}
