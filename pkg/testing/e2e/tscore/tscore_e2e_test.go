//go:build e2e

// Package tscore exercises the TypeScript SDK's core entry (`@hatchet-dev/typescript-sdk/core`,
// the fetch-based Connect client) against an in-process engine started by the test harness.
// A Node script (sdks/typescript/e2e-core) reaches the engine over HTTP/1.1 only, the way
// fetch does from a serverless runtime, while a Go worker serves the workflow it triggers.
package tscore

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/testing/harness"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

const (
	workflowName = "tscore-echo"
	eventKey     = "tscore:echo"
	testTimeout  = 10 * time.Minute
)

type echoInput struct {
	Message string `json:"message"`
}

type echoOutput struct {
	Message string `json:"message"`
}

// scenarioResult is the JSON line sdks/typescript/e2e-core/run.mjs prints.
type scenarioResult struct {
	RunID    string     `json:"runId"`
	Output   echoOutput `json:"output"`
	Status   string     `json:"status"`
	Done     bool       `json:"done"`
	EventID  string     `json:"eventId"`
	EventKey string     `json:"eventKey"`
}

func TestMain(m *testing.M) {
	harness.RunTestWithEngine(m)
}

func TestCoreClientOverHTTP1(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on PATH")
	}
	pnpm, err := exec.LookPath("pnpm")
	if err != nil {
		t.Skip("pnpm is not on PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	sdkDir := filepath.Join(repoRoot(t), "sdks", "typescript")
	runCommand(t, ctx, sdkDir, nil, pnpm, "install", "--frozen-lockfile")
	runCommand(t, ctx, sdkDir, nil, pnpm, "run", "tsc:build")

	require.NoError(t, harness.WaitEngineReady(ctx, 60*time.Second))

	client, err := hatchet.NewClient()
	require.NoError(t, err)

	// Buffered so the worker never blocks on the test reading the inputs it handled.
	received := make(chan string, 16)
	echo := client.NewStandaloneTask(
		workflowName,
		func(ctx hatchet.Context, input echoInput) (*echoOutput, error) {
			received <- input.Message
			return &echoOutput{Message: input.Message}, nil
		},
		hatchet.WithWorkflowEvents(eventKey),
	)

	worker, err := client.NewWorker("tscore-e2e-worker", hatchet.WithWorkflows(echo))
	require.NoError(t, err)

	stopWorker, err := worker.Start()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, stopWorker())
	}()

	env := []string{
		"HATCHET_CLIENT_HOST_PORT=" + os.Getenv("SERVER_GRPC_BROADCAST_ADDRESS"),
		"HATCHET_E2E_WORKFLOW=" + workflowName,
		"HATCHET_E2E_EVENT=" + eventKey,
	}
	stdout := runCommand(t, ctx, sdkDir, env, node, filepath.Join("e2e-core", "run.mjs"))

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	resultLine := lines[len(lines)-1]
	t.Logf("scenario result: %s", resultLine)

	var result scenarioResult
	require.NoError(t, json.Unmarshal([]byte(resultLine), &result), "last stdout line is not the result JSON: %q", stdout)

	require.NotEmpty(t, result.RunID)
	require.True(t, strings.HasPrefix(result.Output.Message, "hello from core "), "output: %+v", result.Output)
	require.Equal(t, "COMPLETED", result.Status)
	require.True(t, result.Done)
	require.NotEmpty(t, result.EventID)
	require.Equal(t, eventKey, result.EventKey)

	// The order is fixed: the scenario pushes the event only after the direct run completed.
	require.Equal(t, result.Output.Message, waitForMessage(t, received))
	require.Equal(t, "event "+result.Output.Message, waitForMessage(t, received))
}

func waitForMessage(t *testing.T, received <-chan string) string {
	t.Helper()
	select {
	case message := <-received:
		return message
	case <-time.After(60 * time.Second):
		t.Fatal("the worker did not receive a run in time")
		return ""
	}
}

// runCommand keeps stdout apart from stderr because the scenario's result is the last stdout
// line; pnpm's and Node's progress output goes to stderr and is only logged. The inherited
// environment stays in place so the harness's HATCHET_CLIENT_* variables reach Node.
func runCommand(t *testing.T, ctx context.Context, dir string, extraEnv []string, name string, args ...string) string {
	t.Helper()

	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- test tooling with fixed arguments
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	t.Logf("running %s %s in %s", name, strings.Join(args, " "), dir)
	err := cmd.Run()
	if stderr.Len() > 0 {
		t.Logf("%s stderr:\n%s", name, stderr.String())
	}
	require.NoError(t, err, "%s %s failed; stdout:\n%s", name, strings.Join(args, " "), stdout.String())

	return stdout.String()
}

// repoRoot is derived from this file's path rather than the working directory, which `go test`
// sets to the package directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}
