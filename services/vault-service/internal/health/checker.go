package health

import (
	"context"
	"time"

	"google.golang.org/grpc/health/grpc_health_v1"

	vaultnats "github.com/nomados/nomados/services/vault-service/internal/nats"
)

// Checker performs health checks for the vault-service dependencies.
type Checker struct {
	publisher *vaultnats.Publisher
}

// NewChecker creates a new health Checker with the given NATS publisher.
func NewChecker(publisher *vaultnats.Publisher) *Checker {
	return &Checker{publisher: publisher}
}

// Check verifies that all dependencies are reachable.
// Returns SERVING if all checks pass, NOT_SERVING otherwise.
func (c *Checker) Check(ctx context.Context) grpc_health_v1.HealthCheckResponse_ServingStatus {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Check NATS connectivity by verifying the connection is still active.
	// The NATS client maintains a persistent connection; if it's disconnected
	// the service should report NOT_SERVING.
	if c.publisher != nil {
		if !c.publisher.IsConnected() {
			return grpc_health_v1.HealthCheckResponse_NOT_SERVING
		}
	}

	// Use the context deadline to ensure we don't block indefinitely.
	select {
	case <-ctx.Done():
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING
	default:
	}

	return grpc_health_v1.HealthCheckResponse_SERVING
}