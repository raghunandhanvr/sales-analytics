package handler

import (
	"database/sql"
	"net/http"

	apierr "sales-analytics/internal/errors"
	"sales-analytics/internal/repository"
	"sales-analytics/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Status struct {
	Jobs repository.JobRepository
	Log  *zap.Logger
}

func (
	h Status,
) Get(
	c *gin.Context,
) {
	id := c.Param("id")
	if id == "" {
		utils.JSON(c, apierr.BadRequest.Code, apierr.BadRequest)
		return
	}

	// get job status from repo
	job, err := h.Jobs.Get(c.Request.Context(), id)

	// better error handling with logging
	if err != nil {
		// log the error
		if h.Log != nil {
			h.Log.Error("job status retrieval failed",
				zap.String("job_id", id),
				zap.Error(err))
		}

		// handle no rows error separately
		if err == sql.ErrNoRows {
			utils.JSON(c, apierr.NotFound.Code, gin.H{
				"error":  "job not found",
				"job_id": id,
			})
			return
		}

		// handle other database errors
		utils.JSON(c, apierr.Internal.Code, apierr.Internal)
		return
	}

	utils.JSON(c, http.StatusOK, job)
}
