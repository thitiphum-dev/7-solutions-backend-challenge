package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	ServiceName   string `env:"SERVICE_NAME,required"`
	Port          string `env:"PORT,required"`
	MongoURI      string `env:"MONGO_URI,required"`
	MongoDatabase string `env:"MONGO_DATABASE,required"`
	JWTSecret     string `env:"JWT_SECRET,required"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return &cfg, nil
}
