package repository

import (
	"context"
	"database/sql"
	"fmt"

	"sales-analytics/internal/models"
)

type analyticsRepository struct {
	db *sql.DB
}

func NewAnalyticsRepo(db *sql.DB) AnalyticsRepo {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) GetTotalRevenue(
	ctx context.Context,
	start, end string,
) (float64, error) {
	query := `
		select sum(oi.quantity * (oi.unit_price * (1-oi.discount)) + oi.shipping_cost)
		from order_items oi 
		join orders o on o.id = oi.order_id
		where o.order_date between ? and ?`

	var rev sql.NullFloat64
	err := r.db.QueryRowContext(ctx, query, start, end).Scan(&rev)
	if err != nil {
		return 0, fmt.Errorf("failed to get total revenue: %w", err)
	}

	if rev.Valid {
		return rev.Float64, nil
	}
	return 0, nil
}

func (r *analyticsRepository) GetRevenueByProduct(
	ctx context.Context,
	start, end string,
) ([]models.ProductRevenue, error) {
	query := `
		select p.id, p.name, sum(oi.quantity * (oi.unit_price * (1-oi.discount)) + oi.shipping_cost) as revenue
		from order_items oi 
		join orders o on o.id = oi.order_id
		join products p on p.id = oi.product_id
		where o.order_date between ? and ?
		group by p.id, p.name
		order by revenue desc`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue by product: %w", err)
	}
	defer rows.Close()

	var result []models.ProductRevenue
	for rows.Next() {
		var rev models.ProductRevenue
		if err := rows.Scan(&rev.ProductID, &rev.ProductName, &rev.Revenue); err != nil {
			return nil, fmt.Errorf("failed to scan product revenue row: %w", err)
		}
		result = append(result, rev)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product revenue rows: %w", err)
	}

	return result, nil
}

func (r *analyticsRepository) GetRevenueByCategory(
	ctx context.Context,
	start, end string,
) ([]models.CategoryRevenue, error) {
	query := `
		select p.category, sum(oi.quantity * (oi.unit_price * (1-oi.discount)) + oi.shipping_cost) as revenue
		from order_items oi 
		join orders o on o.id = oi.order_id
		join products p on p.id = oi.product_id
		where o.order_date between ? and ?
		group by p.category
		order by revenue desc`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue by category: %w", err)
	}
	defer rows.Close()

	var result []models.CategoryRevenue
	for rows.Next() {
		var rev models.CategoryRevenue
		if err := rows.Scan(&rev.Category, &rev.Revenue); err != nil {
			return nil, fmt.Errorf("failed to scan category revenue row: %w", err)
		}
		result = append(result, rev)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category revenue rows: %w", err)
	}

	return result, nil
}

func (r *analyticsRepository) GetRevenueByRegion(
	ctx context.Context,
	start, end string,
) ([]models.RegionRevenue, error) {
	query := `
		select c.region, sum(oi.quantity * (oi.unit_price * (1-oi.discount)) + oi.shipping_cost) as revenue
		from order_items oi 
		join orders o on o.id = oi.order_id
		join customers c on c.id = o.customer_id
		where o.order_date between ? and ?
		group by c.region
		order by revenue desc`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue by region: %w", err)
	}
	defer rows.Close()

	var result []models.RegionRevenue
	for rows.Next() {
		var rev models.RegionRevenue
		if err := rows.Scan(&rev.Region, &rev.Revenue); err != nil {
			return nil, fmt.Errorf("failed to scan region revenue row: %w", err)
		}
		result = append(result, rev)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating region revenue rows: %w", err)
	}

	return result, nil
}

func (r *analyticsRepository) GetTopProducts(
	ctx context.Context,
	start, end string,
	limit int,
) ([]models.TopProduct, error) {
	query := `
		select p.id, p.name, sum(oi.quantity) as qty_sold,
		       sum(oi.quantity * (oi.unit_price * (1-oi.discount)) + oi.shipping_cost) as revenue
		from order_items oi 
		join orders o on o.id = oi.order_id
		join products p on p.id = oi.product_id
		where o.order_date between ? and ?
		group by p.id, p.name
		order by qty_sold desc
		limit ?`

	rows, err := r.db.QueryContext(ctx, query, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top products: %w", err)
	}
	defer rows.Close()

	var result []models.TopProduct
	for rows.Next() {
		var product models.TopProduct
		if err := rows.Scan(&product.ProductID, &product.ProductName, &product.QuantitySold, &product.Revenue); err != nil {
			return nil, fmt.Errorf("failed to scan top product row: %w", err)
		}
		result = append(result, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating top product rows: %w", err)
	}

	return result, nil
}

func (r *analyticsRepository) GetCustomerCount(
	ctx context.Context,
	start, end string,
) (int, error) {
	query := `
		select count(distinct o.customer_id)
		from orders o
		where o.order_date between ? and ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, start, end).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get customer count: %w", err)
	}

	return count, nil
}

func (r *analyticsRepository) GetOrderCount(
	ctx context.Context,
	start, end string,
) (int, error) {
	query := `
		select count(*)
		from orders o
		where o.order_date between ? and ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, start, end).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get order count: %w", err)
	}

	return count, nil
}

func (r *analyticsRepository) GetAverageOrderValue(
	ctx context.Context,
	start, end string,
) (float64, error) {
	query := `
		select avg(total_amount)
		from orders o
		where o.order_date between ? and ?`

	var avg sql.NullFloat64
	err := r.db.QueryRowContext(ctx, query, start, end).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("failed to get average order value: %w", err)
	}

	if avg.Valid {
		return avg.Float64, nil
	}
	return 0, nil
}
