package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	httpadapter "github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/http"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/mongodb"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/config"
)

type dependencies struct {
	router      http.Handler
	mongoClient *mongodb.Client
}

func setup(ctx context.Context, cfg *config.Config) (*dependencies, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	mongoClient, err := mongodb.Connect(connectCtx, cfg.MongoURI)
	if err != nil {
		return nil, err
	}

	slog.Info(
		"MongoDB connection established",
	)

	router := httpadapter.NewRouter()

	return &dependencies{
		router:      router,
		mongoClient: mongoClient,
	}, nil
}

func (d *dependencies) close(ctx context.Context) error {
	if err := d.mongoClient.Disconnect(ctx); err != nil {
		return fmt.Errorf("disconnect mongodb: %w", err)
	}

	return nil
}
