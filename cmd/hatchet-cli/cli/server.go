package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/drivers/docker"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/drivers/local"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

var serverCmd = &cobra.Command{
	Use:     "server",
	Short:   "Commands to manage a local Hatchet server",
	Aliases: []string{"servers", "srv"},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a local Hatchet server",
	Long: `Start a local Hatchet server environment. By default, uses Docker containers.

Use --local to run without Docker:
  - Downloads and runs hatchet-api and hatchet-engine binaries
  - Uses embedded PostgreSQL by default (no external database needed)
  - Headless mode (no web UI) - use TUI or SDK to interact
  - Runs in foreground, press Ctrl+C to stop

Database options for --local mode:
  - By default: Uses embedded PostgreSQL (zero configuration)
  - With --database-url: Uses external PostgreSQL (embedded PG disabled)
  - With --no-embedded-postgres: Requires external PostgreSQL`,
	Example: `  # Start server with Docker (default)
  hatchet server start

  # Start server without Docker (uses embedded PostgreSQL)
  hatchet server start --local

  # Start local server with external PostgreSQL
  hatchet server start --local --database-url "postgresql://user:pass@localhost:5432/hatchet"

  # Start local server with custom embedded postgres port
  hatchet server start --local --postgres-port 5434

  # Start local server on custom ports
  hatchet server start --local --api-port 9080 --grpc-port 9077

  # Start Docker server with custom dashboard port
  hatchet server start --dashboard-port 9000`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if --local flag is set
		localMode, _ := cmd.Flags().GetBool("local")
		profileName, _ := cmd.Flags().GetString("profile")

		if localMode {
			// Local mode (no Docker) - runs in foreground
			err := runLocalServerNative(cmd, profileName)
			if err != nil {
				cli.Logger.Fatalf("%v", err)
			}
		} else {
			// Docker mode (default)
			dashboardPort, _ := cmd.Flags().GetInt("dashboard-port")
			grpcPort, _ := cmd.Flags().GetInt("grpc-port")
			projectName, _ := cmd.Flags().GetString("project-name")

			result, err := startLocalServer(cmd, profileName, dashboardPort, grpcPort, projectName)
			if err != nil {
				cli.Logger.Fatalf("%v", err)
			}

			// Render styled output
			fmt.Println(serverStartedView(result.ProfileName, result.DashboardPort, result.GrpcPort, ""))
		}
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a local Hatchet server started with 'hatchet server start'",
	Long: `Stop a local Hatchet server environment. Auto-detects whether the server
was started with Docker or --local mode and stops it appropriately.`,
	Example: `  # Stop the local Hatchet server (auto-detects mode)
  hatchet server stop

  # Stop a Docker server with a custom project name
  hatchet server stop --project-name my-hatchet

  # Explicitly stop a local (non-Docker) server
  hatchet server stop --local`,
	Run: func(cmd *cobra.Command, args []string) {
		localMode, _ := cmd.Flags().GetBool("local")

		// Auto-detect: check if local server is running
		if !localMode && local.IsLocalServerRunning() {
			localMode = true
		}

		if localMode {
			// Stop local server
			localDriver := local.NewLocalDriver()
			err := localDriver.Stop()
			if err != nil {
				cli.Logger.Fatalf("could not stop local server: %v\n", err)
			}
			fmt.Println(styles.SuccessBox.Render("Local Hatchet server stopped successfully!"))
			return
		}

		// Docker mode
		dockerDriver, err := docker.NewDockerDriver(cmd.Context())
		if err != nil {
			cli.Logger.Fatalf("Docker is required to run this command. Please ensure Docker is installed and running.\nError: %v\n", err)
		}

		// Get flag values
		projectName, _ := cmd.Flags().GetString("project-name")

		// Build options for StopHatchetLite
		opts := []docker.HatchetLiteOpt{}

		if projectName != "" {
			opts = append(opts, docker.WithProjectName(projectName))
		}

		err = dockerDriver.StopHatchetLite(cmd.Context(), opts...)

		if err != nil {
			cli.Logger.Fatalf("could not stop hatchet-lite container: %v\n", err)
		}

		fmt.Println(styles.SuccessBox.Render("Hatchet server stopped successfully!"))
	},
}

// ServerStartResult contains the result of starting a local server
type ServerStartResult struct {
	ProfileName   string
	Token         string
	DashboardPort int
	GrpcPort      int
}

