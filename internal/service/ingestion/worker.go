package ingestion

import (
	"context"
	"sync/atomic"
	"time"

	"sales-analytics/internal/models"
	"sales-analytics/internal/repository"

	"go.uber.org/zap"
)

// worker processes data from the csv reader
func (s *service) worker(
	ctx context.Context,
	jobID string,
	rows <-chan []string,
	stats *struct {
		rows      int64
		customers int64
		products  int64
		orders    int64
		items     int64
	},
	workerID int,
) {
	// track stats for this worker
	startTime := time.Now()
	var processed, failed int

	// maintain maps for deduplication
	seenCustomers := make(map[string]bool)
	seenProducts := make(map[string]bool)
	seenOrders := make(map[string]bool)

	// batch accumulation
	var customerBatch []models.Customer
	var productBatch []models.Product
	var orderBatch []Sale

	// helper to flush batches when they reach the threshold
	flushBatches := func() {
		if len(customerBatch) > 0 {
			count := s.insertCustomerBatch(ctx, customerBatch, jobID, workerID)
			atomic.AddInt64(&stats.customers, int64(count))
			customerBatch = customerBatch[:0]
		}

		if len(productBatch) > 0 {
			count := s.insertProductBatch(ctx, productBatch, jobID, workerID)
			atomic.AddInt64(&stats.products, int64(count))
			productBatch = productBatch[:0]
		}

		if len(orderBatch) > 0 {
			orders, items := s.insertOrderBatch(ctx, orderBatch, jobID, workerID)
			atomic.AddInt64(&stats.orders, int64(orders))
			atomic.AddInt64(&stats.items, int64(items))
			orderBatch = orderBatch[:0]
		}
	}

	// process rows received from the channel
	for record := range rows {
		sale, err := parseRow(record)
		if err != nil {
			s.log.Warn("failed to parse row",
				zap.String("job_id", jobID),
				zap.Error(err),
				zap.Strings("record", record))
			failed++
			continue
		}

		// add to batches, handle deduplication
		if !seenCustomers[sale.CustomerID] {
			seenCustomers[sale.CustomerID] = true
			customerBatch = append(customerBatch, sale.ToCustomer())

			// flush customer batch if it reaches batch size
			if len(customerBatch) >= s.batchSize {
				count := s.insertCustomerBatch(ctx, customerBatch, jobID, workerID)
				atomic.AddInt64(&stats.customers, int64(count))
				customerBatch = customerBatch[:0]
			}
		}

		if !seenProducts[sale.ProductID] {
			seenProducts[sale.ProductID] = true
			productBatch = append(productBatch, sale.ToProduct())

			// flush product batch if it reaches batch size
			if len(productBatch) >= s.batchSize {
				count := s.insertProductBatch(ctx, productBatch, jobID, workerID)
				atomic.AddInt64(&stats.products, int64(count))
				productBatch = productBatch[:0]
			}
		}

		// add to order batch (even if we've seen this order before,
		// as order items might be different)
		orderBatch = append(orderBatch, sale)
		if !seenOrders[sale.OrderID] {
			seenOrders[sale.OrderID] = true
		}

		// flush order batch if it reaches batch size
		if len(orderBatch) >= s.batchSize {
			orders, items := s.insertOrderBatch(ctx, orderBatch, jobID, workerID)
			atomic.AddInt64(&stats.orders, int64(orders))
			atomic.AddInt64(&stats.items, int64(items))
			orderBatch = orderBatch[:0]
		}

		processed++
	}

	// flush any remaining batches at the end
	flushBatches()

	s.log.Info("worker finished",
		zap.String("job_id", jobID),
		zap.Int("worker_id", workerID),
		zap.Int("processed", processed),
		zap.Int("failed", failed),
		zap.Duration("duration", time.Since(startTime)))
}

