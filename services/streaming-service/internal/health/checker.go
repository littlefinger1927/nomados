package health

import (
	"context"
	"time"

	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/nomados/nomados/services/streaming-service/internal/nats"
)

// Checker performs health checks for the streaming-service dependencies.
type Checker struct {
	sub *nats.Subscriber
}

// NewChecker creates a new health Checker with the given NATS subscriber.
func NewChecker(sub *nats.Subscriber) *Checker {
	return &Checker{sub: sub}
}

// Check verifies that all dependencies are reachable.
// Returns SERVING if all checks pass, NOT_SERVING otherwise.
func (c *Checker) Check(ctx context.Context) grpc_health_v1.HealthCheckResponse_ServingStatus {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Check NATS connectivity by verifying the connection is still active.
	if c.sub != nil {
		if !c.sub.IsConnected() {
			return grpc_health_v1.HealthCheckResponse_NOT_SERVING
		}
	}

	select {
	case <-ctx.Done():
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING
	default:
	}

	return grpc_health_v1.HealthCheckResponse_SERVING
}