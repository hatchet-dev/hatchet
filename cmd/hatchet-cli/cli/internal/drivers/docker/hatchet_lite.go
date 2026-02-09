package docker

import (
	"context"
	"fmt"
	"maps"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
)

// Progress message styling
var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#3392FF")).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#A5C5E9"))
)

func printSuccess(message string) {
	fmt.Println(successStyle.Render(fmt.Sprintf("✓ %s", message)))
}

func printInfo(message string) {
	fmt.Println(infoStyle.Render(fmt.Sprintf("  %s", message)))
}

// hatchet lite opts
const (
	defaultpostgresName          = "postgres"
	defaulthatchetName           = "hatchet"
	defaultprojectName           = "hatchet-cli"
	defaultserviceName           = "hatchet"
	startingDashboardPort        = 8888
	startingGrpcPort             = 7077
	hatchetInternalDashboardPort = 8888
	hatchetInternalGrpcPort      = 7077
)

type HatchetLiteOpts struct {
	tokenCb               func(string)
	portsCb               func(dashboardPort, grpcPort int)
	postgresName          string
	hatchetName           string
	projectName           string
	serviceName           string
	overrideDashboardPort int
	overrideGrpcPort      int
}

func initDefaultHatchetLiteOpts() *HatchetLiteOpts {
	return &HatchetLiteOpts{
		postgresName: defaultpostgresName,
		hatchetName:  defaulthatchetName,
		projectName:  defaultprojectName,
		serviceName:  defaultserviceName,
	}
}

type HatchetLiteOpt func(*HatchetLiteOpts) error

func WithPostgresName(name string) HatchetLiteOpt {
	return func(d *HatchetLiteOpts) error {
		d.postgresName = name
		return nil
	}
}

func WithHatchetName(name string) HatchetLiteOpt {
	return func(d *HatchetLiteOpts) error {
		d.hatchetName = name
		return nil
	}
}

func WithProjectName(name string) HatchetLiteOpt {
	return func(d *HatchetLiteOpts) error {
		d.projectName = name
		return nil
	}
}

func WithServiceName(name string) HatchetLiteOpt {
	return func(d *HatchetLiteOpts) error {
		d.serviceName = name
		return nil
	}
}

func WithCreateTokenCallback(cb func(string)) HatchetLiteOpt {
	return func(o *HatchetLiteOpts) error {
		o.tokenCb = cb
		return nil
	}
}

func WithPortsCallback(cb func(dashboardPort, grpcPort int)) HatchetLiteOpt {
	return func(o *HatchetLiteOpts) error {
		o.portsCb = cb
		return nil
	}
}

// WithOverrideDashboardPort sets the override dashboard port
func WithOverrideDashboardPort(port int) HatchetLiteOpt {
	return func(o *HatchetLiteOpts) error {
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid port: %d", port)
		}

		o.overrideDashboardPort = port
		return nil
	}
}

// WithOverrideGrpcPort sets the override grpc port
func WithOverrideGrpcPort(port int) HatchetLiteOpt {
	return func(o *HatchetLiteOpts) error {
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid port: %d", port)
		}

		o.overrideGrpcPort = port
		return nil
	}
}

