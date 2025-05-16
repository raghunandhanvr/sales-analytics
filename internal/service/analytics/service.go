package analytics

import (
	"context"
	"database/sql"
	"fmt"

	"sales-analytics/internal/models"
	"sales-analytics/internal/repository"

	"go.uber.org/zap"
)

type service struct {
	repo repository.AnalyticsRepo
	log  *zap.Logger
}

func New(db *sql.DB, log *zap.Logger) Service {
	repo := repository.NewAnalyticsRepo(db)
	return &service{
		repo: repo,
		log:  log,
	}
}

func (s *service) Total(ctx context.Context, start, end string) (float64, error) {
	revenue, err := s.repo.GetTotalRevenue(ctx, start, end)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate total revenue: %w", err)
	}

	s.log.Debug("Total revenue calculated",
		zap.String("start_date", start),
		zap.String("end_date", end),
		zap.Float64("revenue", revenue))

	return revenue, nil
}

func (s *service) ByProduct(ctx context.Context, start, end string) ([]models.ProductRevenue, error) {
	products, err := s.repo.GetRevenueByProduct(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate revenue by product: %w", err)
	}

	s.log.Debug("Revenue by product calculated",
		zap.String("start_date", start),
		zap.String("end_date", end),
		zap.Int("product_count", len(products)))

	return products, nil
}

func (s *service) ByCategory(ctx context.Context, start, end string) ([]models.CategoryRevenue, error) {
	categories, err := s.repo.GetRevenueByCategory(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate revenue by category: %w", err)
	}

	s.log.Debug("Revenue by category calculated",
		zap.String("start_date", start),
		zap.String("end_date", end),
		zap.Int("category_count", len(categories)))

	return categories, nil
}

func (s *service) ByRegion(ctx context.Context, start, end string) ([]models.RegionRevenue, error) {
	regions, err := s.repo.GetRevenueByRegion(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate revenue by region: %w", err)
	}

	s.log.Debug("Revenue by region calculated",
		zap.String("start_date", start),
		zap.String("end_date", end),
		zap.Int("region_count", len(regions)))

	return regions, nil
}

func (s *service) TopProducts(ctx context.Context, start, end string, limit int) ([]models.TopProduct, error) {
	products, err := s.repo.GetTopProducts(ctx, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate top products: %w", err)
	}

	s.log.Debug("Top products calculated",
		zap.String("start_date", start),
		zap.String("end_date", end),
		zap.Int("limit", limit),
		zap.Int("product_count", len(products)))

	return products, nil
}

func (s *service) CustomerCount(ctx context.Context, start, end string) (int, error) {
	count, err := s.repo.GetCustomerCount(ctx, start, end)
	if err != nil {
		return 0, fmt.Errorf("failed to count customers: %w", err)
	}

	s.log.Debug("Customer count calculated",
		zap.String("start_date", start),
		zap.String("end_date", end),
		zap.Int("count", count))

	return count, nil
}

func (s *service) OrderCount(ctx context.Context, start, end string) (int, error) {
	count, err := s.repo.GetOrderCount(ctx, start, end)
	if err != nil {
		return 0, fmt.Errorf("failed to count orders: %w", err)
	}

	s.log.Debug("Order count calculated",
		zap.String("start_date", start),
		zap.String("end_date", end),
		zap.Int("count", count))

	return count, nil
}

func (s *service) AverageOrderValue(ctx context.Context, start, end string) (float64, error) {
	avg, err := s.repo.GetAverageOrderValue(ctx, start, end)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate average order value: %w", err)
	}

	s.log.Debug("Average order value calculated",
		zap.String("start_date", start),
		zap.String("end_date", end),
		zap.Float64("average", avg))

	return avg, nil
}
