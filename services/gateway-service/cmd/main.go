package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	authsdk "github.com/nomados/nomados/packages/auth-sdk"
	"github.com/nomados/nomados/packages/logging"

	authv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/auth/v1"
	browserv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/browser_manager/v1"
	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/session/v1"
	streamingv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/streaming/v1"
	workspacev1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/workspace/v1"
	filev1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/file/v1"
	vaultv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/vault/v1"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/nomados/nomados/services/gateway-service/internal/config"
	gwhealth "github.com/nomados/nomados/services/gateway-service/internal/health"
	"github.com/nomados/nomados/services/gateway-service/internal/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load()

	logger := logging.NewLogger("gateway-service", nil)

	ctx := context.Background()

	// Determine gRPC transport credentials based on TLS config.
	var transportCreds credentials.TransportCredentials
	if cfg.TLSBackendCertPath != "" {
		creds, err := credentials.NewClientTLSFromFile(cfg.TLSBackendCertPath, "")
		if err != nil {
			logger.Error("failed to load TLS backend cert", "error", err)
			os.Exit(1)
		}
		transportCreds = creds
		logger.Info("using TLS for backend connections", "cert_path", cfg.TLSBackendCertPath)
	} else {
		transportCreds = insecure.NewCredentials()
	}

	// Create gRPC client connections to backend services.
	authConn, err := grpc.NewClient(cfg.AuthAddr, grpc.WithTransportCredentials(transportCreds))
	if err != nil {
		logger.Error("failed to connect to auth service", "error", err)
		os.Exit(1)
	}

	sessionConn, err := grpc.NewClient(cfg.SessionAddr, grpc.WithTransportCredentials(transportCreds))
	if err != nil {
		logger.Error("failed to connect to session service", "error", err)
		os.Exit(1)
	}

	workspaceConn, err := grpc.NewClient(cfg.WorkspaceAddr, grpc.WithTransportCredentials(transportCreds))
	if err != nil {
		logger.Error("failed to connect to workspace service", "error", err)
		os.Exit(1)
	}

	fileConn, err := grpc.NewClient(cfg.FileAddr, grpc.WithTransportCredentials(transportCreds))
	if err != nil {
		logger.Error("failed to connect to file service", "error", err)
		os.Exit(1)
	}

	vaultConn, err := grpc.NewClient(cfg.VaultAddr, grpc.WithTransportCredentials(transportCreds))
	if err != nil {
		logger.Error("failed to connect to vault service", "error", err)
		os.Exit(1)
	}

	browserConn, err := grpc.NewClient(cfg.BrowserAddr, grpc.WithTransportCredentials(transportCreds))
	if err != nil {
		logger.Error("failed to connect to browser-manager service", "error", err)
		os.Exit(1)
	}

	streamingConn, err := grpc.NewClient(cfg.StreamingAddr, grpc.WithTransportCredentials(transportCreds))
	if err != nil {
		logger.Error("failed to connect to streaming service", "error", err)
		os.Exit(1)
	}

	// Build grpc-gateway mux.
	mux := runtime.NewServeMux()

	// Register each service handler from endpoint using the gRPC connections.
	if err := authv1.RegisterAuthServiceHandler(ctx, mux, authConn); err != nil {
		logger.Error("failed to register auth handler", "error", err)
		os.Exit(1)
	}
	if err := sessionv1.RegisterSessionServiceHandler(ctx, mux, sessionConn); err != nil {
		logger.Error("failed to register session handler", "error", err)
		os.Exit(1)
	}
	if err := workspacev1.RegisterWorkspaceServiceHandler(ctx, mux, workspaceConn); err != nil {
		logger.Error("failed to register workspace handler", "error", err)
		os.Exit(1)
	}
	if err := filev1.RegisterFileServiceHandler(ctx, mux, fileConn); err != nil {
		logger.Error("failed to register file handler", "error", err)
		os.Exit(1)
	}
	if err := vaultv1.RegisterVaultServiceHandler(ctx, mux, vaultConn); err != nil {
		logger.Error("failed to register vault handler", "error", err)
		os.Exit(1)
	}
	if err := browserv1.RegisterBrowserManagerServiceHandler(ctx, mux, browserConn); err != nil {
		logger.Error("failed to register browser-manager handler", "error", err)
		os.Exit(1)
	}
	if err := streamingv1.RegisterStreamingServiceHandler(ctx, mux, streamingConn); err != nil {
		logger.Error("failed to register streaming handler", "error", err)
		os.Exit(1)
	}

	// Set up health checker for downstream services.
	healthChecker := gwhealth.NewChecker([]gwhealth.BackendConfig{
		{Name: "auth", Addr: cfg.AuthAddr},
		{Name: "session", Addr: cfg.SessionAddr},
		{Name: "workspace", Addr: cfg.WorkspaceAddr},
		{Name: "browser", Addr: cfg.BrowserAddr},
		{Name: "streaming", Addr: cfg.StreamingAddr},
		{Name: "file", Addr: cfg.FileAddr},
		{Name: "vault", Addr: cfg.VaultAddr},
	})

	// Track serving state for the health endpoint.
	var serving atomic.Bool
	serving.Store(false)

	// Set up middleware.
	validator := authsdk.NewTokenValidator(cfg.SigningSecret)
	rateLimiter := middleware.NewIPRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	// Wire middleware chain: logging -> rate limit -> request ID -> CORS -> auth -> grpc-gateway mux.
	corsConfig := middleware.CORSConfig{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:  []string{"Content-Type", "Authorization", "X-Requested-With"},
	}

	// Request logging middleware injects request context into structured logs.
	requestLogger := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqLogger := logger.
				WithRequestID(middleware.GetRequestID(r.Context())).
				WithMethod(r.Method).
				WithPath(r.URL.Path)
			reqLogger.Info("request received")
			next.ServeHTTP(w, r)
		})
	}

	handler := middleware.RateLimitMiddleware(rateLimiter, middleware.RequestIDMiddleware(requestLogger(middleware.CORSMiddleware(corsConfig, middleware.AuthMiddleware(validator, mux)))))

	// Add /health endpoint with SERVING/NOT_SERVING lifecycle.
	httpHandler := http.NewServeMux()
	httpHandler.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !serving.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, `{"status":"NOT_SERVING"}`)
			return
		}
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
		logger.Info("gateway service starting", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// All dependencies connected — mark as SERVING.
	serving.Store(true)
	log.Println("gateway service is SERVING")

	// Wait for interrupt signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("received %s, shutting down...", sig)

	// Mark as NOT_SERVING before stopping.
	serving.Store(false)

	// Shutdown with timeout (15s for slow operations).
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}

	// Close resources explicitly.
	log.Println("closing auth service connection...")
	authConn.Close()
	log.Println("closing session service connection...")
	sessionConn.Close()
	log.Println("closing workspace service connection...")
	workspaceConn.Close()
	log.Println("closing file service connection...")
	fileConn.Close()
	log.Println("closing vault service connection...")
	vaultConn.Close()
	log.Println("closing browser-manager service connection...")
	browserConn.Close()
	log.Println("closing streaming service connection...")
	streamingConn.Close()

	log.Println("shutdown complete")
}