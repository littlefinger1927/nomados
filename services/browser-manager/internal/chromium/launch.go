package chromium

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/nomados/nomados/packages/logging"
)

// BrowserInstance represents a running Chromium process for a workspace.
type BrowserInstance struct {
	PID         int
	Cmd         *exec.Cmd
	WorkspaceID string
	Fingerprint Fingerprint
	ProfilePath string
}

// CommandRunner abstracts exec.Command for testability.
type CommandRunner interface {
	Start(cmd *exec.Cmd) error
	ProcessKill(process *os.Process) error
}

// RealCommandRunner is the production implementation that uses exec.Command.
type RealCommandRunner struct{}

// Start starts a command using the real exec package.
func (r *RealCommandRunner) Start(cmd *exec.Cmd) error {
	return cmd.Start()
}

// ProcessKill kills a process using the real os package.
func (r *RealCommandRunner) ProcessKill(process *os.Process) error {
	return process.Kill()
}

// ChromiumLauncher manages Chromium browser instances across workspaces.
// It launches headless Chromium with anti-fingerprinting flags and tracks
// running instances for lifecycle management.
type ChromiumLauncher struct {
	instances sync.Map // workspaceID -> *BrowserInstance
	runner    CommandRunner
	binPath   string
	profile   *ProfileManager
	logger    *logging.Logger
	mu        sync.Mutex
}

// LauncherConfig holds configuration for the ChromiumLauncher.
type LauncherConfig struct {
	// BinPath is the path to the Chromium binary. Defaults to "chromium-browser".
	BinPath string
	// ProfileBasePath is the base directory for workspace profile directories.
	ProfileBasePath string
}

// NewChromiumLauncher creates a new ChromiumLauncher with the given configuration.
func NewChromiumLauncher(config LauncherConfig, logger *logging.Logger) *ChromiumLauncher {
	binPath := config.BinPath
	if binPath == "" {
		binPath = "chromium-browser"
	}

	profileBasePath := config.ProfileBasePath
	if profileBasePath == "" {
		profileBasePath = "/tmp/nomados-profiles"
	}

	return &ChromiumLauncher{
		runner:  &RealCommandRunner{},
		binPath: binPath,
		profile: NewProfileManager(profileBasePath),
		logger:  logger,
	}
}

// NewChromiumLauncherWithRunner creates a ChromiumLauncher with a custom command
// runner for testing. Production code should use NewChromiumLauncher.
func NewChromiumLauncherWithRunner(config LauncherConfig, logger *logging.Logger, runner CommandRunner) *ChromiumLauncher {
	binPath := config.BinPath
	if binPath == "" {
		binPath = "chromium-browser"
	}

	profileBasePath := config.ProfileBasePath
	if profileBasePath == "" {
		profileBasePath = "/tmp/nomados-profiles"
	}

	return &ChromiumLauncher{
		runner:  runner,
		binPath: binPath,
		profile: NewProfileManager(profileBasePath),
		logger:  logger,
	}
}

