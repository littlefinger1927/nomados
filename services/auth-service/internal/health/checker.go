package health

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// Checker performs health checks for the auth-service dependencies.
type Checker struct {
	pool     *pgxpool.Pool
	redisURL string
}

// NewChecker creates a new health Checker with the given PostgreSQL connection pool
// and optional Redis URL for future Redis health checks.
func NewChecker(pool *pgxpool.Pool, opts ...CheckerOption) *Checker {
	c := &Checker{pool: pool}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// CheckerOption is a functional option for configuring the Checker.
type CheckerOption func(*Checker)

// WithRedisURL sets the Redis URL for future Redis health checks.
// When REDIS_URL is configured, the health checker will verify Redis connectivity
// once Redis integration is implemented.
func WithRedisURL(url string) CheckerOption {
	return func(c *Checker) {
		c.redisURL = url
	}
}

// Check verifies that all dependencies are reachable.
// Returns SERVING if all checks pass, NOT_SERVING otherwise.
func (c *Checker) Check(ctx context.Context) grpc_health_v1.HealthCheckResponse_ServingStatus {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Check PostgreSQL connectivity.
	if err := c.pool.Ping(ctx); err != nil {
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}

	// Redis check: if REDIS_URL is configured, verify connectivity.
	// This will be implemented when Redis integration is added.
	// For now, if REDIS_URL is set but Redis is not yet integrated,
	// we skip the check (log a warning in the future).
	_ = c.redisURL

	return grpc_health_v1.HealthCheckResponse_SERVING
}