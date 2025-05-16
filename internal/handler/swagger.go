package handler

import "github.com/gin-gonic/gin"

func Swagger(
	c *gin.Context,
) {
	c.File("./docs/swagger/swagger.yaml")
}
