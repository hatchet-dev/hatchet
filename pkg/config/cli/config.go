package cli

import (
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// CLIConfig represents the global configuration for the Hatchet CLI. This is distinct from the
// CWD project configuration.
type CLIConfig struct {
	// ProfileFileName is the name of the profile file, defaults to "profiles.yaml"
	ProfileFileName string `mapstructure:"profileFileName" json:"profileFileName,omitempty" default:"profiles.yaml"`

	// Logger is the logging configuration for the CLI
	Logger CLIConfigLogger `mapstructure:"logger" json:"logger,omitempty"`
}

// ProfileFile represents a list of profiles in the profiles config file
type ProfileFile struct {
	Profiles map[string]Profile `mapstructure:"profiles"`
}

// Profile represents a single profile configuration
type Profile struct {
	TenantId     uuid.UUID `mapstructure:"tenantId" json:"tenantId"`
	Name         string    `mapstructure:"name" json:"name"`
	Token        string    `mapstructure:"token" json:"token"`
	ExpiresAt    time.Time `mapstructure:"expiresAt" json:"expiresAt"`
	ApiServerURL string    `mapstructure:"apiServerURL" json:"apiServerURL"`
	GrpcHostPort string    `mapstructure:"grpcHostPort" json:"grpcHostPort"`
	TLSStrategy  string    `mapstructure:"tlsStrategy" json:"tlsStrategy" default:"tls"`
}

type CLIConfigLogger struct {
	// Level is the logging level for the CLI. Possible values are: debug, info, warn, error
	Level string `mapstructure:"level" json:"level,omitempty" default:"warn"`

	// Format is the logging format for the CLI. Possible values are: text, json
	Format string `mapstructure:"format" json:"format,omitempty" default:"text"`

	// Prefix is an optional prefix for log lines
	Prefix string `mapstructure:"prefix" json:"prefix,omitempty"`
}

func BindAllEnv(v *viper.Viper) {
	_ = v.BindEnv("profileFileName", "HATCHET_CLI_PROFILE_FILE_NAME")

	// logger options
	_ = v.BindEnv("logger.level", "HATCHET_CLI_LOGGER_LEVEL")
	_ = v.BindEnv("logger.format", "HATCHET_CLI_LOGGER_FORMAT")
	_ = v.BindEnv("logger.prefix", "HATCHET_CLI_LOGGER_PREFIX")
}
