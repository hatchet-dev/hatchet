package worker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type WorkerConfig struct {
	Triggers []Trigger       `mapstructure:"triggers" json:"triggers,omitempty"`
	Dev      WorkerDevConfig `mapstructure:"dev" json:"dev,omitempty"`
}

type Trigger struct {
	// command to execute
	Command string `mapstructure:"command" json:"command"`

	// optional name (defaults to command)
	Name string `mapstructure:"name" json:"name,omitempty"`

	// optional description
	Description string `mapstructure:"description" json:"description,omitempty"`
}

type WorkerDevConfig struct {
	// commands to run before starting the worker
	PreCmds []string `mapstructure:"preCmds" json:"preCmds,omitempty"`

	// command to run the worker
	RunCmd string `mapstructure:"runCmd" json:"runCmd,omitempty"`

	// list of glob files to watch for reloads
	Files []string `mapstructure:"files" json:"files,omitempty"`

	// whether to reload on file changes
	Reload bool `mapstructure:"reload" json:"reload,omitempty"`
}

var workerViperConfig *viper.Viper

// LoadWorkerConfig loads the worker configuration from a hatchet.yaml file in the current working directory.
// Returns nil config if the file doesn't exist.
func LoadWorkerConfig() (*WorkerConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get current working directory: %w", err)
	}

	configPath := filepath.Join(cwd, "hatchet.yaml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}

	workerViperConfig = viper.New()
	workerViperConfig.SetConfigFile(configPath)
	workerViperConfig.SetConfigType("yaml")

	if err := workerViperConfig.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("could not read worker config file: %w", err)
	}

	config := &WorkerConfig{}
	if err := workerViperConfig.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("could not unmarshal worker config: %w", err)
	}

	return config, nil
}

// GetWorkerConfig returns the current worker viper config instance.
// Returns nil if LoadWorkerConfig has not been called.
func GetWorkerConfig() *viper.Viper {
	return workerViperConfig
}
