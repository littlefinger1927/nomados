package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	filev1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/file/v1"
	"github.com/nomados/nomados/services/file-service/internal/handler"
	"github.com/nomados/nomados/services/file-service/internal/health"
	"github.com/nomados/nomados/services/file-service/internal/repository"
	"github.com/nomados/nomados/services/file-service/internal/storage"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	// Database configuration.
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://nomados:nomados_dev@localhost:5432/nomados?sslmode=disable"
	}

	// MinIO configuration.
	minioCfg := storage.MinIOConfig{
		Endpoint:  os.Getenv("MINIO_ENDPOINT"),
		AccessKey: os.Getenv("MINIO_ACCESS_KEY"),
		SecretKey: os.Getenv("MINIO_SECRET_KEY"),
		Bucket:    os.Getenv("MINIO_BUCKET"),
		UseSSL:    os.Getenv("MINIO_USE_SSL") == "true",
	}
	if minioCfg.Endpoint == "" {
		minioCfg.Endpoint = "localhost:9000"
	}
	if minioCfg.AccessKey == "" {
		minioCfg.AccessKey = "nomados"
	}
	if minioCfg.SecretKey == "" {
		minioCfg.SecretKey = "nomados_dev_key"
	}
	if minioCfg.Bucket == "" {
		minioCfg.Bucket = "nomados-files"
	}

	// gRPC server address.
	listenAddr := os.Getenv("FILE_SERVICE_ADDR")
	if listenAddr == "" {
		listenAddr = ":50056"
	}

	ctx := context.Background()

	// Connect to PostgreSQL.
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("connected to PostgreSQL")

	// Connect to MinIO.
	store, err := storage.NewMinIOStorage(minioCfg)
	if err != nil {
		log.Fatalf("failed to create MinIO storage client: %v", err)
	}

	// Ensure the bucket exists.
	if err := store.EnsureBucket(ctx); err != nil {
		log.Fatalf("failed to ensure bucket: %v", err)
	}
	log.Println("MinIO bucket ensured")

	// Initialize repository and handler.
	repo := repository.NewPostgresRepository(pool)
	fileHandler := handler.NewFileServiceHandler(store, repo)

	// Create gRPC adapter that wraps the handler to implement FileServiceServer.
	grpcAdapter := handler.NewFileServiceGRPCAdapter(fileHandler)

	// Set up gRPC server.
	grpcServer := grpc.NewServer()
	filev1.RegisterFileServiceServer(grpcServer, grpcAdapter)

	// Register gRPC health server.
	hs := grpchealth.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Create health checker for dependency verification.
	checker := health.NewChecker(pool, store)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Graceful shutdown.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		// Mark as NOT_SERVING before stopping.
		hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		log.Println("shutting down file-service...")
		grpcServer.GracefulStop()
	}()

	// All dependencies connected — mark as SERVING.
	if checker.Check(context.Background()) == grpc_health_v1.HealthCheckResponse_SERVING {
		hs.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	} else {
		log.Println("warning: health check failed at startup, reporting NOT_SERVING")
	}

	log.Printf("file-service listening on %s", listenAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}