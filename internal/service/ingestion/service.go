package ingestion

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"sales-analytics/internal/constants"
	"sales-analytics/internal/models"
	"sales-analytics/internal/repository"

	"go.uber.org/zap"
)

// TODO: need to move this to config.yaml
const (
	defaultBatchSize  = 2000  // increased from 500 to 2000
	defaultBufferSize = 50000 // increased channel buffer for better throughput
	defaultWorkers    = 0     // 0 means use NumCPU()
	maxDBConnections  = 30    // increased from 20 to 30
	minBatchSize      = 100   // minimum batch size for very small datasets
	maxBatchSize      = 5000  // increased max batch size for better performance
)

type service struct {
	db      *sql.DB
	jobRepo repository.JobRepository
	log     *zap.Logger
	csvPath string

	// processing options
	batchSize  int
	bufferSize int
	workers    int
}

func New(
	db *sql.DB,
	jobRepo repository.JobRepository,
	log *zap.Logger,
	csvPath string,
) Service {
	db.SetMaxOpenConns(maxDBConnections)
	db.SetMaxIdleConns(maxDBConnections / 2)
	db.SetConnMaxLifetime(time.Minute * 5)

	return &service{
		db:         db,
		jobRepo:    jobRepo,
		log:        log,
		csvPath:    csvPath,
		batchSize:  defaultBatchSize,
		bufferSize: defaultBufferSize,
		workers:    defaultWorkers,
	}
}

// ImportFromPath imports a CSV file from the configured path
func (s *service) ImportFromPath(
	ctx context.Context,
	jobID, mode string,
) error {
	startTime := time.Now()
	s.log.Info("starting import from path",
		zap.String("path", s.csvPath),
		zap.String("job_id", jobID),
		zap.String("mode", mode))

	file, err := os.Open(s.csvPath)
	if err != nil {
		s.log.Error("failed to open csv file", zap.Error(err))
		s.jobRepo.SetFailed(ctx, jobID, fmt.Sprintf("failed to open csv file: %s", err))
		return err
	}
	defer file.Close()

	s.process(ctx, file, jobID, mode)

	s.log.Info("import completed",
		zap.String("job_id", jobID),
		zap.Duration("total_duration", time.Since(startTime)))
	return nil
}

// ImportFile imports a CSV from a reader (typically a multipart file upload)
func (s *service) ImportFile(
	ctx context.Context,
	r io.Reader,
	jobID, mode string,
) error {
	startTime := time.Now()
	s.log.Info("starting import from file upload",
		zap.String("job_id", jobID),
		zap.String("mode", mode))

	s.process(ctx, r, jobID, mode)

	s.log.Info("import completed",
		zap.String("job_id", jobID),
		zap.Duration("total_duration", time.Since(startTime)))
	return nil
}

// process handles the ingestion workflow
func (s *service) process(
	ctx context.Context,
	r io.Reader,
	jobID, mode string,
) {
	start := time.Now()
	s.log.Info(constants.LogIngestStart, zap.String("job_id", jobID), zap.String("mode", mode))

	if err := s.optimizeDBForBulkLoad(ctx, jobID); err != nil {
		return
	}
	defer s.restoreDBSettings(ctx, jobID)

	if mode == "overwrite" {
		if err := s.truncateTables(ctx, jobID); err != nil {
			return
		}
	}

	var stats struct {
		rows      int64
		customers int64
		products  int64
		orders    int64
		items     int64
		parseTime int64
		dbTime    int64
	}

	// Choose optimal worker count based on cpu cores
	workerCount := s.workers
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()

		// If multicore system, leave 1-2 cores for OS
		if workerCount > 4 {
			workerCount -= 1
		}

		maxWorkers := maxDBConnections / 3
		if workerCount > maxWorkers {
			workerCount = maxWorkers
		}
	}

	s.log.Info("starting csv ingestion with optimized settings",
		zap.String("job_id", jobID),
		zap.Int("batch_size", s.batchSize),
		zap.Int("buffer_size", s.bufferSize),
		zap.Int("workers", workerCount),
		zap.Int("max_db_connections", maxDBConnections))

	rawRows := make(chan []string, s.bufferSize)
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go func(workerID int) {
			defer wg.Done()
			s.worker(ctx, jobID, rawRows, &stats, workerID)
		}(i + 1)
	}

	// Start csv reader in a goroutine
	go func() {
		defer close(rawRows) // signal workers when done
		rowCount := s.readCSV(ctx, r, rawRows, jobID)
		atomic.StoreInt64(&stats.rows, int64(rowCount))
	}()

	go func() {
		wg.Wait()
		close(done)
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastUpdate := start
	lastRows := int64(0)

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			elapsed := now.Sub(lastUpdate)
			currentRows := atomic.LoadInt64(&stats.rows)

			rowDelta := currentRows - lastRows
			rate := float64(rowDelta) / elapsed.Seconds()

			s.jobRepo.Bump(ctx, jobID, int(currentRows))
			s.log.Info("ingestion progress",
				zap.String("job_id", jobID),
				zap.Int64("rows", currentRows),
				zap.Int64("customers", atomic.LoadInt64(&stats.customers)),
				zap.Int64("products", atomic.LoadInt64(&stats.products)),
				zap.Int64("orders", atomic.LoadInt64(&stats.orders)),
				zap.Int64("items", atomic.LoadInt64(&stats.items)),
				zap.Float64("rows_per_second", rate),
				zap.Duration("elapsed", time.Since(start)))

			lastUpdate = now
			lastRows = currentRows

		case <-done:
			// processing complete
			goto finish
		}
	}

