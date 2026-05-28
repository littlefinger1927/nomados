package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/browser-manager/internal/chromium"
	bmnats "github.com/nomados/nomados/services/browser-manager/internal/nats"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	bmhealth "github.com/nomados/nomados/services/browser-manager/internal/health"
)

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	listenAddr := os.Getenv("BROWSER_MANAGER_ADDR")
	if listenAddr == "" {
		listenAddr = ":50054"
	}

	binPath := os.Getenv("CHROMIUM_BIN_PATH")
	profileBasePath := os.Getenv("BROWSER_PROFILE_BASE_PATH")

	// Create logger
	logger := logging.NewLogger("browser-manager", nil)

	// Create Chromium launcher
	config := chromium.LauncherConfig{
		BinPath:         binPath,
		ProfileBasePath: profileBasePath,
	}
	launcher := chromium.NewChromiumLauncher(config, logger)

	// Connect to NATS and subscribe to workspace events
	sub, err := bmnats.NewSubscriber(natsURL, launcher, logger)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer sub.Close()

	// Set up graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start gRPC server (health check / management endpoint for Phase 1)
	grpcServer := grpc.NewServer()

	// Register gRPC health server.
	hs := grpchealth.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Create health checker for dependency verification.
	checker := bmhealth.NewChecker(sub)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		sig := <-sigCh
		logger.Info("received shutdown signal", "signal", sig)

		// Mark as NOT_SERVING before stopping.
		hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

		// Stop all running browser instances
		ctx := context.Background()
		if err := launcher.StopAll(ctx); err != nil {
			logger.Error("error stopping browser instances during shutdown", "error", err)
		}

		// Close NATS subscription
		sub.Close()
		os.Exit(0)
	}()

	// Subscribe to workspace events
	ctx := context.Background()
	if err := sub.SubscribeAll(ctx); err != nil {
		log.Fatalf("failed to subscribe to NATS events: %v", err)
	}

	// All dependencies connected — mark as SERVING.
	if checker.Check(context.Background()) == grpc_health_v1.HealthCheckResponse_SERVING {
		hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	} else {
		log.Println("warning: health check failed at startup, reporting NOT_SERVING")
	}

	logger.Info("browser-manager listening", "address", listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}