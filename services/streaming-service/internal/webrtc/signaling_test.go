package webrtc

import (
	"context"
	"testing"

	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/streaming-service/internal/turn"
)

func newTestSignalingServer() *SignalingServer {
	config := turn.DefaultConfig()
	turnRelay := turn.NewTURNRelay(config)
	logger := logging.NewLogger("streaming-test", nil)
	return NewSignalingServer(turnRelay, logger)
}

func TestProcessOffer(t *testing.T) {
	server := newTestSignalingServer()
	ctx := context.Background()

	answer, err := server.ProcessOffer(ctx, "ws-1", "v=0\r\no=- 12345 2 IN IP4 0.0.0.0\r\ns=-\r\nt=0 0\r\n")
	if err != nil {
		t.Fatalf("ProcessOffer returned error: %v", err)
	}

	if answer.Type != SignalTypeAnswer {
		t.Errorf("expected answer type, got %s", answer.Type)
	}
	if answer.WorkspaceID != "ws-1" {
		t.Errorf("expected workspace ws-1, got %s", answer.WorkspaceID)
	}
}

func TestGetPendingOffer(t *testing.T) {
	server := newTestSignalingServer()
	ctx := context.Background()

	sdp := "v=0\r\no=- 12345 2 IN IP4 0.0.0.0\r\ns=-\r\nt=0 0\r\n"
	server.ProcessOffer(ctx, "ws-1", sdp)

	offer, err := server.GetPendingOffer("ws-1")
	if err != nil {
		t.Fatalf("GetPendingOffer returned error: %v", err)
	}
	if offer.Offer.SDP != sdp {
		t.Errorf("expected SDP %q, got %q", sdp, offer.Offer.SDP)
	}
	if offer.Answer == nil {
		t.Error("expected answer to be set")
	}
}

func TestGetPendingOfferNotFound(t *testing.T) {
	server := newTestSignalingServer()

	_, err := server.GetPendingOffer("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent workspace, got nil")
	}
}

func TestProcessICECandidate(t *testing.T) {
	server := newTestSignalingServer()
	ctx := context.Background()

	err := server.ProcessICECandidate(ctx, "ws-1", "candidate:0 1 UDP 2122252543 192.168.1.1 5000 typ host")
	if err != nil {
		t.Fatalf("ProcessICECandidate returned error: %v", err)
	}

	candidates := server.GetICECandidates("ws-1")
	if len(candidates) != 1 {
		t.Errorf("expected 1 candidate, got %d", len(candidates))
	}
}

func TestMultipleICECandidates(t *testing.T) {
	server := newTestSignalingServer()
	ctx := context.Background()

	server.ProcessICECandidate(ctx, "ws-1", "candidate:0 1 UDP 2122252543 192.168.1.1 5000 typ host")
	server.ProcessICECandidate(ctx, "ws-1", "candidate:1 1 UDP 2122252543 10.0.0.1 5001 typ host")

	candidates := server.GetICECandidates("ws-1")
	if len(candidates) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(candidates))
	}
}

func TestGetICECandidatesEmpty(t *testing.T) {
	server := newTestSignalingServer()

	candidates := server.GetICECandidates("nonexistent")
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates, got %d", len(candidates))
	}
}

func TestClearWorkspace(t *testing.T) {
	server := newTestSignalingServer()
	ctx := context.Background()

	server.ProcessOffer(ctx, "ws-1", "v=0\r\ns=-\r\n")
	server.ProcessICECandidate(ctx, "ws-1", "candidate:0 1 UDP 2122252543 192.168.1.1 5000 typ host")

	server.ClearWorkspace("ws-1")

	_, err := server.GetPendingOffer("ws-1")
	if err == nil {
		t.Error("expected error after clearing workspace, got nil")
	}

	candidates := server.GetICECandidates("ws-1")
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates after clearing, got %d", len(candidates))
	}
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