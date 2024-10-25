package config

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
	"github.com/tierklinik-dobersberg/apis/gen/go/tkd/calendar/v1/calendarv1connect"
	"github.com/tierklinik-dobersberg/apis/gen/go/tkd/events/v1/eventsv1connect"
	"github.com/tierklinik-dobersberg/apis/gen/go/tkd/idm/v1/idmv1connect"
	"github.com/tierklinik-dobersberg/apis/pkg/cli"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/repo"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/resolver"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/watcher"
)

type Config struct {
	AllowedOrigins  []string `env:"ALLOWED_ORIGINS,default=*"`
	ListenAddress   string   `env:"LISTEN,default=:8081"`
	IdmURL          string   `env:"IDM_URL"`
	EventsService   string   `env:"EVENT_SERVICE"`
	CalendarService string   `env:"CALENDAR_SERVICE"`
	MongoURL        string   `env:"MONGO_URL,required"`
	Database        string   `env:"DATABASE,default=cis"`
}

func LoadConfig(ctx context.Context) (*Config, error) {
	var cfg Config

	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (cfg *Config) ConfigureProviders(ctx context.Context) (*Providers, error) {
	hcli := cli.NewInsecureHttp2Client()

	repo, err := repo.NewRepo(ctx, cfg.MongoURL, cfg.Database)
	if err != nil {
		return nil, err
	}

	resolver := resolver.NewResolver(repo, calendarv1connect.NewHolidayServiceClient(hcli, cfg.CalendarService))

	var w *watcher.Watcher
	if cfg.EventsService != "" {
		w = watcher.New(
			resolver,
			eventsv1connect.NewEventServiceClient(hcli, cfg.EventsService),
		)

		// Immediately start the watcher
		w.Start(ctx)
	}

	return &Providers{
		Config:   cfg,
		Repo:     repo,
		Resolver: resolver,
		Watcher:  w,

		UserServiceClient: idmv1connect.NewUserServiceClient(hcli, cfg.IdmURL),
		RoleServiceClient: idmv1connect.NewRoleServiceClient(hcli, cfg.IdmURL),
	}, nil
}
