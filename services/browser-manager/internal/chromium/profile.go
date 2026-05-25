package chromium

import (
	"fmt"
	"os"
	"path/filepath"
)

// ProfileManager manages isolated Chromium user data directories.
// Each workspace receives its own profile directory for cookies,
// localStorage, cache, and other browser state, preventing any
// cross-workspace data leakage.
type ProfileManager struct {
	basePath string
}

// NewProfileManager creates a new ProfileManager with the given base directory.
// Profile directories are created under basePath/<workspaceID>.
func NewProfileManager(basePath string) *ProfileManager {
	return &ProfileManager{
		basePath: basePath,
	}
}

// GetProfilePath returns the absolute path to a workspace's profile directory.
func (pm *ProfileManager) GetProfilePath(workspaceID string) string {
	return filepath.Join(pm.basePath, workspaceID)
}

// CreateProfile creates an isolated profile directory for a workspace.
// It creates the directory and any necessary parent directories.
// Returns the path to the created directory.
func (pm *ProfileManager) CreateProfile(workspaceID string) (string, error) {
	profilePath := pm.GetProfilePath(workspaceID)

	if err := os.MkdirAll(profilePath, 0700); err != nil {
		return "", fmt.Errorf("failed to create profile directory for workspace %s: %w", workspaceID, err)
	}

	return profilePath, nil
}

// DeleteProfile removes a workspace's profile directory and all its contents.
// This permanently deletes cookies, localStorage, cache, and any other
// browser state for the workspace.
func (pm *ProfileManager) DeleteProfile(workspaceID string) error {
	profilePath := pm.GetProfilePath(workspaceID)

	if err := os.RemoveAll(profilePath); err != nil {
		return fmt.Errorf("failed to delete profile directory for workspace %s: %w", workspaceID, err)
	}

	return nil
}

// ProfileExists checks whether a profile directory exists for a workspace.
func (pm *ProfileManager) ProfileExists(workspaceID string) bool {
	profilePath := pm.GetProfilePath(workspaceID)

	_, err := os.Stat(profilePath)
	return err == nil
}