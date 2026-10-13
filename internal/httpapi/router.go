package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mulki214/gdcpay_test_project/internal/auth"
	"github.com/mulki214/gdcpay_test_project/internal/middleware"
	"github.com/mulki214/gdcpay_test_project/internal/response"
)

func NewRouter(db *pgxpool.Pool, authService *auth.Service, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(middleware.RequestID(), middleware.AccessLog(logger), middleware.Recovery(logger))

	health := NewHealthHandler(db)
	authHandler := NewAuthHandler(authService)
	router.GET("/health", health.Health)
	router.GET("/ready", health.Ready)
	router.POST("/auth/register", authHandler.Register)
	router.POST("/auth/login", authHandler.Login)
	router.GET("/openapi.yaml", openAPISpec)
	router.GET("/swagger/*any", swaggerUI)
	router.NoRoute(func(c *gin.Context) {
		response.WriteError(c, http.StatusNotFound, "NOT_FOUND", "The requested resource was not found")
	})
	router.NoMethod(func(c *gin.Context) {
		response.WriteError(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "The request method is not allowed")
	})
	return router
}
