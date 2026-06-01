package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nomados/nomados/packages/logging"
	workspacev1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/workspace/v1"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/docker"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/handler"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/health"
	wsnats "github.com/nomados/nomados/services/workspace-orchestrator/internal/nats"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/repository"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/service"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://nomados:nomados_dev@localhost:5432/nomados?sslmode=disable"
	}
	listenAddr := os.Getenv("WORKSPACE_SERVICE_ADDR")
	if listenAddr == "" {
		listenAddr = ":50053"
	}

	ctx := context.Background()

	// Connect to PostgreSQL
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// Connect to Docker
	var dockerClient docker.DockerClient
	if os.Getenv("DOCKER_ENABLED") != "false" {
		dc, err := docker.NewDockerClient()
		if err != nil {
			log.Fatalf("failed to connect to Docker: %v", err)
		}
		dockerClient = dc
	} else {
		log.Println("DOCKER_ENABLED=false, using mock Docker client")
		dockerClient = docker.NewMockClient()
	}
	// Connect to NATS
	pub, err := wsnats.NewPublisher(natsURL)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	// Create services
	logger := logging.NewLogger("workspace-orchestrator", nil)
	repo := repository.NewPostgresRepository(pool)
	svc := service.NewWorkspaceService(dockerClient, repo, logger, pub)
	workspaceHandler := handler.NewWorkspaceServiceHandler(svc)

	// Start gRPC server
	grpcServer := grpc.NewServer()
	workspacev1.RegisterWorkspaceServiceServer(grpcServer, workspaceHandler)

	// Register gRPC health server.
	hs := grpchealth.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Create health checker for dependency verification.
	checker := health.NewChecker(dockerClient, pub)

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

	log.Printf("workspace-orchestrator listening on %s", listenAddr)

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("received %s, shutting down...", sig)

	// Mark as NOT_SERVING before stopping.
	hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Shutdown with timeout (15s for slow container operations).
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
	log.Println("closing NATS publisher...")
	pub.Close()
	log.Println("closing Docker client...")
	dockerClient.Close()
	log.Println("closing database connection...")
	pool.Close()

	log.Println("shutdown complete")
}