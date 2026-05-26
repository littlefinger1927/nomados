package aggregator

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAggregator(t *testing.T) {
	agg := NewAggregator()
	if agg == nil {
		t.Fatal("expected non-nil aggregator")
	}
}

func TestProcessEventAudit(t *testing.T) {
	agg := NewAggregator()
	agg.ProcessEvent("audit.login", []byte(`{"user_id":"u1"}`))

	// Verify the auth counter was incremented by checking the Prometheus output
	handler := agg.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "nomados_auth_requests_total") {
		t.Error("expected nomados_auth_requests_total in metrics output")
	}
}

func TestProcessEventSessionCreated(t *testing.T) {
	agg := NewAggregator()
	agg.ProcessEvent("session.created", []byte(`{"session_id":"s1"}`))

	handler := agg.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "nomados_active_sessions 1") {
		t.Errorf("expected nomados_active_sessions 1, got:\n%s", body)
	}
}

func TestProcessEventSessionRevoked(t *testing.T) {
	agg := NewAggregator()
	agg.ProcessEvent("session.created", []byte(`{}`))
	agg.ProcessEvent("session.created", []byte(`{}`))
	agg.ProcessEvent("session.revoked", []byte(`{}`))

	handler := agg.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "nomados_active_sessions 1") {
		t.Errorf("expected nomados_active_sessions 1, got:\n%s", body)
	}
}

func TestProcessEventWorkspaceCreated(t *testing.T) {
	agg := NewAggregator()
	agg.ProcessEvent("workspace.created", []byte(`{"workspace_id":"w1"}`))

	handler := agg.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "nomados_active_workspaces 1") {
		t.Errorf("expected nomados_active_workspaces 1, got:\n%s", body)
	}
	if !strings.Contains(body, `nomados_workspace_operations_total{operation="created"}`) {
		t.Errorf("expected workspace_operations_total with operation=created, got:\n%s", body)
	}
}

func TestProcessEventWorkspaceLifecycle(t *testing.T) {
	agg := NewAggregator()

	// Create two workspaces
	agg.ProcessEvent("workspace.created", []byte(`{}`))
	agg.ProcessEvent("workspace.created", []byte(`{}`))

	// Pause one
	agg.ProcessEvent("workspace.paused", []byte(`{}`))
	// Resume it
	agg.ProcessEvent("workspace.resumed", []byte(`{}`))
	// Stop one
	agg.ProcessEvent("workspace.stopped", []byte(`{}`))
	// Destroy one
	agg.ProcessEvent("workspace.destroyed", []byte(`{}`))

	handler := agg.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// 2 created, 1 stopped => 1 active
	if !strings.Contains(body, "nomados_active_workspaces 1") {
		t.Errorf("expected 1 active workspace, got:\n%s", body)
	}

	// Check all operation counters
	ops := []string{"created", "paused", "resumed", "stopped", "destroyed"}
	for _, op := range ops {
		label := `nomados_workspace_operations_total{operation="` + op + `"}`
		if !strings.Contains(body, label) {
			t.Errorf("expected %s in output", label)
		}
	}
}

func TestProcessEventVault(t *testing.T) {
	agg := NewAggregator()
	// Vault events should not panic and should not affect any gauge
	agg.ProcessEvent("vault.rotated", []byte(`{"key_id":"k1"}`))

	handler := agg.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	// Vault events should not increment any counter
	if strings.Contains(body, "nomados_active_sessions") && strings.Contains(body, "nomados_active_sessions 1") {
		t.Error("vault event should not change session gauge")
	}
}

func TestProcessEventInvalidSubject(t *testing.T) {
	agg := NewAggregator()
	// Invalid subjects should be ignored without panicking
	agg.ProcessEvent("invalid", []byte(`{}`))
	agg.ProcessEvent("", []byte(`{}`))
}

func TestIncrementErrors(t *testing.T) {
	agg := NewAggregator()
	agg.IncrementErrors("auth-service")
	agg.IncrementErrors("auth-service")
	agg.IncrementErrors("workspace-orchestrator")

	handler := agg.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `nomados_errors_total{service="auth-service"}`) {
		t.Errorf("expected errors_total for auth-service, got:\n%s", body)
	}
	if !strings.Contains(body, `nomados_errors_total{service="workspace-orchestrator"}`) {
		t.Errorf("expected errors_total for workspace-orchestrator, got:\n%s", body)
	}
}

func TestSetActiveSessions(t *testing.T) {
	agg := NewAggregator()
	agg.SetActiveSessions(5)

	handler := agg.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "nomados_active_sessions 5") {
		t.Errorf("expected 5 active sessions, got:\n%s", body)
	}
}

func TestSetActiveWorkspaces(t *testing.T) {
	agg := NewAggregator()
	agg.SetActiveWorkspaces(3)

	handler := agg.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "nomados_active_workspaces 3") {
		t.Errorf("expected 3 active workspaces, got:\n%s", body)
	}
}

func TestHealthHandler(t *testing.T) {
	agg := NewAggregator()
	handler := agg.HealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body != `{"status":"ok"}` {
		t.Errorf("expected health response, got: %s", body)
	}
}

func TestReset(t *testing.T) {
	agg := NewAggregator()
	agg.SetActiveSessions(5)
	agg.SetActiveWorkspaces(3)
	agg.Reset()

	handler := agg.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "nomados_active_sessions 0") {
		t.Errorf("expected 0 active sessions after reset, got:\n%s", body)
	}
	if !strings.Contains(body, "nomados_active_workspaces 0") {
		t.Errorf("expected 0 active workspaces after reset, got:\n%s", body)
	}
}

func TestPrometheusOutputFormat(t *testing.T) {
	agg := NewAggregator()
	agg.ProcessEvent("audit.login", []byte(`{}`))
	agg.ProcessEvent("session.created", []byte(`{}`))
	agg.ProcessEvent("workspace.created", []byte(`{}`))
	agg.IncrementErrors("test-service")

	handler := agg.PrometheusHandler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("expected text/plain content type, got %s", contentType)
	}

	body := rec.Body.String()
	expectedMetrics := []string{
		"nomados_active_sessions",
		"nomados_active_workspaces",
		"nomados_auth_requests_total",
		"nomados_workspace_operations_total",
		"nomados_errors_total",
	}
	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("expected metric %s in output", metric)
		}
	}
}