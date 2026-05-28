package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authsdk "github.com/nomados/nomados/packages/auth-sdk"

	authv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/auth/v1"
	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/session/v1"
	workspacev1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/workspace/v1"
	filev1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/file/v1"
	vaultv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/vault/v1"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/nomados/nomados/services/gateway-service/internal/config"
	gwhealth "github.com/nomados/nomados/services/gateway-service/internal/health"
	"github.com/nomados/nomados/services/gateway-service/internal/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()

	// Create gRPC client connections to backend services.
	authConn, err := grpc.NewClient(cfg.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to auth service: %v", err)
	}
	defer authConn.Close()

	sessionConn, err := grpc.NewClient(cfg.SessionAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to session service: %v", err)
	}
	defer sessionConn.Close()

	workspaceConn, err := grpc.NewClient(cfg.WorkspaceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to workspace service: %v", err)
	}
	defer workspaceConn.Close()

	fileConn, err := grpc.NewClient(cfg.FileAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to file service: %v", err)
	}
	defer fileConn.Close()

	vaultConn, err := grpc.NewClient(cfg.VaultAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to vault service: %v", err)
	}
	defer vaultConn.Close()

	// Build grpc-gateway mux.
	mux := runtime.NewServeMux()

	// Register each service handler from endpoint using the gRPC connections.
	if err := authv1.RegisterAuthServiceHandler(ctx, mux, authConn); err != nil {
		log.Fatalf("failed to register auth handler: %v", err)
	}
	if err := sessionv1.RegisterSessionServiceHandler(ctx, mux, sessionConn); err != nil {
		log.Fatalf("failed to register session handler: %v", err)
	}
	if err := workspacev1.RegisterWorkspaceServiceHandler(ctx, mux, workspaceConn); err != nil {
		log.Fatalf("failed to register workspace handler: %v", err)
	}
	if err := filev1.RegisterFileServiceHandler(ctx, mux, fileConn); err != nil {
		log.Fatalf("failed to register file handler: %v", err)
	}
	if err := vaultv1.RegisterVaultServiceHandler(ctx, mux, vaultConn); err != nil {
		log.Fatalf("failed to register vault handler: %v", err)
	}

	// Set up health checker for downstream services.
	healthChecker := gwhealth.NewChecker([]gwhealth.BackendConfig{
		{Name: "auth", Addr: cfg.AuthAddr},
		{Name: "session", Addr: cfg.SessionAddr},
		{Name: "workspace", Addr: cfg.WorkspaceAddr},
		{Name: "file", Addr: cfg.FileAddr},
		{Name: "vault", Addr: cfg.VaultAddr},
	})

	// Set up middleware.
	validator := authsdk.NewTokenValidator(cfg.SigningSecret)
	rateLimiter := middleware.NewIPRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	// Wire middleware chain: rate limit -> auth -> grpc-gateway mux.
	handler := middleware.RateLimitMiddleware(rateLimiter, middleware.AuthMiddleware(validator, mux))

	// Add /health endpoint.
	httpHandler := http.NewServeMux()
	httpHandler.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := healthChecker.Check(r.Context())
		if status == "SERVING" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"status":"SERVING"}`)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, `{"status":"NOT_SERVING"}`)
		}
	}))
	httpHandler.Handle("/", handler)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine.
	go func() {
		log.Printf("gateway service starting on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for interrupt signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("received signal %v, shutting down gracefully...", sig)

	// Give outstanding requests 10 seconds to complete.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "server forced to shutdown: %v\n", err)
	}

	log.Println("gateway service stopped")
}