// startLocalServer starts a local Hatchet server and returns connection details
func startLocalServer(cmd *cobra.Command, profileName string, dashboardPort, grpcPort int, projectName string) (*ServerStartResult, error) {
	dockerDriver, err := docker.NewDockerDriver(cmd.Context())
	if err != nil {
		return nil, fmt.Errorf("Docker is required to run a local server. Please ensure Docker is installed and running: %w", err)
	}

	var token string
	var actualDashboardPort, actualGrpcPort int

	// Build options for RunHatchetLite
	opts := []docker.HatchetLiteOpt{
		docker.WithCreateTokenCallback(func(tok string) {
			token = tok
		}),
		docker.WithPortsCallback(func(dashboard, grpc int) {
			actualDashboardPort = dashboard
			actualGrpcPort = grpc
		}),
	}

	if dashboardPort != 0 {
		opts = append(opts, docker.WithOverrideDashboardPort(dashboardPort))
	}

	if grpcPort != 0 {
		opts = append(opts, docker.WithOverrideGrpcPort(grpcPort))
	}

	if projectName != "" {
		opts = append(opts, docker.WithProjectName(projectName))
	}

	err = dockerDriver.RunHatchetLite(cmd.Context(), opts...)
	if err != nil {
		return nil, fmt.Errorf("could not start hatchet-lite container: %w", err)
	}

	// Create profile from the token
	profile, err := getProfileFromToken(cmd, token, profileName, "none")
	if err != nil {
		return nil, fmt.Errorf("could not get profile from token: %w", err)
	}

	err = cli.AddProfile(profileName, profile)
	if err != nil {
		return nil, fmt.Errorf("could not add profile: %w", err)
	}

	return &ServerStartResult{
		ProfileName:   profileName,
		Token:         token,
		DashboardPort: actualDashboardPort,
		GrpcPort:      actualGrpcPort,
	}, nil
}

// serverStartedView renders the server started message for Docker mode
func serverStartedView(profileName string, dashboardPort, grpcPort int, additionalMessage string) string {
	var lines []string

	lines = append(lines, styles.SuccessMessage("Hatchet server started successfully!"))
	lines = append(lines, "")
	lines = append(lines, styles.Success.Render(fmt.Sprintf("✓ Created local profile '%s'", profileName)))
	lines = append(lines, "")
	lines = append(lines, styles.KeyValue("Dashboard", fmt.Sprintf("http://localhost:%d", dashboardPort)))
	lines = append(lines, styles.KeyValue("gRPC Port", fmt.Sprintf("%d", grpcPort)))
	lines = append(lines, "")
	lines = append(lines, styles.Success.Render(fmt.Sprintf("Visit the dashboard at http://localhost:%d to get started!", dashboardPort)))
	lines = append(lines, styles.Muted.Render("Admin credentials: email 'admin@example.com', password 'Admin123!!'"))

	if additionalMessage != "" {
		lines = append(lines, "")
		lines = append(lines, styles.Muted.Render(additionalMessage))
	}

	return styles.SuccessBox.Render(strings.Join(lines, "\n"))
}

