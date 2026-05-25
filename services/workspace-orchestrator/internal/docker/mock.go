package docker

import (
	"context"
	"fmt"
)

// MockClient implements a mock Docker client for unit testing.
// It is exported so other packages can use it in their tests.
type MockClient struct {
	Workspaces   map[string]string // workspaceID -> state
	CreateError  error
	PauseError   error
	ResumeError  error
	StopError    error
	RemoveError  error
	StatusError  error
	StatusResult ContainerState
}

// NewMockClient creates a new MockClient for testing.
func NewMockClient() *MockClient {
	return &MockClient{
		Workspaces: make(map[string]string),
	}
}

// CreateWorkspace simulates creating a workspace.
func (m *MockClient) CreateWorkspace(ctx context.Context, workspaceID, userID string) error {
	if m.CreateError != nil {
		return m.CreateError
	}
	m.Workspaces[workspaceID] = string(StateRunning)
	return nil
}

// PauseWorkspace simulates pausing a workspace.
func (m *MockClient) PauseWorkspace(ctx context.Context, workspaceID string) error {
	if m.PauseError != nil {
		return m.PauseError
	}
	if _, ok := m.Workspaces[workspaceID]; !ok {
		return fmt.Errorf("workspace %s not found", workspaceID)
	}
	m.Workspaces[workspaceID] = string(StatePaused)
	return nil
}

// ResumeWorkspace simulates resuming a workspace.
func (m *MockClient) ResumeWorkspace(ctx context.Context, workspaceID string) error {
	if m.ResumeError != nil {
		return m.ResumeError
	}
	if _, ok := m.Workspaces[workspaceID]; !ok {
		return fmt.Errorf("workspace %s not found", workspaceID)
	}
	m.Workspaces[workspaceID] = string(StateRunning)
	return nil
}

// StopWorkspace simulates stopping a workspace.
func (m *MockClient) StopWorkspace(ctx context.Context, workspaceID string) error {
	if m.StopError != nil {
		return m.StopError
	}
	if _, ok := m.Workspaces[workspaceID]; !ok {
		return fmt.Errorf("workspace %s not found", workspaceID)
	}
	m.Workspaces[workspaceID] = string(StateStopped)
	return nil
}

// RemoveWorkspace simulates removing a workspace.
func (m *MockClient) RemoveWorkspace(ctx context.Context, workspaceID string) error {
	if m.RemoveError != nil {
		return m.RemoveError
	}
	if _, ok := m.Workspaces[workspaceID]; !ok {
		return fmt.Errorf("workspace %s not found", workspaceID)
	}
	delete(m.Workspaces, workspaceID)
	return nil
}

// GetWorkspaceStatus simulates getting workspace status.
func (m *MockClient) GetWorkspaceStatus(ctx context.Context, workspaceID string) (ContainerState, error) {
	if m.StatusError != nil {
		return StateUnknown, m.StatusError
	}
	if state, ok := m.Workspaces[workspaceID]; ok {
		return ContainerState(state), nil
	}
	return StateUnknown, fmt.Errorf("workspace %s not found", workspaceID)
}

// Close simulates closing the mock client.
func (m *MockClient) Close() error {
	return nil
}

// Ensure MockClient implements DockerClient interface.
var _ DockerClient = (*MockClient)(nil)