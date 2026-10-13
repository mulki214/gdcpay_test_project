package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mulki214/gdcpay_test_project/internal/response"
)

func TestErrorHandlerReturnsSafeStructuredResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ErrorHandler())
	router.GET("/failure", func(c *gin.Context) {
		_ = c.Error(errors.New("database password must not be exposed"))
	})

	result := httptest.NewRecorder()
	router.ServeHTTP(result, httptest.NewRequest(http.MethodGet, "/failure", nil))

	assertStructuredInternalError(t, result)
}

func TestRecoveryReturnsSafeStructuredResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router.Use(Recovery(logger))
	router.GET("/panic", func(c *gin.Context) {
		panic("internal panic detail")
	})

	result := httptest.NewRecorder()
	router.ServeHTTP(result, httptest.NewRequest(http.MethodGet, "/panic", nil))

	assertStructuredInternalError(t, result)
}

func assertStructuredInternalError(t *testing.T, result *httptest.ResponseRecorder) {
	t.Helper()
	if result.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", result.Code, http.StatusInternalServerError)
	}
	if contentType := result.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
	var payload response.Error
	if err := json.Unmarshal(result.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if payload.Status != http.StatusInternalServerError || payload.Code != "INTERNAL_ERROR" || payload.Message != "An unexpected error occurred" || payload.Timestamp.IsZero() {
		t.Fatalf("unexpected error payload: %+v", payload)
	}
	if strings.Contains(result.Body.String(), "database password") || strings.Contains(result.Body.String(), "internal panic detail") {
		t.Fatalf("internal error details leaked in response: %s", result.Body.String())
	}
}
