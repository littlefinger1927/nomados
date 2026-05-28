package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	authv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/auth/v1"
	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/session/v1"
	"github.com/nomados/nomados/services/auth-service/internal/handler"
	"github.com/nomados/nomados/services/auth-service/internal/health"
	"github.com/nomados/nomados/services/auth-service/internal/repository"
	"github.com/nomados/nomados/services/auth-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://nomados:nomados_dev@localhost:5432/nomados?sslmode=disable"
	}
	listenAddr := os.Getenv("AUTH_SERVICE_ADDR")
	if listenAddr == "" {
		listenAddr = ":50051"
	}
	sessionAddr := os.Getenv("SESSION_SERVICE_ADDR")
	if sessionAddr == "" {
		sessionAddr = "localhost:50052"
	}
	redisURL := os.Getenv("REDIS_URL")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// Connect to session-service for real token generation.
	sessionConn, err := grpc.NewClient(sessionAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to session service at %s: %v", sessionAddr, err)
	}
	defer sessionConn.Close()

	sessionClient := sessionv1.NewSessionServiceClient(sessionConn)

	repo := repository.NewPostgresRepository(pool)
	svc := service.NewAuthService(repo)
	authHandler := handler.NewAuthServiceHandler(svc, sessionClient)

	grpcServer := grpc.NewServer()
	authv1.RegisterAuthServiceServer(grpcServer, authHandler)

	// Register gRPC health server.
	hs := grpchealth.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Create health checker for dependency verification.
	checker := health.NewChecker(pool, health.WithRedisURL(redisURL))

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

	log.Printf("auth-service listening on %s", listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}