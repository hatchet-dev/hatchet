//go:build e2e_cli

package cli

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

type celDebugResult struct {
	Status string  `json:"status"`
	Output *bool   `json:"output"`
	Error  *string `json:"error"`
}

func TestCelDebugSuccessJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("cel", "debug",
		"--expression", `input.message == "hi"`,
		"--input", `{"message":"hi"}`,
	)

	var result celDebugResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal cel debug output: %v\nOutput: %s", err, out)
	}
	if result.Status != "SUCCESS" {
		t.Errorf("expected status SUCCESS, got %q (output: %s)", result.Status, out)
	}
	if result.Output == nil || !*result.Output {
		t.Errorf("expected output true, got: %s", out)
	}
}

func TestCelDebugFalseJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("cel", "debug",
		"--expression", `input.message == "hi"`,
		"--input", `{"message":"bye"}`,
	)

	var result celDebugResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal cel debug output: %v\nOutput: %s", err, out)
	}
	if result.Status != "SUCCESS" {
		t.Errorf("expected status SUCCESS, got %q (output: %s)", result.Status, out)
	}
	if result.Output == nil || *result.Output {
		t.Errorf("expected output false, got: %s", out)
	}
}

func TestCelDebugErrorJSON(t *testing.T) {
	h := testharness.New(t)

	cmd := exec.Command(h.BinaryPath, "cel", "debug",
		"--expression", "undefined_variable.foo == 1",
		"--input", `{"message":"hi"}`,
		"-o", "json",
		"--profile", h.Profile,
	)
	stdout, err := cmd.Output()

	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		} else {
			t.Fatalf("failed to run cel debug: %v", err)
		}
		combined := string(stdout) + stderr
		if !strings.Contains(strings.ToLower(combined), "undefined") &&
			!strings.Contains(strings.ToLower(combined), "error") {
			t.Errorf("expected error output mentioning the failure, got:\nStdout: %s\nStderr: %s", stdout, stderr)
		}
		return
	}

	var result celDebugResult
	if err := json.Unmarshal(stdout, &result); err != nil {
		t.Fatalf("failed to unmarshal cel debug output: %v\nOutput: %s", err, stdout)
	}
	if result.Status != "ERROR" {
		t.Errorf("expected status ERROR, got %q (output: %s)", result.Status, stdout)
	}
	if result.Error == nil || *result.Error == "" {
		t.Errorf("expected non-empty error message, got: %s", stdout)
	}
}
