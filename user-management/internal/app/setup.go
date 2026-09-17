package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	httpadapter "github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/http"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/http/handler"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/http/middleware"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/mongodb"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/security"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/application"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/config"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/user"
)

type dependencies struct {
	router         http.Handler
	mongoClient    *mongodb.Client
	userRepository user.Repository
}

func setup(ctx context.Context, cfg *config.Config) (*dependencies, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	mongoClient, err := mongodb.Connect(connectCtx, cfg.MongoURI)
	if err != nil {
		return nil, err
	}

	userRepository := mongodb.NewUserRepository(
		mongoClient.Database(cfg.MongoDatabase),
	)

	indexCtx, cancelIndex := context.WithTimeout(ctx, 5*time.Second)
	err = userRepository.EnsureIndexes(indexCtx)
	cancelIndex()
	if err != nil {
		closeCtx, cancelClose := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancelClose()

		_ = mongoClient.Disconnect(closeCtx)
		return nil, fmt.Errorf("ensure user indexes: %w", err)
	}

	slog.Info(
		"MongoDB connection and indexes established",
	)

	hasher := security.NewBcryptHasher()
	jwtIssuer := security.NewJWT(cfg.JWTSecret, 24*time.Hour)

	authService := application.NewAuthService(
		userRepository,
		hasher,
		jwtIssuer,
	)
	userService := application.NewUserService(userRepository)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	authMiddleware := middleware.NewAuth(jwtIssuer)
	router := httpadapter.NewRouter(
		authHandler,
		userHandler,
		authMiddleware,
	)

	return &dependencies{
		router:         router,
		mongoClient:    mongoClient,
		userRepository: userRepository,
	}, nil
}

func (d *dependencies) close(ctx context.Context) error {
	if err := d.mongoClient.Disconnect(ctx); err != nil {
		return fmt.Errorf("disconnect mongodb: %w", err)
	}

	return nil
}
