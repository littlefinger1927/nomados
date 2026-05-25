package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen int64
}

type IPRateLimiter struct {
	limiters map[string]*ipLimiter
	mu       sync.Mutex
	rps      float64
	burst    int
}

func NewIPRateLimiter(rps float64, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*ipLimiter),
		rps:      rps,
		burst:    burst,
	}
}

func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	if limiter, exists := l.limiters[ip]; exists {
		return limiter.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(l.rps), l.burst)
	l.limiters[ip] = &ipLimiter{limiter: limiter}
	return limiter
}

func RateLimitMiddleware(limiter *IPRateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		ipLimiter := limiter.GetLimiter(ip)

		if !ipLimiter.Allow() {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, use the first one
		ips := strings.SplitN(forwarded, ",", 2)
		ip := strings.TrimSpace(ips[0])
		if ip != "" {
			return ip
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}