// this file takes care of providing the dependencies to the wire.go file, which is the entry point of the application

package di

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"sales-analytics/config"
	"sales-analytics/internal"
	"sales-analytics/internal/handler"
	"sales-analytics/internal/repository"
	"sales-analytics/internal/service/analytics"
	"sales-analytics/internal/service/ingestion"
	"sales-analytics/pkg/orm"
)

func ProvideLogger(
	config config.Config,
) (*zap.Logger, error) {
	var zapConfig zap.Config
	if config.App.Mode == "production" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}
	return zapConfig.Build()
}

func ProvideDB(
	config config.Config,
	logger *zap.Logger,
) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		config.DB.User, config.DB.Password, config.DB.Host, config.DB.Port, config.DB.Name)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Connected to MySQL database",
		zap.String("host", config.DB.Host),
		zap.String("port", config.DB.Port),
		zap.String("name", config.DB.Name))

	return db, nil
}

func ProvideStore(
	db *sql.DB,
) *orm.Store {
	return orm.New(db)
}

func ProvideJobRepository(
	store *orm.Store,
) repository.JobRepository {
	return repository.NewJobRepo(store)
}

func ProvideAnalyticsService(
	db *sql.DB,
	logger *zap.Logger,
) analytics.Service {
	return analytics.New(db, logger)
}

func ProvideIngestionService(
	db *sql.DB,
	jobRepo repository.JobRepository,
	logger *zap.Logger,
	csvPath string,
) ingestion.Service {
	return ingestion.New(db, jobRepo, logger, csvPath)
}

func ProvideGin(
	config config.Config,
	logger *zap.Logger,
	jobRepo repository.JobRepository,
	ingestionSvc ingestion.Service,
	analyticsSvc analytics.Service,
) *gin.Engine {
	r := gin.New()

	ingHandler := handler.Ingestion{Service: ingestionSvc, Jobs: jobRepo, Log: logger}
	statusHandler := handler.Status{Jobs: jobRepo, Log: logger}
	analyticsHandler := handler.Analytics{Service: analyticsSvc, Log: logger}

	internal.RegisterRoutes(r, ingHandler, statusHandler, analyticsHandler)

	return r
}

func ProvideHTTP(
	engine *gin.Engine,
	config config.Config,
	logger *zap.Logger,
) *http.Server {
	address := fmt.Sprintf(":%d", config.App.Port)
	logger.Info("Configuring HTTP server", zap.String("address", address))

	return &http.Server{
		Addr:    address,
		Handler: engine,
	}
}

func ProvideCsvPath(
	config config.Config,
) string {
	return config.CSV.Path
}
