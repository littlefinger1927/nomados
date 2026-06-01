package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/observability/internal/aggregator"
	"github.com/nomados/nomados/services/observability/internal/subscriber"
)

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	listenAddr := os.Getenv("OBSERVABILITY_ADDR")
	if listenAddr == "" {
		listenAddr = ":9090"
	}

	logger := logging.NewLogger("observability", nil)

	// Create aggregator
	agg := aggregator.NewAggregator()

	// Connect to NATS and create subscriber
	sub, err := subscriber.NewSubscriber(natsURL, agg, logger)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}

	// Subscribe to all event subjects
	ctx := context.Background()
	if err := sub.SubscribeAll(ctx); err != nil {
		log.Fatalf("failed to subscribe to NATS events: %v", err)
	}

	// Set up HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", agg.PrometheusHandler())
	mux.HandleFunc("/health", agg.HealthHandler())

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server in a goroutine.
	go func() {
		logger.Info("observability service listening", "address", listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("received %s, shutting down...", sig)

	// Shutdown with timeout (15s for slow operations).
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("error shutting down HTTP server: %v", err)
	}

	// Close resources explicitly.
	log.Println("closing NATS subscription...")
	sub.Close()

	log.Println("shutdown complete")
}