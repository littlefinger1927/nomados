package webrtc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/streaming-service/internal/turn"
)

// SignalType represents the type of signaling message.
type SignalType string

const (
	// SignalTypeOffer represents a WebRTC offer SDP.
	SignalTypeOffer SignalType = "offer"
	// SignalTypeAnswer represents a WebRTC answer SDP.
	SignalTypeAnswer SignalType = "answer"
	// SignalTypeICECandidate represents an ICE candidate.
	SignalTypeICECandidate SignalType = "ice-candidate"
)

// SignalMessage represents a WebRTC signaling message exchanged
// between the Tauri client and the browser runtime.
type SignalMessage struct {
	Type        SignalType `json:"type"`
	WorkspaceID string     `json:"workspace_id"`
	SDP         string     `json:"sdp,omitempty"`
	Candidate   string     `json:"candidate,omitempty"`
	Timestamp   time.Time  `json:"timestamp"`
}

// PeerConnectionConfig holds configuration for creating a WebRTC
// peer connection with TURN-only relay.
type PeerConnectionConfig struct {
	ICEServers []turn.ICEServer
}

// PendingOffer tracks an in-progress offer/answer exchange for a workspace.
type PendingOffer struct {
	Offer     SignalMessage
	Answer    *SignalMessage
	CreatedAt time.Time
}

// SignalingServer handles WebRTC offer/answer exchange and ICE candidate
// relay between the Tauri client and the browser runtime.
type SignalingServer struct {
	mu             sync.RWMutex
	pendingOffers  map[string]*PendingOffer // keyed by workspace ID
	iceCandidates  map[string][]string     // keyed by workspace ID
	turnRelay      *turn.TURNRelay
	logger         *logging.Logger
}

// NewSignalingServer creates a new SignalingServer with the given TURN relay.
func NewSignalingServer(turnRelay *turn.TURNRelay, logger *logging.Logger) *SignalingServer {
	return &SignalingServer{
		pendingOffers: make(map[string]*PendingOffer),
		iceCandidates: make(map[string][]string),
		turnRelay:      turnRelay,
		logger:         logger,
	}
}

// ProcessOffer processes a WebRTC offer from the Tauri client and generates
// an answer for the browser runtime. In Phase 1, the offer is stored and
// a placeholder answer is created; full browser peer connection wiring will
// be completed when browser-manager integration is done.
func (s *SignalingServer) ProcessOffer(ctx context.Context, workspaceID, sdp string) (*SignalMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	offer := SignalMessage{
		Type:        SignalTypeOffer,
		WorkspaceID: workspaceID,
		SDP:         sdp,
		Timestamp:   time.Now().UTC(),
	}

	// Store the pending offer
	s.pendingOffers[workspaceID] = &PendingOffer{
		Offer:     offer,
		CreatedAt: time.Now().UTC(),
	}

	// In Phase 1, generate a placeholder answer signaling message.
	// The actual WebRTC answer will be generated when the browser peer
	// connection is fully wired.
	answer := &SignalMessage{
		Type:        SignalTypeAnswer,
		WorkspaceID: workspaceID,
		SDP:         "", // Will be populated by browser peer connection
		Timestamp:   time.Now().UTC(),
	}

	s.pendingOffers[workspaceID].Answer = answer
	s.logger.Info("processed WebRTC offer", "workspace_id", workspaceID)

	return answer, nil
}

// ProcessICECandidate relays an ICE candidate between peers.
// In Phase 1, candidates are stored for later retrieval; the actual
// relay to the browser peer will be wired in a later phase.
func (s *SignalingServer) ProcessICECandidate(ctx context.Context, workspaceID, candidate string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.iceCandidates[workspaceID] = append(s.iceCandidates[workspaceID], candidate)
	s.logger.Debug("stored ICE candidate", "workspace_id", workspaceID, "candidate_count", len(s.iceCandidates[workspaceID]))

	return nil
}

// GetICECandidates returns all stored ICE candidates for a workspace.
func (s *SignalingServer) GetICECandidates(workspaceID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	candidates := s.iceCandidates[workspaceID]
	result := make([]string, len(candidates))
	copy(result, candidates)
	return result
}

// GetPendingOffer returns the pending offer for a workspace, if any.
func (s *SignalingServer) GetPendingOffer(workspaceID string) (*PendingOffer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	offer, exists := s.pendingOffers[workspaceID]
	if !exists {
		return nil, fmt.Errorf("no pending offer for workspace %s", workspaceID)
	}
	return offer, nil
}

// ClearWorkspace removes all signaling state for a workspace.
func (s *SignalingServer) ClearWorkspace(workspaceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.pendingOffers, workspaceID)
	delete(s.iceCandidates, workspaceID)
	s.logger.Info("cleared signaling state for workspace", "workspace_id", workspaceID)
}

// CreatePeerConnectionConfig creates a WebRTC configuration with TURN-only
// servers (no STUN, per the TURN-only relay requirement).
func (s *SignalingServer) CreatePeerConnectionConfig() *PeerConnectionConfig {
	iceServers := s.turnRelay.ICEServers()
	return &PeerConnectionConfig{
		ICEServers: iceServers,
	}
}