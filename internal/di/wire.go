//go:build wireinject
// +build wireinject

package di

import (
	"net/http"

	"github.com/google/wire"

	"sales-analytics/config"
)

func Init() (*http.Server, error) {
	wire.Build(
		config.Load,
		ProvideLogger,
		ProvideDB,
		ProvideStore,
		ProvideJobRepository,
		ProvideCsvPath,
		ProvideIngestionService,
		ProvideAnalyticsService,
		ProvideGin,
		ProvideHTTP,
	)
	return nil, nil
}
