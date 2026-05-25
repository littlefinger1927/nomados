package relay

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/streaming-service/internal/webrtc"
)

// StreamStatus represents the current status of a stream.
type StreamStatus string

const (
	// StreamActive indicates the stream is actively relaying.
	StreamActive StreamStatus = "active"
	// StreamPaused indicates the stream is temporarily paused.
	StreamPaused StreamStatus = "paused"
	// StreamClosed indicates the stream has been closed.
	StreamClosed StreamStatus = "closed"
)

// Stream represents an active WebRTC stream for a workspace.
type Stream struct {
	ID          string
	WorkspaceID string
	Status      StreamStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Peer connection placeholders for Phase 1.
	// The browser-manager owns the Chromium process; the streaming service
	// handles signaling and relay setup.
	ClientPeerConfig  *webrtc.PeerConnectionConfig
	BrowserPeerConfig *webrtc.PeerConnectionConfig
}

// StreamRelay manages active WebRTC streams per workspace.
type StreamRelay struct {
	mu      sync.RWMutex
	streams map[string]*Stream // keyed by workspace ID
	logger  *logging.Logger
}

// NewStreamRelay creates a new StreamRelay.
func NewStreamRelay(logger *logging.Logger) *StreamRelay {
	return &StreamRelay{
		streams: make(map[string]*Stream),
		logger:  logger,
	}
}

// CreateStream creates a new stream for a workspace. If a stream already
// exists for the workspace, it returns an error.
func (r *StreamRelay) CreateStream(ctx context.Context, workspaceID string) (*Stream, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.streams[workspaceID]; exists {
		return nil, fmt.Errorf("stream already exists for workspace %s", workspaceID)
	}

	now := time.Now().UTC()
	stream := &Stream{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		Status:      StreamActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	r.streams[workspaceID] = stream
	r.logger.Info("stream created", "workspace_id", workspaceID, "stream_id", stream.ID)

	return stream, nil
}

// CloseStream closes and removes the stream for the given workspace.
func (r *StreamRelay) CloseStream(ctx context.Context, workspaceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.streams[workspaceID]
	if !exists {
		return fmt.Errorf("no stream found for workspace %s", workspaceID)
	}

	stream.Status = StreamClosed
	stream.UpdatedAt = time.Now().UTC()
	delete(r.streams, workspaceID)

	r.logger.Info("stream closed", "workspace_id", workspaceID, "stream_id", stream.ID)
	return nil
}

// GetStream returns the stream for the given workspace ID.
func (r *StreamRelay) GetStream(workspaceID string) (*Stream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stream, exists := r.streams[workspaceID]
	if !exists {
		return nil, fmt.Errorf("no stream found for workspace %s", workspaceID)
	}
	return stream, nil
}

// ListStreams returns all active (non-closed) streams.
func (r *StreamRelay) ListStreams() []*Stream {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Stream, 0, len(r.streams))
	for _, s := range r.streams {
		result = append(result, s)
	}
	return result
}

// RouteInput routes input events (mouse/keyboard) from the client to the
// browser peer. In Phase 1, this logs the input event; the actual WebRTC
// data channel routing will be wired when browser-manager integration is
// complete.
func (r *StreamRelay) RouteInput(ctx context.Context, workspaceID string, data []byte) error {
	r.mu.RLock()
	_, exists := r.streams[workspaceID]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no stream found for workspace %s", workspaceID)
	}

	r.logger.Debug("routing input event", "workspace_id", workspaceID, "bytes", len(data))
	return nil
}

// PauseStream pauses the stream for the given workspace.
func (r *StreamRelay) PauseStream(ctx context.Context, workspaceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.streams[workspaceID]
	if !exists {
		return fmt.Errorf("no stream found for workspace %s", workspaceID)
	}

	stream.Status = StreamPaused
	stream.UpdatedAt = time.Now().UTC()
	r.logger.Info("stream paused", "workspace_id", workspaceID)
	return nil
}

// ResumeStream resumes a paused stream for the given workspace.
func (r *StreamRelay) ResumeStream(ctx context.Context, workspaceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.streams[workspaceID]
	if !exists {
		return fmt.Errorf("no stream found for workspace %s", workspaceID)
	}

	if stream.Status != StreamPaused {
		return fmt.Errorf("stream for workspace %s is not paused (status: %s)", workspaceID, stream.Status)
	}

	stream.Status = StreamActive
	stream.UpdatedAt = time.Now().UTC()
	r.logger.Info("stream resumed", "workspace_id", workspaceID)
	return nil
}

// CloseAll closes all streams and cleans up resources. Used for graceful shutdown.
func (r *StreamRelay) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for workspaceID, stream := range r.streams {
		stream.Status = StreamClosed
		stream.UpdatedAt = time.Now().UTC()
		r.logger.Info("stream closed during shutdown", "workspace_id", workspaceID, "stream_id", stream.ID)
	}

	r.streams = make(map[string]*Stream)
	r.logger.Info("all streams closed")
}