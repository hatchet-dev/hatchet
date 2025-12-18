package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

func (d *DockerDriver) RunHatchetLite(ctx context.Context) error {
	imageName := "ghcr.io/hatchet-dev/hatchet/hatchet-lite:latest"

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
				"DATABASE_URL=postgresql://hatchet:hatchet@" + d.opts.PostgresName + ":5432/hatchet?sslmode=disable",
				"SERVER_AUTH_COOKIE_DOMAIN=localhost",
				"SERVER_AUTH_COOKIE_INSECURE=t",
				"SERVER_GRPC_BIND_ADDRESS=0.0.0.0",
				"SERVER_GRPC_INSECURE=t",
				"SERVER_GRPC_BROADCAST_ADDRESS=localhost:7077",
				"SERVER_GRPC_PORT=7077",
				"SERVER_URL=http://localhost:8888",
				"SERVER_AUTH_SET_EMAIL_VERIFIED=t",
				"SERVER_DEFAULT_ENGINE_VERSION=V1",
				"SERVER_INTERNAL_CLIENT_INTERNAL_GRPC_BROADCAST_ADDRESS=localhost:7077",
			},
			ExposedPorts: nat.PortSet{
				"8888/tcp": struct{}{},
				"7077/tcp": struct{}{},
			},
			Labels: map[string]string{
				"com.docker.compose.project":          d.opts.ProjectName,
				"com.docker.compose.service":          d.opts.HatchetName,
				"com.docker.compose.container-number": "1",
				"com.docker.compose.oneoff":           "False",
			},
		},
		&container.HostConfig{
			RestartPolicy: container.RestartPolicy{
				Name: container.RestartPolicyAlways,
			},
			PortBindings: nat.PortMap{
				"8888/tcp": []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: "8888"},
				},
				"7077/tcp": []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: "7077"},
				},
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeVolume,
					Source: "hatchet_lite_config",
					Target: "/config",
				},
			},
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				d.initNetworkId: {
					NetworkID: d.initNetworkId,
					Aliases:   []string{d.opts.HatchetName},
				},
			},
		},
		nil,
		canonicalContainerName(d.opts.ProjectName, d.opts.HatchetName),
	)

	if err != nil {
		return fmt.Errorf("could not create hatchet-lite container: %w", err)
	}

	if err := d.apiClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("could not start hatchet-lite container: %w", err)
	}

	fmt.Println(resp.ID)

	return nil
}
