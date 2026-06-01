package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/docker"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/nats"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/repository"
)

// WorkspaceState represents the lifecycle state of a workspace.
type WorkspaceState string

const (
	StateCreating WorkspaceState = "creating"
	StateRunning  WorkspaceState = "running"
	StatePaused   WorkspaceState = "paused"
	StateStopping WorkspaceState = "stopping"
	StateStopped  WorkspaceState = "stopped"
)

// validTransitions defines the allowed state transitions.
var validTransitions = map[WorkspaceState][]WorkspaceState{
	StateCreating: {StateRunning},
	StateRunning:  {StatePaused, StateStopping},
	StatePaused:   {StateRunning, StateStopping},
	StateStopping: {StateStopped},
	StateStopped:  {StateCreating},
}

// Workspace represents a user workspace.
type Workspace struct {
	ID        string
	UserID    string
	Name      string
	State     WorkspaceState
	CreatedAt int64
	UpdatedAt int64
	NoVNCPort int
}

// persistState writes the current workspace state to the database.
func (s *WorkspaceService) persistState(ctx context.Context, workspaceID string, state WorkspaceState) {
	if s.repo == nil {
		return
	}
	wsUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		s.logger.Warn("failed to parse workspace ID for persistence", "workspace_id", workspaceID, "error", err)
		return
	}
	if err := s.repo.UpdateWorkspaceState(ctx, wsUUID, string(state)); err != nil {
		s.logger.Warn("failed to persist workspace state to database", "workspace_id", workspaceID, "state", state, "error", err)
	}
}

// IsValidTransition checks whether a state transition is valid.
func IsValidTransition(from, to WorkspaceState) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// WorkspaceService provides business logic for workspace lifecycle.
type WorkspaceService struct {
	docker docker.DockerClient
	repo   *repository.PostgresRepository
	logger *logging.Logger
	nats  *nats.Publisher
	mu    sync.RWMutex
	store map[string]*Workspace // in-memory cache
}

// NewWorkspaceService creates a new WorkspaceService.
func NewWorkspaceService(dockerClient docker.DockerClient, repo *repository.PostgresRepository, logger *logging.Logger, pub *nats.Publisher) *WorkspaceService {
	return &WorkspaceService{
		docker: dockerClient,
		repo:   repo,
		logger: logger,
		nats:   pub,
		store:  make(map[string]*Workspace),
	}
}

