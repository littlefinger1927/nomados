package docker

import (
	"context"
	"fmt"
)

// ContainerState represents the state of a workspace container.
type ContainerState string

const (
	StateCreating ContainerState = "creating"
	StateRunning  ContainerState = "running"
	StatePaused   ContainerState = "paused"
	StateStopped  ContainerState = "stopped"
	StateUnknown  ContainerState = "unknown"
)

const (
	// LabelPrefix is the prefix for all NomadOS labels on Docker objects.
	LabelPrefix = "nomados.workspace"

	// NetworkPrefix is the prefix for workspace Docker networks.
	NetworkPrefix = "nomados-ws-"

	// VolumePrefix is the prefix for workspace Docker volumes.
	VolumePrefix = "nomados-ws-"

	// ChromiumImage is the Docker image used for the Chromium container.
	ChromiumImage = "chromium/chromium:latest"

	// AgentImage is the Docker image used for the workspace agent container.
	AgentImage = "nomados/workspace-agent:latest"
)

// DockerClient interface abstracts Docker operations for testability.
// The real implementation uses the Docker SDK; the mock is used for unit tests.
type DockerClient interface {
	CreateWorkspace(ctx context.Context, workspaceID, userID string) error
	PauseWorkspace(ctx context.Context, workspaceID string) error
	ResumeWorkspace(ctx context.Context, workspaceID string) error
	StopWorkspace(ctx context.Context, workspaceID string) error
	RemoveWorkspace(ctx context.Context, workspaceID string) error
	GetWorkspaceStatus(ctx context.Context, workspaceID string) (ContainerState, error)
	Close() error
}

// Client wraps the Docker SDK for workspace container lifecycle management.
// It requires a running Docker daemon to function.
type Client struct {
	// The Docker SDK client will be initialized here when connecting to a real daemon.
	// For Phase 1, the actual Docker API calls are stubbed through the interface.
	// When deploying, NewDockerClient() will initialize the real SDK client.
}

// NewDockerClient creates a new Docker client using environment configuration.
// It connects to the Docker daemon via the DOCKER_HOST environment variable.
// Returns an error if the Docker daemon is not available.
func NewDockerClient() (*Client, error) {
	// In production, this would call:
	//   cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	// For Phase 1, we return a stub that needs to be replaced with the real SDK.
	return &Client{}, nil
}

// CreateWorkspace creates an isolated Docker network + containers for a workspace.
// Each workspace gets:
//  1. Its own Docker network (isolated networking) named "nomados-ws-{workspaceID}"
//  2. Its own volume (encrypted filesystem) named "nomados-ws-{workspaceID}"
//  3. A Chromium container running headless with anti-fingerprint flags
//  4. A workspace agent container for orchestration commands
func (c *Client) CreateWorkspace(ctx context.Context, workspaceID, userID string) error {
	// TODO: Implement with real Docker SDK client when Docker daemon is available.
	// Steps:
	// 1. Create isolated Docker network named "nomados-ws-{workspaceID}"
	// 2. Create volume for workspace data named "nomados-ws-{workspaceID}"
	// 3. Create and start Chromium container with anti-fingerprint flags
	// 4. Create and start workspace agent container
	return fmt.Errorf("docker client: CreateWorkspace not yet connected to daemon")
}

// PauseWorkspace freezes a container using cgroups freezer.
func (c *Client) PauseWorkspace(ctx context.Context, workspaceID string) error {
	// TODO: Implement with real Docker SDK client
	return fmt.Errorf("docker client: PauseWorkspace not yet connected to daemon")
}

// ResumeWorkspace unpauses a container.
func (c *Client) ResumeWorkspace(ctx context.Context, workspaceID string) error {
	// TODO: Implement with real Docker SDK client
	return fmt.Errorf("docker client: ResumeWorkspace not yet connected to daemon")
}

// StopWorkspace gracefully stops all containers for a workspace.
func (c *Client) StopWorkspace(ctx context.Context, workspaceID string) error {
	// TODO: Implement with real Docker SDK client
	return fmt.Errorf("docker client: StopWorkspace not yet connected to daemon")
}

// RemoveWorkspace removes containers, network, and volume for a workspace.
func (c *Client) RemoveWorkspace(ctx context.Context, workspaceID string) error {
	// TODO: Implement with real Docker SDK client
	return fmt.Errorf("docker client: RemoveWorkspace not yet connected to daemon")
}

// GetWorkspaceStatus returns the container status for a workspace.
func (c *Client) GetWorkspaceStatus(ctx context.Context, workspaceID string) (ContainerState, error) {
	// TODO: Implement with real Docker SDK client
	return StateUnknown, fmt.Errorf("docker client: GetWorkspaceStatus not yet connected to daemon")
}

// Close closes the Docker client.
func (c *Client) Close() error {
	return nil
}

// Ensure Client implements DockerClient interface.
var _ DockerClient = (*Client)(nil)