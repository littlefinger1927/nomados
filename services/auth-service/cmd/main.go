package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	authv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/auth/v1"
	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/session/v1"
	"github.com/nomados/nomados/services/auth-service/internal/handler"
	"github.com/nomados/nomados/services/auth-service/internal/repository"
	"github.com/nomados/nomados/services/auth-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	log.Printf("auth-service listening on %s", listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}