package analytics

import (
	"context"
	"sales-analytics/internal/models"
)

type Service interface {
	Total(ctx context.Context, start, end string) (float64, error)
	ByProduct(ctx context.Context, start, end string) ([]models.ProductRevenue, error)
	ByCategory(ctx context.Context, start, end string) ([]models.CategoryRevenue, error)
	ByRegion(ctx context.Context, start, end string) ([]models.RegionRevenue, error)
	TopProducts(ctx context.Context, start, end string, limit int) ([]models.TopProduct, error)
	CustomerCount(ctx context.Context, start, end string) (int, error)
	OrderCount(ctx context.Context, start, end string) (int, error)
	AverageOrderValue(ctx context.Context, start, end string) (float64, error)
}
