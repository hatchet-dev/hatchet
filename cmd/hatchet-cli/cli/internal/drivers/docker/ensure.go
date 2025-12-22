package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"

	cerrdefs "github.com/containerd/errdefs"
)

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
	// Check if container already exists
	existing, err := d.apiClient.ContainerInspect(ctx, containerName)

	if err == nil {
		// Container exists - check if it needs to be recreated
		needsRecreate := false

		// Check if image has changed
		if existing.Config.Image != imageName {
			needsRecreate = true
		}

		// Check if the image ID has changed (for :latest tags)
		imageInspect, err := d.apiClient.ImageInspect(ctx, imageName)
		if err == nil && existing.Image != imageInspect.ID {
			needsRecreate = true
		}

		// TODO: more checks, ports, volumes, etc
		// We can potentially hash the inputs and write them to a label, then compare the label

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
