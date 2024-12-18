package config

import (
	"os"
	"path/filepath"

	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
	"github.com/spf13/viper"
)

const DEFAULT_PROFILE_CONFIG_DIR = ".hatchet"
const DEFAULT_PROFILE_CONFIG_FILE = "profiles.yaml"

type EngineMode string

const (
	EngineModeLocal EngineMode = "local"
	EngineModeCloud EngineMode = "cloud"
)

type ProfileConfigFile struct {
	Name       string     `mapstructure:"name" json:"name"`
	TenantId   string     `mapstructure:"tenantId" json:"tenantId,omitempty"`
	Token      string     `mapstructure:"token" json:"token,omitempty"`
	EngineMode EngineMode `mapstructure:"engineMode" json:"engineMode,omitempty" oneof:"cloud,local"`
	IsDefault  bool       `mapstructure:"isDefault" json:"isDefault,omitempty"`
	Namespace  string     `mapstructure:"namespace" json:"namespace,omitempty"`
}

type ProfilesConfigFile struct {
	Profiles []ProfileConfigFile `mapstructure:"profiles" json:"profiles,omitempty"`
}

// LoadProfiles loads the profiles configuration from the given files.
//
// Parameters:
//   - files: Optional byte slices representing the configuration files to load.
//
// Returns:
//   - *ProfilesConfigFile: The loaded ProfilesConfigFile.
//   - error: Any error that occurred during loading the configuration.
func LoadProfiles(files ...[]byte) (*ProfilesConfigFile, error) {
	configFile := &ProfilesConfigFile{}

	_, err := loaderutils.LoadConfigFromViper(BindAllEnv, configFile, files...)

	return configFile, err
}

// WriteProfiles writes the provided profiles configuration to a YAML file. If no path is provided,
// it writes to the default profile path (~/.hatchet/profiles.yaml). The profiles are written using
// Viper's YAML configuration format.
//
// Parameters:
//   - profiles: The ProfilesConfigFile containing the profiles to write
//   - path: Optional custom file path to write to. If nil, uses config.defaultProfilePath() path
//
// Returns:
//   - error: Any error that occurred during writing the configuration
func WriteProfiles(profiles *ProfilesConfigFile, path *string) error {
	v := viper.New()
	v.SetConfigType("yaml")

	v.Set("profiles", profiles.Profiles)

	if path == nil {
		p, err := defaultProfilePath()

		if err != nil {
			return err
		}

		path = &p
	}

	return v.WriteConfigAs(*path)
}

// BindAllEnv binds all environment variables to the given Viper instance.
// This function is used to configure Viper to automatically bind environment variables.
// It is typically used in the context of a command or application that needs to bind environment variables.
//
// Parameters:
//   - v: The Viper instance to bind environment variables to.
func BindAllEnv(v *viper.Viper) {
}

// defaultProfilePath returns the default path for the Hatchet profiles configuration file.
// It creates the necessary directory structure if it doesn't exist.
//
// The default path is ~/.hatchet/profiles.yaml where ~ represents the user's home directory as returned by os.UserHomeDir().
//
// Returns:
//   - string: The full path to the profiles configuration file
//   - error: Any error encountered while getting the home directory or creating directories
func defaultProfilePath() (string, error) {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		return "", err
	}

	profileDir := filepath.Join(homeDir, DEFAULT_PROFILE_CONFIG_DIR)

	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		if err := os.MkdirAll(profileDir, 0755); err != nil {
			return "", err
		}
	}

	profileFile := filepath.Join(profileDir, DEFAULT_PROFILE_CONFIG_FILE)

	return profileFile, nil
}
