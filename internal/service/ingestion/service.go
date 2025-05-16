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

// reasonable defaults that work well in most environments
const (
	defaultBatchSize  = 500   // default rows per batch
	defaultBufferSize = 10000 // channel buffer size
	defaultWorkers    = 0     // 0 means use NumCPU()
	maxDBConnections  = 20    // prevent DB connection overload
	minBatchSize      = 100   // minimum batch size for very small datasets
	maxBatchSize      = 1000  // maximum batch size for very large datasets
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
	file, err := os.Open(s.csvPath)
	if err != nil {
		s.log.Error("failed to open csv file", zap.Error(err))
		s.jobRepo.SetFailed(ctx, jobID, fmt.Sprintf("failed to open csv file: %s", err))
		return err
	}
	defer file.Close()

	s.process(ctx, file, jobID, mode)
	return nil
}

// ImportFile imports a CSV from a reader (typically a multipart file upload)
func (s *service) ImportFile(
	ctx context.Context,
	r io.Reader,
	jobID, mode string,
) error {
	s.process(ctx, r, jobID, mode)
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

	// step 1: clear existing data if in overwrite mode
	if mode == "overwrite" {
		if err := s.truncateTables(ctx, jobID); err != nil {
			return // error already logged
		}
	}

	// step 2: initialize counters for tracking progress
	var stats struct {
		rows      int64
		customers int64
		products  int64
		orders    int64
		items     int64
	}

	// step 3: choose optimal worker count based on cpu cores
	workerCount := s.workers
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
		// limit to a reasonable number to prevent db connection overload
		if workerCount > maxDBConnections/2 {
			workerCount = maxDBConnections / 2
		}
	}

	s.log.Info("starting csv ingestion",
		zap.String("job_id", jobID),
		zap.Int("batch_size", s.batchSize),
		zap.Int("workers", workerCount))

	// step 4: create worker pool and channels
	rawRows := make(chan []string, s.bufferSize)
	done := make(chan struct{})

	// step 5: start worker pool
	var wg sync.WaitGroup
	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go func(workerID int) {
			defer wg.Done()
			s.worker(ctx, jobID, rawRows, &stats, workerID)
		}(i + 1)
	}

	// step 6: start csv reader in a goroutine
	go func() {
		defer close(rawRows) // signal workers when done
		rowCount := s.readCSV(ctx, r, rawRows, jobID)
		atomic.StoreInt64(&stats.rows, int64(rowCount))
	}()

	// step 7: wait for completion in a goroutine
	go func() {
		wg.Wait()
		close(done)
	}()

	// step 8: periodically update job status while processing
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentRows := atomic.LoadInt64(&stats.rows)
			s.jobRepo.Bump(ctx, jobID, int(currentRows))
			s.log.Info("ingestion progress",
				zap.String("job_id", jobID),
				zap.Int64("rows", currentRows),
				zap.Int64("customers", atomic.LoadInt64(&stats.customers)),
				zap.Int64("products", atomic.LoadInt64(&stats.products)),
				zap.Int64("orders", atomic.LoadInt64(&stats.orders)),
				zap.Int64("items", atomic.LoadInt64(&stats.items)))
		case <-done:
			// processing complete
			goto finish
		}
	}

finish:
	// step 9: mark job as completed
	s.jobRepo.SetCompleted(ctx, jobID, int(atomic.LoadInt64(&stats.rows)))

	// step 10: log final stats
	duration := time.Since(start)
	s.log.Info(constants.LogIngestDone,
		zap.String("job_id", jobID),
		zap.Int64("rows", stats.rows),
		zap.Int64("customers", stats.customers),
		zap.Int64("products", stats.products),
		zap.Int64("orders", stats.orders),
		zap.Int64("items", stats.items),
		zap.Duration("duration", duration),
		zap.Float64("rows_per_sec", float64(stats.rows)/duration.Seconds()))
}

// truncateTables clears all tables if in overwrite mode
func (s *service) truncateTables(
	ctx context.Context,
	jobID string,
) error {
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
