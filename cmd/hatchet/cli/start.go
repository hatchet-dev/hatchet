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
	"github.com/hatchet-dev/hatchet/cmd/hatchet/glob"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a Hatchet worker",
	Args:  cobra.MinimumNArgs(1),
	Run:   startWorker,
}

var reload bool

func init() {
	startCmd.PersistentFlags().BoolVarP(&reload, "reload", "r", false, "Reload the worker automatically when the source code changes")

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
