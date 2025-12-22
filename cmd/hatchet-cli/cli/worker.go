package cli

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/worker"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/patternmatcher"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

var c *worker.WorkerConfig

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Commands for managing Hatchet workers",
	Long:  `Manage Hatchet workers with commands for development and testing.`,
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

		startWorker(&devConfig, profileFlag)
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)

	workerCmd.AddCommand(devCmd)

	// Add flags for dev command
	devCmd.Flags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	devCmd.Flags().Bool("no-reload", false, "Disable automatic reloading on file changes")
	devCmd.Flags().StringP("run-cmd", "r", "", "Override the run command from hatchet.yaml")
}

var procCmd *exec.Cmd
var procLk sync.Mutex

func startWorker(devConfig *worker.WorkerDevConfig, profileFlag string) {
	var selectedProfile string

	// Use profile from flag if provided, otherwise show selection form
	if profileFlag != "" {
		selectedProfile = profileFlag
	} else {
		selectedProfile = selectProfileForm()

		if selectedProfile == "" {
			cli.Logger.Fatal("no profiles found. please create a profile using '%s'", profileAddCmd.CommandPath())
		}
	}

	profile, err := cli.GetProfile(selectedProfile)

	if err != nil {
		cli.Logger.Fatalf("could not get profile '%s': %v", selectedProfile, err)
	}

	apiToken := profile.Token

	interruptChan := cmdutils.InterruptChan()

	if devConfig.PreCmds != nil {
		for _, preCmdStr := range devConfig.PreCmds {
			fmt.Println(styles.InfoMessage(fmt.Sprintf("Running pre-command: %s", preCmdStr)))

			preCmdArgs, err := shellquote.Split(preCmdStr)
			if err != nil {
				cli.Logger.Fatalf("error parsing pre-command '%s': %v", preCmdStr, err)
			}

			preCmd := exec.Command(preCmdArgs[0], preCmdArgs[1:]...)
			preCmd.Stdout = os.Stdout
			preCmd.Stderr = os.Stderr
			preCmd.Env = os.Environ()

			err = preCmd.Run()
			if err != nil {
				cli.Logger.Fatalf("error running pre-command '%s': %v", preCmdStr, err)
			}
		}
	}

	if devConfig.Reload {
		fmt.Println(styles.InfoMessage("Starting file watcher for automatic reloads"))

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		cwd, err := os.Getwd()
		if err != nil {
			cli.Logger.Fatalf("error getting cwd: %v", err)
		}

		// Create pattern matcher for file watching
		for _, pattern := range devConfig.Files {
			fmt.Println(styles.Muted.Render(fmt.Sprintf("  Watching pattern: %s", pattern)))
		}

		pm, err := patternmatcher.New(devConfig.Files)
		if err != nil {
			cli.Logger.Fatalf("error parsing file patterns: %v", err)
		}

		// Track directories to watch for new files
		watchedDirs := make(map[string]bool)

		// Walk the directory tree and watch matching files and all directories
		err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Watch all directories so we can detect new files
			if info.IsDir() {
				// Make path relative to cwd for pattern matching
				relPath, err := filepath.Rel(cwd, path)
				if err != nil {
					relPath = path
				}

				// Watch files that match any pattern (and aren't excluded)
				matched, err := pm.DirMatches(relPath)

				if err != nil {
					return err
				}

				if !matched {
					return filepath.SkipDir
				}

				watcher.Add(path)
				watchedDirs[path] = true
				return nil
			}

			// Make path relative to cwd for pattern matching
			relPath, err := filepath.Rel(cwd, path)
			if err != nil {
				relPath = path
			}

			// Watch files that match any pattern (and aren't excluded)
			matched, err := pm.MatchesOrParentMatches(relPath)
			if err == nil && matched {
				watcher.Add(path)
			}

			return nil
		})

		if err != nil {
			cli.Logger.Fatalf("error walking path: %v", err)
		}

		watchCount := len(watcher.WatchList())
		fmt.Println(styles.Muted.Render(fmt.Sprintf("  Watching %d file(s) and directories", watchCount)))

		go func() {
			for {
				select {
				case <-interruptChan:
					return
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}

					// Check if this is a write to an existing file or a newly created file
					shouldReload := false

					// Make path relative to cwd for pattern matching
					relPath, err := filepath.Rel(cwd, event.Name)
					if err != nil {
						relPath = event.Name
					}

					if event.Has(fsnotify.Write) {
						// Check if it's a file we're already watching (matches a pattern)
						matched, err := pm.MatchesOrParentMatches(relPath)
						if err == nil && matched {
							shouldReload = true
						}
					} else if event.Has(fsnotify.Create) {
						// New file or directory created
						info, err := os.Stat(event.Name)
						if err == nil {
							if info.IsDir() {
								// determine if the directory matches
								dirRelPath, err := filepath.Rel(cwd, event.Name)
								if err != nil {
									dirRelPath = event.Name
								}

								dirMatched, err := pm.DirMatches(dirRelPath)
								if err == nil && dirMatched {
									// Watch new directory, but don't reload
									watcher.Add(event.Name)
									watchedDirs[event.Name] = true
								}
							} else {
								// New file created - check if it matches any pattern
								matched, err := pm.MatchesOrParentMatches(relPath)
								if err == nil && matched {
									watcher.Add(event.Name)
									shouldReload = true
								}
							}
						}
					}

					if shouldReload {
						fmt.Println(styles.InfoMessage(fmt.Sprintf("File change detected: %s", event.Name)))
						fmt.Println(styles.InfoMessage("Reloading worker..."))

						killProcess()

						args, err := shellquote.Split(devConfig.RunCmd)
						if err != nil {
							cli.Logger.Fatalf("error parsing run command: %v", err)
						}

						err = startProcess(args, apiToken)
						if err != nil {
							cli.Logger.Fatalf("error restarting worker: %v", err)
						}
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					fmt.Printf("Watcher error: %v\n", err)
				}
			}
		}()
	}

	fmt.Println(workerStartingView(selectedProfile, devConfig.Reload))

	args, err := shellquote.Split(devConfig.RunCmd)
	if err != nil {
		cli.Logger.Fatalf("error parsing run command: %v", err)
	}

	err = startProcess(args, apiToken)
	if err != nil {
		cli.Logger.Fatalf("error starting worker: %v", err)
	}

	<-interruptChan

	killProcess()
}

