package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/worker"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/pm"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/tui"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	profileconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

var c *worker.WorkerConfig

var workerCmd = &cobra.Command{
	Use:     "worker",
	Short:   "Commands for managing Hatchet workers",
	Long:    `Manage Hatchet workers with commands for development and testing.`,
	Aliases: []string{"workers", "wrk"},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var err error

		c, err = worker.LoadWorkerConfig()

		if err != nil {
			log.Fatalf("could not load worker config: %v", err)
		}

		if c == nil {
			fmt.Println(workerConfigMissingView())
			os.Exit(1)
		}
	},
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start a development environment for the Hatchet worker",
	Long:  `Start a Hatchet worker in development mode with automatic reloading on file changes. This command connects to your Hatchet instance using a profile and runs your worker with the configuration specified in hatchet.yaml.`,
	Example: `  # Start worker in dev mode (prompts for profile selection)
  hatchet worker dev

  # Start worker with a specific profile
  hatchet worker dev --profile local

  # Start worker with profile and disable auto-reload
  hatchet worker dev --profile production --no-reload

  # Override the run command
  hatchet worker dev --run-cmd "npm run dev"`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get flag values
		profileFlag, _ := cmd.Flags().GetString("profile")
		noReload, _ := cmd.Flags().GetBool("no-reload")
		runCmd, _ := cmd.Flags().GetString("run-cmd")

		// Override config with flags if provided
		devConfig := c.Dev
		if noReload {
			devConfig.Reload = false
		}
		if runCmd != "" {
			devConfig.RunCmd = runCmd
		}

		startWorker(cmd, &devConfig, profileFlag)
	},
}

var workerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workers",
	Long:  `List workers. Without --output json, launches the interactive TUI. With --output json, outputs raw JSON.`,
	Example: `  # Launch interactive TUI (default)
  hatchet worker list --profile local

  # JSON output
  hatchet worker list -o json`,
	// Override parent PersistentPreRun — no hatchet.yaml required for listing
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		selectedProfile, hatchetClient := clientFromCmd(cmd)

		if !isJSON {
			tuiM := newTUIModel(selectedProfile, hatchetClient)
			tuiM.currentViewType = ViewTypeWorkers
			tuiM.currentView = tui.NewWorkersView(tuiM.ctx)
			p := tea.NewProgram(tuiM, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				cli.Logger.Fatalf("error running TUI: %v", err)
			}
			return
		}

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)
		resp, err := hatchetClient.API().WorkerListWithResponse(ctx, tenantUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to list workers: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var workerGetCmd = &cobra.Command{
	Use:   "get <worker-id>",
	Short: "Get worker details",
	Long:  `Get details about a worker. Without --output json, launches the TUI navigated to the worker. With --output json, outputs raw JSON.`,
	Args:  cobra.ExactArgs(1),
	Example: `  # Launch TUI for a specific worker
  hatchet worker get <worker-id> --profile local

  # JSON output
  hatchet worker get <worker-id> -o json`,
	// Override parent PersistentPreRun — no hatchet.yaml required for get
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		workerID := args[0]
		isJSON := isJSONOutput(cmd)
		selectedProfile, hatchetClient := clientFromCmd(cmd)

		if !isJSON {
			base := newTUIModel(selectedProfile, hatchetClient)
			base.currentViewType = ViewTypeWorkers
			base.currentView = tui.NewWorkersView(base.ctx)
			model := tuiModelWithInitialWorker{
				tuiModel:        base,
				initialWorkerID: workerID,
			}
			p := tea.NewProgram(model, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				cli.Logger.Fatalf("error running TUI: %v", err)
			}
			return
		}

		workerUUID, err := uuid.Parse(workerID)
		if err != nil {
			cli.Logger.Fatalf("invalid worker ID %q: %v", workerID, err)
		}

		ctx := cmd.Context()
		resp, err := hatchetClient.API().WorkerGetWithResponse(ctx, workerUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to get worker: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("worker not found (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

// tuiModelWithInitialWorker wraps tuiModel to navigate to a specific worker on init
type tuiModelWithInitialWorker struct {
	initialWorkerID string
	tuiModel
}

func (m tuiModelWithInitialWorker) Init() tea.Cmd {
	return tea.Batch(
		m.tuiModel.Init(),
		func() tea.Msg { return tui.NavigateToWorkerMsg{WorkerID: m.initialWorkerID} },
	)
}

func init() {
	rootCmd.AddCommand(workerCmd)

	workerCmd.AddCommand(devCmd)
	workerCmd.AddCommand(workerListCmd, workerGetCmd)

	// Add flags for dev command
	devCmd.Flags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	devCmd.Flags().Bool("no-reload", false, "Disable automatic reloading on file changes")
	devCmd.Flags().StringP("run-cmd", "r", "", "Override the run command from hatchet.yaml")

	// Add flags for list/get commands
	workerListCmd.Flags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	workerListCmd.Flags().StringP("output", "o", "", "Output format: json (skips interactive TUI)")
	workerGetCmd.Flags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	workerGetCmd.Flags().StringP("output", "o", "", "Output format: json (skips interactive TUI)")
}

func startWorker(cmd *cobra.Command, devConfig *worker.WorkerDevConfig, profileFlag string) {
	var selectedProfile string

	// Use profile from flag if provided, otherwise use default or show selection form
	if profileFlag != "" {
		selectedProfile = profileFlag
	} else {
		selectedProfile = selectProfileForm(true)

		if selectedProfile == "" {
			// No profiles found - prompt user to either start local server or add a profile
			selectedProfile = handleNoProfiles(cmd)
			if selectedProfile == "" {
				cli.Logger.Fatal("no profile selected or created")
			}
		}
	}

	profile, err := cli.GetProfile(selectedProfile)

	if err != nil {
		cli.Logger.Fatalf("could not get profile '%s': %v", selectedProfile, err)
	}

	ctx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	fmt.Println(workerStartingView(selectedProfile, devConfig.Reload))

	if err := RunWorkerDev(ctx, profile, devConfig, nil); err != nil {
		cli.Logger.Fatalf("error running worker: %v", err)
	}
}

// RunWorkerDev runs the worker in dev mode with the given profile and config.
// This function can be called directly from tests without interactive forms.
// If preCmdsCompleteChan is provided, it will be signaled when pre-commands complete.
func RunWorkerDev(ctx context.Context, profile *profileconfig.Profile, devConfig *worker.WorkerDevConfig, preCmdsCompleteChan chan<- struct{}) error {
	// Run pre-commands if any
	if devConfig.PreCmds != nil {
		for _, preCmdStr := range devConfig.PreCmds {
			fmt.Println(styles.InfoMessage(fmt.Sprintf("Running pre-command: %s", preCmdStr)))

			err := pm.Exec(ctx, preCmdStr, profile)

			if err != nil {
				return fmt.Errorf("error running pre-command '%s': %w", preCmdStr, err)
			}
		}
	}

	// Signal that pre-commands are complete
	if preCmdsCompleteChan != nil {
		preCmdsCompleteChan <- struct{}{}
	}

	proc := pm.NewProcessManager(devConfig.RunCmd, profile)

	if devConfig.Reload {
		cleanup := pm.WatchFiles(ctx, devConfig.Files, proc)

		<-ctx.Done()
		<-cleanup
	} else {
		err := proc.StartProcess(ctx)

		if err != nil {
			return fmt.Errorf("error starting worker: %w", err)
		}

		<-ctx.Done()
		proc.KillProcess()
	}

	return nil
}

// handleNoProfiles prompts the user to either start a local server or add a profile with an API token
func handleNoProfiles(cmd *cobra.Command) string {
	var choice string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("No profiles found. How would you like to proceed?").
				Options(
					huh.NewOption("Start a local Hatchet server (requires Docker)", "local"),
					huh.NewOption("Connect to an existing Hatchet instance with an API token", "remote"),
					huh.NewOption("Cancel", "cancel"),
				).
				Value(&choice),
		),
	).WithTheme(styles.HatchetTheme())

	err := form.Run()
	if err != nil {
		cli.Logger.Fatalf("could not run profile setup form: %v", err)
	}

	switch choice {
	case "local":
		return startLocalServerAndCreateProfile(cmd)
	case "remote":
		return createProfileFromToken(cmd)
	case "cancel":
		return ""
	default:
		return ""
	}
}

// startLocalServerAndCreateProfile starts a local Hatchet server and creates a profile
func startLocalServerAndCreateProfile(cmd *cobra.Command) string {
	fmt.Println(styles.InfoMessage("Starting local Hatchet server..."))

	result, err := startLocalServer(cmd, "local", 0, 0, "")
	if err != nil {
		cli.Logger.Errorf("%v", err)
		fmt.Println()
		fmt.Println(styles.Muted.Render("Alternatively, you can connect to an existing Hatchet instance."))
		return ""
	}

	// Show success message
	fmt.Println(serverStartedView(result.ProfileName, result.DashboardPort, result.GrpcPort, "Starting worker..."))

	return result.ProfileName
}

// createProfileFromToken prompts for an API token and creates a profile
func createProfileFromToken(cmd *cobra.Command) string {
	name, err := addProfileFromToken(cmd)
	if err != nil {
		cli.Logger.Errorf("could not add profile: %v", err)
		return ""
	}

	fmt.Println(styles.SuccessMessage(fmt.Sprintf("Profile '%s' created successfully", name)))

	return name
}

// workerConfigMissingView renders the missing config message
func workerConfigMissingView() string {
	var output []string

	// Info box with instructions
	var lines []string
	lines = append(lines, styles.H2.Render("No worker configuration found"))
	lines = append(lines, "")
	lines = append(lines, "To get started with Hatchet workers, you need a "+styles.Code.Render("hatchet.yaml")+" file.")
	lines = append(lines, "")
	lines = append(lines, styles.Section("Quick Start"))
	lines = append(lines, "")
	lines = append(lines, styles.Accent.Render("1.")+" Generate a new worker project:")
	lines = append(lines, styles.Code.Render("   hatchet quickstart"))
	lines = append(lines, "")
	lines = append(lines, styles.Accent.Render("2.")+" Or create a "+styles.Code.Render("hatchet.yaml")+" file manually with this example:")

	output = append(output, styles.InfoBox.Render(strings.Join(lines, "\n")))
	output = append(output, "")

	// Example configuration outside the box for easy copying
	exampleConfig := `dev:
  preCmds: ["poetry install"]
  runCmd: "poetry run python src/worker.py"
  files:
    - "**/*.py"
  reload: true`

	output = append(output, exampleConfig)
	output = append(output, "")
	output = append(output, styles.Muted.Render("Adjust the commands and file patterns for your language and project structure."))

	return strings.Join(output, "\n")
}

// workerStartingView renders the worker starting message
func workerStartingView(profile string, reloadEnabled bool) string {
	var lines []string

	lines = append(lines, styles.SuccessMessage("Starting Hatchet worker"))
	lines = append(lines, "")
	lines = append(lines, styles.KeyValue("Profile", profile))

	reloadStatus := "disabled"
	if reloadEnabled {
		reloadStatus = "enabled"
	}
	lines = append(lines, styles.KeyValue("Auto-reload", reloadStatus))
	lines = append(lines, "")
	lines = append(lines, styles.Muted.Render("Press Ctrl+C to stop the worker"))

	return styles.SuccessBox.Render(strings.Join(lines, "\n"))
}