// Launch starts a headless Chromium instance for the given workspace.
// It creates an isolated profile directory and applies anti-fingerprinting
// flags derived from the workspace ID.
func (l *ChromiumLauncher) Launch(ctx context.Context, workspaceID string) (*BrowserInstance, error) {
	// Check if instance already running
	if _, exists := l.instances.Load(workspaceID); exists {
		return nil, fmt.Errorf("browser instance already running for workspace %s", workspaceID)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring lock
	if _, exists := l.instances.Load(workspaceID); exists {
		return nil, fmt.Errorf("browser instance already running for workspace %s", workspaceID)
	}

	// Generate deterministic fingerprint for this workspace
	fp := GenerateFingerprint(workspaceID)

	// Create isolated profile directory
	profilePath, err := l.profile.CreateProfile(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile for workspace %s: %w", workspaceID, err)
	}

	// Build Chromium args with anti-fingerprinting flags
	args := l.buildArgs(fp, profilePath)

	// Create and configure the command
	cmd := exec.CommandContext(ctx, l.binPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group for clean shutdown
	}

	// Apply timezone spoofing via TZ environment variable.
	// Chromium does not have a --timezone flag, so we set TZ at the
	// process level to override the system timezone for this workspace.
	cmd.Env = append(os.Environ(), "TZ="+fp.Timezone)

	l.logger.Info("launching chromium", "workspace_id", workspaceID, "profile_path", profilePath, "pid_pending", true)

	// Start the process
	if err := l.runner.Start(cmd); err != nil {
		l.logger.Error("failed to start chromium", "workspace_id", workspaceID, "error", err)
		return nil, fmt.Errorf("failed to start chromium for workspace %s: %w", workspaceID, err)
	}

	if cmd.Process == nil {
		return nil, fmt.Errorf("chromium process is nil after start for workspace %s", workspaceID)
	}

	instance := &BrowserInstance{
		PID:         cmd.Process.Pid,
		Cmd:         cmd,
		WorkspaceID: workspaceID,
		Fingerprint: fp,
		ProfilePath: profilePath,
	}

	l.instances.Store(workspaceID, instance)

	l.logger.Info("chromium launched", "workspace_id", workspaceID, "pid", instance.PID, "viewport", fmt.Sprintf("%dx%d", fp.ViewportWidth, fp.ViewportHeight))

	return instance, nil
}

// Stop terminates a running Chromium instance for the given workspace.
// It kills the process group and removes the profile directory.
func (l *ChromiumLauncher) Stop(ctx context.Context, workspaceID string) error {
	val, exists := l.instances.LoadAndDelete(workspaceID)
	if !exists {
		return fmt.Errorf("no browser instance running for workspace %s", workspaceID)
	}

	instance, ok := val.(*BrowserInstance)
	if !ok {
		return fmt.Errorf("invalid instance type for workspace %s", workspaceID)
	}

	l.logger.Info("stopping chromium", "workspace_id", workspaceID, "pid", instance.PID)

	// Kill the process
	if instance.Cmd != nil && instance.Cmd.Process != nil {
		if err := l.runner.ProcessKill(instance.Cmd.Process); err != nil {
			l.logger.Warn("failed to kill chromium process", "workspace_id", workspaceID, "pid", instance.PID, "error", err)
		}
	}

	// Clean up profile directory
	if err := l.profile.DeleteProfile(workspaceID); err != nil {
		l.logger.Warn("failed to delete profile directory", "workspace_id", workspaceID, "error", err)
	} else {
		l.logger.Info("profile directory deleted", "workspace_id", workspaceID)
	}

	l.logger.Info("chromium stopped", "workspace_id", workspaceID, "pid", instance.PID)

	return nil
}

// GetInstance returns the running browser instance for a workspace, if any.
func (l *ChromiumLauncher) GetInstance(workspaceID string) (*BrowserInstance, bool) {
	val, exists := l.instances.Load(workspaceID)
	if !exists {
		return nil, false
	}
	instance, ok := val.(*BrowserInstance)
	if !ok {
		return nil, false
	}
	return instance, true
}

// StopAll terminates all running Chromium instances. Used for graceful shutdown.
func (l *ChromiumLauncher) StopAll(ctx context.Context) error {
	var lastErr error

	l.instances.Range(func(key, val interface{}) bool {
		workspaceID, ok := key.(string)
		if !ok {
			return true
		}

		if err := l.Stop(ctx, workspaceID); err != nil {
			l.logger.Error("failed to stop chromium during shutdown", "workspace_id", workspaceID, "error", err)
			lastErr = err
		}
		return true
	})

	return lastErr
}

// buildArgs constructs the Chromium command-line arguments with anti-fingerprinting flags.
func (l *ChromiumLauncher) buildArgs(fp Fingerprint, profilePath string) []string {
	return []string{
		// Headless mode
		"--headless=new",
		"--no-sandbox",
		"--disable-gpu",

		// Anti-fingerprinting: disable leaky APIs
		"--disable-web-security",
		"--disable-features=WebRTC,QUIC,TranslateUI",
		"--disable-dns-prefetch",
		"--no-first-run",
		"--no-default-browser-check",

		// Deterministic viewport
		fmt.Sprintf("--window-size=%d,%d", fp.ViewportWidth, fp.ViewportHeight),

		// Isolated profile
		fmt.Sprintf("--user-data-dir=%s", profilePath),

		// Spoofed user agent
		fmt.Sprintf("--user-agent=%s", fp.UserAgent),

		// Language spoofing (timezone applied via TZ env var on the process)
		fmt.Sprintf("--lang=%s", fp.Language),

		// Canvas fingerprint mitigation: prevent canvas-based fingerprinting.
		// Phase 2 will add a Chromium extension for canvas noise injection
		// that adds random pixel noise to canvas reads.
		"--disable-reading-from-canvas",

		// WebGL mitigation: disabled to prevent WebGL fingerprinting.
		// Phase 2 will add a Chromium extension for WebGL noise injection
		// that adds random noise to rendering output.
		"--disable-webgl",
		"--disable-webgl-image-chromium",

		// Font fingerprint baseline: reduce font enumeration surface.
		// Phase 2 will add per-workspace custom font lists for full
		// font virtualization.
		"--disable-font-subpixel-positioning",
		"--disable-local-fonts",

		// Disable background timers and other noise sources
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-renderer-backgrounding",
		"--disable-ipc-flooding-protection",

		// Disable various features that can leak identity
		"--disable-component-update",
		"--disable-default-apps",
		"--disable-extensions",
		"--disable-sync",
		"--disable-translate",
		"--metrics-recording-only",
		"--no-pings",

		// Remote debugging port (for streaming service to connect)
		"--remote-debugging-port=0",

		// Use a blank page to start
		"about:blank",
	}
}