func (d *DockerDriver) RunHatchetLite(ctx context.Context, opts ...HatchetLiteOpt) error {
	hatchetLiteOpts := initDefaultHatchetLiteOpts()

	for _, fn := range opts {
		if err := fn(hatchetLiteOpts); err != nil {
			return err
		}
	}

	printInfo("Starting Hatchet Lite with Docker driver")

	sharedLabels := getSharedLabels(hatchetLiteOpts)

	// find or create network (use Docker Compose naming convention)
	networkName := fmt.Sprintf("%s_default", hatchetLiteOpts.projectName)
	networkId, err := d.initNetwork(ctx, networkName, hatchetLiteOpts.projectName)
	if err != nil {
		return fmt.Errorf("could not initialize network: %w", err)
	}

	printSuccess("Network ready")

	// Check if hatchet container already exists and get its ports
	hatchetContainerName := canonicalContainerName(hatchetLiteOpts.projectName, hatchetLiteOpts.hatchetName)
	existingContainer, err := d.apiClient.ContainerInspect(ctx, hatchetContainerName)

	var dashboardPort, grpcPort int

	switch {
	case err == nil && existingContainer.State.Running:
		// Container exists and is running - use its existing ports
		dashboardPort, grpcPort, err = extractPortsFromContainer(&existingContainer)
		if err != nil {
			return fmt.Errorf("could not extract ports from existing container: %w", err)
		}

		if hatchetLiteOpts.overrideDashboardPort != 0 {
			dashboardPort = hatchetLiteOpts.overrideDashboardPort
		}

		if hatchetLiteOpts.overrideGrpcPort != 0 {
			grpcPort = hatchetLiteOpts.overrideGrpcPort
		}
	case !cerrdefs.IsNotFound(err) && err != nil:
		return fmt.Errorf("could not inspect hatchet container: %w", err)
	default:
		// Container doesn't exist or isn't running - find available ports
		dashboardPort = hatchetLiteOpts.overrideDashboardPort
		if dashboardPort == 0 {
			dashboardPort, err = findAvailablePort(startingDashboardPort)
			if err != nil {
				return fmt.Errorf("could not find available dashboard port: %w", err)
			}
		}

		grpcPort = hatchetLiteOpts.overrideGrpcPort
		if grpcPort == 0 {
			grpcPort, err = findAvailablePort(startingGrpcPort)
			if err != nil {
				return fmt.Errorf("could not find available grpc port: %w", err)
			}
		}
	}

	// start postgres container
	if err := d.startPostgresContainer(ctx, hatchetLiteOpts, networkId, sharedLabels); err != nil {
		return fmt.Errorf("could not start postgres container: %w", err)
	}

	// start hatchet-lite container
	if err := d.startHatchetLiteContainer(ctx, hatchetLiteOpts, networkId, dashboardPort, grpcPort, sharedLabels); err != nil {
		return fmt.Errorf("could not start hatchet-lite container: %w", err)
	}

	// notify caller of actual ports used
	if hatchetLiteOpts.portsCb != nil {
		hatchetLiteOpts.portsCb(dashboardPort, grpcPort)
	}

	if hatchetLiteOpts.tokenCb != nil {
		// create token
		token, err := d.createHatchetToken(ctx, hatchetLiteOpts)
		if err != nil {
			return fmt.Errorf("could not create hatchet token: %w", err)
		}

		hatchetLiteOpts.tokenCb(token)
	}

	return nil
}

func (d *DockerDriver) StopHatchetLite(ctx context.Context, opts ...HatchetLiteOpt) error {
	hatchetLiteOpts := initDefaultHatchetLiteOpts()

	for _, fn := range opts {
		if err := fn(hatchetLiteOpts); err != nil {
			return err
		}
	}

	if err := d.stopHatchetLiteContainer(ctx, hatchetLiteOpts); err != nil {
		return fmt.Errorf("could not stop hatchet-lite container: %w", err)
	}

	if err := d.stopPostgresContainer(ctx, hatchetLiteOpts); err != nil {
		return fmt.Errorf("could not stop postgres container: %w", err)
	}

	return nil
}

func (d *DockerDriver) startPostgresContainer(ctx context.Context, opts *HatchetLiteOpts, networkId string, sharedLabels map[string]string) error {
	imageName := "postgres:17"
	containerName := canonicalContainerName(opts.projectName, opts.postgresName)

	out, err := d.apiClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("could not pull image %s: %w", imageName, err)
	}
	defer out.Close()

	// Display progress while pulling the image
	displayImagePullProgress(out, imageName)

	// Get image details for proper labeling
	imageInspect, err := d.apiClient.ImageInspect(ctx, imageName)
	if err != nil {
		return fmt.Errorf("could not inspect image %s: %w", imageName, err)
	}

	labels := map[string]string{}

	// copy shared labels
	maps.Copy(labels, sharedLabels)

	labels["com.docker.compose.service"] = opts.postgresName
	labels["com.docker.compose.container-number"] = "1"
	labels["com.docker.compose.oneoff"] = "False"
	labels["com.docker.compose.depends_on"] = ""
	labels["com.docker.compose.image"] = imageInspect.ID

	containerConfig := &container.Config{
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
		Labels: labels,
	}

	// Create volume with proper labels for Docker Compose compatibility
	postgresVolumeName := fmt.Sprintf("%s_postgres_data", opts.projectName)
	_, err = d.apiClient.VolumeCreate(ctx, volume.CreateOptions{
		Name: postgresVolumeName,
		Labels: map[string]string{
			"com.docker.compose.project": opts.projectName,
			"com.docker.compose.volume":  "postgres_data",
		},
	})
	// Ignore error if volume already exists
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("could not create postgres volume: %w", err)
	}

	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyUnlessStopped,
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: postgresVolumeName,
				Target: "/var/lib/postgresql/data",
			},
		},
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkId: {
				NetworkID: networkId,
				Aliases:   []string{opts.postgresName},
			},
		},
	}

	containerID, err := d.ensureContainer(ctx, containerName, imageName, containerConfig, hostConfig, networkConfig)
	if err != nil {
		return fmt.Errorf("could not ensure postgres container: %w", err)
	}

	// wait for postgres to be healthy
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	err = d.ensureContainerIsHealthy(ctx, containerID)

	if err != nil {
		return fmt.Errorf("postgres container did not become healthy: %w", err)
	}

	printSuccess("PostgreSQL ready")

	return nil
}