// runLocalServerNative starts a local Hatchet server without Docker in foreground mode
// This function blocks until the server is stopped via Ctrl+C
func runLocalServerNative(cmd *cobra.Command, profileName string) error {
	databaseURL, _ := cmd.Flags().GetString("database-url")
	apiPort, _ := cmd.Flags().GetInt("api-port")
	grpcPort, _ := cmd.Flags().GetInt("grpc-port")
	healthcheckPort, _ := cmd.Flags().GetInt("healthcheck-port")
	noEmbeddedPG, _ := cmd.Flags().GetBool("no-embedded-postgres")
	postgresPort, _ := cmd.Flags().GetInt("postgres-port")
	binaryVersion, _ := cmd.Flags().GetString("binary-version")

	localDriver := local.NewLocalDriver()

	// Build options
	opts := []local.LocalOpt{
		local.WithProfileName(profileName),
	}

	if databaseURL != "" {
		opts = append(opts, local.WithDatabaseURL(databaseURL))
	}

	// Embedded postgres is enabled by default unless:
	// 1. A database URL is provided (handled in local.Run)
	// 2. --no-embedded-postgres is set
	if noEmbeddedPG {
		opts = append(opts, local.WithEmbeddedPostgres(false))
	}

	if postgresPort != 0 {
		opts = append(opts, local.WithPostgresPort(uint32(postgresPort)))
	}

	if apiPort != 0 {
		opts = append(opts, local.WithAPIPort(apiPort))
	}

	if grpcPort != 0 {
		opts = append(opts, local.WithGRPCPort(grpcPort))
	}

	if healthcheckPort != 0 {
		opts = append(opts, local.WithHealthcheckPort(healthcheckPort))
	}

	// Set binary version (defaults to CLI version)
	if binaryVersion != "" {
		opts = append(opts, local.WithBinaryVersion(binaryVersion))
	} else {
		opts = append(opts, local.WithBinaryVersion(Version))
	}

	// Setup: migrations, keys, seed, etc.
	result, err := localDriver.Run(cmd.Context(), opts...)
	if err != nil {
		return err
	}

	// Create profile from the result
	profile, err := local.CreateProfileFromResult(result)
	if err != nil {
		return fmt.Errorf("could not create profile: %w", err)
	}

	err = cli.AddProfile(profileName, profile)
	if err != nil {
		return fmt.Errorf("could not add profile: %w", err)
	}

	// Setup interrupt handler
	interruptCh := cmdutils.InterruptChan()

	// Determine postgres mode for display
	pgMode := "external"
	if localDriver.IsEmbeddedPostgresEnabled() {
		pgMode = "embedded"
	}

	// Start server (downloads and runs api/engine binaries)
	onReady := func() {
		fmt.Println(localServerStartedView(result.ProfileName, result.APIPort, result.GRPCPort, pgMode))
	}

	return localDriver.StartServer(cmd.Context(), interruptCh, onReady, binaryVersion)
}

// localServerStartedView renders the server started message for local mode
func localServerStartedView(profileName string, apiPort, grpcPort int, pgMode string) string {
	var lines []string

	lines = append(lines, styles.SuccessMessage("Local Hatchet server started successfully!"))
	lines = append(lines, "")
	lines = append(lines, styles.KeyValue("Mode", "Local (headless, no Docker)"))
	lines = append(lines, styles.KeyValue("PostgreSQL", pgMode))
	lines = append(lines, styles.KeyValue("Profile", profileName))
	lines = append(lines, styles.KeyValue("API Port", fmt.Sprintf("%d", apiPort)))
	lines = append(lines, styles.KeyValue("gRPC Port", fmt.Sprintf("%d", grpcPort)))
	lines = append(lines, "")
	lines = append(lines, styles.Muted.Render("Note: Running in headless mode (no web UI)."))
	lines = append(lines, styles.Muted.Render("Use the TUI or SDK to interact with the server."))
	lines = append(lines, "")
	lines = append(lines, styles.Muted.Render("Press Ctrl+C to stop the server."))

	return styles.SuccessBox.Render(strings.Join(lines, "\n"))
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.AddCommand(startCmd)
	serverCmd.AddCommand(stopCmd)

	// Flags for start command
	// Local mode flags
	startCmd.Flags().BoolP("local", "l", false, "Run without Docker (uses embedded PostgreSQL by default)")
	startCmd.Flags().String("database-url", "", "PostgreSQL connection string (disables embedded PostgreSQL)")
	startCmd.Flags().Int("api-port", 0, "Port for the API server in --local mode (default: 8080)")
	startCmd.Flags().Int("healthcheck-port", 0, "Port for the healthcheck server in --local mode (default: 8733)")

	// Embedded Postgres flags
	startCmd.Flags().Bool("no-embedded-postgres", false, "Disable embedded PostgreSQL (requires external PostgreSQL)")
	startCmd.Flags().Int("postgres-port", 0, "Port for embedded PostgreSQL (default: 5433)")

	// Binary version flag
	startCmd.Flags().String("binary-version", "", "Version of hatchet-api/engine binaries to download (default: CLI version)")

	// Docker mode flags
	startCmd.Flags().IntP("dashboard-port", "d", 0, "Port for the Hatchet dashboard in Docker mode (default: auto-detect starting at 8888)")
	startCmd.Flags().IntP("grpc-port", "g", 0, "Port for the Hatchet gRPC server (default: auto-detect starting at 7077)")
	startCmd.Flags().StringP("project-name", "p", "", "Docker project name for containers (default: hatchet-cli)")

	// Common flags
	startCmd.Flags().StringP("profile", "n", "local", "Name for the local profile (default: local)")

	// Flags for stop command
	stopCmd.Flags().BoolP("local", "l", false, "Explicitly stop a local (non-Docker) server")
	stopCmd.Flags().StringP("project-name", "p", "", "Docker project name for containers (default: hatchet-cli)")
}
