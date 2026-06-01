package logging

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestLoggerInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test-service", &buf)
	logger.Info("test message", "key", "value")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if entry["service"] != "test-service" {
		t.Errorf("expected service=test-service, got %v", entry["service"])
	}
	if entry["msg"] != "test message" {
		t.Errorf("expected msg='test message', got %v", entry["msg"])
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test-service", &buf)
	child := logger.With("request_id", "abc-123")
	child.Info("child message")

	var entry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &entry)
	if entry["request_id"] != "abc-123" {
		t.Errorf("expected request_id=abc-123, got %v", entry["request_id"])
	}
}

func TestLoggerWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test-service", &buf)
	child := logger.WithRequestID("req-456")
	child.Info("with request id")

	var entry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &entry)
	if entry["request_id"] != "req-456" {
		t.Errorf("expected request_id=req-456, got %v", entry["request_id"])
	}
}

func TestLoggerWithMethod(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test-service", &buf)
	child := logger.WithMethod("GET")
	child.Info("with method")

	var entry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &entry)
	if entry["method"] != "GET" {
		t.Errorf("expected method=GET, got %v", entry["method"])
	}
}

func TestLoggerWithPath(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test-service", &buf)
	child := logger.WithPath("/api/v1/sessions")
	child.Info("with path")

	var entry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &entry)
	if entry["path"] != "/api/v1/sessions" {
		t.Errorf("expected path=/api/v1/sessions, got %v", entry["path"])
	}
}

func TestLoggerWithImmutability(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test-service", &buf)

	// With* methods should return a new Logger without mutating the original.
	child1 := logger.WithRequestID("req-1")
	child2 := logger.WithRequestID("req-2")

	child1.Info("first")
	child2.Info("second")

	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	var first, second map[string]interface{}
	json.Unmarshal(lines[0], &first)
	json.Unmarshal(lines[1], &second)

	if first["request_id"] != "req-1" {
		t.Errorf("expected first request_id=req-1, got %v", first["request_id"])
	}
	if second["request_id"] != "req-2" {
		t.Errorf("expected second request_id=req-2, got %v", second["request_id"])
	}
}

func TestLoggerChainedWith(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test-service", &buf)

	// Chaining With* methods should accumulate all fields.
	child := logger.WithRequestID("req-1").WithMethod("POST").WithPath("/api/v1/workspaces")
	child.Info("chained request")

	var entry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &entry)
	if entry["request_id"] != "req-1" {
		t.Errorf("expected request_id=req-1, got %v", entry["request_id"])
	}
	if entry["method"] != "POST" {
		t.Errorf("expected method=POST, got %v", entry["method"])
	}
	if entry["path"] != "/api/v1/workspaces" {
		t.Errorf("expected path=/api/v1/workspaces, got %v", entry["path"])
	}
}