func (d *DockerDriver) stopPostgresContainer(ctx context.Context, opts *HatchetLiteOpts) error {
	containerName := canonicalContainerName(opts.projectName, opts.postgresName)

	return d.stopContainer(ctx, containerName)
}

func (d *DockerDriver) startHatchetLiteContainer(ctx context.Context, opts *HatchetLiteOpts, networkId string, dashboardPort, grpcPort int, sharedLabels map[string]string) error {
	imageName := "ghcr.io/hatchet-dev/hatchet/hatchet-lite:latest"
	containerName := canonicalContainerName(opts.projectName, opts.hatchetName)

	out, err := d.apiClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("could not pull image %s: %w", imageName, err)
	}
	defer out.Close()

	// Display progress while pulling the image
	displayImagePullProgress(out, imageName)

	// Get image details for proper labeling
	imageInspect, err := d.apiClient.ImageInspect(ctx, imageName)
	if err != nil {
		return fmt.Errorf("could not inspect image %s: %w", imageName, err)
	}

	// ExposedPorts are the INTERNAL container ports
	exposedPorts := nat.PortSet{
		nat.Port(fmt.Sprintf("%d/tcp", hatchetInternalDashboardPort)): struct{}{},
		nat.Port(fmt.Sprintf("%d/tcp", hatchetInternalGrpcPort)):      struct{}{},
	}

	labels := map[string]string{}
	maps.Copy(labels, sharedLabels)

	labels["com.docker.compose.service"] = opts.hatchetName
	labels["com.docker.compose.container-number"] = "1"
	labels["com.docker.compose.oneoff"] = "False"
	labels["com.docker.compose.depends_on"] = opts.postgresName
	labels["com.docker.compose.image"] = imageInspect.ID

	containerConfig := &container.Config{
		Image: imageName,
		Env: []string{
			"DATABASE_URL=postgresql://hatchet:hatchet@" + opts.postgresName + ":5432/hatchet?sslmode=disable",
			"SERVER_AUTH_COOKIE_DOMAIN=localhost",
			"SERVER_AUTH_COOKIE_INSECURE=t",
			"SERVER_GRPC_BIND_ADDRESS=0.0.0.0",
			"SERVER_GRPC_INSECURE=t",
			"SERVER_GRPC_BROADCAST_ADDRESS=localhost:" + fmt.Sprintf("%d", grpcPort),
			"SERVER_GRPC_PORT=" + fmt.Sprintf("%d", hatchetInternalGrpcPort),
			"SERVER_URL=http://localhost:" + fmt.Sprintf("%d", dashboardPort),
			"SERVER_AUTH_SET_EMAIL_VERIFIED=t",
			"SERVER_DEFAULT_ENGINE_VERSION=V1",
			"SERVER_INTERNAL_CLIENT_INTERNAL_GRPC_BROADCAST_ADDRESS=localhost:" + fmt.Sprintf("%d", hatchetInternalGrpcPort),
		},
		ExposedPorts: exposedPorts,
		Labels:       labels,
		Healthcheck: &container.HealthConfig{
			Test:        []string{"CMD-SHELL", "curl -f http://localhost:8733/ready || exit 1"},
			Interval:    2 * time.Second,
			Timeout:     5 * time.Second,
			Retries:     5,
			StartPeriod: 5 * time.Second,
		},
	}

	// Create volume with proper labels for Docker Compose compatibility
	hatchetVolumeName := fmt.Sprintf("%s_hatchet_config", opts.projectName)
	_, err = d.apiClient.VolumeCreate(ctx, volume.CreateOptions{
		Name: hatchetVolumeName,
		Labels: map[string]string{
			"com.docker.compose.project": opts.projectName,
			"com.docker.compose.volume":  "hatchet_config",
		},
	})
	// Ignore error if volume already exists
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("could not create hatchet volume: %w", err)
	}

	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyUnlessStopped,
		},
		// PortBindings maps INTERNAL container port -> EXTERNAL host port
		PortBindings: nat.PortMap{
			nat.Port(fmt.Sprintf("%d/tcp", hatchetInternalDashboardPort)): []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", dashboardPort)}, // 8888 -> 8889
			},
			nat.Port(fmt.Sprintf("%d/tcp", hatchetInternalGrpcPort)): []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", grpcPort)}, // 7077 -> defaultGrpcPort
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: hatchetVolumeName,
				Target: "/config",
			},
		},
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkId: {
				NetworkID: networkId,
				Aliases:   []string{opts.hatchetName},
			},
		},
	}

	containerId, err := d.ensureContainer(ctx, containerName, imageName, containerConfig, hostConfig, networkConfig)
	if err != nil {
		return fmt.Errorf("could not ensure hatchet-lite container: %w", err)
	}

	// wait for hatchet-lite to be healthy
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	err = d.ensureContainerIsHealthy(ctx, containerId)

	if err != nil {
		return fmt.Errorf("hatchet-lite container did not become healthy: %w", err)
	}

	printSuccess("Hatchet Lite ready")

	return nil
}

