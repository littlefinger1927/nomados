package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ChallengeStore stores WebAuthn challenges with TTL-based expiry and one-time-use semantics.
type ChallengeStore interface {
	// Put stores a challenge value under the given key with the specified TTL.
	Put(ctx context.Context, key string, value string, ttl time.Duration) error
	// Get retrieves and deletes a challenge value (one-time use).
	// Returns ("", nil) if the key does not exist or has expired.
	Get(ctx context.Context, key string) (string, error)
}

// MemoryChallengeStore is an in-memory ChallengeStore backed by a map with sync.RWMutex.
type MemoryChallengeStore struct {
	mu      sync.RWMutex
	entries map[string]*challengeEntry
	done    chan struct{}
}

type challengeEntry struct {
	value  string
	expiry time.Time
}

// NewMemoryChallengeStore creates a new MemoryChallengeStore and starts a background
// cleanup goroutine that removes expired entries every minute.
func NewMemoryChallengeStore() *MemoryChallengeStore {
	s := &MemoryChallengeStore{
		entries: make(map[string]*challengeEntry),
		done:    make(chan struct{}),
	}
	go s.cleanupLoop()
	return s
}

// Close stops the background cleanup goroutine.
func (s *MemoryChallengeStore) Close() {
	close(s.done)
}

func (s *MemoryChallengeStore) Put(_ context.Context, key string, value string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = &challengeEntry{
		value:  value,
		expiry: time.Now().Add(ttl),
	}
	return nil
}

func (s *MemoryChallengeStore) Get(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[key]
	if !ok {
		return "", nil
	}
	// Delete on read (one-time use)
	delete(s.entries, key)

	if time.Now().After(entry.expiry) {
		return "", nil
	}
	return entry.value, nil
}

func (s *MemoryChallengeStore) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.removeExpired()
		}
	}
}

func (s *MemoryChallengeStore) removeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, v := range s.entries {
		if now.After(v.expiry) {
			delete(s.entries, k)
		}
	}
}

// RedisChallengeStore is a Redis-backed ChallengeStore using go-redis.
type RedisChallengeStore struct {
	client *redis.Client
}

// NewRedisChallengeStore creates a new RedisChallengeStore with the given redis.Client.
func NewRedisChallengeStore(client *redis.Client) *RedisChallengeStore {
	return &RedisChallengeStore{client: client}
}

func (s *RedisChallengeStore) Put(ctx context.Context, key string, value string, ttl time.Duration) error {
	redisKey := fmt.Sprintf("challenge:%s", key)
	return s.client.Set(ctx, redisKey, value, ttl).Err()
}

func (s *RedisChallengeStore) Get(ctx context.Context, key string) (string, error) {
	redisKey := fmt.Sprintf("challenge:%s", key)
	val, err := s.client.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("redis get: %w", err)
	}
	// Delete on read (one-time use) — best-effort; ignore error since TTL will clean up
	_ = s.client.Del(ctx, redisKey).Err()
	return val, nil
}