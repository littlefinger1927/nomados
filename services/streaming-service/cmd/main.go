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
	streamingv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/streaming/v1"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nomados/nomados/services/streaming-service/internal/handler"
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

	// Connect to NATS for signaling and workspace events
	natsConn, err := natsgo.Connect(natsURL)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}

	// Create signaling server with NATS connection
	signaling := webrtc.NewSignalingServer(turnRelay, natsConn, logger)

	// Subscribe to workspace events via NATS
	sub, err := nats.NewSubscriber(natsURL, streamRelay, logger)
	if err != nil {
		log.Fatalf("failed to connect to NATS for subscriptions: %v", err)
	}

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

	// Register the streaming service
	grpcAdapter := handler.NewStreamingServiceGRPCAdapter(signaling)
	streamingv1.RegisterStreamingServiceServer(grpcServer, grpcAdapter)

	// Create health checker for dependency verification.
	checker := health.NewChecker(sub)

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

	logger.Info("streaming-service listening", "address", listenAddr)

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

	// Close all active streams
	streamRelay.CloseAll()

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
	log.Println("closing NATS subscription...")
	sub.Close()
	log.Println("closing NATS connection...")
	natsConn.Close()

	log.Println("shutdown complete")
}