func (d *DockerDriver) stopHatchetLiteContainer(ctx context.Context, opts *HatchetLiteOpts) error {
	containerName := canonicalContainerName(opts.projectName, opts.hatchetName)
	return d.stopContainer(ctx, containerName)
}

// findAvailablePort returns an available port starting from the given port
// It will only attempt a maximum of 100 ports before giving up
func findAvailablePort(startPort int) (int, error) {
	port := startPort
	maxAttempts := 100
	// Use min to limit the search to either port+100 or 65535, whichever is smaller
	maxPort := min(startPort+maxAttempts-1, 65535)

	for port <= maxPort {
		addr := ":" + strconv.Itoa(port)
		listener, err := net.Listen("tcp", addr)

		if err == nil {
			// Port is available, close the listener and return the port
			listener.Close()
			return port, nil
		}

		// Try the next port
		port++
	}

	return 0, fmt.Errorf("no available port found in range %d-%d", startPort, maxPort)
}

func canonicalContainerName(projectName, serviceName string) string {
	return fmt.Sprintf("%s-%s-1", projectName, serviceName)
}

// extractPortsFromContainer extracts the dashboard and gRPC ports from an existing container
func extractPortsFromContainer(inspect *container.InspectResponse) (dashboardPort, grpcPort int, err error) {
	if inspect.NetworkSettings == nil || inspect.NetworkSettings.Ports == nil {
		return 0, 0, fmt.Errorf("container has no network settings")
	}

	// Extract dashboard port (internal port 8888)
	dashboardPortKey := nat.Port(fmt.Sprintf("%d/tcp", hatchetInternalDashboardPort))
	if bindings, ok := inspect.NetworkSettings.Ports[dashboardPortKey]; ok && len(bindings) > 0 {
		port, parseErr := strconv.Atoi(bindings[0].HostPort)
		if parseErr != nil {
			return 0, 0, fmt.Errorf("could not parse dashboard port: %w", parseErr)
		}
		dashboardPort = port
	} else {
		return 0, 0, fmt.Errorf("dashboard port not found in container")
	}

	// Extract gRPC port (internal port 7077)
	grpcPortKey := nat.Port(fmt.Sprintf("%d/tcp", hatchetInternalGrpcPort))
	if bindings, ok := inspect.NetworkSettings.Ports[grpcPortKey]; ok && len(bindings) > 0 {
		port, parseErr := strconv.Atoi(bindings[0].HostPort)
		if parseErr != nil {
			return 0, 0, fmt.Errorf("could not parse gRPC port: %w", parseErr)
		}
		grpcPort = port
	} else {
		return 0, 0, fmt.Errorf("gRPC port not found in container")
	}

	return dashboardPort, grpcPort, nil
}

func getSharedLabels(opts *HatchetLiteOpts) map[string]string {
	return map[string]string{
		"com.docker.compose.project": opts.projectName,
	}
}
