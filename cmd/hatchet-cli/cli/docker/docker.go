package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
)

// defaults
const (
	defaultNetworkName  = "hatchet_network"
	defaultPostgresName = "postgres"
	defaultHatchetName  = "hatchet"
	defaultProjectName  = "hatchet"
	defaultServiceName  = "hatchet"
)

type Opt func(*DockerDriver) error

type DockerDriverOpts struct {
	NetworkName  string
	PostgresName string
	HatchetName  string
	ProjectName  string
	ServiceName  string
}

type DockerDriver struct {
	l             *zerolog.Logger // nolint: unused
	apiClient     *client.Client
	initNetworkId string

	opts DockerDriverOpts
}

func WithNetworkName(name string) Opt {
	return func(d *DockerDriver) error {
		d.opts.NetworkName = name
		return nil
	}
}

func WithPostgresName(name string) Opt {
	return func(d *DockerDriver) error {
		d.opts.PostgresName = name
		return nil
	}
}

func WithHatchetName(name string) Opt {
	return func(d *DockerDriver) error {
		d.opts.HatchetName = name
		return nil
	}
}

func WithProjectName(name string) Opt {
	return func(d *DockerDriver) error {
		d.opts.ProjectName = name
		return nil
	}
}

func WithServiceName(name string) Opt {
	return func(d *DockerDriver) error {
		d.opts.ServiceName = name
		return nil
	}
}

func NewDockerDriver(ctx context.Context, optFns ...Opt) (*DockerDriver, error) {
	d := &DockerDriver{
		opts: DockerDriverOpts{
			NetworkName:  defaultNetworkName,
			PostgresName: defaultPostgresName,
			HatchetName:  defaultHatchetName,
			ProjectName:  defaultProjectName,
			ServiceName:  defaultServiceName,
		},
	}

	for _, fn := range optFns {
		if err := fn(d); err != nil {
			return nil, err
		}
	}

	if err := d.init(ctx); err != nil {
		return nil, fmt.Errorf("could not initialize docker driver: %w", err)
	}

	return d, nil
}

func (d *DockerDriver) init(ctx context.Context) error {
	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("could not create docker client: %w", err)
	}
	defer apiClient.Close()

	d.apiClient = apiClient

	return d.initNetwork(ctx)
}

// initNetwork atomically creates the docker network if it does not already exist
func (d *DockerDriver) initNetwork(ctx context.Context) error {
	// check if network already exists
	networks, err := d.apiClient.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", d.opts.NetworkName)),
	})
	if err != nil {
		return fmt.Errorf("could not list docker networks: %w", err)
	}
	var networkId string

	if len(networks) > 0 {
		networkId = networks[0].ID
	} else {
		networkResp, err := d.apiClient.NetworkCreate(ctx, d.opts.NetworkName, network.CreateOptions{
			Driver: "bridge",
			Labels: map[string]string{
				"com.docker.compose.project": d.opts.ProjectName, // optional: mimic compose labels
			},
		})

		if err != nil {
			return fmt.Errorf("could not create network: %w", err)
		}

		networkId = networkResp.ID
	}

	d.initNetworkId = networkId
	return nil
}

func canonicalContainerName(projectName, serviceName string) string {
	return fmt.Sprintf("%s-%s-1", projectName, serviceName)
}
