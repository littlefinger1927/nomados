package health

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/nomados/nomados/services/file-service/internal/storage"
)

// Checker performs health checks for the file-service dependencies.
type Checker struct {
	pool  *pgxpool.Pool
	store storage.Storage
}

// NewChecker creates a new health Checker with the given dependencies.
func NewChecker(pool *pgxpool.Pool, store storage.Storage) *Checker {
	return &Checker{
		pool:  pool,
		store: store,
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

	// Check MinIO connectivity by ensuring the bucket exists/is accessible.
	if err := c.store.EnsureBucket(ctx); err != nil {
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}

	return grpc_health_v1.HealthCheckResponse_SERVING
}