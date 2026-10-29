package docker

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"
)

// initNetwork atomically creates the docker network if it does not already exist
func (d *DockerDriver) initNetwork(ctx context.Context, name, projectName string) (string, error) {
	// check if network already exists
	networks, err := d.apiClient.NetworkList(ctx, client.NetworkListOptions{
		Filters: client.Filters{}.Add("name", name),
	})
	if err != nil {
		return "", fmt.Errorf("could not list docker networks: %w", err)
	}
	var networkId string

	if len(networks.Items) > 0 {
		networkId = networks.Items[0].ID
	} else {
		networkResp, err := d.apiClient.NetworkCreate(ctx, name, client.NetworkCreateOptions{
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
