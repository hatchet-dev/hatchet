package docker

import (
	"context"
	"fmt"

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
	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("could not create docker client: %w", err)
	}
	defer apiClient.Close()

	d.apiClient = apiClient

	return nil
}
