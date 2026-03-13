package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	dockercontext "github.com/docker/go-sdk/context"
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

	// Resolve Docker host from the current Docker context.
	// This respects DOCKER_HOST, DOCKER_CONTEXT, and the active context in ~/.docker/config.json.
	if host, err := dockercontext.CurrentDockerHost(); err == nil && host != "" {
		clientOpts = append(clientOpts, client.WithHost(host))
	}

	apiClient, err := client.NewClientWithOpts(clientOpts...)
	if err != nil {
		return fmt.Errorf("could not create docker client: %w", err)
	}

	d.apiClient = apiClient

	return nil
}
