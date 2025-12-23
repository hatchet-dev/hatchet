package pm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kballard/go-shellquote"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

// ProcessManager manages the lifecycle of a child process which can be started, stopped, and restarted.
type ProcessManager struct {
	procLk sync.Mutex

	proc    *exec.Cmd
	cmd     string
	profile *cliconfig.Profile
}

func NewProcessManager(cmd string, profile *cliconfig.Profile) *ProcessManager {
	return &ProcessManager{
		cmd:     cmd,
		profile: profile,
	}
}

func (pm *ProcessManager) StartProcess(ctx context.Context) error {
	pm.KillProcess()

	pm.procLk.Lock()
	defer pm.procLk.Unlock()

	args, err := shellquote.Split(pm.cmd)

	if err != nil {
		return fmt.Errorf("could not parse command '%s': %w", pm.cmd, err)
	}

	pm.proc = exec.Command(args[0], args[1:]...) // nolint: gosec

	// Make process its own process group so we can kill it and all children
	pm.proc.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	pm.proc.Env = prepareEnviron(pm.profile)

	pm.proc.Stdout = os.Stdout
	pm.proc.Stderr = os.Stderr
	err = pm.proc.Start()

	if err != nil {
		pm.proc = nil
		return fmt.Errorf("error starting worker: %v", err)
	}

	// Don't wait here - we'll wait when killing or when the process exits itself
	// This is a non-blocking start
	go func() {
		waitProc := pm.proc // Capture the current process
		err := waitProc.Wait()
		pm.procLk.Lock()
		defer pm.procLk.Unlock()

		// Only clear procCmd if it's still the same process we started
		if pm.proc != nil && pm.proc == waitProc {
			pm.proc = nil
		}

		if err != nil && !strings.Contains(err.Error(), "signal: killed") {
			fmt.Printf("Worker exited with error: %v\n", err)
		}
	}()

	return nil
}

func (pm *ProcessManager) KillProcess() {
	pm.procLk.Lock()
	defer pm.procLk.Unlock()

	if pm.proc != nil {
		fmt.Println(styles.InfoMessage("Stopping worker"))

		// Create a process group for easier cleanup
		pgid, err := syscall.Getpgid(pm.proc.Process.Pid)
		if err == nil {
			// First try graceful shutdown with SIGTERM
			_ = syscall.Kill(-pgid, syscall.SIGTERM)

			// Give it a chance to exit cleanly
			done := make(chan error, 1)
			go func() {
				done <- pm.proc.Wait()
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
			_ = pm.proc.Process.Signal(syscall.SIGTERM)

			// Wait for it to exit or force kill after timeout
			done := make(chan error, 1)
			go func() {
				done <- pm.proc.Wait()
			}()

			select {
			case <-done:
				// Process exited, all good
			case <-time.After(3 * time.Second):
				// Process didn't exit in time, force kill
				fmt.Println(styles.Muted.Render("Worker didn't exit gracefully, force killing"))
				_ = pm.proc.Process.Kill()
				<-done // Still wait for the process to be fully gone
			}
		}

		pm.proc = nil
	}
}

// Exec implements a platform-agnostic child process execution
func Exec(ctx context.Context, cmd string, profile *cliconfig.Profile) error {
	env := prepareEnviron(profile)

	preCmdArgs, err := shellquote.Split(cmd)
	if err != nil {
		return fmt.Errorf("could not parse command '%s': %w", cmd, err)
	}

	preCmd := exec.CommandContext(ctx, preCmdArgs[0], preCmdArgs[1:]...) // nolint: gosec
	preCmd.Stdout = os.Stdout
	preCmd.Stderr = os.Stderr
	preCmd.Env = env

	err = preCmd.Run()
	if err != nil {
		cli.Logger.Fatalf("error running command '%s': %v", cmd, err)
	}

	return nil
}

func prepareEnviron(profile *cliconfig.Profile) []string {
	env := append(os.Environ(), fmt.Sprintf("HATCHET_CLIENT_TOKEN=%s", profile.Token))

	if profile.TLSStrategy != "tls" { // tls is the default on all SDKs
		env = append(env, "HATCHET_CLIENT_TLS_STRATEGY="+profile.TLSStrategy)
	}

	return env
}
