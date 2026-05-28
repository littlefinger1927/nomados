package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	workspacev1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/workspace/v1"
	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/docker"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/handler"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/health"
	wsnats "github.com/nomados/nomados/services/workspace-orchestrator/internal/nats"
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
	listenAddr := os.Getenv("WORKSPACE_SERVICE_ADDR")
	if listenAddr == "" {
		listenAddr = ":50053"
	}

	// Connect to Docker
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		log.Fatalf("failed to connect to Docker: %v", err)
	}
	defer dockerClient.Close()

	// Connect to NATS
	pub, err := wsnats.NewPublisher(natsURL)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer pub.Close()

	// Create services
	logger := logging.NewLogger("workspace-orchestrator", nil)
	svc := service.NewWorkspaceService(dockerClient, logger, pub)
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

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		// Mark as NOT_SERVING before stopping.
		hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		grpcServer.GracefulStop()
	}()

	// All dependencies connected — mark as SERVING.
	if checker.Check(context.Background()) == grpc_health_v1.HealthCheckResponse_SERVING {
		hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	} else {
		log.Println("warning: health check failed at startup, reporting NOT_SERVING")
	}

	log.Printf("workspace-orchestrator listening on %s", listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}