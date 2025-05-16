package handler

import (
	"net/http"
	"strconv"
	"time"

	"sales-analytics/internal/service/analytics"
	"sales-analytics/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Analytics struct {
	Service analytics.Service
	Log     *zap.Logger
}

// Revenue is a unified endpoint for all revenue-related analytics
// Query parameters:
// - start_date: start of the date range (default: 1 year ago)
// - end_date: end of the date range (default: today)
// - type: revenue calculation type (total, product, category, region, trend)
// - interval: for trend analysis (monthly, quarterly, yearly)
// - limit: number of items to return (default: 10)
func (
	h *Analytics,
) Revenue(
	c *gin.Context,
) {
	start, end := h.getDateRange(c)
	calculationType := c.DefaultQuery("type", "total")

	logFields := []zap.Field{
		zap.String("calculation", calculationType),
		zap.String("start_date", start),
		zap.String("end_date", end),
	}

	var result interface{}

	switch calculationType {
	case "total":
		revenue, err := h.Service.Total(c.Request.Context(), start, end)
		if err != nil {
			h.Log.Error("Failed to calculate total revenue", append(logFields, zap.Error(err))...)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate total revenue"})
			return
		}

		result = gin.H{
			"calculation": "total_revenue",
			"result":      revenue,
			"period": gin.H{
				"start_date": start,
				"end_date":   end,
			},
		}

	case "product":
		products, err := h.Service.ByProduct(c.Request.Context(), start, end)
		if err != nil {
			h.Log.Error("Failed to calculate revenue by product", append(logFields, zap.Error(err))...)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate revenue by product"})
			return
		}

		result = gin.H{
			"calculation": "revenue_by_product",
			"count":       len(products),
			"products":    products,
			"period": gin.H{
				"start_date": start,
				"end_date":   end,
			},
		}

	case "category":
		categories, err := h.Service.ByCategory(c.Request.Context(), start, end)
		if err != nil {
			h.Log.Error("Failed to calculate revenue by category", append(logFields, zap.Error(err))...)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate revenue by category"})
			return
		}

		result = gin.H{
			"calculation": "revenue_by_category",
			"count":       len(categories),
			"categories":  categories,
			"period": gin.H{
				"start_date": start,
				"end_date":   end,
			},
		}

	case "region":
		regions, err := h.Service.ByRegion(c.Request.Context(), start, end)
		if err != nil {
			h.Log.Error("Failed to calculate revenue by region", append(logFields, zap.Error(err))...)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate revenue by region"})
			return
		}

		result = gin.H{
			"calculation": "revenue_by_region",
			"count":       len(regions),
			"regions":     regions,
			"period": gin.H{
				"start_date": start,
				"end_date":   end,
			},
		}

	case "top_products":
		limit := h.getLimit(c)
		logFields = append(logFields, zap.Int("limit", limit))

		products, err := h.Service.TopProducts(c.Request.Context(), start, end, limit)
		if err != nil {
			h.Log.Error("Failed to get top products", append(logFields, zap.Error(err))...)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate top products"})
			return
		}

		result = gin.H{
			"calculation": "top_products",
			"count":       len(products),
			"limit":       limit,
			"products":    products,
			"period": gin.H{
				"start_date": start,
				"end_date":   end,
			},
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid calculation type"})
		return
	}

	h.Log.Info("Revenue calculation completed", logFields...)
	c.JSON(http.StatusOK, utils.SuccessResponse("data", result))
}

func (
	h *Analytics,
) getDateRange(
	c *gin.Context,
) (string, string) {
	start := c.Query("start_date")
	end := c.Query("end_date")

	if start == "" {
		start = time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	}
	if end == "" {
		end = time.Now().Format("2006-01-02")
	}

	return start, end
}

func (
	h *Analytics,
) getLimit(
	c *gin.Context,
) int {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		return 10
	}
	return limit
}
