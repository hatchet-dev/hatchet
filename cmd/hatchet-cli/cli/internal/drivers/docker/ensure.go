package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/opencontainers/go-digest"

	cerrdefs "github.com/containerd/errdefs"
)

const hatchetHashLabel = "hatchet-cli.opts.hash"

// ensureContainer creates a container if it doesn't exist, or updates it if it already exists.
// Returns the container ID.
func (d *DockerDriver) ensureContainer(
	ctx context.Context,
	containerName string,
	imageName string,
	containerConfig *container.Config,
	hostConfig *container.HostConfig,
	networkConfig *network.NetworkingConfig,
) (string, error) {
	newHash := getHashLabel(containerName, imageName, containerConfig, hostConfig, networkConfig)
	containerConfig.Labels[hatchetHashLabel] = newHash

	// Check if container already exists
	existing, err := d.apiClient.ContainerInspect(ctx, containerName)

	if err == nil {
		// Container exists - check if it needs to be recreated
		needsRecreate := false

		// get the current hash label
		currentHash, ok := existing.Config.Labels[hatchetHashLabel]

		if !ok || currentHash != newHash {
			needsRecreate = true
		}

		if needsRecreate {
			// Stop and remove existing container
			timeout := 10
			if err := d.apiClient.ContainerStop(ctx, existing.ID, container.StopOptions{Timeout: &timeout}); err != nil && !cerrdefs.IsNotFound(err) {
				return "", fmt.Errorf("could not stop existing container: %w", err)
			}

			if err := d.apiClient.ContainerRemove(ctx, existing.ID, container.RemoveOptions{Force: true}); err != nil && !cerrdefs.IsNotFound(err) {
				return "", fmt.Errorf("could not remove existing container: %w", err)
			}
		} else {
			// Container exists and doesn't need recreation - just ensure it's running
			if !existing.State.Running {
				if err := d.apiClient.ContainerStart(ctx, existing.ID, container.StartOptions{}); err != nil {
					return "", fmt.Errorf("could not start existing container: %w", err)
				}
			}
			return existing.ID, nil
		}
	} else if !cerrdefs.IsNotFound(err) {
		return "", fmt.Errorf("could not inspect container: %w", err)
	}

	// Create new container
	resp, err := d.apiClient.ContainerCreate(ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		containerName,
	)
	if err != nil {
		return "", fmt.Errorf("could not create container: %w", err)
	}

	if err := d.apiClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("could not start container: %w", err)
	}

	return resp.ID, nil
}

func (d *DockerDriver) stopContainer(ctx context.Context, containerName string) error {
	containers, err := d.apiClient.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", containerName)),
	})

	if err != nil {
		return fmt.Errorf("could not get container ID: %w", err)
	}

	if len(containers) == 0 {
		// Container doesn't exist
		return nil
	}

	containerId := containers[0].ID

	timeout := 10
	if err := d.apiClient.ContainerStop(ctx, containerId, container.StopOptions{Timeout: &timeout}); err != nil && !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("could not stop container: %w", err)
	}

	return nil
}

func (d *DockerDriver) ensureContainerIsHealthy(ctx context.Context, containerId string) error {
	inspect, err := d.apiClient.ContainerInspect(ctx, containerId)
	if err != nil {
		return fmt.Errorf("could not inspect container: %w", err)
	}

	if inspect.State.Health == nil {
		// no healthcheck defined
		return nil
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		inspect, err = d.apiClient.ContainerInspect(ctx, containerId)
		if err != nil {
			return fmt.Errorf("could not inspect container: %w", err)
		}

		if inspect.State.Health.Status == "healthy" {
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}

type Config struct {
	ContainerConfig *container.Config
	HostConfig      *container.HostConfig
	NetworkConfig   *network.NetworkingConfig
	ContainerName   string
	ImageName       string
}

func (c *Config) Hash() (string, error) {
	valueBytes, err := json.Marshal(c)

	if err != nil {
		return "", err
	}

	digester := digest.SHA512.Digester()

	if _, err := digester.Hash().Write(valueBytes); err != nil {
		return "", err
	}

	return digester.Digest().String(), nil
}

func getHashLabel(
	containerName string,
	imageName string,
	containerConfig *container.Config,
	hostConfig *container.HostConfig,
	networkConfig *network.NetworkingConfig,
) string {
	config := &Config{
		ContainerName:   containerName,
		ImageName:       imageName,
		ContainerConfig: containerConfig,
		HostConfig:      hostConfig,
		NetworkConfig:   networkConfig,
	}

	hash, err := config.Hash()

	if err != nil {
		// panic is ok here, we're in the CLI and this should never happen
		panic(fmt.Sprintf("could not hash hatchet-lite opts: %v", err))
	}

	return hash
}
