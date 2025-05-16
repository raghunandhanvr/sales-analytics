package errors

import "github.com/gin-gonic/gin"

type APIError struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
}

func (
	e APIError,
) Error() string {
	return e.Message
}

var (
	FileRequired = APIError{Code: 400, Message: "file required"}
	BadRequest   = APIError{Code: 400, Message: "bad_request"}
	Internal     = APIError{Code: 500, Message: "internal"}
	NotFound     = APIError{Code: 404, Message: "not_found"}
	Unauthorized = APIError{Code: 401, Message: "unauthorized"}
)

func ServerError(
	message string,
) gin.H {
	return gin.H{
		"error": message,
	}
}
