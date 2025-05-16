package repository

import (
	"context"
	"database/sql"
	"time"

	"sales-analytics/internal/models"
)

type Database interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type CustomerRepo interface {
	Upsert(ctx context.Context, customer models.Customer) error
}

type ProductRepo interface {
	Upsert(ctx context.Context, product models.Product) error
}

type OrderRepo interface {
	Upsert(ctx context.Context, id, custID string, date time.Time, total float64) error
}

type ItemRepo interface {
	Upsert(ctx context.Context, orderID, prodID string, qty int, price, disc, ship float64) error
}

type JobRepository interface {
	Insert(ctx context.Context, id string)
	SetFailed(ctx context.Context, id, msg string)
	SetCompleted(ctx context.Context, id string, rows int)
	Bump(ctx context.Context, id string, rows int)
	Get(ctx context.Context, id string) (models.IngestionJob, error)
}

type AnalyticsRepo interface {
	GetTotalRevenue(ctx context.Context, start, end string) (float64, error)
	GetRevenueByProduct(ctx context.Context, start, end string) ([]models.ProductRevenue, error)
	GetRevenueByCategory(ctx context.Context, start, end string) ([]models.CategoryRevenue, error)
	GetRevenueByRegion(ctx context.Context, start, end string) ([]models.RegionRevenue, error)
	GetTopProducts(ctx context.Context, start, end string, limit int) ([]models.TopProduct, error)
	GetCustomerCount(ctx context.Context, start, end string) (int, error)
	GetOrderCount(ctx context.Context, start, end string) (int, error)
	GetAverageOrderValue(ctx context.Context, start, end string) (float64, error)
}

type Store interface {
	Exec(query string, args ...any) error
	Query(dest any, query string, args ...any) error
}
