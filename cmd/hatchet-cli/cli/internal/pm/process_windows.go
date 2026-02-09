//go:build windows

package pm

import (
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

func (pm *ProcessManager) killProcessPlatform() error {
	if pm.proc == nil {
		return nil
	}

	fmt.Println(styles.InfoMessage("Stopping worker"))

	pid := pm.proc.Process.Pid

	// First try graceful shutdown using taskkill with /T (tree kill)
	killCmd := exec.Command("taskkill", "/T", "/PID", strconv.Itoa(pid))
	_ = killCmd.Run() // Ignore error as process might already be dead

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
		forceKillCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
		_ = forceKillCmd.Run()
		<-done // Still wait for the process to be fully gone
	}

	pm.proc = nil
	return nil
}

func (pm *ProcessManager) setPlatformAttributes() {
	// Windows doesn't need special process group handling
	// The taskkill /T flag handles killing child processes
}
