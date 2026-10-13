package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Error struct {
	Status    int       `json:"status"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func WriteError(c *gin.Context, status int, code, message string) {
	c.JSON(status, Error{
		Status:    status,
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC(),
	})
}

func Internal(c *gin.Context) {
	WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
}
