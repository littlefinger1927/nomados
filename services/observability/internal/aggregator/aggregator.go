package aggregator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Aggregator tracks metrics from NATS events and exposes them
// in Prometheus format.
type Aggregator struct {
	activeSessions     prometheus.Gauge
	activeWorkspaces   prometheus.Gauge
	authRequestsTotal  prometheus.Counter
	workspaceOpsTotal  *prometheus.CounterVec
	errorsTotal        *prometheus.CounterVec
	registry           *prometheus.Registry
}

// NewAggregator creates a new Aggregator with Prometheus metrics registered
// in an isolated registry (does not pollute the default registry).
func NewAggregator() *Aggregator {
	reg := prometheus.NewRegistry()

	activeSessions := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nomados_active_sessions",
		Help: "Current number of active sessions",
	})
	activeWorkspaces := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nomados_active_workspaces",
		Help: "Current number of active workspaces",
	})
	authRequestsTotal := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nomados_auth_requests_total",
		Help: "Total number of authentication requests processed",
	})
	workspaceOpsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nomados_workspace_operations_total",
		Help: "Total number of workspace operations by type",
	}, []string{"operation"})
	errorsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nomados_errors_total",
		Help: "Total number of errors by service",
	}, []string{"service"})

	reg.MustRegister(activeSessions)
	reg.MustRegister(activeWorkspaces)
	reg.MustRegister(authRequestsTotal)
	reg.MustRegister(workspaceOpsTotal)
	reg.MustRegister(errorsTotal)

	return &Aggregator{
		activeSessions:    activeSessions,
		activeWorkspaces:  activeWorkspaces,
		authRequestsTotal: authRequestsTotal,
		workspaceOpsTotal: workspaceOpsTotal,
		errorsTotal:       errorsTotal,
		registry:          reg,
	}
}

// ProcessEvent updates metrics based on the NATS subject and event data.
// It parses the subject to determine the event category and increments
// the appropriate counters or adjusts gauges.
func (a *Aggregator) ProcessEvent(subject string, data []byte) {
	parts := strings.SplitN(subject, ".", 2)
	if len(parts) < 2 {
		return
	}
	category := parts[0]
	action := parts[1]

	switch category {
	case "audit":
		a.authRequestsTotal.Inc()
	case "session":
		a.processSessionEvent(action, data)
	case "workspace":
		a.processWorkspaceEvent(action, data)
	case "vault":
		// Vault events are logged but no specific metric beyond errors
	}
}

// processSessionEvent handles session.* events.
func (a *Aggregator) processSessionEvent(action string, data []byte) {
	switch action {
	case "created":
		a.activeSessions.Inc()
	case "revoked":
		a.activeSessions.Dec()
		// "refreshed" does not change the gauge count
	}
}

// processWorkspaceEvent handles workspace.* events.
func (a *Aggregator) processWorkspaceEvent(action string, data []byte) {
	switch action {
	case "created":
		a.activeWorkspaces.Inc()
		a.workspaceOpsTotal.WithLabelValues("created").Inc()
	case "paused":
		a.workspaceOpsTotal.WithLabelValues("paused").Inc()
	case "resumed":
		a.workspaceOpsTotal.WithLabelValues("resumed").Inc()
	case "stopped":
		a.activeWorkspaces.Dec()
		a.workspaceOpsTotal.WithLabelValues("stopped").Inc()
	case "destroyed":
		a.workspaceOpsTotal.WithLabelValues("destroyed").Inc()
	}
}

// IncrementErrors increments the error counter for a given service.
func (a *Aggregator) IncrementErrors(service string) {
	a.errorsTotal.WithLabelValues(service).Inc()
}

// SetActiveSessions sets the gauge to an absolute value (useful for testing).
func (a *Aggregator) SetActiveSessions(count float64) {
	a.activeSessions.Set(count)
}

// SetActiveWorkspaces sets the gauge to an absolute value (useful for testing).
func (a *Aggregator) SetActiveWorkspaces(count float64) {
	a.activeWorkspaces.Set(count)
}

// PrometheusHandler returns an HTTP handler that serves Prometheus-format metrics.
func (a *Aggregator) PrometheusHandler() http.Handler {
	return promhttp.HandlerFor(a.registry, promhttp.HandlerOpts{})
}

// HealthHandler returns an HTTP handler for the /health endpoint.
func (a *Aggregator) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	}
}

// Reset resets all counters and gauges. This is primarily for testing.
func (a *Aggregator) Reset() {
	a.activeSessions.Set(0)
	a.activeWorkspaces.Set(0)
	// Note: Prometheus counters cannot be reset once incremented.
	// For testing, create a new Aggregator instead.
}

// Registry returns the underlying Prometheus registry (for advanced use).
func (a *Aggregator) Registry() *prometheus.Registry {
	return a.registry
}

// EventData is a generic event data structure used for parsing incoming NATS messages.
type EventData struct {
	Subject string          `json:"subject"`
	Data    json.RawMessage `json:"data"`
	Source  string          `json:"source,omitempty"`
}