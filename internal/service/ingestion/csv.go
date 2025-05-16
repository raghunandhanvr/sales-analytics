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
	readerBuf = 4 << 20 // 4MB buffer for CSV reading for better performance
)

// readCSV reads CSV data and sends rows to the worker pool
func (s *service) readCSV(ctx context.Context, r io.Reader, rows chan<- []string, jobID string) int {
	// use buffered reader for better performance
	bufReader := bufio.NewReaderSize(r, readerBuf)
	csvReader := csv.NewReader(bufReader)
	csvReader.ReuseRecord = true // reuse memory for better performance

	// track progress
	startTime := time.Now()
	lastLogTime := startTime
	rowCount := 0

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

		// log progress periodically
		if rowCount%10000 == 0 || time.Since(lastLogTime) > 5*time.Second {
			s.log.Info("csv reading progress",
				zap.String("job_id", jobID),
				zap.Int("rows_read", rowCount),
				zap.Duration("elapsed", time.Since(startTime)))
			lastLogTime = time.Now()
		}
	}

	s.log.Info("csv reading completed",
		zap.String("job_id", jobID),
		zap.Int("total_rows", rowCount),
		zap.Duration("duration", time.Since(startTime)))

	return rowCount
}

// parseRow converts a CSV row into structured data
func parseRow(rec []string) (Sale, error) {
	var s Sale

	// extract basic identifiers
	s.OrderID = rec[0]
	s.ProductID = rec[1]
	s.CustomerID = rec[2]

	// parse string values
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

	// parse date
	date, err := time.Parse("2006-01-02", rec[6])
	if err != nil {
		return s, fmt.Errorf("invalid date: %w", err)
	}
	s.OrderDate = date

	// calculate order total
	s.OrderTotal = float64(s.Quantity)*s.Price*(1-s.Discount) + s.Shipping

	return s, nil
}
