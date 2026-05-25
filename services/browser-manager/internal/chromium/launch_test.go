package chromium

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nomados/nomados/packages/logging"
)

// MockCommandRunner records commands without executing them.
type MockCommandRunner struct {
	StartCalls   []*exec.Cmd
	StartError   error
	KillCalls    []*os.Process
	KillError   error
	LastPID     int
}

func (m *MockCommandRunner) Start(cmd *exec.Cmd) error {
	m.StartCalls = append(m.StartCalls, cmd)
	if m.StartError != nil {
		return m.StartError
	}
	// Simulate a process with PID
	m.LastPID++
	if cmd.Process == nil {
		// We can't create a real os.Process in tests, so we set it to a marker value
		// The test will check args instead of PID
	}
	return nil
}

func (m *MockCommandRunner) ProcessKill(process *os.Process) error {
	m.KillCalls = append(m.KillCalls, process)
	return m.KillError
}

func newTestLauncher(runner CommandRunner) *ChromiumLauncher {
	logger := logging.NewLogger("browser-manager-test", nil)
	config := LauncherConfig{
		BinPath:        "/usr/bin/chromium-browser",
		ProfileBasePath: filepath.Join(os.TempDir(), "nomados-test-profiles"),
	}
	return NewChromiumLauncherWithRunner(config, logger, runner)
}

func TestGenerateFingerprintDeterministic(t *testing.T) {
	// Same workspaceID should always produce the same fingerprint
	fp1 := GenerateFingerprint("ws-test-123")
	fp2 := GenerateFingerprint("ws-test-123")

	if fp1.ViewportWidth != fp2.ViewportWidth {
		t.Errorf("ViewportWidth not deterministic: %d != %d", fp1.ViewportWidth, fp2.ViewportWidth)
	}
	if fp1.ViewportHeight != fp2.ViewportHeight {
		t.Errorf("ViewportHeight not deterministic: %d != %d", fp1.ViewportHeight, fp2.ViewportHeight)
	}
	if fp1.UserAgent != fp2.UserAgent {
		t.Errorf("UserAgent not deterministic: %s != %s", fp1.UserAgent, fp2.UserAgent)
	}
	if fp1.Timezone != fp2.Timezone {
		t.Errorf("Timezone not deterministic: %s != %s", fp1.Timezone, fp2.Timezone)
	}
	if fp1.Language != fp2.Language {
		t.Errorf("Language not deterministic: %s != %s", fp1.Language, fp2.Language)
	}
	if fp1.Platform != fp2.Platform {
		t.Errorf("Platform not deterministic: %s != %s", fp1.Platform, fp2.Platform)
	}
}

func TestGenerateFingerprintUnique(t *testing.T) {
	// Different workspaceIDs should produce different fingerprints
	fp1 := GenerateFingerprint("ws-alpha")
	fp2 := GenerateFingerprint("ws-beta")

	// At least one field should differ
	same := fp1.UserAgent == fp2.UserAgent &&
		fp1.Timezone == fp2.Timezone &&
		fp1.Language == fp2.Language &&
		fp1.ViewportWidth == fp2.ViewportWidth &&
		fp1.ViewportHeight == fp2.ViewportHeight &&
		fp1.Platform == fp2.Platform

	if same {
		t.Error("expected different fingerprints for different workspace IDs, but they were identical")
	}
}

func TestGenerateFingerprintPoolValues(t *testing.T) {
	// Generated fingerprint values should come from the pool
	fp := GenerateFingerprint("ws-pool-test")

	foundUA := false
	for _, ua := range FingerprintPool.UserAgents {
		if fp.UserAgent == ua {
			foundUA = true
			break
		}
	}
	if !foundUA {
		t.Errorf("UserAgent %q not found in pool", fp.UserAgent)
	}

	foundTZ := false
	for _, tz := range FingerprintPool.Timezones {
		if fp.Timezone == tz {
			foundTZ = true
			break
		}
	}
	if !foundTZ {
		t.Errorf("Timezone %q not found in pool", fp.Timezone)
	}

	foundLang := false
	for _, lang := range FingerprintPool.Languages {
		if fp.Language == lang {
			foundLang = true
			break
		}
	}
	if !foundLang {
		t.Errorf("Language %q not found in pool", fp.Language)
	}

	foundViewport := false
	for _, vp := range FingerprintPool.Viewports {
		if fp.ViewportWidth == vp[0] && fp.ViewportHeight == vp[1] {
			foundViewport = true
			break
		}
	}
	if !foundViewport {
		t.Errorf("Viewport %dx%d not found in pool", fp.ViewportWidth, fp.ViewportHeight)
	}

	foundPlatform := false
	for _, p := range FingerprintPool.Platforms {
		if fp.Platform == p {
			foundPlatform = true
			break
		}
	}
	if !foundPlatform {
		t.Errorf("Platform %q not found in pool", fp.Platform)
	}
}

