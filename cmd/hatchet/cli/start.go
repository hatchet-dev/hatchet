package cli

import (
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/fsnotify/fsnotify"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-migrate/migrate"
	"github.com/hatchet-dev/hatchet/cmd/hatchet/glob"
	"github.com/hatchet-dev/hatchet/cmd/internal"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed dist
var distDir embed.FS

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Hatchet services",
}

var reload bool

var startWorkerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start a Hatchet worker",
	Args:  cobra.MinimumNArgs(1),
	Run:   startWorker,
}

var startLiteCmd = &cobra.Command{
	Use:   "lite",
	Short: "Start a Hatchet Lite instance for local development",
	Run:   startLite,
}

func init() {
	startWorkerCmd.PersistentFlags().BoolVarP(&reload, "reload", "r", false, "Reload the worker automatically when the source code changes")

	startCmd.AddCommand(startWorkerCmd)
	startCmd.AddCommand(startLiteCmd)
	rootCmd.AddCommand(startCmd)
}

var procCmd *exec.Cmd
var procLk sync.Mutex

func startWorker(cmd *cobra.Command, args []string) {
	apiToken := viper.GetString("token")
	if apiToken == "" {
		log.Fatalf("API token not set. Please set the API token by running `hatchet config set-token <token>`")
	}

	interruptChan := cmdutils.InterruptChan()

	if reload {
		fmt.Println("Detecting runtime ...")
		runtimes := detectRuntimes()

		for _, runtime := range runtimes {
			fmt.Printf("Detected runtime: %s\n", runtime)
		}

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		go func() {
			for {
				select {
				case <-interruptChan:
					return
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Has(fsnotify.Write) {
						fmt.Println("Detected file change")
						fmt.Println("Restarting worker")

						killProcess()

						err := startProcess(args, apiToken)
						if err != nil {
							log.Fatalf("error restarting worker: %v", err)
						}
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Println("watcher error:", err)
				}
			}
		}()

		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("error getting cwd: %v", err)
		}

		for _, runtime := range runtimes {
			runtimeGlob, err := glob.Parse(runtimeGlobWatcher[runtime])
			if err != nil {
				log.Fatalf("error parsing runtime glob: %v", err)
			}

			err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if runtimeGlob.Match(path) {
					watcher.Add(path)
				}

				return nil
			})

			if err != nil {
				log.Fatalf("error walking path: %v", err)
			}
		}

		fmt.Printf("Watching for changes to %v\n", watcher.WatchList())
	}

	fmt.Println("Starting worker")

	err := startProcess(args, apiToken)
	if err != nil {
		log.Fatalf("error starting worker: %v", err)
	}

	<-interruptChan

	killProcess()
}

func killProcess() {
	procLk.Lock()
	defer procLk.Unlock()

	if procCmd != nil {
		fmt.Println("Stopping worker")

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
				fmt.Println("Worker didn't exit gracefully, force killing")
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
				fmt.Println("Worker didn't exit gracefully, force killing")
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

	procCmd.Env = append(os.Environ(), fmt.Sprintf("HATCHET_CLIENT_TOKEN=%s", apiToken))
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
			log.Printf("Worker exited with error: %v", err)
		}
	}()

	return nil
}

func startLite(cmd *cobra.Command, args []string) {
	fmt.Println("setting up postgres ...")

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error getting user home directory: %v", err)
	}

	hatchetDir := filepath.Join(home, ".hatchet")

	postgres := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V17).
			Username("hatchet").
			Password("hatchet").
			Database("hatchet").
			Port(55432).
			DataPath(filepath.Join(hatchetDir, "postgres-data")),
	)
	err = postgres.Start()
	if err != nil {
		log.Fatalf("error starting postgres: %v", err)
	}
	defer postgres.Stop()

	// make sure to populate the dist directory
	frontendDistDir := filepath.Join(hatchetDir, "dist")
	err = os.MkdirAll(frontendDistDir, os.ModePerm)
	if err != nil {
		log.Fatalf("error creating dist directory: %v", err)
	}

	// Extract embedded dist files to the hatchet dir
	// This function recursively extracts files from the embedded FS
	var extractDir func(string, string) error
	extractDir = func(embedPath string, targetPath string) error {
		entries, err := distDir.ReadDir(embedPath)
		if err != nil {
			return fmt.Errorf("error reading embedded directory %s: %v", embedPath, err)
		}

		for _, entry := range entries {
			entryEmbed := filepath.Join(embedPath, entry.Name())
			entryTarget := filepath.Join(targetPath, entry.Name())

			if entry.IsDir() {
				// Create the directory
				err := os.MkdirAll(entryTarget, os.ModePerm)
				if err != nil {
					return fmt.Errorf("error creating directory %s: %v", entryTarget, err)
				}

				// Recursively extract subdirectory
				err = extractDir(entryEmbed, entryTarget)
				if err != nil {
					return err
				}
			} else {
				// Read the embedded file
				content, err := distDir.ReadFile(entryEmbed)
				if err != nil {
					return fmt.Errorf("error reading embedded file %s: %v", entryEmbed, err)
				}

				// Write the file to disk
				err = os.WriteFile(entryTarget, content, 0644)
				if err != nil {
					return fmt.Errorf("error writing file %s: %v", entryTarget, err)
				}
			}
		}

		return nil
	}

	// Start the extraction from the dist root
	err = extractDir("dist", frontendDistDir)
	if err != nil {
		log.Fatalf("error extracting dist: %v", err)
	}

	var envVars = map[string]string{
		"LITE_STATIC_ASSET_DIR":          frontendDistDir,
		"LITE_FRONTEND_PORT":             "8081",
		"LITE_RUNTIME_PORT":              "8888",
		"DATABASE_URL":                   "postgresql://hatchet:hatchet@localhost:55432/hatchet?sslmode=disable",
		"DATABASE_POSTGRES_PORT":         "55432",
		"DATABASE_POSTGRES_HOST":         "localhost",
		"SERVER_GRPC_BIND_ADDRESS":       "0.0.0.0",
		"SERVER_GRPC_INSECURE":           "t",
		"SERVER_GRPC_BROADCAST_ADDRESS":  "localhost:7077",
		"SERVER_GRPC_PORT":               "7077",
		"SERVER_URL":                     "http://localhost:8888",
		"SERVER_AUTH_SET_EMAIL_VERIFIED": "t",
		"SERVER_DEFAULT_ENGINE_VERSION":  "V1",
		"SERVER_AUTH_COOKIE_SECRETS":     "NwzMhVEEg9UhMhmh lrV93PsZ0c9fKmIx",
		"SERVER_AUTH_COOKIE_DOMAIN":      "localhost",
		"SERVER_AUTH_COOKIE_INSECURE":    "true",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}

	ctx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	fmt.Println("running migrations ...")

	migrate.RunMigrations(ctx)

	err = internal.RunQuickstart(&internal.QuickstartOpts{
		ConfigDirectory:    filepath.Join(hatchetDir, "config"),
		GeneratedConfigDir: filepath.Join(hatchetDir, "config"),
		Overwrite:          false,
		CertDir:            filepath.Join(hatchetDir, "certs"),
	})
	if err != nil {
		log.Fatalf("error setting up hatchet-lite config: %v", err)
	}

	cf := loader.NewConfigLoader(filepath.Join(hatchetDir, "config"))
	interruptCh := cmdutils.InterruptChan()

	fmt.Println("starting Hatchet Lite at http://localhost:8888 ...")

	if err := internal.StartLite(cf, interruptCh, Version); err != nil {
		log.Fatalln("error starting Hatchet Lite:", err)
	}
}
