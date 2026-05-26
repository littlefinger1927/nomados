package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	vaultv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/vault/v1"
	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/vault-service/internal/handler"
	vaultnats "github.com/nomados/nomados/services/vault-service/internal/nats"
	"github.com/nomados/nomados/services/vault-service/internal/keyderivation"
	"google.golang.org/grpc"
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

	// Initialize logger.
	logger := logging.NewLogger("vault-service", nil)

	// Initialize key deriver with default Argon2id parameters.
	kd := keyderivation.NewKeyDeriver()

	// Connect to NATS for publishing key rotation events.
	publisher, err := vaultnats.NewPublisher(natsURL)
	if err != nil {
		log.Printf("warning: failed to connect to NATS at %s: %v (rotation events will not be published)", natsURL, err)
		// Continue without NATS — rotation events will be logged but not published.
		publisher = nil
	} else {
		log.Println("connected to NATS")
	}
	if publisher != nil {
		defer publisher.Close()
	}

	// Initialize handler and gRPC adapter.
	vaultHandler := handler.NewVaultServiceHandler(kd, publisher, logger)
	grpcAdapter := handler.NewVaultServiceGRPCAdapter(vaultHandler)

	// Set up gRPC server.
	grpcServer := grpc.NewServer()
	vaultv1.RegisterVaultServiceServer(grpcServer, grpcAdapter)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Graceful shutdown.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down vault-service...")
		grpcServer.GracefulStop()
	}()

	logger.Info("vault-service starting", "addr", listenAddr)
	log.Printf("vault-service listening on %s", listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}