//go:build !windows

package pm

import (
	"fmt"
	"syscall"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

func (pm *ProcessManager) killProcessPlatform() error {
	if pm.proc == nil {
		return nil
	}

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
	return nil
}

func (pm *ProcessManager) setPlatformAttributes() {
	// Make process its own process group so we can kill it and all children
	pm.proc.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
