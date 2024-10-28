package config

import (
	"github.com/tierklinik-dobersberg/apis/pkg/discovery"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/repo"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/resolver"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/watcher"
)

type Providers struct {
	*Config

	Repo     *repo.Repo
	Resolver *resolver.Resolver
	Watcher  *watcher.Watcher

	Catalog discovery.Discoverer
}
