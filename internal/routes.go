package internal

import (
	"sales-analytics/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	r *gin.Engine,
	ing handler.Ingestion,
	st handler.Status,
	an handler.Analytics,
) {
	v1 := r.Group("/api/v1")
	{
		// Ingestion endpoints
		v1.POST("/ingestion/upload", ing.Upload)
		v1.GET("/ingestion/status/:id", st.Get)
		v1.POST("/ingestion/refresh", ing.Refresh)

		// Analytics endpoints
		v1.GET("/analytics/revenue", an.Revenue)
	}
	r.GET("/swagger", handler.Swagger)
}
