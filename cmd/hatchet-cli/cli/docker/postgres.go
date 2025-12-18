package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
)

func (d *DockerDriver) RunPostgresContainer(ctx context.Context) error {
	imageName := "postgres:17"

	out, err := d.apiClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("could not pull image %s: %w", imageName, err)
	}
	defer out.Close()

	// TODO: write these logs to logger
	io.Copy(io.Discard, out) // nolint: errcheck

	resp, err := d.apiClient.ContainerCreate(ctx,
		&container.Config{
			Image: imageName,
			Env: []string{
				"POSTGRES_USER=hatchet",
				"POSTGRES_PASSWORD=hatchet",
				"POSTGRES_DB=hatchet",
			},
			Cmd: []string{"postgres", "-c", "max_connections=200"},
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", "pg_isready -d hatchet -U hatchet"},
				Interval:    10 * time.Second,
				Timeout:     10 * time.Second,
				Retries:     5,
				StartPeriod: 10 * time.Second,
			},
			Labels: map[string]string{
				"com.docker.compose.project": d.opts.ProjectName,
				"com.docker.compose.service": d.opts.ServiceName,
				// needed for full compose compatibility:
				"com.docker.compose.container-number": "1",
				"com.docker.compose.oneoff":           "False",
			},
		},
		&container.HostConfig{
			RestartPolicy: container.RestartPolicy{
				Name: container.RestartPolicyAlways,
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeVolume,
					Source: "hatchet_lite_postgres_data",
					Target: "/var/lib/postgresql/data",
				},
			},
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				d.initNetworkId: {
					NetworkID: d.initNetworkId,
					Aliases:   []string{d.opts.PostgresName}, // other containers can reach this as "postgres"
				},
			},
		},
		// platform-specific settings
		nil, // TODO: pass platform somewhere?
		canonicalContainerName(d.opts.ProjectName, d.opts.PostgresName),
	)

	if err != nil {
		return fmt.Errorf("could not create postgres container: %w", err)
	}

	if err := d.apiClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("could not start postgres container: %w", err)
	}

	fmt.Println(resp.ID)

	return nil
}
