package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/mulki214/gdcpay_test_project/internal/response"
)

// ErrorHandler converts errors left on the Gin context into a safe API response.
// Handlers that already wrote a response keep their explicit status and body.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		response.Internal(c)
	}
}
