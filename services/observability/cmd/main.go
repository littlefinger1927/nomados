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
	defer sub.Close()

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

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Info("received shutdown signal, draining connections", "signal", sig)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("error shutting down HTTP server", "error", err)
		}

		sub.Close()
		os.Exit(0)
	}()

	logger.Info("observability service listening", "address", listenAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}