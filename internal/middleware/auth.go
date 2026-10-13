package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mulki214/gdcpay_test_project/internal/auth"
	"github.com/mulki214/gdcpay_test_project/internal/response"
)

const (
	AuthenticatedUserIDKey = "authenticated_user_id"
	AuthenticatedEmailKey  = "authenticated_user_email"
)

func JWT(manager *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		const prefix = "Bearer "
		header := c.GetHeader("Authorization")
		if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
			response.WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "A valid bearer token is required")
			c.Abort()
			return
		}
		claims, err := manager.Parse(header[len(prefix):])
		if err != nil {
			response.WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "A valid bearer token is required")
			c.Abort()
			return
		}
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			response.WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "A valid bearer token is required")
			c.Abort()
			return
		}
		c.Set(AuthenticatedUserIDKey, userID)
		c.Set(AuthenticatedEmailKey, claims.Email)
		c.Next()
	}
}
