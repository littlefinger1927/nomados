package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nomados/nomados/packages/logging"
	vaultv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/vault/v1"
	"github.com/nomados/nomados/services/vault-service/internal/handler"
	"github.com/nomados/nomados/services/vault-service/internal/health"
	"github.com/nomados/nomados/services/vault-service/internal/keyderivation"
	vaultnats "github.com/nomados/nomados/services/vault-service/internal/nats"
	"github.com/nomados/nomados/services/vault-service/internal/service"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	// NATS configuration.
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	// gRPC server address.
	listenAddr := os.Getenv("VAULT_SERVICE_ADDR")
	if listenAddr == "" {
		listenAddr = ":50057"
	}

	// Session service address for token validation.
	sessionAddr := os.Getenv("SESSION_SERVICE_ADDR")
	if sessionAddr == "" {
		sessionAddr = "localhost:50052"
	}

	// Initialize logger.
	logger := logging.NewLogger("vault-service", nil)

	// Initialize key deriver with default Argon2id parameters.
	kd := keyderivation.NewKeyDeriver()

	// Connect to NATS for publishing key rotation events.
	var publisher *vaultnats.Publisher
	publisher, err := vaultnats.NewPublisher(natsURL)
	if err != nil {
		log.Printf("warning: failed to connect to NATS at %s: %v (rotation events will not be published)", natsURL, err)
		// Continue without NATS — rotation events will be logged but not published.
		publisher = nil
	} else {
		log.Println("connected to NATS")
	}

	// Initialize session validator for authenticating requests.
	sessionValidator, err := service.NewSessionValidator(sessionAddr)
	if err != nil {
		log.Fatalf("failed to initialize session validator: %v", err)
	}

	// Initialize handler and gRPC adapter.
	vaultHandler := handler.NewVaultServiceHandler(kd, publisher, logger)
	grpcAdapter := handler.NewVaultServiceGRPCAdapter(vaultHandler, sessionValidator)

	// Set up gRPC server.
	grpcServer := grpc.NewServer()
	vaultv1.RegisterVaultServiceServer(grpcServer, grpcAdapter)

	// Register gRPC health server.
	hs := grpchealth.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Create health checker for dependency verification.
	checker := health.NewChecker(publisher)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Start gRPC server in a goroutine.
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// All dependencies connected — mark as SERVING.
	if checker.Check(context.Background()) == grpc_health_v1.HealthCheckResponse_SERVING {
		hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	} else {
		log.Println("warning: health check failed at startup, reporting NOT_SERVING")
	}

	logger.Info("vault-service starting", "addr", listenAddr)
	log.Printf("vault-service listening on %s", listenAddr)

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("received %s, shutting down...", sig)

	// Mark as NOT_SERVING before stopping.
	hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Shutdown with timeout (15s for slow operations).
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	// Stop accepting new requests with timeout enforcement.
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Println("graceful stop completed")
	case <-shutdownCtx.Done():
		log.Println("shutdown timeout exceeded, forcing stop")
		grpcServer.Stop()
	}

	// Close resources explicitly.
	if publisher != nil {
		log.Println("closing NATS publisher...")
		publisher.Close()
	}
	log.Println("closing session validator...")
	sessionValidator.Close()

	log.Println("shutdown complete")
}