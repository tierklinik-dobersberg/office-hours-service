package config

import (
	"github.com/tierklinik-dobersberg/apis/gen/go/tkd/idm/v1/idmv1connect"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/repo"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/resolver"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/watcher"
)

type Providers struct {
	*Config

	Repo     *repo.Repo
	Resolver *resolver.Resolver
	Watcher  *watcher.Watcher

	idmv1connect.UserServiceClient
	idmv1connect.RoleServiceClient
}
