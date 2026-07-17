package cli

import (
	"fmt"

	"github.com/google/uuid"
)

// TelemetryEnabled reports whether anonymous usage telemetry is enabled. It
// defaults to true when unset in the config file.
func TelemetryEnabled() bool {
	if CLIConfig == nil {
		return true
	}
	if CLIConfig.Telemetry.Enabled == nil {
		return true
	}
	return *CLIConfig.Telemetry.Enabled
}

// TelemetryEndpoint returns the configured telemetry endpoint.
func TelemetryEndpoint() string {
	if CLIConfig != nil && CLIConfig.Telemetry.Endpoint != "" {
		return CLIConfig.Telemetry.Endpoint
	}
	return "https://security.hatchet.run"
}

// EnsureAnonymousID returns the persistent, randomly generated install ID,
// generating and writing one to the CLI config file on first use. The ID is not
// derived from any machine, user or network identity. Returns an empty string if
// it cannot be persisted, in which case callers should skip telemetry.
func EnsureAnonymousID() string {
	if CLIConfig != nil && CLIConfig.Telemetry.AnonymousID != "" {
		return CLIConfig.Telemetry.AnonymousID
	}

	id := uuid.NewString()

	if err := persistAnonymousID(id); err != nil {
		return ""
	}

	if CLIConfig != nil {
		CLIConfig.Telemetry.AnonymousID = id
	}

	return id
}

func persistAnonymousID(id string) error {
	if CLIConfigViper == nil || CLIConfigFilePath == "" {
		return fmt.Errorf("cli config not initialized")
	}

	release, err := acquireLock()
	if err != nil {
		return err
	}
	defer release()

	CLIConfigViper.Set("telemetry.anonymousId", id)

	return CLIConfigViper.WriteConfigAs(CLIConfigFilePath)
}
