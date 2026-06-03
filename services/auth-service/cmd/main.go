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
	authv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/auth/v1"
	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/session/v1"
	"github.com/nomados/nomados/services/auth-service/internal/handler"
	"github.com/nomados/nomados/services/auth-service/internal/health"
	"github.com/nomados/nomados/services/auth-service/internal/repository"
	"github.com/nomados/nomados/services/auth-service/internal/service"
	"github.com/redis/go-redis/v9"
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
	sessionAddr := os.Getenv("SESSION_SERVICE_CLIENT_ADDR")
	if sessionAddr == "" {
		sessionAddr = os.Getenv("SESSION_SERVICE_ADDR")
		if sessionAddr == "" {
			sessionAddr = "localhost:50052"
		}
	}
	redisURL := os.Getenv("REDIS_URL")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// Connect to session-service for real token generation.
	sessionConn, err := grpc.NewClient(sessionAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to session service at %s: %v", sessionAddr, err)
	}
	sessionClient := sessionv1.NewSessionServiceClient(sessionConn)

	// Set up challenge store: Redis if REDIS_URL is provided, otherwise in-memory.
	var challengeStore service.ChallengeStore
	if redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Printf("WARNING: failed to parse REDIS_URL (%s): %v; falling back to in-memory challenge store", redisURL, err)
			challengeStore = service.NewMemoryChallengeStore()
		} else {
			redisClient := redis.NewClient(opts)
			if err := redisClient.Ping(ctx).Err(); err != nil {
				log.Printf("WARNING: failed to connect to Redis at %s: %v; falling back to in-memory challenge store", redisURL, err)
				challengeStore = service.NewMemoryChallengeStore()
			} else {
				challengeStore = service.NewRedisChallengeStore(redisClient)
				log.Printf("using Redis challenge store at %s", redisURL)
			}
		}
	} else {
		challengeStore = service.NewMemoryChallengeStore()
		log.Printf("using in-memory challenge store (set REDIS_URL for Redis-backed store)")
	}

	repo := repository.NewPostgresRepository(pool)
	svc := service.NewAuthService(repo, challengeStore)
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

	log.Printf("auth-service listening on %s", listenAddr)

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
	log.Println("closing session service connection...")
	sessionConn.Close()
	log.Println("closing challenge store...")
	challengeStore.Close()
	log.Println("closing database connection...")
	pool.Close()

	log.Println("shutdown complete")
}
