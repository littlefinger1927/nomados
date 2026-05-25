package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitMiddleware_WithinLimit(t *testing.T) {
	limiter := NewIPRateLimiter(100, 5)

	called := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimitMiddleware(limiter, next)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, rec.Code)
		}
	}

	if called != 5 {
		t.Errorf("expected 5 calls, got %d", called)
	}
}

func TestRateLimitMiddleware_ExceedsLimit(t *testing.T) {
	// Very low rate limit: 1 request per second, burst of 2
	limiter := NewIPRateLimiter(1, 2)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimitMiddleware(limiter, next)

	// First 2 requests should succeed (burst)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("burst request %d: expected 200, got %d", i, rec.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("rate limited request: expected 429, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_DifferentIPs(t *testing.T) {
	// Low rate limit per IP
	limiter := NewIPRateLimiter(1, 1)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimitMiddleware(limiter, next)

	ips := []string{"192.0.2.1:1234", "192.0.2.2:1234", "192.0.2.3:1234"}
	for _, addr := range ips {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = addr
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("IP %s: expected 200, got %d", addr, rec.Code)
		}
	}
}

func TestRateLimitMiddleware_XForwardedFor(t *testing.T) {
	limiter := NewIPRateLimiter(1, 1)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimitMiddleware(limiter, next)

	// First request with X-Forwarded-For should succeed
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Second request from same forwarded IP should be rate limited
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "10.0.0.2:1234" // different remote addr
	req2.Header.Set("X-Forwarded-For", "203.0.113.50") // same forwarded IP
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 for same forwarded IP, got %d", rec2.Code)
	}
}

func TestRateLimitMiddleware_Recovery(t *testing.T) {
	// 10 rps with burst of 1
	limiter := NewIPRateLimiter(10, 1)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimitMiddleware(limiter, next)

	// Exhaust burst
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Immediate next request should be rate limited
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.0.2.1:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec2.Code)
	}

	// After waiting, should recover
	time.Sleep(150 * time.Millisecond)
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "192.0.2.1:1234"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Errorf("expected 200 after recovery, got %d", rec3.Code)
	}
}