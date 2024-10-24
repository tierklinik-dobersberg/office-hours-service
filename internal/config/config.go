package config

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	IdmURL        string `env:"IDM_URL"`
	EventsService string `env:"EVENT_SERVICE"`
	MongoURL      string `env:"MONGO_URL,required"`
	Database      string `env:"DATABASE,default=cis"`
}

func LoadConfig(ctx context.Context) (*Config, error) {
	var cfg Config

	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}
