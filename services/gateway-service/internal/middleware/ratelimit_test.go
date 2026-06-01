package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
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

// --- Eviction tests ---

func TestEvictStaleEntries_RemovesOldEntries(t *testing.T) {
	limiter := NewIPRateLimiter(100, 5)

	// Add entries for several IPs
	ips := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4", "10.0.0.5"}
	for _, ip := range ips {
		limiter.GetLimiter(ip)
	}

	// All 5 entries should exist
	totalBefore := limiter.totalEntries()
	if totalBefore != 5 {
		t.Fatalf("expected 5 entries before eviction, got %d", totalBefore)
	}

	// Manually age the entries by overwriting lastSeen to be older than maxEntryAge
	cutoff := time.Now().Add(-maxEntryAge - time.Second)
	for i := range limiter.shards {
		s := &limiter.shards[i]
		s.mu.Lock()
		for _, entry := range s.limiters {
			entry.lastSeen = cutoff
		}
		s.mu.Unlock()
	}

	// Evict stale entries
	limiter.EvictStaleEntries()

	// All entries should be removed
	totalAfter := limiter.totalEntries()
	if totalAfter != 0 {
		t.Errorf("expected 0 entries after eviction, got %d", totalAfter)
	}
}

func TestEvictStaleEntries_KeepsRecentEntries(t *testing.T) {
	limiter := NewIPRateLimiter(100, 5)

	// Add entries
	limiter.GetLimiter("10.0.0.1")
	limiter.GetLimiter("10.0.0.2")
	limiter.GetLimiter("10.0.0.3")

	// Age only the first entry past the threshold
	cutoff := time.Now().Add(-maxEntryAge - time.Second)
	idx := getShardIndex("10.0.0.1")
	limiter.shards[idx].mu.Lock()
	limiter.shards[idx].limiters["10.0.0.1"].lastSeen = cutoff
	limiter.shards[idx].mu.Unlock()

	// Evict
	limiter.EvictStaleEntries()

	// Stale entry should be gone, recent entries should remain
	total := limiter.totalEntries()
	if total != 2 {
		t.Errorf("expected 2 entries after eviction, got %d", total)
	}

	// Verify the stale IP is gone
	idx = getShardIndex("10.0.0.1")
	limiter.shards[idx].mu.RLock()
	_, exists := limiter.shards[idx].limiters["10.0.0.1"]
	limiter.shards[idx].mu.RUnlock()
	if exists {
		t.Error("expected stale IP 10.0.0.1 to be evicted")
	}

	// Verify recent IPs still exist
	for _, ip := range []string{"10.0.0.2", "10.0.0.3"} {
		idx := getShardIndex(ip)
		limiter.shards[idx].mu.RLock()
		_, exists := limiter.shards[idx].limiters[ip]
		limiter.shards[idx].mu.RUnlock()
		if !exists {
			t.Errorf("expected recent IP %s to still exist", ip)
		}
	}
}

func TestEvictStaleEntries_PartialEvictionAcrossShards(t *testing.T) {
	limiter := NewIPRateLimiter(100, 5)

	// Add enough IPs to cover multiple shards
	for i := 0; i < 32; i++ {
		ip := fmt.Sprintf("10.0.%d.%d", i/256, i%256)
		limiter.GetLimiter(ip)
	}

	totalBefore := limiter.totalEntries()
	if totalBefore != 32 {
		t.Fatalf("expected 32 entries before eviction, got %d", totalBefore)
	}

	// Age every other entry
	cutoff := time.Now().Add(-maxEntryAge - time.Second)
	for i := 0; i < 32; i += 2 {
		ip := fmt.Sprintf("10.0.%d.%d", i/256, i%256)
		idx := getShardIndex(ip)
		limiter.shards[idx].mu.Lock()
		limiter.shards[idx].limiters[ip].lastSeen = cutoff
		limiter.shards[idx].mu.Unlock()
	}

	limiter.EvictStaleEntries()

	totalAfter := limiter.totalEntries()
	if totalAfter != 16 {
		t.Errorf("expected 16 entries after partial eviction, got %d", totalAfter)
	}
}