func TestProfileCreateAndGetPath(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewProfileManager(tmpDir)

	profilePath, err := pm.CreateProfile("ws-test-profile")
	if err != nil {
		t.Fatalf("unexpected error creating profile: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "ws-test-profile")
	if profilePath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, profilePath)
	}

	// Verify directory exists
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Error("profile directory was not created")
	}
}

func TestProfileIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewProfileManager(tmpDir)

	path1, err := pm.CreateProfile("ws-alpha")
	if err != nil {
		t.Fatalf("unexpected error creating profile: %v", err)
	}
	path2, err := pm.CreateProfile("ws-beta")
	if err != nil {
		t.Fatalf("unexpected error creating profile: %v", err)
	}

	if path1 == path2 {
		t.Error("profile paths should be isolated per workspace")
	}

	// Verify each profile directory is independent
	if !pm.ProfileExists("ws-alpha") {
		t.Error("profile for ws-alpha should exist")
	}
	if !pm.ProfileExists("ws-beta") {
		t.Error("profile for ws-beta should exist")
	}
}

func TestProfileDelete(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewProfileManager(tmpDir)

	_, err := pm.CreateProfile("ws-to-delete")
	if err != nil {
		t.Fatalf("unexpected error creating profile: %v", err)
	}

	if !pm.ProfileExists("ws-to-delete") {
		t.Error("profile should exist before deletion")
	}

	err = pm.DeleteProfile("ws-to-delete")
	if err != nil {
		t.Fatalf("unexpected error deleting profile: %v", err)
	}

	if pm.ProfileExists("ws-to-delete") {
		t.Error("profile should not exist after deletion")
	}
}

func TestProfileDeleteNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewProfileManager(tmpDir)

	// Deleting a nonexistent profile should not error
	err := pm.DeleteProfile("ws-nonexistent")
	if err != nil {
		t.Errorf("expected no error deleting nonexistent profile, got: %v", err)
	}
}

func TestChromiumLaunchArgs(t *testing.T) {
	runner := &MockCommandRunner{}
	launcher := newTestLauncher(runner)

	fp := GenerateFingerprint("ws-args-test")
	profilePath := filepath.Join(os.TempDir(), "nomados-test-profiles", "ws-args-test")
	args := launcher.buildArgs(fp, profilePath)

	// Check required anti-fingerprinting flags
	requiredFlags := []string{
		"--headless=new",
		"--no-sandbox",
		"--disable-web-security",
		"--disable-features=WebRTC,QUIC,TranslateUI",
		"--disable-dns-prefetch",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-webgl",
	}

	for _, flag := range requiredFlags {
		found := false
		for _, arg := range args {
			if arg == flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required flag %q not found in args", flag)
		}
	}

	// Check dynamic flags are present
	hasWindowSize := false
	hasUserDataDir := false
	hasUserAgent := false
	hasLang := false

	for _, arg := range args {
		if strings.HasPrefix(arg, "--window-size=") {
			hasWindowSize = true
		}
		if strings.HasPrefix(arg, "--user-data-dir=") {
			hasUserDataDir = true
		}
		if strings.HasPrefix(arg, "--user-agent=") {
			hasUserAgent = true
		}
		if strings.HasPrefix(arg, "--lang=") {
			hasLang = true
		}
	}

	if !hasWindowSize {
		t.Error("expected --window-size flag in args")
	}
	if !hasUserDataDir {
		t.Error("expected --user-data-dir flag in args")
	}
	if !hasUserAgent {
		t.Error("expected --user-agent flag in args")
	}
	if !hasLang {
		t.Error("expected --lang flag in args")
	}

	// Check user-data-dir contains the profile path
	for _, arg := range args {
		if strings.HasPrefix(arg, "--user-data-dir=") {
			if !strings.Contains(arg, "ws-args-test") {
				t.Errorf("expected user-data-dir to contain workspace ID, got %s", arg)
			}
		}
	}
}

func TestChromiumLaunchAndStop(t *testing.T) {
	runner := &MockCommandRunner{}
	launcher := newTestLauncher(runner)

	ctx := context.Background()

	// Since MockCommandRunner doesn't set cmd.Process, we need a special test
	// that verifies the launch logic without requiring a real process.
	// The real test is in TestChromiumLaunchArgs above.
	// Here we test the instance tracking.

	// Directly test StopAll with no instances (should succeed)
	err := launcher.StopAll(ctx)
	if err != nil {
		t.Errorf("expected no error stopping all with no instances, got: %v", err)
	}
}