finish:
	s.jobRepo.SetCompleted(ctx, jobID, int(atomic.LoadInt64(&stats.rows)))

	duration := time.Since(start)
	s.log.Info(constants.LogIngestDone,
		zap.String("job_id", jobID),
		zap.Int64("rows", stats.rows),
		zap.Int64("customers", stats.customers),
		zap.Int64("products", stats.products),
		zap.Int64("orders", stats.orders),
		zap.Int64("items", stats.items),
		zap.Duration("duration", duration),
		zap.Float64("rows_per_sec", float64(stats.rows)/duration.Seconds()),
		zap.Duration("parsing_time", time.Duration(atomic.LoadInt64(&stats.parseTime))),
		zap.Duration("db_time", time.Duration(atomic.LoadInt64(&stats.dbTime))),
		zap.Float64("db_time_percent", float64(atomic.LoadInt64(&stats.dbTime))/float64(duration.Nanoseconds())*100))
}

// optimizeDBForBulkLoad disables checks to improve bulk load performance
func (s *service) optimizeDBForBulkLoad(ctx context.Context, jobID string) error {
	optimizations := []string{
		"set unique_checks=0",
		"set foreign_key_checks=0",
		"set session transaction_isolation='READ-UNCOMMITTED'",
	}

	for _, stmt := range optimizations {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			s.log.Error("failed to set DB optimization",
				zap.String("job_id", jobID),
				zap.String("statement", stmt),
				zap.Error(err))
			s.jobRepo.SetFailed(ctx, jobID, fmt.Sprintf("failed to optimize DB: %s", err))
			return err
		}
	}

	s.log.Info("database optimized for bulk loading", zap.String("job_id", jobID))
	return nil
}

// restoreDBSettings restores default DB settings after bulk load
func (s *service) restoreDBSettings(ctx context.Context, jobID string) {
	restorations := []string{
		"set unique_checks=1",
		"set foreign_key_checks=1",
		"set session transaction_isolation='REPEATABLE-READ'",
	}

	for _, stmt := range restorations {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			s.log.Warn("failed to restore DB setting",
				zap.String("job_id", jobID),
				zap.String("statement", stmt),
				zap.Error(err))
		}
	}

	s.log.Info("database settings restored", zap.String("job_id", jobID))
}

// truncateTables clears all tables if in overwrite mode
func (s *service) truncateTables(
	ctx context.Context,
	jobID string,
) error {
	s.log.Info("truncating tables for overwrite mode", zap.String("job_id", jobID))

	// clear in reverse dependency order
	tables := []string{"order_items", "orders", "products", "customers"}
	for _, t := range tables {
		if _, err := s.db.ExecContext(ctx, "truncate table "+t); err != nil {
			s.log.Error("failed to truncate table",
				zap.String("table", t),
				zap.Error(err))
			s.jobRepo.SetFailed(ctx, jobID, fmt.Sprintf("failed to truncate table %s: %s", t, err))
			return err
		}
	}
	return nil
}

// GetJobStatus returns the current status of a job
func (s *service) GetJobStatus(
	ctx context.Context,
	jobID string,
) (models.IngestionJob, error) {
	return s.jobRepo.Get(ctx, jobID)
}
