package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestIdempotencyKeyRequiresCanonicalUUID(t *testing.T) {
	validKey := uuid.New()
	tests := []struct {
		name      string
		value     string
		wantValid bool
	}{
		{name: "valid UUID", value: validKey.String(), wantValid: true},
		{name: "uppercase UUID", value: strings.ToUpper(validKey.String()), wantValid: true},
		{name: "missing", value: "", wantValid: false},
		{name: "invalid", value: "not-a-uuid", wantValid: false},
		{name: "UUID without hyphens", value: validKey.String()[:8] + validKey.String()[9:13] + validKey.String()[14:18] + validKey.String()[19:23] + validKey.String()[24:], wantValid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest("POST", "/tasks", nil)
			context.Request.Header.Set("Idempotency-Key", test.value)

			key, valid := idempotencyKey(context)
			if valid != test.wantValid {
				t.Fatalf("valid = %t, want %t", valid, test.wantValid)
			}
			if valid && key != validKey {
				t.Fatalf("key = %s, want %s", key, validKey)
			}
		})
	}
}
