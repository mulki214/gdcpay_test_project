package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mulki214/gdcpay_test_project/internal/response"
)

func TestRouterReturnsStructuredErrorsForRoutingFailures(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   string
	}{
		{name: "unknown path", method: http.MethodGet, path: "/missing", wantStatus: http.StatusNotFound, wantCode: "NOT_FOUND"},
		{name: "unknown Swagger UI asset", method: http.MethodGet, path: "/swagger/missing.js", wantStatus: http.StatusNotFound, wantCode: "NOT_FOUND"},
		{name: "unsupported method", method: http.MethodPost, path: "/health", wantStatus: http.StatusMethodNotAllowed, wantCode: "METHOD_NOT_ALLOWED"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.path, nil)
			router.ServeHTTP(result, request)

			if result.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", result.Code, test.wantStatus)
			}
			var payload response.Error
			if err := json.Unmarshal(result.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if payload.Status != test.wantStatus || payload.Code != test.wantCode || payload.Message == "" || payload.Timestamp.IsZero() {
				t.Fatalf("unexpected error payload: %+v", payload)
			}
		})
	}
}
