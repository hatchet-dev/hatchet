package hatchet

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// MinEngineVersion defines minimum engine versions for feature support.
var MinEngineVersion = struct {
	SlotConfig      string
	DurableEviction string
	Observability   string
}{
	SlotConfig:      "v0.78.23",
	DurableEviction: "v0.80.0",
	Observability:   "v0.82.0",
}

// GetEngineVersion retrieves the engine version from the server.
func (c *Client) GetEngineVersion(ctx context.Context) (string, error) {
	version, err := c.legacyClient.Dispatcher().GetVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get engine version: %w", err)
	}
	return version, nil
}

// SupportsDurableEviction checks whether the engine version supports durable eviction.
func SupportsDurableEviction(engineVersion string) (bool, error) {
	return !semverLessThan(engineVersion, MinEngineVersion.DurableEviction), nil
}

// parseSemver parses a semver string like "v0.80.0" into (major, minor, patch).
func parseSemver(v string) (int, int, int, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid semver: %s", v)
	}

	// Handle pre-release suffix in patch (e.g. "0-alpha.1")
	patchStr := strings.SplitN(parts[2], "-", 2)[0]

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid semver major: %s", parts[0])
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid semver minor: %s", parts[1])
	}
	patch, err := strconv.Atoi(patchStr)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid semver patch: %s", patchStr)
	}
	return major, minor, patch, nil
}

// semverLessThan returns true if version a is strictly less than version b.
func semverLessThan(a, b string) bool {
	aMaj, aMin, aPatch, err := parseSemver(a)
	if err != nil {
		return false
	}
	bMaj, bMin, bPatch, err := parseSemver(b)
	if err != nil {
		return false
	}
	if aMaj != bMaj {
		return aMaj < bMaj
	}
	if aMin != bMin {
		return aMin < bMin
	}
	return aPatch < bPatch
}
