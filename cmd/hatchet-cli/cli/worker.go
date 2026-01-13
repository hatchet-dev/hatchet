package cli

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/worker"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/pm"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
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

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a script defined in hatchet.yaml",
	Long:  `Execute a script defined in the scripts section of hatchet.yaml. If no script name is provided, displays an interactive list of available scripts to choose from.`,
	Example: `  # Show interactive list of scripts
  hatchet worker run

  # Run a specific script by name
  hatchet worker run --script simple

  # Run with a specific profile
  hatchet worker run --script bulk --profile local`,
	Run: func(cmd *cobra.Command, args []string) {
		scriptFlag, _ := cmd.Flags().GetString("script")
		profileFlag, _ := cmd.Flags().GetString("profile")

		runScript(cmd, scriptFlag, profileFlag)
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)

	workerCmd.AddCommand(devCmd)
	workerCmd.AddCommand(runCmd)

	// Add flags for dev command
	devCmd.Flags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	devCmd.Flags().Bool("no-reload", false, "Disable automatic reloading on file changes")
	devCmd.Flags().StringP("run-cmd", "r", "", "Override the run command from hatchet.yaml")

	// Add flags for run command
	runCmd.Flags().StringP("script", "s", "", "Name of the script to run (default: prompts for selection)")
	runCmd.Flags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
}

func startWorker(cmd *cobra.Command, devConfig *worker.WorkerDevConfig, profileFlag string) {
	var selectedProfile string

	// Use profile from flag if provided, otherwise show selection form
	if profileFlag != "" {
		selectedProfile = profileFlag
	} else {
		selectedProfile = selectProfileForm()

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

	if devConfig.PreCmds != nil {
		for _, preCmdStr := range devConfig.PreCmds {
			fmt.Println(styles.InfoMessage(fmt.Sprintf("Running pre-command: %s", preCmdStr)))

			err := pm.Exec(ctx, preCmdStr, profile)

			if err != nil {
				cli.Logger.Fatalf("error running pre-command '%s': %v", preCmdStr, err)
			}
		}
	}

	fmt.Println(workerStartingView(selectedProfile, devConfig.Reload))

	proc := pm.NewProcessManager(devConfig.RunCmd, profile)

	if devConfig.Reload {
		cleanup := pm.WatchFiles(ctx, devConfig.Files, proc)

		<-ctx.Done()
		<-cleanup
	} else {
		err = proc.StartProcess(ctx)

		if err != nil {
			cli.Logger.Fatalf("error starting worker: %v", err)
		}

		<-ctx.Done()
		proc.KillProcess()
	}
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

// runScript executes a script from the worker configuration
func runScript(cmd *cobra.Command, scriptFlag string, profileFlag string) {
	// Check if scripts are configured
	if len(c.Scripts) == 0 {
		fmt.Println(styles.H2.Render("No scripts configured"))
		fmt.Println()
		fmt.Println(styles.Muted.Render("Add scripts to your hatchet.yaml file:"))
		fmt.Println()
		fmt.Println(`scripts:
  - name: "simple"
    command: "poetry run simple"
    description: "Trigger a simple workflow"
  - name: "bulk"
    command: "poetry run python -m src.bulk_trigger"
    description: "Trigger multiple workflow runs"`)
		os.Exit(1)
	}

	// Get profile first
	var selectedProfile string
	if profileFlag != "" {
		selectedProfile = profileFlag
	} else {
		selectedProfile = selectProfileForm()

		if selectedProfile == "" {
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

	// Then get script
	var selectedScript *worker.Script

	// If script flag is provided, find it by name
	if scriptFlag != "" {
		for i := range c.Scripts {
			script := &c.Scripts[i]
			scriptName := script.Name
			if scriptName == "" {
				scriptName = script.Command
			}
			if scriptName == scriptFlag {
				selectedScript = script
				break
			}
		}

		if selectedScript == nil {
			cli.Logger.Fatalf("script '%s' not found in hatchet.yaml", scriptFlag)
		}
	} else {
		// Show interactive selection form
		selectedScript = selectScriptForm()
		if selectedScript == nil {
			cli.Logger.Fatal("no script selected")
		}
	}

	// Display script info
	scriptName := selectedScript.Name
	if scriptName == "" {
		scriptName = selectedScript.Command
	}

	fmt.Println(styles.InfoMessage(fmt.Sprintf("Running script: %s", scriptName)))
	if selectedScript.Description != "" {
		fmt.Println(styles.Muted.Render(selectedScript.Description))
	}
	fmt.Println(styles.Muted.Render(fmt.Sprintf("Command: %s", selectedScript.Command)))
	fmt.Println()

	// Execute the script
	ctx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	err = pm.Exec(ctx, selectedScript.Command, profile)
	if err != nil {
		cli.Logger.Fatalf("error running script '%s': %v", scriptName, err)
	}

	fmt.Println()
	fmt.Println(styles.SuccessMessage("Script completed successfully"))
}

// selectScriptForm displays an interactive form to select a script
func selectScriptForm() *worker.Script {
	if len(c.Scripts) == 1 {
		return &c.Scripts[0]
	}

	// Build options from scripts
	options := make([]huh.Option[int], 0, len(c.Scripts))
	for i, script := range c.Scripts {
		scriptName := script.Name
		if scriptName == "" {
			scriptName = script.Command
		}

		label := scriptName
		if script.Description != "" {
			label = fmt.Sprintf("%s - %s", scriptName, script.Description)
		}

		options = append(options, huh.NewOption(label, i))
	}

	var selectedIndex int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select a script to run:").
				Options(options...).
				Value(&selectedIndex),
		),
	).WithTheme(styles.HatchetTheme())

	err := form.Run()
	if err != nil {
		cli.Logger.Fatalf("could not run script selection form: %v", err)
	}

	return &c.Scripts[selectedIndex]
}