func TestGetLimiter_UpdatesLastSeen(t *testing.T) {
	limiter := NewIPRateLimiter(100, 5)

	// Get a limiter for an IP
	l1 := limiter.GetLimiter("10.0.0.1")

	// Record the initial lastSeen
	idx := getShardIndex("10.0.0.1")
	limiter.shards[idx].mu.RLock()
	initial := limiter.shards[idx].limiters["10.0.0.1"].lastSeen
	limiter.shards[idx].mu.RUnlock()

	// Wait a tiny bit and get the limiter again
	time.Sleep(2 * time.Millisecond)
	l2 := limiter.GetLimiter("10.0.0.1")

	// Should be the same limiter object
	if l1 != l2 {
		t.Error("expected same rate.Limiter for same IP")
	}

	// lastSeen should have been updated
	limiter.shards[idx].mu.RLock()
	updated := limiter.shards[idx].limiters["10.0.0.1"].lastSeen
	limiter.shards[idx].mu.RUnlock()

	if !updated.After(initial) {
		t.Error("expected lastSeen to be updated after GetLimiter call")
	}
}

func TestEvictionConcurrency(t *testing.T) {
	limiter := NewIPRateLimiter(1000, 10)

	// Hammer GetLimiter from multiple goroutines while eviction runs
	var wg sync.WaitGroup
	done := make(chan struct{})

	// Writers: continuously add/access limiters
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					ip := fmt.Sprintf("192.0.2.%d", g%10)
					l := limiter.GetLimiter(ip)
					_ = l.Allow()
				}
			}
		}(g)
	}

	// Run a few eviction cycles
	for i := 0; i < 5; i++ {
		limiter.EvictStaleEntries()
		time.Sleep(10 * time.Millisecond)
	}

	close(done)
	wg.Wait()
}

func TestEvictStaleEntries_EmptyLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(100, 5)

	// Should not panic on empty limiter
	limiter.EvictStaleEntries()

	if limiter.totalEntries() != 0 {
		t.Errorf("expected 0 entries, got %d", limiter.totalEntries())
	}
}

func TestShardDistribution(t *testing.T) {
	limiter := NewIPRateLimiter(100, 5)

	// Add entries for many IPs and verify they spread across shards
	for i := 0; i < 100; i++ {
		ip := fmt.Sprintf("10.%d.%d.%d", i/65536, (i/256)%256, i%256)
		limiter.GetLimiter(ip)
	}

	nonEmptyShards := 0
	for i := range limiter.shards {
		limiter.shards[i].mu.RLock()
		if len(limiter.shards[i].limiters) > 0 {
			nonEmptyShards++
		}
		limiter.shards[i].mu.RUnlock()
	}

	// With 100 entries across 16 shards, we should see entries in most shards
	if nonEmptyShards < 8 {
		t.Errorf("expected entries in at least 8 shards, got %d", nonEmptyShards)
	}
}

func TestGetShardIndex_Deterministic(t *testing.T) {
	// Same IP should always map to the same shard
	ip := "192.0.2.1"
	idx1 := getShardIndex(ip)
	idx2 := getShardIndex(ip)
	if idx1 != idx2 {
		t.Errorf("expected same shard index for same IP, got %d and %d", idx1, idx2)
	}

	// Different IPs should generally map to different shards
	differentShard := false
	for i := 0; i < 100; i++ {
		otherIP := fmt.Sprintf("192.0.2.%d", i+2)
		if getShardIndex(otherIP) != idx1 {
			differentShard = true
			break
		}
	}
	if !differentShard {
		t.Error("expected at least some IPs to map to different shards")
	}
}

// totalEntries is a test helper that counts all entries across all shards.
func (l *IPRateLimiter) totalEntries() int {
	total := 0
	for i := range l.shards {
		l.shards[i].mu.RLock()
		total += len(l.shards[i].limiters)
		l.shards[i].mu.RUnlock()
	}
	return total
}