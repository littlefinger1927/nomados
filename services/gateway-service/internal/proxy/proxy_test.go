package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nomados/nomados/services/gateway-service/internal/config"
)

func TestNewRouter(t *testing.T) {
	cfg := &config.Config{
		AuthAddr:      "localhost:50051",
		SessionAddr:   "localhost:50052",
		WorkspaceAddr: "localhost:50053",
		BrowserAddr:   "localhost:50054",
		StreamingAddr: "localhost:50055",
		FileAddr:      "localhost:50056",
		VaultAddr:     "localhost:50057",
	}

	router := NewRouter(cfg)
	if len(router.routes) != 7 {
		t.Errorf("expected 7 routes, got %d", len(router.routes))
	}

	expectedPrefixes := []string{"/auth", "/session", "/workspace", "/browser", "/stream", "/file", "/vault"}
	for i, prefix := range expectedPrefixes {
		if router.routes[i].Prefix != prefix {
			t.Errorf("route %d: expected prefix %s, got %s", i, prefix, router.routes[i].Prefix)
		}
	}
}

func TestRouter_UnknownPath(t *testing.T) {
	cfg := &config.Config{
		AuthAddr:      "localhost:50051",
		SessionAddr:   "localhost:50052",
		WorkspaceAddr: "localhost:50053",
		BrowserAddr:   "localhost:50054",
		StreamingAddr: "localhost:50055",
		FileAddr:      "localhost:50056",
		VaultAddr:     "localhost:50057",
	}

	router := NewRouter(cfg)
	req := httptest.NewRequest("GET", "/unknown/path", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown path, got %d", rec.Code)
	}
}

func TestRouter_RoutesToBackend(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("auth response"))
	}))
	defer backend.Close()

	// Parse the backend address from the test server URL
	cfg := &config.Config{
		AuthAddr:      backend.Listener.Addr().String(),
		SessionAddr:   "localhost:50052",
		WorkspaceAddr: "localhost:50053",
		BrowserAddr:   "localhost:50054",
		StreamingAddr: "localhost:50055",
		FileAddr:      "localhost:50056",
		VaultAddr:     "localhost:50057",
	}

	router := NewRouter(cfg)
	req := httptest.NewRequest("GET", "/auth/login", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRouter_ExactPrefixMatch(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	cfg := &config.Config{
		AuthAddr:      backend.Listener.Addr().String(),
		SessionAddr:   "localhost:50052",
		WorkspaceAddr: "localhost:50053",
		BrowserAddr:   "localhost:50054",
		StreamingAddr: "localhost:50055",
		FileAddr:      "localhost:50056",
		VaultAddr:     "localhost:50057",
	}

	router := NewRouter(cfg)

	// Exact prefix match without trailing slash
	req := httptest.NewRequest("GET", "/auth", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for exact prefix /auth, got %d", rec.Code)
	}

	// Prefix with subpath
	req2 := httptest.NewRequest("GET", "/auth/login", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200 for /auth/login, got %d", rec2.Code)
	}
}