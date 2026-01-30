//go:build windows

package local

import (
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

// setPlatformAttributes sets Windows-specific process attributes
func setPlatformAttributes(cmd *exec.Cmd) {
	// Windows doesn't need special process group handling
	// The taskkill /T flag handles killing child processes
}

// stopProcess uses taskkill to stop the process and its children on Windows
func stopProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	pid := cmd.Process.Pid

	// First try graceful shutdown using taskkill with /T (tree kill)
	killCmd := exec.Command("taskkill", "/T", "/PID", strconv.Itoa(pid))
	_ = killCmd.Run() // Ignore error as process might already be dead

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		_, err := cmd.Process.Wait()
		done <- err
	}()

	select {
	case <-done:
		return nil
	case <-time.After(10 * time.Second):
		// Force kill if graceful shutdown times out
		fmt.Println(styles.Muted.Render("Graceful shutdown timed out, force killing..."))
		forceKillCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
		_ = forceKillCmd.Run()
		<-done // Wait for the process to be fully gone
		return nil
	}
}
