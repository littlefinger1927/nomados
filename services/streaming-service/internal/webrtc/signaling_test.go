package webrtc

import (
	"testing"

	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/streaming-service/internal/turn"
)

func newTestSignalingServer() *SignalingServer {
	config := turn.DefaultConfig()
	turnRelay := turn.NewTURNRelay(config)
	logger := logging.NewLogger("streaming-test", nil)
	// NATS connection is nil for unit tests that don't require NATS;
	// tests that exercise NATS functionality should use an integration test.
	return NewSignalingServer(turnRelay, nil, logger)
}

func TestCreatePeerConnectionConfig(t *testing.T) {
	server := newTestSignalingServer()

	config := server.CreatePeerConnectionConfig()
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if len(config.ICEServers) != 1 {
		t.Errorf("expected 1 ICE server (TURN-only), got %d", len(config.ICEServers))
	}

	// Verify it's TURN-only (no STUN)
	iceServer := config.ICEServers[0]
	if len(iceServer.URLs) != 1 {
		t.Errorf("expected 1 URL, got %d", len(iceServer.URLs))
	}
	url := iceServer.URLs[0]
	if url[:4] != "turn" {
		t.Errorf("expected TURN URL, got %s", url)
	}
}

func TestSignalTypes(t *testing.T) {
	if SignalTypeOffer != "offer" {
		t.Errorf("expected SignalTypeOffer 'offer', got %s", SignalTypeOffer)
	}
	if SignalTypeAnswer != "answer" {
		t.Errorf("expected SignalTypeAnswer 'answer', got %s", SignalTypeAnswer)
	}
	if SignalTypeICECandidate != "ice-candidate" {
		t.Errorf("expected SignalTypeICECandidate 'ice-candidate', got %s", SignalTypeICECandidate)
	}
}

func TestClearWorkspace(t *testing.T) {
	server := newTestSignalingServer()

	// ClearWorkspace should not panic even with no cached answers
	server.ClearWorkspace("ws-1")

	// With a cached answer, ClearWorkspace should remove it
	server.mu.Lock()
	server.pendingAnswers["ws-1"] = &SignalMessage{
		Type:        SignalTypeAnswer,
		WorkspaceID: "ws-1",
		SDP:         "test-sdp",
	}
	server.mu.Unlock()

	server.ClearWorkspace("ws-1")

	server.mu.RLock()
	_, exists := server.pendingAnswers["ws-1"]
	server.mu.RUnlock()

	if exists {
		t.Error("expected pendingAnswers to be cleared after ClearWorkspace")
	}
}

func TestGetAnswerFromCache(t *testing.T) {
	server := newTestSignalingServer()

	// Pre-populate the cache with a known answer
	expectedSDP := "v=0\r\no=- 12345 2 IN IP4 0.0.0.0\r\ns=-\r\nt=0 0\r\n"
	server.mu.Lock()
	server.pendingAnswers["ws-1"] = &SignalMessage{
		Type:        SignalTypeAnswer,
		WorkspaceID: "ws-1",
		SDP:         expectedSDP,
	}
	server.mu.Unlock()

	// GetAnswer should return the cached answer without needing NATS
	answer, err := server.GetAnswer(nil, "ws-1")
	if err != nil {
		t.Fatalf("GetAnswer returned error: %v", err)
	}
	if answer.WorkspaceID != "ws-1" {
		t.Errorf("expected workspace ws-1, got %s", answer.WorkspaceID)
	}
	if answer.SDP != expectedSDP {
		t.Errorf("expected SDP %q, got %q", expectedSDP, answer.SDP)
	}
}