package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/streaming-service/internal/handler"
	"github.com/nomados/nomados/services/streaming-service/internal/nats"
	"github.com/nomados/nomados/services/streaming-service/internal/relay"
	"github.com/nomados/nomados/services/streaming-service/internal/turn"
	"github.com/nomados/nomados/services/streaming-service/internal/webrtc"
	streamingv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/streaming/v1"
	natsgo "github.com/nats-io/nats.go"
	"google.golang.org/grpc"
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
	defer natsConn.Close()

	// Create signaling server with NATS connection
	signaling := webrtc.NewSignalingServer(turnRelay, natsConn, logger)

	// Subscribe to workspace events via NATS
	sub, err := nats.NewSubscriber(natsURL, streamRelay, logger)
	if err != nil {
		log.Fatalf("failed to connect to NATS for subscriptions: %v", err)
	}
	defer sub.Close()

	ctx := context.Background()
	if err := sub.SubscribeAll(ctx); err != nil {
		log.Fatalf("failed to subscribe to NATS events: %v", err)
	}

	// Start gRPC server
	grpcServer := grpc.NewServer()

	// Register the streaming service
	grpcAdapter := handler.NewStreamingServiceGRPCAdapter(signaling)
	streamingv1.RegisterStreamingServiceServer(grpcServer, grpcAdapter)

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

		// Close all active streams
		streamRelay.CloseAll()

		// Close NATS subscription
		sub.Close()

		// Stop gRPC server
		grpcServer.GracefulStop()
	}()

	logger.Info("streaming-service listening", "address", listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}