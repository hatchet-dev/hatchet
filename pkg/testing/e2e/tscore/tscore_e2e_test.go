//go:build e2e

// Package tscore exercises the TypeScript SDK's core entry (`@hatchet-dev/typescript-sdk/core`,
// the fetch-based Connect client) against an in-process engine started by the test harness.
// A Node script (sdks/typescript/e2e-core) reaches the engine over HTTP/1.1 only, the way
// fetch does from a serverless runtime, while a Go worker serves the workflow it triggers.
// The REST API server is started in-process alongside, since the script also lists runs
// through it.
package tscore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/run"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
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
	RunID              string     `json:"runId"`
	Output             echoOutput `json:"output"`
	Status             string     `json:"status"`
	Done               bool       `json:"done"`
	EventID            string     `json:"eventId"`
	EventKey           string     `json:"eventKey"`
	StreamTopic        string     `json:"streamTopic"`
	StreamMessageCount int64      `json:"streamMessageCount"`
	StreamLatestCursor string     `json:"streamLatestCursor"`
	ListedRun          bool       `json:"listedRun"`
	ListedRunStatus    string     `json:"listedRunStatus"`
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
	apiURL := startAPIServer(t, ctx)

	// The scenario publishes to a durable stream, which the engine gates per tenant.
	entitleDurableStreams(t, ctx)

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
		"HATCHET_CLIENT_API_URL=" + apiURL,
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
	require.Equal(t, "tscore-stream-"+result.RunID, result.StreamTopic)
	require.Equal(t, int64(2), result.StreamMessageCount)
	require.NotEmpty(t, result.StreamLatestCursor)
	require.True(t, result.ListedRun, "runs.list did not return the run over the REST API")
	require.Equal(t, "COMPLETED", result.ListedRunStatus)

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

// startAPIServer runs the REST API server in-process on a free port, against the database and
// keys the harness configured, and returns its base URL once it answers readiness probes. The
// harness itself runs only the engine. The server is built the way cmd/hatchet-api builds it,
// and torn down the way the harness tears down the server it mints the token with, so that
// its repository stops its background loops before the leak check runs.
func startAPIServer(t *testing.T, ctx context.Context) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	t.Setenv("SERVER_PORT", strconv.Itoa(port))

	configCleanup, server, err := loader.NewConfigLoader("").CreateServerFromConfig("testing")
	require.NoError(t, err)

	apiCleanup, err := run.NewAPIServer(server).Run()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, apiCleanup())
		require.NoError(t, configCleanup())
		require.NoError(t, server.Disconnect())
	})

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/ready", nil)
		require.NoError(t, err)
		response, err := http.DefaultClient.Do(request)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return baseURL
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("the API server did not become ready in time")
	return ""
}

// repoRoot is derived from this file's path rather than the working directory, which `go test`
// sets to the package directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

// entitleDurableStreams turns the durable streams entitlement on for every tenant of the
// harness database, the way the SDK e2e jobs do before their durable streams suites.
func entitleDurableStreams(t *testing.T, ctx context.Context) {
	t.Helper()

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	require.NoError(t, err)
	defer pool.Close()

	_, err = pool.Exec(ctx, `INSERT INTO tenant_entitlement (tenant_id, durable_streams)
		SELECT id, TRUE FROM "Tenant"
		ON CONFLICT (tenant_id) DO UPDATE SET durable_streams = TRUE`)
	require.NoError(t, err)
}
