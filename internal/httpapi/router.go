package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mulki214/gdcpay_test_project/internal/auth"
	"github.com/mulki214/gdcpay_test_project/internal/middleware"
	"github.com/mulki214/gdcpay_test_project/internal/response"
	"github.com/mulki214/gdcpay_test_project/internal/task"
)

func NewRouter(db *pgxpool.Pool, authService *auth.Service, taskService *task.Service, tokenManager *auth.TokenManager, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(middleware.RequestID(), middleware.AccessLog(logger), middleware.Recovery(logger), middleware.ErrorHandler())

	health := NewHealthHandler(db)
	authHandler := NewAuthHandler(authService)
	taskHandler := NewTaskHandler(taskService)
	router.GET("/health", health.Health)
	router.GET("/ready", health.Ready)
	router.POST("/auth/register", authHandler.Register)
	router.POST("/auth/login", authHandler.Login)
	tasks := router.Group("/tasks", middleware.JWT(tokenManager))
	tasks.POST("", taskHandler.Create)
	tasks.GET("", taskHandler.List)
	tasks.GET("/:id", taskHandler.Get)
	tasks.PUT("/:id", taskHandler.Update)
	tasks.DELETE("/:id", taskHandler.Delete)
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
