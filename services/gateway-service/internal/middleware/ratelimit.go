package middleware

import (
	"hash/fnv"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	numShards        = 16
	evictionInterval = 60 * time.Second
	maxEntryAge      = 10 * time.Minute
)

type shard struct {
	mu       sync.RWMutex
	limiters map[string]*ipLimiter
}

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	shards [numShards]shard
	rps    float64
	burst  int
	once   sync.Once
}

func NewIPRateLimiter(rps float64, burst int) *IPRateLimiter {
	l := &IPRateLimiter{
		rps:   rps,
		burst: burst,
	}
	for i := range l.shards {
		l.shards[i].limiters = make(map[string]*ipLimiter)
	}
	return l
}

// getShardIndex returns the shard index for a given IP key.
func getShardIndex(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32() % numShards
}

func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	idx := getShardIndex(ip)
	s := &l.shards[idx]

	s.mu.Lock()
	entry, exists := s.limiters[ip]
	if exists {
		entry.lastSeen = time.Now()
		s.mu.Unlock()
		return entry.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(l.rps), l.burst)
	s.limiters[ip] = &ipLimiter{limiter: limiter, lastSeen: time.Now()}
	s.mu.Unlock()
	return limiter
}

// EvictStaleEntries removes entries that have not been seen for longer than
// maxEntryAge. It iterates all shards and is safe to call concurrently.
func (l *IPRateLimiter) EvictStaleEntries() {
	cutoff := time.Now().Add(-maxEntryAge)
	for i := range l.shards {
		s := &l.shards[i]
		s.mu.Lock()
		for ip, entry := range s.limiters {
			if entry.lastSeen.Before(cutoff) {
				delete(s.limiters, ip)
			}
		}
		s.mu.Unlock()
	}
}

func (l *IPRateLimiter) startEviction() {
	go func() {
		ticker := time.NewTicker(evictionInterval)
		defer ticker.Stop()
		for range ticker.C {
			l.EvictStaleEntries()
		}
	}()
}

func RateLimitMiddleware(limiter *IPRateLimiter, next http.Handler) http.Handler {
	limiter.once.Do(limiter.startEviction)
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