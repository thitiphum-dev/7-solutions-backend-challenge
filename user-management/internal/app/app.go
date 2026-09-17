package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	usercountworker "github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/worker"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/config"
)

func Run(ctx context.Context, cfg *config.Config) error {
	deps, err := setup(ctx, cfg)
	if err != nil {
		return err
	}

	defer func() {
		closeCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := deps.close(closeCtx); err != nil {
			slog.Error(
				"failed to close application resources",
				"error", err,
			)
		}
	}()

	workerCtx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	defer func() {
		cancel()
		workers.Wait()
	}()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           deps.router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen http: %w", err)
	}

	workers.Go(func() {
		usercountworker.RunUserCount(
			workerCtx,
			deps.userRepository,
			10*time.Second,
		)
	})

	slog.Info(
		"HTTP server is running",
		"service", cfg.ServiceName,
		"port", cfg.Port,
	)

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.Serve(listener)
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve http: %w", err)
		}

	case <-ctx.Done():
		slog.Info(
			"shutting down application",
			"service", cfg.ServiceName,
		)

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
	}

	return nil
}
