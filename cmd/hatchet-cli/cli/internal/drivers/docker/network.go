package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

// initNetwork atomically creates the docker network if it does not already exist
func (d *DockerDriver) initNetwork(ctx context.Context, name, projectName string) (string, error) {
	// check if network already exists
	networks, err := d.apiClient.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", name)),
	})
	if err != nil {
		return "", fmt.Errorf("could not list docker networks: %w", err)
	}
	var networkId string

	if len(networks) > 0 {
		networkId = networks[0].ID
	} else {
		networkResp, err := d.apiClient.NetworkCreate(ctx, name, network.CreateOptions{
			Driver: "bridge",
			Labels: map[string]string{
				"com.docker.compose.project": projectName,
				"com.docker.compose.network": "default",
				"com.docker.compose.version": "2.31.0",
			},
		})

		if err != nil {
			return "", fmt.Errorf("could not create network: %w", err)
		}

		networkId = networkResp.ID
	}

	return networkId, nil
}
