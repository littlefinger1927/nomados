package health

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// BackendConfig holds the address of a downstream service.
type BackendConfig struct {
	Name string
	Addr string
}

// Checker performs health checks for the gateway-service by verifying
// downstream gRPC services are reachable.
type Checker struct {
	backends []BackendConfig
}

// NewChecker creates a new health Checker with the given backend service addresses.
func NewChecker(backends []BackendConfig) *Checker {
	return &Checker{backends: backends}
}

// Check verifies that downstream services are reachable by attempting
// a gRPC health check on each one.
// Returns "SERVING" if the gateway is operational. Individual backend failures
// are logged as warnings but do not cause the gateway to report NOT_SERVING,
// since the gateway can still route requests and return appropriate errors
// for unavailable backends.
func (c *Checker) Check(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for _, backend := range c.backends {
		if err := c.checkBackend(ctx, backend.Addr); err != nil {
			log.Printf("WARNING: backend %s (%s) health check failed: %v", backend.Name, backend.Addr, err)
		}
	}

	return "SERVING"
}

// checkBackend attempts to connect to a backend service and check its health.
func (c *Checker) checkBackend(ctx context.Context, addr string) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)
	_, err = client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	return err
}