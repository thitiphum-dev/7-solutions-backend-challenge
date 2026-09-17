package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	httpadapter "github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/http"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/config"
)

func Run(ctx context.Context, cfg *config.Config) error {
	router := httpadapter.NewRouter()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen http: %w", err)
	}

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
			"shutting down HTTP server",
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
