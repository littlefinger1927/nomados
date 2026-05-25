package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/session/v1"
	"github.com/nomados/nomados/services/session-service/internal/handler"
	"github.com/nomados/nomados/services/session-service/internal/nats"
	"github.com/nomados/nomados/services/session-service/internal/repository"
	"github.com/nomados/nomados/services/session-service/internal/service"
	"google.golang.org/grpc"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://nomados:nomados_dev@localhost:5432/nomados?sslmode=disable"
	}
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	listenAddr := os.Getenv("SESSION_SERVICE_ADDR")
	if listenAddr == "" {
		listenAddr = ":50052"
	}
	signingSecret := os.Getenv("SIGNING_SECRET")
	if signingSecret == "" {
		signingSecret = "dev-secret-change-me"
	}

	ctx := context.Background()

	// Connect to PostgreSQL
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// Connect to NATS
	pub, err := nats.NewPublisher(natsURL)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer pub.Close()

	// Create services
	repo := repository.NewPostgresRepository(pool)
	svc := service.NewSessionService(repo, pub, signingSecret)
	sessionHandler := handler.NewSessionServiceHandler(svc)

	// Start gRPC server
	grpcServer := grpc.NewServer()
	sessionv1.RegisterSessionServiceServer(grpcServer, sessionHandler)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		grpcServer.GracefulStop()
	}()

	log.Printf("session-service listening on %s", listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}