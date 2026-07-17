package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func TelemetryEnabled() bool {
	if CLIConfig == nil || CLIConfig.Telemetry.Enabled == nil {
		return true
	}
	return *CLIConfig.Telemetry.Enabled
}

func TelemetryEndpoint() string {
	if CLIConfig != nil && CLIConfig.Telemetry.Endpoint != "" {
		return CLIConfig.Telemetry.Endpoint
	}
	return "https://security.hatchet.run"
}

func EnsureAnonymousID() string {
	if CLIConfig != nil && CLIConfig.Telemetry.AnonymousID != "" {
		return CLIConfig.Telemetry.AnonymousID
	}

	if HomeDir == "" {
		return ""
	}

	path := filepath.Join(HomeDir, ".hatchet", "telemetry_id")

	if b, err := os.ReadFile(path); err == nil {
		if id := strings.TrimSpace(string(b)); id != "" {
			return id
		}
	}

	id := uuid.NewString()

	if err := os.WriteFile(path, []byte(id), 0600); err != nil {
		return ""
	}

	return id
}
