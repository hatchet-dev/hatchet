package docker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/client"
)

type DockerDriverOpt func(*DockerDriverOpts) error

type DockerDriverOpts struct {
}

type DockerDriver struct {
	apiClient *client.Client
}

func NewDockerDriver(ctx context.Context, optFns ...DockerDriverOpt) (*DockerDriver, error) {
	opts := &DockerDriverOpts{}

	for _, fn := range optFns {
		if err := fn(opts); err != nil {
			return nil, err
		}
	}

	// TODO: pass opts to init
	d := &DockerDriver{}

	if err := d.init(); err != nil {
		return nil, fmt.Errorf("could not initialize docker driver: %w", err)
	}

	return d, nil
}

func (d *DockerDriver) init() error {
	clientOpts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}

	// If DOCKER_HOST is not set, try to resolve from active Docker context
	if os.Getenv("DOCKER_HOST") == "" {
		if host := resolveDockerHostFromContext(); host != "" {
			clientOpts = append(clientOpts, client.WithHost(host))
		}
	}

	apiClient, err := client.NewClientWithOpts(clientOpts...)
	if err != nil {
		return fmt.Errorf("could not create docker client: %w", err)
	}

	d.apiClient = apiClient

	return nil
}

// dockerConfig represents the relevant fields from ~/.docker/config.json
type dockerConfig struct {
	CurrentContext string `json:"currentContext"`
}

// contextMetadata represents the relevant fields from Docker context metadata
type contextMetadata struct {
	Name      string `json:"Name"`
	Endpoints struct {
		Docker struct {
			Host string `json:"Host"`
		} `json:"docker"`
	} `json:"Endpoints"`
}

// resolveDockerHostFromContext reads the active Docker context and returns
// the Docker host endpoint. Returns empty string if:
// - Docker config cannot be read
// - No context is set or context is "default"
// - Context metadata cannot be read
func resolveDockerHostFromContext() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Read Docker config
	configPath := filepath.Join(homeDir, ".docker", "config.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	var config dockerConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return ""
	}

	// No context or default context - use default behavior
	if config.CurrentContext == "" || config.CurrentContext == "default" {
		return ""
	}

	// Read context metadata
	// Docker stores non-default context metadata under ~/.docker/contexts/meta/.
	hash := sha256.Sum256([]byte(config.CurrentContext))
	hashStr := hex.EncodeToString(hash[:])
	metaPath := filepath.Join(homeDir, ".docker", "contexts", "meta", hashStr, "meta.json")

	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return ""
	}

	var meta contextMetadata
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return ""
	}

	return meta.Endpoints.Docker.Host
}
