package utils

import "github.com/gin-gonic/gin"

func JSON(
	c *gin.Context,
	code int,
	body any,
) {
	c.Header("Content-Type", "application/json")
	c.JSON(code, body)
}

func SuccessResponse(key string, data interface{}) gin.H {
	return gin.H{
		"success": true,
		key:       data,
	}
}
