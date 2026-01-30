//go:build !windows

package local

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

// setPlatformAttributes sets Unix-specific process attributes
func setPlatformAttributes(cmd *exec.Cmd) {
	// Make process its own process group so we can kill it and all children
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// stopProcess sends SIGTERM and waits for process to exit, then SIGKILL if needed
func stopProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	// Send SIGTERM to process group
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}

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
		if pgid, err := syscall.Getpgid(cmd.Process.Pid); err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = cmd.Process.Kill()
		}
		<-done // Wait for the process to be fully gone
		return nil
	}
}
