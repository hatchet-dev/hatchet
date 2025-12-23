package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/drivers/docker"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Commands to manage a local Hatchet server",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a local Hatchet server using Docker",
	Long:  `Start a local Hatchet server environment using Docker containers. This command will start both a PostgreSQL database and a Hatchet server instance, automatically creating a local profile for easy access.`,
	Example: `  # Start server with default settings (port 8888)
  hatchet server start

  # Start server with custom dashboard port
  hatchet server start --dashboard-port 9000

  # Start server with custom ports and project name
  hatchet server start --dashboard-port 9000 --grpc-port 8077 --project-name my-hatchet

  # Start server with custom profile name
  hatchet server start --profile my-local`,
	Run: func(cmd *cobra.Command, args []string) {
		dockerDriver, err := docker.NewDockerDriver(cmd.Context())

		if err != nil {
			cli.Logger.Fatalf("Docker is required to run this command. Please ensure Docker is installed and running.\nError: %v\n", err)
		}

		// Get flag values
		dashboardPort, _ := cmd.Flags().GetInt("dashboard-port")
		grpcPort, _ := cmd.Flags().GetInt("grpc-port")
		projectName, _ := cmd.Flags().GetString("project-name")
		profileName, _ := cmd.Flags().GetString("profile")

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
			cli.Logger.Fatalf("could not start hatchet-lite container: %v\n", err)
		}

		// hook into local profiles to save this token
		profile, err := getProfileFromToken(cmd, token, profileName)

		if err != nil {
			cli.Logger.Fatalf("could not get profile from token: %v", err)
		}

		err = cli.AddProfile(profileName, profile)

		if err != nil {
			cli.Logger.Fatalf("could not add profile: %v", err)
		}

		// Render styled output
		fmt.Println(serverStartedView(profileName, actualDashboardPort, actualGrpcPort))
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a local Hatchet server started with 'hatchet server start'",
	Long:  `Stop a local Hatchet server environment that was started using Docker containers with the 'hatchet server start' command.`,
	Example: `  # Stop the local Hatchet server
  hatchet server stop

  # Stop the local Hatchet server with a custom project name
  hatchet server stop --project-name my-hatchet`,
	Run: func(cmd *cobra.Command, args []string) {
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

// serverStartedView renders the server started message
func serverStartedView(profileName string, dashboardPort, grpcPort int) string {
	var lines []string

	lines = append(lines, styles.SuccessMessage("Hatchet server started successfully!"))
	lines = append(lines, "")
	lines = append(lines, styles.KeyValue("Profile", profileName))
	lines = append(lines, styles.KeyValue("Dashboard", fmt.Sprintf("http://localhost:%d", dashboardPort)))
	lines = append(lines, styles.KeyValue("gRPC Port", fmt.Sprintf("%d", grpcPort)))
	lines = append(lines, "")
	lines = append(lines, styles.Success.Render(fmt.Sprintf("Visit the dashboard URL at http://localhost:%d to get started!", dashboardPort)))
	lines = append(lines, styles.Muted.Render("Admin credentials: email 'admin@example.com', password 'Admin123!!'"))

	return styles.SuccessBox.Render(strings.Join(lines, "\n"))
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.AddCommand(startCmd)
	serverCmd.AddCommand(stopCmd)

	// Add flags for server command
	startCmd.Flags().IntP("dashboard-port", "d", 0, "Port for the Hatchet dashboard (default: auto-detect starting at 8888)")
	startCmd.Flags().IntP("grpc-port", "g", 0, "Port for the Hatchet gRPC server (default: auto-detect starting at 7077)")
	startCmd.Flags().StringP("project-name", "p", "", "Docker project name for containers (default: hatchet-cli)")
	startCmd.Flags().StringP("profile", "n", "local", "Name for the local profile (default: local)")

	stopCmd.Flags().StringP("project-name", "p", "", "Docker project name for containers (default: hatchet-cli)")
}