// insertCustomerBatch inserts a batch of customers
func (s *service) insertCustomerBatch(
	ctx context.Context,
	customers []models.Customer,
	jobID string,
	workerID int,
) int {
	if len(customers) == 0 {
		return 0
	}

	// start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.log.Error("failed to begin transaction",
			zap.String("job_id", jobID),
			zap.Int("worker_id", workerID),
			zap.String("entity", "customer"),
			zap.Error(err))
		return 0
	}
	defer tx.Rollback()

	// insert customers
	repos := repository.NewIngestionRepo(tx)
	inserted := 0

	for _, c := range customers {
		if err := repos.Customers.Upsert(ctx, c); err != nil {
			s.log.Error("failed to insert customer",
				zap.String("job_id", jobID),
				zap.Int("worker_id", workerID),
				zap.String("id", c.ID),
				zap.Error(err))
			continue
		}
		inserted++
	}

	// commit if we inserted anything
	if inserted > 0 {
		if err := tx.Commit(); err != nil {
			s.log.Error("failed to commit transaction",
				zap.String("job_id", jobID),
				zap.Int("worker_id", workerID),
				zap.String("entity", "customer"),
				zap.Error(err))
			return 0
		}
	}

	return inserted
}

// insertProductBatch inserts a batch of products
func (s *service) insertProductBatch(
	ctx context.Context,
	products []models.Product,
	jobID string,
	workerID int,
) int {
	if len(products) == 0 {
		return 0
	}

	// start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.log.Error("failed to begin transaction",
			zap.String("job_id", jobID),
			zap.Int("worker_id", workerID),
			zap.String("entity", "product"),
			zap.Error(err))
		return 0
	}
	defer tx.Rollback()

	// insert products
	repos := repository.NewIngestionRepo(tx)
	inserted := 0

	for _, p := range products {
		if err := repos.Products.Upsert(ctx, p); err != nil {
			s.log.Error("failed to insert product",
				zap.String("job_id", jobID),
				zap.Int("worker_id", workerID),
				zap.String("id", p.ID),
				zap.Error(err))
			continue
		}
		inserted++
	}

	// commit if we inserted anything
	if inserted > 0 {
		if err := tx.Commit(); err != nil {
			s.log.Error("failed to commit transaction",
				zap.String("job_id", jobID),
				zap.Int("worker_id", workerID),
				zap.String("entity", "product"),
				zap.Error(err))
			return 0
		}
	}

	return inserted
}

// insertOrderBatch inserts a batch of orders and their items
func (s *service) insertOrderBatch(
	ctx context.Context,
	sales []Sale,
	jobID string,
	workerID int,
) (int, int) {
	if len(sales) == 0 {
		return 0, 0
	}

	// start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.log.Error("failed to begin transaction",
			zap.String("job_id", jobID),
			zap.Int("worker_id", workerID),
			zap.String("entity", "order"),
			zap.Error(err))
		return 0, 0
	}
	defer tx.Rollback()

	// insert orders and items
	repos := repository.NewIngestionRepo(tx)
	orderCount := 0
	itemCount := 0

	// track processed orders for this batch to avoid duplicates
	processedOrders := make(map[string]bool)

	// process each sale
	for _, sale := range sales {
		// insert order if not already done in this batch
		if !processedOrders[sale.OrderID] {
			if err := repos.Orders.Upsert(
				ctx,
				sale.OrderID,
				sale.CustomerID,
				sale.OrderDate,
				sale.OrderTotal,
			); err != nil {
				s.log.Error("failed to insert order",
					zap.String("job_id", jobID),
					zap.Int("worker_id", workerID),
					zap.String("id", sale.OrderID),
					zap.Error(err))
				continue
			}
			orderCount++
			processedOrders[sale.OrderID] = true
		}

		// insert order item
		if err := repos.Items.Upsert(
			ctx,
			sale.OrderID,
			sale.ProductID,
			sale.Quantity,
			sale.Price,
			sale.Discount,
			sale.Shipping,
		); err != nil {
			s.log.Error("failed to insert order item",
				zap.String("job_id", jobID),
				zap.Int("worker_id", workerID),
				zap.String("order_id", sale.OrderID),
				zap.String("product_id", sale.ProductID),
				zap.Error(err))
			continue
		}
		itemCount++
	}

	// commit if we inserted anything
	if orderCount > 0 || itemCount > 0 {
		if err := tx.Commit(); err != nil {
			s.log.Error("failed to commit transaction",
				zap.String("job_id", jobID),
				zap.Int("worker_id", workerID),
				zap.String("entity", "order"),
				zap.Error(err))
			return 0, 0
		}
	}

	return orderCount, itemCount
}
