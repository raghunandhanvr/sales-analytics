package ingestion

import (
	"context"
	"io"
	"sales-analytics/internal/models"
)

type Service interface {
	ImportFromPath(ctx context.Context, jobID, mode string) error

	ImportFile(ctx context.Context, r io.Reader, jobID, mode string) error

	GetJobStatus(ctx context.Context, jobID string) (models.IngestionJob, error)
}
