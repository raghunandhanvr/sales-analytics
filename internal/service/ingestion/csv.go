package ingestion

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// csvReader constants
const (
	readerBuf = 8 << 20 // 8MB buffer for CSV reading for better performance
)

// readCSV reads CSV data and sends rows to the worker pool
func (s *service) readCSV(ctx context.Context, r io.Reader, rows chan<- []string, jobID string) int {
	// use buffered reader for better performance
	bufReader := bufio.NewReaderSize(r, readerBuf)
	csvReader := csv.NewReader(bufReader)
	csvReader.ReuseRecord = true      // reuse memory for better performance
	csvReader.LazyQuotes = true       // more tolerant parsing
	csvReader.TrimLeadingSpace = true // clean data as we read

	// track performance metrics
	startTime := time.Now()
	lastLogTime := startTime
	rowCount := 0
	parseErrors := 0
	lastBatchTime := startTime
	batchSize := 10000

	// Read and discard header row
	_, err := csvReader.Read()
	if err != nil {
		s.log.Error("failed to read csv header", zap.Error(err))
		s.jobRepo.SetFailed(ctx, jobID, fmt.Sprintf("failed to read csv header: %s", err))
		return 0
	}

	// read all rows and send to worker pool
	for {
		// check if context was canceled
		select {
		case <-ctx.Done():
			return rowCount
		default: // continue reading
		}

		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			parseErrors++
			s.log.Warn("error reading csv line",
				zap.String("job_id", jobID),
				zap.Error(err),
				zap.Int("line", rowCount+1))
			continue
		}

		rowCount++

		// create a copy of the record to avoid race conditions
		// when csv reader reuses the underlying slice
		recordCopy := make([]string, len(record))
		copy(recordCopy, record)

		// send to worker pool with backpressure
		select {
		case rows <- recordCopy:
			// row sent to channel
		case <-ctx.Done():
			return rowCount
		}

		// log progress periodically or after batch
		if rowCount%batchSize == 0 {
			now := time.Now()
			batchDuration := now.Sub(lastBatchTime)
			rowsPerSecond := float64(batchSize) / batchDuration.Seconds()

			s.log.Info("csv reading batch completed",
				zap.String("job_id", jobID),
				zap.Int("rows_read", rowCount),
				zap.Duration("batch_duration", batchDuration),
				zap.Float64("rows_per_second", rowsPerSecond),
				zap.Duration("total_elapsed", time.Since(startTime)))

			lastBatchTime = now
			lastLogTime = now

			if rowsPerSecond > 10000 && batchSize < 50000 {
				batchSize *= 2
				s.log.Debug("increasing progress log batch size",
					zap.String("job_id", jobID),
					zap.Int("new_batch_size", batchSize))
			}
		} else if time.Since(lastLogTime) > 5*time.Second {
			elapsedSinceLastLog := time.Since(lastLogTime)
			rowsSinceLastLog := rowCount % batchSize
			if rowsSinceLastLog == 0 {
				rowsSinceLastLog = batchSize
			}
			rowsPerSecond := float64(rowsSinceLastLog) / elapsedSinceLastLog.Seconds()

			s.log.Info("csv reading progress",
				zap.String("job_id", jobID),
				zap.Int("rows_read", rowCount),
				zap.Float64("rows_per_second", rowsPerSecond),
				zap.Duration("elapsed", time.Since(startTime)))

			lastLogTime = time.Now()
		}
	}

	duration := time.Since(startTime)
	rowsPerSecond := float64(rowCount) / duration.Seconds()

	s.log.Info("csv reading completed",
		zap.String("job_id", jobID),
		zap.Int("total_rows", rowCount),
		zap.Int("parse_errors", parseErrors),
		zap.Duration("duration", duration),
		zap.Float64("rows_per_second", rowsPerSecond),
		zap.Int("buffer_size", readerBuf))

	return rowCount
}

// parseRow converts a CSV row into structured data
func parseRow(rec []string) (Sale, error) {
	var s Sale

	if len(rec) < 15 {
		return s, fmt.Errorf("record has insufficient fields: got %d, need at least 15", len(rec))
	}

	// extract basic identifiers - use direct indexing for performance
	s.OrderID = rec[0]
	s.ProductID = rec[1]
	s.CustomerID = rec[2]

	s.ProductName = rec[3]
	s.ProductCategory = rec[4]
	s.Region = rec[5]
	s.CustomerName = rec[12]
	s.CustomerEmail = rec[13]
	s.CustomerAddress = rec[14]

	// parse numeric values
	qty, err := strconv.Atoi(rec[7])
	if err != nil {
		return s, fmt.Errorf("invalid quantity: %w", err)
	}
	s.Quantity = qty

	price, err := strconv.ParseFloat(rec[8], 64)
	if err != nil {
		return s, fmt.Errorf("invalid price: %w", err)
	}
	s.Price = price

	discount, err := strconv.ParseFloat(rec[9], 64)
	if err != nil {
		return s, fmt.Errorf("invalid discount: %w", err)
	}
	s.Discount = discount

	shipping, err := strconv.ParseFloat(rec[10], 64)
	if err != nil {
		return s, fmt.Errorf("invalid shipping: %w", err)
	}
	s.Shipping = shipping

	// parse date - this is typically the slowest operation
	date, err := time.Parse("2006-01-02", rec[6])
	if err != nil {
		return s, fmt.Errorf("invalid date: %w", err)
	}
	s.OrderDate = date

	s.OrderTotal = float64(s.Quantity)*s.Price*(1-s.Discount) + s.Shipping

	return s, nil
}
