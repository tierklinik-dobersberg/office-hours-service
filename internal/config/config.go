package config

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
	"github.com/tierklinik-dobersberg/apis/pkg/discovery"
	"github.com/tierklinik-dobersberg/apis/pkg/discovery/wellknown"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/repo"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/resolver"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/watcher"
)

type Config struct {
	AllowedOrigins []string `env:"ALLOWED_ORIGINS,default=*"`
	ListenAddress  string   `env:"LISTEN,default=:8081"`

	MongoURL string `env:"MONGO_URL,required"`
	Database string `env:"DATABASE,default=cis"`
}

func LoadConfig(ctx context.Context) (*Config, error) {
	var cfg Config

	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (cfg *Config) ConfigureProviders(ctx context.Context, catalog discovery.Discoverer) (*Providers, error) {
	repo, err := repo.NewRepo(ctx, cfg.MongoURL, cfg.Database)
	if err != nil {
		return nil, err
	}

	resolver := resolver.NewResolver(repo, catalog)

	var w *watcher.Watcher

	cli, err := wellknown.EventService.Create(ctx, catalog)
	if err == nil {
		w = watcher.New(
			resolver,
			cli,
		)

		// Immediately start the watcher
		w.Start(ctx)
	}

	return &Providers{
		Config:   cfg,
		Repo:     repo,
		Resolver: resolver,
		Watcher:  w,

		Catalog: catalog,
	}, nil
}
