package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/streaming-service/internal/health"
	"github.com/nomados/nomados/services/streaming-service/internal/nats"
	"github.com/nomados/nomados/services/streaming-service/internal/relay"
	"github.com/nomados/nomados/services/streaming-service/internal/turn"
	"github.com/nomados/nomados/services/streaming-service/internal/webrtc"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	listenAddr := os.Getenv("STREAMING_SERVICE_ADDR")
	if listenAddr == "" {
		listenAddr = ":50055"
	}

	// Create logger
	logger := logging.NewLogger("streaming-service", nil)

	// Initialize TURN relay with default config (Coturn dev credentials)
	turnConfig := turn.DefaultConfig()
	turnRelay := turn.NewTURNRelay(turnConfig)

	// Create stream relay
	streamRelay := relay.NewStreamRelay(logger)

	// Create signaling server
	_ = webrtc.NewSignalingServer(turnRelay, logger)

	// Connect to NATS and subscribe to workspace events
	sub, err := nats.NewSubscriber(natsURL, streamRelay, logger)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer sub.Close()

	ctx := context.Background()
	if err := sub.SubscribeAll(ctx); err != nil {
		log.Fatalf("failed to subscribe to NATS events: %v", err)
	}

	// Start gRPC server
	grpcServer := grpc.NewServer()

	// Register gRPC health server.
	hs := grpchealth.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Create health checker for dependency verification.
	checker := health.NewChecker(sub)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Set up graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("received shutdown signal", "signal", sig)

		// Mark as NOT_SERVING before stopping.
		hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

		// Close all active streams
		streamRelay.CloseAll()

		// Close NATS subscription
		sub.Close()

		// Stop gRPC server
		grpcServer.GracefulStop()
	}()

	// All dependencies connected — mark as SERVING.
	if checker.Check(context.Background()) == grpc_health_v1.HealthCheckResponse_SERVING {
		hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	} else {
		log.Println("warning: health check failed at startup, reporting NOT_SERVING")
	}

	logger.Info("streaming-service listening", "address", listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}