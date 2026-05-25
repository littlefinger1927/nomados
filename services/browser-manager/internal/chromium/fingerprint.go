package chromium

import (
	"hash/fnv"
	"math/rand"
)

// Fingerprint holds per-workspace anti-fingerprinting browser configuration.
// Each workspace receives a unique but deterministic fingerprint derived from
// its workspace ID, ensuring consistency across restarts while preventing
// cross-workspace correlation.
type Fingerprint struct {
	ViewportWidth  int
	ViewportHeight int
	UserAgent      string
	Timezone       string
	Language       string
	Platform       string
}

// FingerprintPool contains realistic browser fingerprint variations
// to avoid generating obviously synthetic or uniform profiles.
var FingerprintPool = struct {
	UserAgents []string
	Timezones  []string
	Languages  []string
	Platforms  []string
	Viewports  [][2]int
}{
	UserAgents: []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	},
	Timezones: []string{
		"America/New_York",
		"America/Chicago",
		"America/Denver",
		"America/Los_Angeles",
		"America/Phoenix",
		"Europe/London",
		"Europe/Paris",
		"Europe/Berlin",
		"Asia/Tokyo",
		"Asia/Shanghai",
		"Australia/Sydney",
		"Pacific/Auckland",
	},
	Languages: []string{
		"en-US,en;q=0.9",
		"en-GB,en;q=0.9",
		"en-US,en;q=0.8",
		"en,en;q=0.9",
		"en-US,en;q=0.9,fr;q=0.8",
		"en-US,en;q=0.9,de;q=0.7",
		"en-US,en;q=0.9,es;q=0.8",
	},
	Platforms: []string{
		"Win32",
		"MacIntel",
		"Linux x86_64",
	},
	Viewports: [][2]int{
		{1920, 1080},
		{1366, 768},
		{1536, 864},
		{1440, 900},
		{1280, 720},
	},
}

// GenerateFingerprint creates a deterministic but unique fingerprint for
// a workspace. The same workspaceID always produces the same fingerprint,
// preventing fingerprint drift across sessions while ensuring different
// workspaces get different fingerprints.
func GenerateFingerprint(workspaceID string) Fingerprint {
	h := fnv.New64a()
	h.Write([]byte(workspaceID))
	seed := h.Sum64()

	r := rand.New(rand.NewSource(int64(seed))) //nolint:gosec // deterministic generation is intentional

	viewports := FingerprintPool.Viewports
	viewport := viewports[r.Intn(len(viewports))]

	return Fingerprint{
		ViewportWidth:  viewport[0],
		ViewportHeight: viewport[1],
		UserAgent:      FingerprintPool.UserAgents[r.Intn(len(FingerprintPool.UserAgents))],
		Timezone:       FingerprintPool.Timezones[r.Intn(len(FingerprintPool.Timezones))],
		Language:        FingerprintPool.Languages[r.Intn(len(FingerprintPool.Languages))],
		Platform:       FingerprintPool.Platforms[r.Intn(len(FingerprintPool.Platforms))],
	}
}