//go:build e2e_cli

package testharness

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// CLIHarness provides helpers for running CLI commands in tests.
// It builds the hatchet binary once per test run and caches the path.
type CLIHarness struct {
	// BinaryPath is the path to the compiled hatchet CLI binary.
	BinaryPath string
	// Profile is the --profile flag value to pass to commands.
	Profile string
	T       *testing.T
}

var (
	binaryOnce sync.Once
	binaryPath string
	binaryErr  error
)

// New builds the CLI binary to a temp dir and returns a CLIHarness.
// The profile is read from the HATCHET_CLI_PROFILE environment variable.
// If the env var is not set, it defaults to "local".
func New(t *testing.T) *CLIHarness {
	t.Helper()

	profile := os.Getenv("HATCHET_CLI_PROFILE")
	if profile == "" {
		profile = "local"
	}

	// Build binary once per test process.
	// Use os.MkdirTemp (not t.TempDir) so the directory outlives the first test
	// that calls New — t.TempDir cleanup runs when that test finishes, which
	// would delete the binary before subsequent tests can use it.
	binaryOnce.Do(func() {
		dir, mkErr := os.MkdirTemp("", "hatchet-cli-e2e-*")
		if mkErr != nil {
			binaryErr = fmt.Errorf("failed to create temp dir: %w", mkErr)
			return
		}
		out := filepath.Join(dir, "hatchet")

		cmd := exec.Command("go", "build", "-o", out, "./cmd/hatchet-cli")
		cmd.Dir = findModuleRoot(t)
		cmd.Env = os.Environ()

		output, err := cmd.CombinedOutput()
		if err != nil {
			binaryErr = fmt.Errorf("failed to build hatchet CLI: %w\nOutput: %s", err, output)
			return
		}

		binaryPath = out
	})

	if binaryErr != nil {
		t.Fatalf("CLIHarness: %v", binaryErr)
	}

	return &CLIHarness{
		BinaryPath: binaryPath,
		Profile:    profile,
		T:          t,
	}
}

// RunJSON runs a CLI command with -o json appended, asserts exit 0, and returns stdout.
func (h *CLIHarness) RunJSON(args ...string) []byte {
	h.T.Helper()
	fullArgs := append(args, "-o", "json", "--profile", h.Profile)
	return h.run(fullArgs...)
}

// Run runs a CLI command without -o json and returns stdout.
func (h *CLIHarness) Run(args ...string) string {
	h.T.Helper()
	fullArgs := append(args, "--profile", h.Profile)
	return string(h.run(fullArgs...))
}

// run executes the binary with the given args and returns stdout.
// It fatals the test if the command exits non-zero.
func (h *CLIHarness) run(args ...string) []byte {
	h.T.Helper()

	cmd := exec.Command(h.BinaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		h.T.Fatalf("hatchet %s failed: %v\nStderr: %s\nStdout: %s",
			strings.Join(args, " "), err, stderr, output)
	}

	return output
}

// findModuleRoot walks up from the test's working directory to find go.mod
func findModuleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod - not in a Go module?")
		}
		dir = parent
	}
}
