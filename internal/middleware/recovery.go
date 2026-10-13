package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mulki214/gdcpay_test_project/internal/response"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("panic recovered", "request_id", c.GetString(RequestIDKey), "panic", recovered)
		response.Internal(c)
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
