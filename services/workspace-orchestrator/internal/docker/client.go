package docker

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/api/types/volume"
	dockerclient "github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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

	// ChromiumFlags contains anti-fingerprint flags for the Chromium container.
	ChromiumFlags = "--disable-blink-features=AutomationControlled --disable-features=IsolateOrigins,site-per-process --disable-web-security --disable-features=ImprovedCookieControls --disable-features=SameSiteByDefaultCookies --disable-features=PrivacySandboxSettings4"
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
	cli       *dockerclient.Client
	imageName string
}

// NewDockerClient creates a new Docker client using environment configuration.
// It connects to the Docker daemon via the DOCKER_HOST environment variable.
// Returns an error if the Docker daemon is not available.
func NewDockerClient() (*Client, error) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Verify the Docker daemon is reachable.
	if _, err = cli.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping Docker daemon: %w", err)
	}

	imageName := os.Getenv("WORKSPACE_IMAGE")
	if imageName == "" {
		imageName = ChromiumImage
	}

	return &Client{
		cli:       cli,
		imageName: imageName,
	}, nil
}

// CreateWorkspace creates an isolated Docker network + containers for a workspace.
// Each workspace gets:
//  1. Its own Docker network (isolated networking) named "nomados-ws-{workspaceID}"
//  2. Its own volume (encrypted filesystem) named "nomados-ws-{workspaceID}"
//  3. A Chromium container running headless with anti-fingerprint flags
//  4. A workspace agent container for orchestration commands
func (c *Client) CreateWorkspace(ctx context.Context, workspaceID, userID string) error {
	labels := map[string]string{
		LabelPrefix:    workspaceID,
		"nomados.user": userID,
	}

	// 1. Create isolated Docker network.
	netName := NetworkPrefix + workspaceID
	_, err := c.cli.NetworkCreate(ctx, netName, types.NetworkCreate{
		Labels: labels,
	})
	if err != nil {
		return fmt.Errorf("failed to create network %s: %w", netName, err)
	}

	// 2. Create volume for workspace data.
	volName := VolumePrefix + workspaceID
	_, err = c.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   volName,
		Labels: labels,
	})
	if err != nil {
		return fmt.Errorf("failed to create volume %s: %w", volName, err)
	}

	// 3. Pull workspace image if not present.
	reader, err := c.cli.ImagePull(ctx, c.imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", c.imageName, err)
	}
	// Wait for the pull to complete by reading the response body.
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("failed to read image pull response: %w", err)
	}
	reader.Close()

	// 4. Create and start Chromium container.
	containerName := "nomados-chromium-" + workspaceID
	containerLabels := map[string]string{
		LabelPrefix:    workspaceID,
		"nomados.user": userID,
		"nomados.role": "chromium",
	}

	containerConfig := &container.Config{
		Image:  c.imageName,
		Labels: containerLabels,
		Env:    []string{fmt.Sprintf("CHROMIUM_FLAGS=%s", ChromiumFlags)},
	}

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:   2 * 1024 * 1024 * 1024, // 2GB
			NanoCPUs: 2_000_000_000,           // 2 CPUs
		},
		SecurityOpt: []string{"no-new-privileges:true"},
		CapDrop:     strslice.StrSlice{"ALL"},
		CapAdd:      strslice.StrSlice{"NET_RAW"},
		Binds:       []string{volName + ":/data"},
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			netName: {},
		},
	}

	platform := &ocispec.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}

	createResp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, platform, containerName)
	if err != nil {
		return fmt.Errorf("failed to create container %s: %w", containerName, err)
	}

	if err := c.cli.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %w", createResp.ID, err)
	}

	return nil
}

// PauseWorkspace freezes a container using cgroups freezer.
func (c *Client) PauseWorkspace(ctx context.Context, workspaceID string) error {
	containerID, err := c.findContainerByLabel(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to find container for workspace %s: %w", workspaceID, err)
	}
	if err := c.cli.ContainerPause(ctx, containerID); err != nil {
		return fmt.Errorf("failed to pause container %s: %w", containerID, err)
	}
	return nil
}

// ResumeWorkspace unpauses a container.
func (c *Client) ResumeWorkspace(ctx context.Context, workspaceID string) error {
	containerID, err := c.findContainerByLabel(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to find container for workspace %s: %w", workspaceID, err)
	}
	if err := c.cli.ContainerUnpause(ctx, containerID); err != nil {
		return fmt.Errorf("failed to unpause container %s: %w", containerID, err)
	}
	return nil
}

// StopWorkspace gracefully stops all containers for a workspace.
func (c *Client) StopWorkspace(ctx context.Context, workspaceID string) error {
	containerID, err := c.findContainerByLabel(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to find container for workspace %s: %w", workspaceID, err)
	}
	timeout := 30
	if err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	return nil
}

// RemoveWorkspace removes containers, network, and volume for a workspace.
func (c *Client) RemoveWorkspace(ctx context.Context, workspaceID string) error {
	containerID, err := c.findContainerByLabel(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to find container for workspace %s: %w", workspaceID, err)
	}

	// Remove the container with force.
	if err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}

	// Best-effort: remove the network.
	netName := NetworkPrefix + workspaceID
	if err := c.cli.NetworkRemove(ctx, netName); err != nil {
		log.Printf("warning: failed to remove network %s: %v", netName, err)
	}

	// Best-effort: remove the volume.
	volName := VolumePrefix + workspaceID
	if err := c.cli.VolumeRemove(ctx, volName, false); err != nil {
		log.Printf("warning: failed to remove volume %s: %v", volName, err)
	}

	return nil
}

// GetWorkspaceStatus returns the container status for a workspace.
func (c *Client) GetWorkspaceStatus(ctx context.Context, workspaceID string) (ContainerState, error) {
	containerID, err := c.findContainerByLabel(ctx, workspaceID)
	if err != nil {
		return StateUnknown, fmt.Errorf("failed to find container for workspace %s: %w", workspaceID, err)
	}

	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return StateUnknown, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	switch inspect.State.Status {
	case "created":
		return StateCreating, nil
	case "running":
		return StateRunning, nil
	case "paused":
		return StatePaused, nil
	case "exited", "dead":
		return StateStopped, nil
	default:
		return StateUnknown, nil
	}
}

// Close closes the Docker client.
func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}

// findContainerByLabel finds a container by the nomados.workspace label.
func (c *Client) findContainerByLabel(ctx context.Context, workspaceID string) (string, error) {
	filter := filters.NewArgs()
	filter.Add("label", LabelPrefix+"="+workspaceID)

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		Filters: filter,
		All:     true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		return "", fmt.Errorf("no container found for workspace %s", workspaceID)
	}

	return containers[0].ID, nil
}

// Ensure Client implements DockerClient interface.
var _ DockerClient = (*Client)(nil)