// CreateWorkspace creates a new workspace with Docker container isolation.
func (s *WorkspaceService) CreateWorkspace(ctx context.Context, userID, name string) (*Workspace, error) {
	workspaceID := uuid.New().String()

	ws := &Workspace{
		ID:        workspaceID,
		UserID:    userID,
		Name:      name,
		State:     StateCreating,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// Store workspace in creating state
	s.mu.Lock()
	s.store[workspaceID] = ws
	s.mu.Unlock()

	// Persist to database
	if s.repo != nil {
		userUUID, _ := uuid.Parse(userID)
		if _, err := s.repo.CreateWorkspace(ctx, userUUID, name, string(StateCreating), 0); err != nil {
			s.logger.Warn("failed to persist workspace to database", "workspace_id", workspaceID, "error", err)
		}
	}

	s.logger.Info("creating workspace", "workspace_id", workspaceID, "user_id", userID, "name", name)

	// Create Docker container
	if err := s.docker.CreateWorkspace(ctx, workspaceID, userID); err != nil {
		s.logger.Error("failed to create workspace container", "workspace_id", workspaceID, "error", err)
		s.mu.Lock()
		delete(s.store, workspaceID)
		s.mu.Unlock()
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Transition to running state
	ws.State = StateRunning
	ws.UpdatedAt = time.Now().Unix()

	// Persist state change to database
	if s.repo != nil {
		wsUUID, _ := uuid.Parse(workspaceID)
		if err := s.repo.UpdateWorkspaceState(ctx, wsUUID, string(StateRunning)); err != nil {
			s.logger.Warn("failed to persist workspace state to database", "workspace_id", workspaceID, "error", err)
		}
	}

	s.logger.Info("workspace created", "workspace_id", workspaceID, "user_id", userID, "name", name)

	// Publish event
	if s.nats != nil {
		if err := s.nats.PublishWorkspaceCreated(ctx, workspaceID, userID); err != nil {
			s.logger.Warn("failed to publish workspace.created event", "workspace_id", workspaceID, "error", err)
		}
	}

	return ws, nil
}

// GetWorkspace returns a workspace by ID.
func (s *WorkspaceService) GetWorkspace(ctx context.Context, workspaceID string) (*Workspace, error) {
	s.mu.RLock()
	ws, ok := s.store[workspaceID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("workspace %s not found", workspaceID)
	}
	return ws, nil
}

// ListWorkspaces returns all workspaces for a user.
func (s *WorkspaceService) ListWorkspaces(ctx context.Context, userID string) ([]*Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Workspace
	for _, ws := range s.store {
		if ws.UserID == userID {
			result = append(result, ws)
		}
	}
	return result, nil
}

// PauseWorkspace pauses a running workspace.
func (s *WorkspaceService) PauseWorkspace(ctx context.Context, workspaceID string) (*Workspace, error) {
	s.mu.Lock()
	ws, ok := s.store[workspaceID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("workspace %s not found", workspaceID)
	}

	if !IsValidTransition(ws.State, StatePaused) {
		s.mu.Unlock()
		return nil, fmt.Errorf("invalid transition: cannot pause workspace in state %s", ws.State)
	}

	if err := s.docker.PauseWorkspace(ctx, workspaceID); err != nil {
		s.mu.Unlock()
		s.logger.Error("failed to pause workspace container", "workspace_id", workspaceID, "error", err)
		return nil, fmt.Errorf("failed to pause workspace: %w", err)
	}

	ws.State = StatePaused
	ws.UpdatedAt = time.Now().Unix()
	s.mu.Unlock()

	s.persistState(ctx, workspaceID, StatePaused)

	s.logger.Info("workspace paused", "workspace_id", workspaceID)

	if s.nats != nil {
		if err := s.nats.PublishWorkspacePaused(ctx, workspaceID); err != nil {
			s.logger.Warn("failed to publish workspace.paused event", "workspace_id", workspaceID, "error", err)
		}
	}

	return ws, nil
}

// ResumeWorkspace resumes a paused workspace.
func (s *WorkspaceService) ResumeWorkspace(ctx context.Context, workspaceID string) (*Workspace, error) {
	s.mu.Lock()
	ws, ok := s.store[workspaceID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("workspace %s not found", workspaceID)
	}

	if !IsValidTransition(ws.State, StateRunning) {
		s.mu.Unlock()
		return nil, fmt.Errorf("invalid transition: cannot resume workspace in state %s", ws.State)
	}

	if err := s.docker.ResumeWorkspace(ctx, workspaceID); err != nil {
		s.mu.Unlock()
		s.logger.Error("failed to resume workspace container", "workspace_id", workspaceID, "error", err)
		return nil, fmt.Errorf("failed to resume workspace: %w", err)
	}

	ws.State = StateRunning
	ws.UpdatedAt = time.Now().Unix()
	s.mu.Unlock()

	s.persistState(ctx, workspaceID, StateRunning)

	s.logger.Info("workspace resumed", "workspace_id", workspaceID)

	if s.nats != nil {
		if err := s.nats.PublishWorkspaceResumed(ctx, workspaceID); err != nil {
			s.logger.Warn("failed to publish workspace.resumed event", "workspace_id", workspaceID, "error", err)
		}
	}

	return ws, nil
}

// StopWorkspace stops a running or paused workspace.
func (s *WorkspaceService) StopWorkspace(ctx context.Context, workspaceID string) (*Workspace, error) {
	s.mu.Lock()
	ws, ok := s.store[workspaceID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("workspace %s not found", workspaceID)
	}

	if !IsValidTransition(ws.State, StateStopping) {
		s.mu.Unlock()
		return nil, fmt.Errorf("invalid transition: cannot stop workspace in state %s", ws.State)
	}

	// Transition to stopping first
	ws.State = StateStopping
	ws.UpdatedAt = time.Now().Unix()
	s.mu.Unlock()

	if err := s.docker.StopWorkspace(ctx, workspaceID); err != nil {
		s.logger.Error("failed to stop workspace container", "workspace_id", workspaceID, "error", err)
		return nil, fmt.Errorf("failed to stop workspace: %w", err)
	}

	// Transition to stopped
	s.mu.Lock()
	ws.State = StateStopped
	ws.UpdatedAt = time.Now().Unix()
	s.mu.Unlock()

	s.persistState(ctx, workspaceID, StateStopped)

	s.logger.Info("workspace stopped", "workspace_id", workspaceID)

	if s.nats != nil {
		if err := s.nats.PublishWorkspaceStopped(ctx, workspaceID); err != nil {
			s.logger.Warn("failed to publish workspace.stopped event", "workspace_id", workspaceID, "error", err)
		}
	}

	return ws, nil
}

// DestroyWorkspace removes a workspace and its Docker resources.
func (s *WorkspaceService) DestroyWorkspace(ctx context.Context, workspaceID string) error {
	s.mu.Lock()
	ws, ok := s.store[workspaceID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("workspace %s not found", workspaceID)
	}

	// Allow destruction from any state except creating
	if ws.State == StateCreating {
		s.mu.Unlock()
		return fmt.Errorf("cannot destroy workspace while it is being created")
	}
	s.mu.Unlock()

	if err := s.docker.RemoveWorkspace(ctx, workspaceID); err != nil {
		s.logger.Error("failed to remove workspace container", "workspace_id", workspaceID, "error", err)
		return fmt.Errorf("failed to destroy workspace: %w", err)
	}

	s.mu.Lock()
	delete(s.store, workspaceID)
	s.mu.Unlock()

	// Delete from database
	if s.repo != nil {
		wsUUID, _ := uuid.Parse(workspaceID)
		if err := s.repo.DeleteWorkspace(ctx, wsUUID); err != nil {
			s.logger.Warn("failed to delete workspace from database", "workspace_id", workspaceID, "error", err)
		}
	}

	s.logger.Info("workspace destroyed", "workspace_id", workspaceID)

	if s.nats != nil {
		if err := s.nats.PublishWorkspaceDestroyed(ctx, workspaceID); err != nil {
			s.logger.Warn("failed to publish workspace.destroyed event", "workspace_id", workspaceID, "error", err)
		}
	}

	return nil
}

// StartWorkspace starts a stopped workspace by recreating its container.
func (s *WorkspaceService) StartWorkspace(ctx context.Context, workspaceID string) (*Workspace, error) {
	s.mu.Lock()
	ws, ok := s.store[workspaceID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("workspace %s not found", workspaceID)
	}

	if !IsValidTransition(ws.State, StateCreating) {
		s.mu.Unlock()
		return nil, fmt.Errorf("invalid transition: cannot start workspace in state %s", ws.State)
	}

	ws.State = StateCreating
	ws.UpdatedAt = time.Now().Unix()
	s.mu.Unlock()

	// Recreate the container
	if s.docker != nil {
		if err := s.docker.CreateWorkspace(ctx, workspaceID, ws.UserID); err != nil {
			s.logger.Error("failed to recreate workspace container", "workspace_id", workspaceID, "error", err)
			s.mu.Lock()
			ws.State = StateStopped
			ws.UpdatedAt = time.Now().Unix()
			s.mu.Unlock()
			return nil, fmt.Errorf("failed to start workspace: %w", err)
		}
	}

	s.mu.Lock()
	ws.State = StateRunning
	ws.UpdatedAt = time.Now().Unix()
	s.mu.Unlock()

	s.persistState(ctx, workspaceID, StateRunning)

	s.logger.Info("workspace started", "workspace_id", workspaceID)

	if s.nats != nil {
		if err := s.nats.PublishWorkspaceResumed(ctx, workspaceID); err != nil {
			s.logger.Warn("failed to publish workspace.started event", "workspace_id", workspaceID, "error", err)
		}
	}

	return ws, nil
}