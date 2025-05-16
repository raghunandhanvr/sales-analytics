package handler

import (
	"context"
	"net/http"
	"os"

	apierr "sales-analytics/internal/errors"
	"sales-analytics/internal/repository"
	"sales-analytics/internal/service/ingestion"
	"sales-analytics/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Ingestion struct {
	Service ingestion.Service
	Jobs    repository.JobRepository
	Log     *zap.Logger
}

func (
	h Ingestion,
) Upload(
	c *gin.Context,
) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		h.Log.Error("No file uploaded", zap.Error(err))
		utils.JSON(c, apierr.FileRequired.Code, apierr.FileRequired)
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		h.Log.Error("Failed to open uploaded file", zap.Error(err))
		utils.JSON(c, apierr.Internal.Code, apierr.Internal)
		return
	}

	ctx := context.Background()
	jobID := uuid.NewString()
	mode := c.DefaultQuery("mode", "append")

	h.Jobs.Insert(ctx, jobID)

	go func() {
		defer f.Close()
		h.Service.ImportFile(ctx, f, jobID, mode)
	}()

	utils.JSON(c, http.StatusAccepted, gin.H{"job_id": jobID})
}

func (
	h Ingestion,
) ProcessLocal(
	c *gin.Context,
) {
	filePath := c.Query("path")
	if filePath == "" {
		h.Log.Error("No file path provided")
		utils.JSON(c, apierr.BadRequest.Code, apierr.BadRequest)
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		h.Log.Error("File does not exist", zap.String("path", filePath))
		utils.JSON(c, apierr.BadRequest.Code, gin.H{"error": "File does not exist"})
		return
	}

	ctx := context.Background()
	jobID := uuid.NewString()
	mode := c.DefaultQuery("mode", "append")

	h.Jobs.Insert(ctx, jobID)

	go func() {
		file, err := os.Open(filePath)
		if err != nil {
			h.Log.Error("Failed to open file", zap.Error(err), zap.String("path", filePath))
			h.Jobs.SetFailed(ctx, jobID, "Failed to open file: "+err.Error())
			return
		}
		defer file.Close()

		h.Service.ImportFile(ctx, file, jobID, mode)
	}()

	utils.JSON(c, http.StatusAccepted, gin.H{"job_id": jobID})
}

func (
	h Ingestion,
) Refresh(
	c *gin.Context,
) {
	mode := c.DefaultQuery("mode", "append")

	jobID := uuid.NewString()
	ctx := context.Background()

	h.Log.Info("manual refresh triggered",
		zap.String("job_id", jobID),
		zap.String("mode", mode))

	// insert job record first for status tracking
	h.Jobs.Insert(ctx, jobID)

	// run import in background to avoid blocking api
	go func() {
		// direct call to csv path import service
		if err := h.Service.ImportFromPath(ctx, jobID, mode); err != nil {
			h.Log.Error("refresh failed",
				zap.String("job_id", jobID),
				zap.Error(err))
			return
		}

		h.Log.Info("refresh completed successfully",
			zap.String("job_id", jobID))
	}()

	// return immediately with job id for tracking
	utils.JSON(c, http.StatusAccepted, gin.H{
		"job_id":  jobID,
		"message": "Refresh started",
		"mode":    mode,
	})
}
