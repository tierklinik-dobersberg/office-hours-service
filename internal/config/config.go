package config

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
	"github.com/tierklinik-dobersberg/apis/gen/go/tkd/calendar/v1/calendarv1connect"
	"github.com/tierklinik-dobersberg/apis/gen/go/tkd/idm/v1/idmv1connect"
	"github.com/tierklinik-dobersberg/apis/pkg/cli"
)

type Config struct {
	IdmURL          string `env:"IDM_URL"`
	EventsService   string `env:"EVENT_SERVICE"`
	CalendarService string `env:"CALENDAR_SERVICE"`
	MongoURL        string `env:"MONGO_URL,required"`
	Database        string `env:"DATABASE,default=cis"`
}

func LoadConfig(ctx context.Context) (*Config, error) {
	var cfg Config

	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (cfg *Config) ConfigureProviders() *Providers {
	hcli := cli.NewInsecureHttp2Client()
	return &Providers{
		Config:               cfg,
		HolidayServiceClient: calendarv1connect.NewHolidayServiceClient(hcli, cfg.CalendarService),
		UserServiceClient:    idmv1connect.NewUserServiceClient(hcli, cfg.IdmURL),
		RoleServiceClient:    idmv1connect.NewRoleServiceClient(hcli, cfg.IdmURL),
	}
}
