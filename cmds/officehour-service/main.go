package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	connect "github.com/bufbuild/connect-go"
	"github.com/bufbuild/protovalidate-go"
	"github.com/tierklinik-dobersberg/apis/gen/go/tkd/idm/v1/idmv1connect"
	"github.com/tierklinik-dobersberg/apis/gen/go/tkd/office_hours/v1/office_hoursv1connect"
	"github.com/tierklinik-dobersberg/apis/pkg/auth"
	"github.com/tierklinik-dobersberg/apis/pkg/cors"
	"github.com/tierklinik-dobersberg/apis/pkg/log"
	"github.com/tierklinik-dobersberg/apis/pkg/server"
	"github.com/tierklinik-dobersberg/apis/pkg/validator"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/config"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/service"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		slog.Error("failed to load configuration", slog.Any("error", err.Error()))
		os.Exit(-1)
	}

	protoValidator, err := protovalidate.New()
	if err != nil {
		slog.Error("failed to prepare protovalidate", slog.Any("error", err.Error()))
		os.Exit(-1)
	}

	// TODO(ppacher): privacy-interceptor
	interceptors := connect.WithInterceptors(
		log.NewLoggingInterceptor(),
		validator.NewInterceptor(protoValidator),
	)

	if cfg.IdmURL != "" {
		roleClient := idmv1connect.NewRoleServiceClient(http.DefaultClient, cfg.IdmURL)

		authInterceptor := auth.NewAuthAnnotationInterceptor(
			protoregistry.GlobalFiles,
			auth.NewIDMRoleResolver(roleClient),
			auth.RemoteHeaderExtractor,
		)

		interceptors = connect.WithOptions(interceptors, connect.WithInterceptors(authInterceptor))
	}

	corsConfig := cors.Config{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowCredentials: true,
	}

	providers, err := cfg.ConfigureProviders(ctx)
	if err != nil {
		slog.Error("failed to configure providers", slog.Any("error", err.Error()))
		os.Exit(-1)
	}

	svc := service.New(providers)

	serveMux := http.NewServeMux()

	path, handler := office_hoursv1connect.NewOfficeHourServiceHandler(svc, interceptors)
	serveMux.Handle(path, handler)

	loggingHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			slog.Info("handled request", slog.Any("method", r.Method), slog.Any("path", r.URL.Path), slog.Any("duration", time.Since(start).String()))
		})
	}

	// Create the server
	srv, err := server.CreateWithOptions(cfg.ListenAddress, loggingHandler(serveMux), server.WithCORS(corsConfig))
	if err != nil {
		slog.Error("failed to setup server", slog.Any("error", err.Error()))
		os.Exit(-1)
	}

	if err := server.Serve(ctx, srv); err != nil {
		slog.Error("failed to serve", slog.Any("error", err.Error()))
		os.Exit(-1)
	}
}
