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
		parseTime int64
		dbTime    int64
	},
	workerID int,
) {
	// track stats for this worker
	startTime := time.Now()
	var processed, failed int
	var parseTime, dbTime time.Duration

	// maintain maps for deduplication - preallocate with capacity
	seenCustomers := make(map[string]bool, s.batchSize*2)
	seenProducts := make(map[string]bool, s.batchSize*2)
	seenOrders := make(map[string]bool, s.batchSize*2)

	// batch accumulation - preallocate with capacity to reduce allocations
	customerBatch := make([]models.Customer, 0, s.batchSize)
	productBatch := make([]models.Product, 0, s.batchSize)
	orderBatch := make([]Sale, 0, s.batchSize)

	// helper to flush batches when they reach the threshold
	flushBatches := func() {
		dbStart := time.Now()

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

		dbDuration := time.Since(dbStart)
		dbTime += dbDuration
		atomic.AddInt64(&stats.dbTime, dbDuration.Nanoseconds())
	}

	// process rows received from the channel
	for record := range rows {
		parseStart := time.Now()
		sale, err := parseRow(record)
		parseTime += time.Since(parseStart)

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

		if processed > 0 && processed%50000 == 0 {
			s.log.Debug("worker progress",
				zap.String("job_id", jobID),
				zap.Int("worker_id", workerID),
				zap.Int("processed_rows", processed),
				zap.Int("failed_rows", failed),
				zap.Duration("elapsed", time.Since(startTime)),
				zap.Duration("parse_time", parseTime),
				zap.Duration("db_time", dbTime),
				zap.Float64("rows_per_sec", float64(processed)/time.Since(startTime).Seconds()))
		}
	}

	// flush any remaining batches at the end
	flushBatches()

	atomic.AddInt64(&stats.parseTime, parseTime.Nanoseconds())

	s.log.Info("worker finished",
		zap.String("job_id", jobID),
		zap.Int("worker_id", workerID),
		zap.Int("processed", processed),
		zap.Int("failed", failed),
		zap.Duration("duration", time.Since(startTime)),
		zap.Duration("parse_time", parseTime),
		zap.Duration("db_time", dbTime),
		zap.Float64("parse_time_pct", float64(parseTime)/float64(time.Since(startTime))*100),
		zap.Float64("db_time_pct", float64(dbTime)/float64(time.Since(startTime))*100))
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

	repos := repository.NewIngestionRepo(tx)
	inserted, err := repos.Customers.BulkUpsert(ctx, customers)
	if err != nil {
		s.log.Error("failed to bulk insert customers",
			zap.String("job_id", jobID),
			zap.Int("worker_id", workerID),
			zap.Int("batch_size", len(customers)),
			zap.Error(err))
		return 0
	}

	if err := tx.Commit(); err != nil {
		s.log.Error("failed to commit customer transaction",
			zap.String("job_id", jobID),
			zap.Int("worker_id", workerID),
			zap.Error(err))
		return 0
	}

	s.log.Debug("bulk inserted customers",
		zap.String("job_id", jobID),
		zap.Int("worker_id", workerID),
		zap.Int("batch_size", len(customers)),
		zap.Int("inserted", inserted))

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

	repos := repository.NewIngestionRepo(tx)
	inserted, err := repos.Products.BulkUpsert(ctx, products)
	if err != nil {
		s.log.Error("failed to bulk insert products",
			zap.String("job_id", jobID),
			zap.Int("worker_id", workerID),
			zap.Int("batch_size", len(products)),
			zap.Error(err))
		return 0
	}

	if err := tx.Commit(); err != nil {
		s.log.Error("failed to commit product transaction",
			zap.String("job_id", jobID),
			zap.Int("worker_id", workerID),
			zap.Error(err))
		return 0
	}

	s.log.Debug("bulk inserted products",
		zap.String("job_id", jobID),
		zap.Int("worker_id", workerID),
		zap.Int("batch_size", len(products)),
		zap.Int("inserted", inserted))

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

	repos := repository.NewIngestionRepo(tx)

	uniqueOrders := make(map[string]bool)
	var orderParams []models.Order
	var itemParams []models.OrderItem

	for _, sale := range sales {
		if !uniqueOrders[sale.OrderID] {
			uniqueOrders[sale.OrderID] = true

			orderParams = append(orderParams, models.Order{
				ID:          sale.OrderID,
				CustomerID:  sale.CustomerID,
				OrderDate:   sale.OrderDate,
				TotalAmount: sale.OrderTotal,
			})
		}

		itemParams = append(itemParams, models.OrderItem{
			OrderID:      sale.OrderID,
			ProductID:    sale.ProductID,
			Quantity:     sale.Quantity,
			UnitPrice:    sale.Price,
			Discount:     sale.Discount,
			ShippingCost: sale.Shipping,
		})
	}

	orderCount := 0
	if len(orderParams) > 0 {
		inserted, err := repos.Orders.BulkUpsert(ctx, orderParams)
		if err != nil {
			s.log.Error("failed to bulk insert orders",
				zap.String("job_id", jobID),
				zap.Int("worker_id", workerID),
				zap.Int("count", len(orderParams)),
				zap.Error(err))
		} else {
			orderCount = inserted
		}
	}

	itemCount := 0
	if len(itemParams) > 0 {
		inserted, err := repos.Items.BulkUpsert(ctx, itemParams)
		if err != nil {
			s.log.Error("failed to bulk insert order items",
				zap.String("job_id", jobID),
				zap.Int("worker_id", workerID),
				zap.Int("count", len(itemParams)),
				zap.Error(err))
		} else {
			itemCount = inserted
		}
	}

	if orderCount > 0 || itemCount > 0 {
		if err := tx.Commit(); err != nil {
			s.log.Error("failed to commit order transaction",
				zap.String("job_id", jobID),
				zap.Int("worker_id", workerID),
				zap.Error(err))
			return 0, 0
		}

		s.log.Debug("bulk inserted orders and items",
			zap.String("job_id", jobID),
			zap.Int("worker_id", workerID),
			zap.Int("orders", orderCount),
			zap.Int("items", itemCount))
	}

	return orderCount, itemCount
}
