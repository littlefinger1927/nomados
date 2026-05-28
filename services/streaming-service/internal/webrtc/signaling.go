package webrtc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
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

// ICECandidatePayload represents an ICE candidate relayed via NATS.
type ICECandidatePayload struct {
	WorkspaceID   string `json:"workspace_id"`
	Candidate     string `json:"candidate"`
	SDPMid        string `json:"sdp_mid"`
	SDPMlineIndex uint32 `json:"sdp_mline_index"`
}

// SignalingServer handles WebRTC offer/answer exchange and ICE candidate
// relay between the Tauri client and the browser runtime. It uses NATS
// for distributed signaling and local caches for fast lookups.
type SignalingServer struct {
	mu              sync.RWMutex
	pendingAnswers  map[string]*SignalMessage // keyed by workspace ID
	nc              *nats.Conn
	turnRelay       *turn.TURNRelay
	logger          *logging.Logger
}

// NewSignalingServer creates a new SignalingServer with the given TURN relay
// and NATS connection for distributed signaling.
func NewSignalingServer(turnRelay *turn.TURNRelay, nc *nats.Conn, logger *logging.Logger) *SignalingServer {
	return &SignalingServer{
		pendingAnswers: make(map[string]*SignalMessage),
		nc:             nc,
		turnRelay:      turnRelay,
		logger:         logger,
	}
}

// ProcessOffer processes a WebRTC offer from the Tauri client by publishing
// it to NATS and waiting for an answer from the browser runtime.
func (s *SignalingServer) ProcessOffer(ctx context.Context, workspaceID, sdp string) (*SignalMessage, error) {
	offer := SignalMessage{
		Type:        SignalTypeOffer,
		WorkspaceID: workspaceID,
		SDP:         sdp,
		Timestamp:   time.Now().UTC(),
	}

	offerData, err := json.Marshal(offer)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal offer: %w", err)
	}

	subject := fmt.Sprintf("workspace.%s.offer", workspaceID)
	answerSubject := fmt.Sprintf("workspace.%s.answer", workspaceID)

	// Subscribe to the answer subject before publishing the offer
	// to avoid a race condition where the answer arrives before we
	// start listening.
	sub, err := s.nc.SubscribeSync(answerSubject)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to answer subject: %w", err)
	}
	defer sub.Unsubscribe()

	// Publish the offer to NATS so browser-manager can pick it up.
	if err := s.nc.Publish(subject, offerData); err != nil {
		return nil, fmt.Errorf("failed to publish offer to NATS: %w", err)
	}
	s.logger.Info("published offer to NATS", "workspace_id", workspaceID, "subject", subject)

	// Wait for the answer from the browser runtime with a 10-second timeout.
	// Create a derived context with a 10-second deadline.
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	msg, err := sub.NextMsgWithContext(waitCtx)
	if err != nil {
		s.logger.Warn("timed out waiting for answer", "workspace_id", workspaceID, "error", err)
		return nil, fmt.Errorf("timed out waiting for answer for workspace %s: %w", workspaceID, err)
	}

	var answer SignalMessage
	if err := json.Unmarshal(msg.Data, &answer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal answer: %w", err)
	}

	// Cache the answer locally for GetAnswer lookups.
	s.mu.Lock()
	s.pendingAnswers[workspaceID] = &answer
	s.mu.Unlock()

	s.logger.Info("received answer from NATS", "workspace_id", workspaceID)
	return &answer, nil
}

// ProcessICECandidate relays an ICE candidate via NATS to the browser runtime.
func (s *SignalingServer) ProcessICECandidate(ctx context.Context, workspaceID, candidate, sdpMid string, sdpMlineIndex uint32) error {
	payload := ICECandidatePayload{
		WorkspaceID:   workspaceID,
		Candidate:     candidate,
		SDPMid:        sdpMid,
		SDPMlineIndex: sdpMlineIndex,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal ICE candidate: %w", err)
	}

	subject := fmt.Sprintf("workspace.%s.ice", workspaceID)
	if err := s.nc.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish ICE candidate to NATS: %w", err)
	}

	s.logger.Debug("published ICE candidate to NATS", "workspace_id", workspaceID, "subject", subject)
	return nil
}

// GetAnswer retrieves a cached answer for a workspace. If not found locally,
// it waits briefly on NATS for an answer to arrive.
func (s *SignalingServer) GetAnswer(ctx context.Context, workspaceID string) (*SignalMessage, error) {
	// Check local cache first.
	s.mu.RLock()
	if answer, ok := s.pendingAnswers[workspaceID]; ok {
		s.mu.RUnlock()
		return answer, nil
	}
	s.mu.RUnlock()

	// Not in cache; wait briefly on NATS for an answer.
	answerSubject := fmt.Sprintf("workspace.%s.answer", workspaceID)
	sub, err := s.nc.SubscribeSync(answerSubject)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to answer subject: %w", err)
	}
	defer sub.Unsubscribe()

	// Wait up to 5 seconds for an answer.
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	msg, err := sub.NextMsgWithContext(waitCtx)
	if err != nil {
		return nil, fmt.Errorf("no answer found for workspace %s: %w", workspaceID, err)
	}

	var answer SignalMessage
	if err := json.Unmarshal(msg.Data, &answer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal answer: %w", err)
	}

	// Cache for future lookups.
	s.mu.Lock()
	s.pendingAnswers[workspaceID] = &answer
	s.mu.Unlock()

	return &answer, nil
}

// ClearWorkspace removes all signaling state for a workspace.
func (s *SignalingServer) ClearWorkspace(workspaceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.pendingAnswers, workspaceID)
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