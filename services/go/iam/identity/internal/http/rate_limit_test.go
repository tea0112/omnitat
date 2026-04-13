package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterBlocksAfterLimit(t *testing.T) {
	limiter := NewRateLimiter(RateLimitConfig{
		Requests: 2,
		Window:   time.Minute,
		Burst:    2,
		Message:  "too many signin attempts",
	})
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", nil)
		req.RemoteAddr = "198.51.100.20"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected status 204 on allowed request, got %d", rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signin", nil)
	req.RemoteAddr = "198.51.100.20"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rec.Code)
	}
}
