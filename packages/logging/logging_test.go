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