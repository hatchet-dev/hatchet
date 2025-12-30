package docker

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

const defaultTenantID = "707d0855-80ab-4e1f-a156-f1c4546cbf52"

func (d *DockerDriver) createHatchetToken(ctx context.Context, hatchetLiteOpts *HatchetLiteOpts) (string, error) {
	containerName := canonicalContainerName(hatchetLiteOpts.projectName, hatchetLiteOpts.hatchetName)

	execConfig := container.ExecOptions{
		Cmd: []string{
			"./hatchet-admin",
			"token",
			"create",
			"--config", "./config",
			"--tenant-id", defaultTenantID,
			"--expiresIn", (100 * 365 * 24 * time.Hour).String(),
		},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := d.apiClient.ContainerExecCreate(ctx, containerName, execConfig)
	if err != nil {
		return "", fmt.Errorf("could not create exec: %w", err)
	}

	attachResp, err := d.apiClient.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("could not attach to exec: %w", err)
	}
	defer attachResp.Close()

	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, attachResp.Reader)
	if err != nil {
		return "", fmt.Errorf("could not read exec output: %w", err)
	}

	inspectResp, err := d.apiClient.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return "", fmt.Errorf("could not inspect exec: %w", err)
	}

	if inspectResp.ExitCode != 0 {
		return "", fmt.Errorf("hatchet-admin exited with code %d: %s", inspectResp.ExitCode, stderr.String())
	}

	token := strings.TrimSpace(stdout.String())

	return token, nil
}