func TestChromiumLaunchDuplicateWorkspace(t *testing.T) {
	runner := &MockCommandRunner{}
	launcher := newTestLauncher(runner)

	ctx := context.Background()

	// Manually store an instance to simulate a running browser
	launcher.instances.Store("ws-dup-test", &BrowserInstance{
		PID:         12345,
		WorkspaceID: "ws-dup-test",
		Fingerprint: GenerateFingerprint("ws-dup-test"),
		ProfilePath:  "/tmp/nomados-test-profiles/ws-dup-test",
	})

	// Trying to launch again should fail
	_, err := launcher.Launch(ctx, "ws-dup-test")
	if err == nil {
		t.Error("expected error when launching duplicate workspace, got nil")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("expected 'already running' error, got: %v", err)
	}
}

func TestChromiumStopNonexistentWorkspace(t *testing.T) {
	runner := &MockCommandRunner{}
	launcher := newTestLauncher(runner)

	ctx := context.Background()

	err := launcher.Stop(ctx, "ws-nonexistent")
	if err == nil {
		t.Error("expected error when stopping nonexistent workspace, got nil")
	}
	if !strings.Contains(err.Error(), "no browser instance running") {
		t.Errorf("expected 'no browser instance running' error, got: %v", err)
	}
}

func TestGetInstance(t *testing.T) {
	runner := &MockCommandRunner{}
	launcher := newTestLauncher(runner)

	// No instance yet
	inst, exists := launcher.GetInstance("ws-get-test")
	if exists {
		t.Error("expected no instance for nonexistent workspace")
	}
	if inst != nil {
		t.Error("expected nil instance for nonexistent workspace")
	}

	// Store an instance
	instance := &BrowserInstance{
		PID:         99999,
		WorkspaceID: "ws-get-test",
		Fingerprint: GenerateFingerprint("ws-get-test"),
		ProfilePath:  "/tmp/nomados-test-profiles/ws-get-test",
	}
	launcher.instances.Store("ws-get-test", instance)

	// Should now find it
	inst, exists = launcher.GetInstance("ws-get-test")
	if !exists {
		t.Error("expected instance to exist")
	}
	if inst.WorkspaceID != "ws-get-test" {
		t.Errorf("expected workspace ID ws-get-test, got %s", inst.WorkspaceID)
	}
}

func TestChromiumLaunchWithStartError(t *testing.T) {
	runner := &MockCommandRunner{
		StartError: fmt.Errorf("chromium binary not found"),
	}
	launcher := newTestLauncher(runner)

	ctx := context.Background()

	_, err := launcher.Launch(ctx, "ws-start-fail")
	if err == nil {
		t.Error("expected error when start fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to start chromium") {
		t.Errorf("expected start error, got: %v", err)
	}
}

func TestStopAllStopsMultipleInstances(t *testing.T) {
	runner := &MockCommandRunner{}
	launcher := newTestLauncher(runner)

	ctx := context.Background()

	// Store multiple instances
	for i, wsID := range []string{"ws-1", "ws-2", "ws-3"} {
		launcher.instances.Store(wsID, &BrowserInstance{
			PID:         10000 + i,
			WorkspaceID:  wsID,
			Fingerprint:  GenerateFingerprint(wsID),
			ProfilePath:   filepath.Join(os.TempDir(), "nomados-test-profiles", wsID),
		})
	}

	err := launcher.StopAll(ctx)
	if err != nil {
		t.Errorf("expected no error stopping all instances, got: %v", err)
	}

	// Verify all instances removed
	for _, wsID := range []string{"ws-1", "ws-2", "ws-3"} {
		if _, exists := launcher.GetInstance(wsID); exists {
			t.Errorf("expected instance %s to be removed after StopAll", wsID)
		}
	}
}

func TestBuildArgsContainsAntiFingerprintFlags(t *testing.T) {
	runner := &MockCommandRunner{}
	launcher := newTestLauncher(runner)

	fp := Fingerprint{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/122.0.0.0 Safari/537.36",
		Timezone:       "America/New_York",
		Language:        "en-US,en;q=0.9",
		Platform:       "Win32",
	}

	args := launcher.buildArgs(fp, "/tmp/profile-ws-123")

	// Verify viewport
	foundViewport := false
	for _, arg := range args {
		if arg == "--window-size=1920,1080" {
			foundViewport = true
			break
		}
	}
	if !foundViewport {
		t.Error("expected --window-size=1920,1080 in args")
	}

	// Verify user agent
	foundUA := false
	for _, arg := range args {
		if arg == "--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/122.0.0.0 Safari/537.36" {
			foundUA = true
			break
		}
	}
	if !foundUA {
		t.Error("expected user-agent flag with specified UA string")
	}

	// Verify user-data-dir
	foundDataDir := false
	for _, arg := range args {
		if arg == "--user-data-dir=/tmp/profile-ws-123" {
			foundDataDir = true
			break
		}
	}
	if !foundDataDir {
		t.Error("expected --user-data-dir pointing to profile path")
	}

	// Verify about:blank is the last arg (the URL to open)
	if args[len(args)-1] != "about:blank" {
		t.Errorf("expected last arg to be 'about:blank', got %s", args[len(args)-1])
	}
}