func killProcess() {
	procLk.Lock()
	defer procLk.Unlock()

	if procCmd != nil {
		fmt.Println(styles.InfoMessage("Stopping worker"))

		// Create a process group for easier cleanup
		pgid, err := syscall.Getpgid(procCmd.Process.Pid)
		if err == nil {
			// First try graceful shutdown with SIGTERM
			_ = syscall.Kill(-pgid, syscall.SIGTERM)

			// Give it a chance to exit cleanly
			done := make(chan error, 1)
			go func() {
				done <- procCmd.Wait()
			}()

			select {
			case <-done:
				// Process exited, all good
			case <-time.After(3 * time.Second):
				// Process didn't exit in time, force kill
				fmt.Println(styles.Muted.Render("Worker didn't exit gracefully, force killing"))
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
				<-done // Still wait for the process to be fully gone
			}
		} else {
			// Fallback if we couldn't get the process group
			_ = procCmd.Process.Signal(syscall.SIGTERM)

			// Wait for it to exit or force kill after timeout
			done := make(chan error, 1)
			go func() {
				done <- procCmd.Wait()
			}()

			select {
			case <-done:
				// Process exited, all good
			case <-time.After(3 * time.Second):
				// Process didn't exit in time, force kill
				fmt.Println(styles.Muted.Render("Worker didn't exit gracefully, force killing"))
				_ = procCmd.Process.Kill()
				<-done // Still wait for the process to be fully gone
			}
		}

		procCmd = nil
	}
}

func startProcess(args []string, apiToken string) error {
	procLk.Lock()
	defer procLk.Unlock()

	// If there's a running process, kill it first
	if procCmd != nil {
		procLk.Unlock()
		killProcess() // This acquires the lock itself
		procLk.Lock()
	}

	procCmd = exec.Command(args[0], args[1:]...)

	// Make process its own process group so we can kill it and all children
	procCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	procCmd.Env = append(os.Environ(), fmt.Sprintf("HATCHET_CLIENT_TOKEN=%s", apiToken), "HATCHET_CLIENT_TLS_STRATEGY=none")
	procCmd.Stdout = os.Stdout
	procCmd.Stderr = os.Stderr

	err := procCmd.Start()
	if err != nil {
		procCmd = nil
		return fmt.Errorf("error starting worker: %v", err)
	}

	// Don't wait here - we'll wait when killing or when the process exits itself
	// This is a non-blocking start
	go func() {
		waitProc := procCmd // Capture the current process
		err := waitProc.Wait()
		procLk.Lock()
		defer procLk.Unlock()

		// Only clear procCmd if it's still the same process we started
		if procCmd != nil && procCmd == waitProc {
			procCmd = nil
		}

		if err != nil && !strings.Contains(err.Error(), "signal: killed") {
			fmt.Printf("Worker exited with error: %v\n", err)
		}
	}()

	return nil
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
