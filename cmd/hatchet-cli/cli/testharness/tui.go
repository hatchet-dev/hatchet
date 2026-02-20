//go:build e2e_cli

package testharness

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TUIHarness provides helpers for testing the TUI via tmux.
// It requires tmux to be installed on the test machine.
type TUIHarness struct {
	harness     *CLIHarness
	sessionName string
	paneID      string
	T           *testing.T
}

// NewTUI creates a new TUIHarness using the given CLIHarness.
func NewTUI(t *testing.T, h *CLIHarness) *TUIHarness {
	t.Helper()
	return &TUIHarness{
		harness:     h,
		sessionName: fmt.Sprintf("hatchet-tui-test-%d", time.Now().UnixNano()),
		T:           t,
	}
}

// Start launches the TUI command in a new tmux session.
// args are passed to the hatchet binary (e.g., "tui", "--profile", "local").
func (th *TUIHarness) Start(args ...string) {
	th.T.Helper()

	fullArgs := append([]string{th.harness.BinaryPath}, args...)
	fullArgs = append(fullArgs, "--profile", th.harness.Profile)
	cmdStr := strings.Join(fullArgs, " ")

	// Create a new tmux session (detached)
	cmd := exec.Command("tmux", "new-session",
		"-d",
		"-s", th.sessionName,
		"-x", "220",
		"-y", "50",
		cmdStr,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		th.T.Fatalf("failed to start tmux session: %v\nOutput: %s", err, output)
	}

	th.paneID = th.sessionName + ":0"
}

// CaptureAfter waits for the given duration and then captures the pane contents.
func (th *TUIHarness) CaptureAfter(d time.Duration) string {
	th.T.Helper()

	time.Sleep(d)

	cmd := exec.Command("tmux", "capture-pane",
		"-t", th.paneID,
		"-p",
		"-e",
	)

	output, err := cmd.Output()
	if err != nil {
		th.T.Fatalf("failed to capture tmux pane: %v", err)
	}

	return string(output)
}

// SendKey sends a key sequence to the tmux pane.
// key should be a tmux key name (e.g., "q", "Enter", "ctrl+c").
func (th *TUIHarness) SendKey(key string) {
	th.T.Helper()

	cmd := exec.Command("tmux", "send-keys", "-t", th.paneID, key, "Enter")
	if output, err := cmd.CombinedOutput(); err != nil {
		th.T.Fatalf("failed to send key %q: %v\nOutput: %s", key, err, output)
	}
}

// Stop kills the tmux session and cleans up.
func (th *TUIHarness) Stop() {
	if th.sessionName == "" {
		return
	}
	// Kill the tmux session, ignore errors (session may already be dead)
	exec.Command("tmux", "kill-session", "-t", th.sessionName).Run() //nolint:errcheck
}
