package handler

import (
	"context"
	"time"

	"sales-analytics/internal/service/ingestion"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Cron struct {
	Service ingestion.Service
	Log     *zap.Logger
}

func (
	c *Cron,
) RunImport(
	ctx context.Context,
) {
	jobID := uuid.NewString()

	c.Log.Info("Starting scheduled CSV import",
		zap.String("job_id", jobID),
		zap.Time("start_time", time.Now()))

	if err := c.Service.ImportFromPath(ctx, jobID, "append"); err != nil {
		c.Log.Error("Scheduled CSV import failed",
			zap.String("job_id", jobID),
			zap.Error(err))
		return
	}

	c.Log.Info("Scheduled CSV import completed",
		zap.String("job_id", jobID))
}
