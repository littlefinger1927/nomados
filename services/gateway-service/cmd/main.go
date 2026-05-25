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

	"github.com/nomados/nomados/services/gateway-service/internal/config"
	"github.com/nomados/nomados/services/gateway-service/internal/middleware"
	"github.com/nomados/nomados/services/gateway-service/internal/proxy"
)

func main() {
	cfg := config.Load()

	validator := authsdk.NewTokenValidator(cfg.SigningSecret)
	rateLimiter := middleware.NewIPRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
	router := proxy.NewRouter(cfg)

	// Wire middleware chain: rate limit -> auth -> proxy
	handler := middleware.RateLimitMiddleware(rateLimiter, middleware.AuthMiddleware(validator, router))

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:     handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("gateway service starting on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("received signal %v, shutting down gracefully...", sig)

	// Give outstanding requests 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "server forced to shutdown: %v\n", err)
	}

	log.Println("gateway service stopped")
}