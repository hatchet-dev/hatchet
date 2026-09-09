package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

const (
	defaultWaitTimeout = 60 * time.Second
	maxWaitTimeout     = 10 * time.Minute
	runPollInterval    = 750 * time.Millisecond

	// embeddedDetectionTTL bounds how often the embedded probe runs so tool
	// calls stay fast.
	embeddedDetectionTTL = 15 * time.Second
)

// Deps are the injectable dependencies of the MCP server. Everything the tool
// handlers touch goes through these, so handlers are testable with fakes.
type Deps struct {
	// Version is the CLI version reported to MCP clients.
	Version string

	// Profiles provides read access to the CLI profile store.
	Profiles ProfileSource

	// Grants is the MCP grant store.
	Grants *GrantStore

	// NewEngine builds an Engine for a resolved profile.
	NewEngine EngineFactory

	// DetectEmbedded looks for a running embedded instance. Defaults to
	// DetectEmbedded over Profiles when nil.
	DetectEmbedded func(ctx context.Context) *EmbeddedDetection

	// Feedback delivers submit_feedback events.
	Feedback FeedbackSender

	// AnonymousID is the CLI's anonymous telemetry ID, used as the feedback
	// distinct ID. Never a user identifier.
	AnonymousID string
}

// Server is the `hatchet mcp serve` stdio server.
type Server struct {
	deps Deps

	mu         sync.Mutex
	engines    map[string]Engine
	detection  *EmbeddedDetection
	detectedAt time.Time
}

// NewServer builds the MCP server and registers its tools.
func NewServer(deps Deps) *Server {
	if deps.DetectEmbedded == nil {
		deps.DetectEmbedded = func(ctx context.Context) *EmbeddedDetection {
			return DetectEmbedded(ctx, deps.Profiles)
		}
	}

	return &Server{
		deps:    deps,
		engines: make(map[string]Engine),
	}
}

// Run serves MCP over stdio until ctx is cancelled or the client disconnects.
func (s *Server) Run(ctx context.Context) error {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "hatchet",
		Title:   "Hatchet",
		Version: s.deps.Version,
	}, &mcpsdk.ServerOptions{
		Instructions: "Local operations for a Hatchet deployment: trigger and inspect workflow runs, list workers, " +
			"replay runs, and check engine status. Tools run against CLI profiles the user has granted with " +
			"`hatchet mcp auth`; a running embedded instance is usable without a grant.",
	})

	s.register(server)

	return server.Run(ctx, &mcpsdk.StdioTransport{})
}

// detect returns the (briefly cached) embedded detection result.
func (s *Server) detect(ctx context.Context) *EmbeddedDetection {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.detection != nil && time.Since(s.detectedAt) < embeddedDetectionTTL {
		return s.detection
	}

	s.detection = s.deps.DetectEmbedded(ctx)
	s.detectedAt = time.Now()

	return s.detection
}

// resolve loads grants, resolves the requested profile, and returns an engine
// for it. Grants are re-read on every call so `hatchet mcp auth` takes effect
// without restarting the server.
func (s *Server) resolve(ctx context.Context, requested string) (*resolvedProfile, Engine, error) {
	grants, err := s.deps.Grants.Load()
	if err != nil {
		return nil, nil, err
	}

	rp, err := resolveProfile(requested, s.deps.Profiles, grants, s.detect(ctx))
	if err != nil {
		return nil, nil, err
	}

	engine, err := s.engine(rp)
	if err != nil {
		return nil, nil, err
	}

	return rp, engine, nil
}

// engine returns a cached engine for the resolved profile, creating one if
// needed. The cache key includes the token so rotated credentials are not
// reused.
func (s *Server) engine(rp *resolvedProfile) (Engine, error) {
	key := rp.Name + "|" + rp.Profile.ApiServerURL + "|" + rp.Profile.GrpcHostPort + "|" + rp.Profile.Token

	s.mu.Lock()
	defer s.mu.Unlock()

	if engine, ok := s.engines[key]; ok {
		return engine, nil
	}

	engine, err := s.deps.NewEngine(rp.Profile)
	if err != nil {
		return nil, fmt.Errorf("could not create a client for profile %q: %w", rp.Name, err)
	}

	s.engines[key] = engine

	return engine, nil
}

// textResult renders a tool result as a profile/tenant prefix line followed by
// an indented JSON payload.
func textResult(rp *resolvedProfile, payload any) (*mcpsdk.CallToolResult, any, error) {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("could not encode result: %w", err)
	}

	text := fmt.Sprintf("profile: %s (tenant: %s)\n%s", rp.Name, rp.Profile.TenantId, data)

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}},
	}, nil, nil
}

// ---------------------------------------------------------------------------
// Tool registration
// ---------------------------------------------------------------------------

type triggerRunArgs struct {
	Profile            string         `json:"profile,omitempty" jsonschema:"profile to run against (default: the granted default profile, else a detected embedded instance)"`
	Workflow           string         `json:"workflow" jsonschema:"name of the workflow or task to trigger"`
	Input              map[string]any `json:"input,omitempty" jsonschema:"JSON input object for the run (default: {})"`
	AdditionalMetadata map[string]any `json:"additional_metadata,omitempty" jsonschema:"optional metadata key/value pairs attached to the run"`
	Wait               *bool          `json:"wait,omitempty" jsonschema:"wait for the run to finish and include its result (default: true)"`
	TimeoutSeconds     int            `json:"timeout_seconds,omitempty" jsonschema:"max seconds to wait for completion (default: 60)"`
}

type getRunArgs struct {
	Profile string `json:"profile,omitempty" jsonschema:"profile to run against (default: the granted default profile, else a detected embedded instance)"`
	RunID   string `json:"run_id" jsonschema:"external ID of the run"`
}

type listRunEventsArgs struct {
	Profile string `json:"profile,omitempty" jsonschema:"profile to run against (default: the granted default profile, else a detected embedded instance)"`
	RunID   string `json:"run_id" jsonschema:"external ID of the run"`
}

type listWorkersArgs struct {
	Profile string `json:"profile,omitempty" jsonschema:"profile to run against (default: the granted default profile, else a detected embedded instance)"`
}

type replayRunArgs struct {
	Profile        string         `json:"profile,omitempty" jsonschema:"profile to run against (default: the granted default profile, else a detected embedded instance)"`
	RunID          string         `json:"run_id" jsonschema:"external ID of the run to replay"`
	NewInput       map[string]any `json:"new_input,omitempty" jsonschema:"optional replacement input; when set, a fresh run of the same workflow is triggered instead of replaying in place"`
	Wait           *bool          `json:"wait,omitempty" jsonschema:"wait for the replayed run to finish (default: true)"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty" jsonschema:"max seconds to wait for completion (default: 60)"`
}

type engineStatusArgs struct {
	Profile string `json:"profile,omitempty" jsonschema:"profile to report on (default: the granted default profile, else a detected embedded instance)"`
}

type submitFeedbackArgs struct {
	Profile  string `json:"profile,omitempty" jsonschema:"profile whose deployment the feedback is about (optional)"`
	Category string `json:"category" jsonschema:"kind of feedback"`
	Summary  string `json:"summary" jsonschema:"one-line summary"`
	Detail   string `json:"detail" jsonschema:"what happened, what was expected, and any error text (no secrets)"`
	Context  string `json:"context,omitempty" jsonschema:"what you were attempting when the issue came up"`
}

var feedbackCategories = []string{"docs-gap", "api-confusion", "error-message", "bug", "other"}

func (s *Server) register(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "trigger_run",
		Description: "Trigger a Hatchet workflow run by name with a JSON input. By default waits for the run to " +
			"finish and returns its status and output.",
	}, s.handleTriggerRun)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_run",
		Description: "Get the status, timing, output, and error message of a workflow run by its run ID.",
	}, s.handleGetRun)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_run_events",
		Description: "List the lifecycle event timeline of a run (scheduling, assignment, retries, failures).",
	}, s.handleListRunEvents)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_workers",
		Description: "List the workers registered with the tenant, including status and slot usage.",
	}, s.handleListWorkers)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "replay_run",
		Description: "Replay a run by ID, or trigger a fresh run of the same workflow when new_input is given. " +
			"By default waits for the result.",
	}, s.handleReplayRun)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "engine_status",
		Description: "Report which granted profiles exist, whether the selected engine is reachable, its version, " +
			"and whether an embedded instance was detected.",
	}, s.handleEngineStatus)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "submit_feedback",
		Description: "Send product feedback about Hatchet (docs gaps, confusing APIs, bad error messages, bugs) " +
			"to the Hatchet team. Only sends what you put in the arguments, plus the deployment type.",
		InputSchema: submitFeedbackSchema(),
	}, s.handleSubmitFeedback)
}

// submitFeedbackSchema builds the submit_feedback input schema with the
// category enum applied.
func submitFeedbackSchema() *jsonschema.Schema {
	schema, err := jsonschema.For[submitFeedbackArgs](nil)
	if err != nil {
		// jsonschema.For over a static struct type cannot fail at runtime.
		panic(err)
	}

	enum := make([]any, len(feedbackCategories))
	for i, c := range feedbackCategories {
		enum[i] = c
	}
	schema.Properties["category"].Enum = enum

	return schema
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func (s *Server) handleTriggerRun(ctx context.Context, _ *mcpsdk.CallToolRequest, args triggerRunArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.Workflow == "" {
		return nil, nil, fmt.Errorf("workflow is required")
	}

	rp, engine, err := s.resolve(ctx, args.Profile)
	if err != nil {
		return nil, nil, err
	}

	input := args.Input
	if input == nil {
		input = map[string]any{}
	}

	runID, err := engine.TriggerWorkflow(ctx, args.Workflow, input, args.AdditionalMetadata)
	if err != nil {
		return nil, nil, fmt.Errorf("could not trigger workflow %q: %w", args.Workflow, err)
	}

	if !waitRequested(args.Wait) {
		return textResult(rp, map[string]any{"runId": runID, "workflow": args.Workflow, "status": "TRIGGERED"})
	}

	return s.waitResult(ctx, rp, engine, runID, args.TimeoutSeconds, nil)
}

func (s *Server) handleGetRun(ctx context.Context, _ *mcpsdk.CallToolRequest, args getRunArgs) (*mcpsdk.CallToolResult, any, error) {
	rp, engine, err := s.resolve(ctx, args.Profile)
	if err != nil {
		return nil, nil, err
	}

	details, err := engine.GetRun(ctx, args.RunID)
	if err != nil {
		return nil, nil, err
	}

	return textResult(rp, runSummary(details))
}

func (s *Server) handleListRunEvents(ctx context.Context, _ *mcpsdk.CallToolRequest, args listRunEventsArgs) (*mcpsdk.CallToolResult, any, error) {
	rp, engine, err := s.resolve(ctx, args.Profile)
	if err != nil {
		return nil, nil, err
	}

	var events []rest.V1TaskEvent

	// A DAG run embeds its task events in the run details; otherwise treat the
	// ID as a task run and use the task event endpoint (mirrors `hatchet runs
	// events`).
	if details, detailsErr := engine.GetRun(ctx, args.RunID); detailsErr == nil && len(details.TaskEvents) > 0 {
		events = details.TaskEvents
	} else {
		events, err = engine.ListTaskEvents(ctx, args.RunID)
		if err != nil {
			return nil, nil, err
		}
	}

	timeline := make([]map[string]any, 0, len(events))
	for _, evt := range events {
		entry := map[string]any{
			"timestamp": evt.Timestamp,
			"event":     evt.EventType,
			"message":   evt.Message,
		}
		if evt.TaskDisplayName != nil && *evt.TaskDisplayName != "" {
			entry["task"] = *evt.TaskDisplayName
		}
		if evt.ErrorMessage != nil && *evt.ErrorMessage != "" {
			entry["errorMessage"] = *evt.ErrorMessage
		}
		timeline = append(timeline, entry)
	}

	return textResult(rp, map[string]any{"runId": args.RunID, "events": timeline})
}

func (s *Server) handleListWorkers(ctx context.Context, _ *mcpsdk.CallToolRequest, args listWorkersArgs) (*mcpsdk.CallToolResult, any, error) {
	rp, engine, err := s.resolve(ctx, args.Profile)
	if err != nil {
		return nil, nil, err
	}

	workers, err := engine.ListWorkers(ctx)
	if err != nil {
		return nil, nil, err
	}

	rows := make([]map[string]any, 0, len(workers))
	for _, w := range workers {
		row := map[string]any{
			"id":   w.Metadata.Id,
			"name": w.Name,
			"type": w.Type,
		}
		if w.Status != nil {
			row["status"] = *w.Status
		}
		if w.LastHeartbeatAt != nil {
			row["lastHeartbeatAt"] = *w.LastHeartbeatAt
		}
		if w.SlotConfig != nil {
			row["slots"] = *w.SlotConfig
		}
		if w.Actions != nil {
			row["actions"] = *w.Actions
		}
		rows = append(rows, row)
	}

	return textResult(rp, map[string]any{"workers": rows, "count": len(rows)})
}

func (s *Server) handleReplayRun(ctx context.Context, _ *mcpsdk.CallToolRequest, args replayRunArgs) (*mcpsdk.CallToolResult, any, error) {
	rp, engine, err := s.resolve(ctx, args.Profile)
	if err != nil {
		return nil, nil, err
	}

	before, err := engine.GetRun(ctx, args.RunID)
	if err != nil {
		return nil, nil, err
	}

	// With replacement input, trigger a fresh run of the same workflow: the
	// replay API always reuses the original input.
	if args.NewInput != nil {
		workflowName, nameErr := engine.WorkflowName(ctx, before.Run.WorkflowId.String())
		if nameErr != nil {
			return nil, nil, fmt.Errorf("could not resolve the run's workflow: %w", nameErr)
		}

		runID, triggerErr := engine.TriggerWorkflow(ctx, workflowName, args.NewInput, nil)
		if triggerErr != nil {
			return nil, nil, fmt.Errorf("could not trigger workflow %q: %w", workflowName, triggerErr)
		}

		if !waitRequested(args.Wait) {
			return textResult(rp, map[string]any{"runId": runID, "workflow": workflowName, "status": "TRIGGERED", "replayOf": args.RunID})
		}

		return s.waitResult(ctx, rp, engine, runID, args.TimeoutSeconds, map[string]any{"replayOf": args.RunID})
	}

	if err := engine.ReplayRun(ctx, args.RunID); err != nil {
		return nil, nil, err
	}

	if !waitRequested(args.Wait) {
		return textResult(rp, map[string]any{"runId": args.RunID, "status": "REPLAYED"})
	}

	details, timedOut, err := waitForReplay(ctx, engine, args.RunID, before.Run.FinishedAt, waitTimeout(args.TimeoutSeconds))
	if err != nil {
		return nil, nil, err
	}
	if timedOut {
		return nil, nil, fmt.Errorf("run %s did not finish its replay within the timeout; use get_run to check on it", args.RunID)
	}

	summary := runSummary(details)
	summary["replayed"] = true

	return textResult(rp, summary)
}

func (s *Server) handleEngineStatus(ctx context.Context, _ *mcpsdk.CallToolRequest, args engineStatusArgs) (*mcpsdk.CallToolResult, any, error) {
	grants, err := s.deps.Grants.Load()
	if err != nil {
		return nil, nil, err
	}

	profiles := regularProfiles(s.deps.Profiles)
	embedded := s.detect(ctx)

	status := map[string]any{
		"grantedProfiles":    grantedStatusRows(profiles, grants, effectiveDefault(profiles, s.deps.Profiles.DefaultProfile())),
		"allProfilesGranted": grants.HasWildcard(),
		"embeddedDetected":   embedded.Detected,
	}
	if embedded.Detected {
		status["embeddedApiUrl"] = embedded.APIURL
		// Which mechanism found the instance: "profile" for the engine's own
		// registration in the profile store, or one of the detection fallbacks
		// (handshake-env, token-env, port-probe).
		status["embeddedSource"] = embedded.Source
	}
	if embedded.Note != "" {
		status["embeddedNote"] = embedded.Note
	}

	// Reachability of the selected engine. Resolution failures are reported in
	// the status rather than failing the tool: the error text only ever names
	// granted profiles.
	rp, engine, err := s.resolve(ctx, args.Profile)
	if err != nil {
		status["selected"] = nil
		status["note"] = err.Error()
		return statusResult(status)
	}

	selected := map[string]any{
		"profile":  rp.Name,
		"tenantId": rp.Profile.TenantId,
		"apiUrl":   rp.Profile.ApiServerURL,
		"embedded": rp.Embedded,
	}

	if _, metaErr := engine.Meta(ctx); metaErr != nil {
		selected["reachable"] = false
		selected["error"] = metaErr.Error()
	} else {
		selected["reachable"] = true
		if version, versionErr := engine.Version(ctx); versionErr == nil && version != "" {
			selected["engineVersion"] = version
		}
	}

	status["selected"] = selected

	return statusResult(status)
}

func (s *Server) handleSubmitFeedback(ctx context.Context, _ *mcpsdk.CallToolRequest, args submitFeedbackArgs) (*mcpsdk.CallToolResult, any, error) {
	if !validFeedbackCategory(args.Category) {
		return nil, nil, fmt.Errorf("invalid category %q (valid: %s)", args.Category, strings.Join(feedbackCategories, ", "))
	}
	if strings.TrimSpace(args.Summary) == "" {
		return nil, nil, fmt.Errorf("summary is required")
	}

	// Resolve best-effort: feedback is still useful when no profile is granted.
	deployment := "unknown"
	target := FeedbackTarget{APIKey: PosthogAPIKey, Endpoint: PosthogEndpoint}

	rp, engine, resolveErr := s.resolve(ctx, args.Profile)
	if resolveErr == nil {
		meta, metaErr := engine.Meta(ctx)
		deployment = classifyDeployment(rp, meta)

		// Fall back to the public frontend PostHog key served by the connected
		// engine (set on Hatchet Cloud) when no build-time key is present.
		if target.APIKey == "" && metaErr == nil && meta.Posthog != nil && meta.Posthog.ApiKey != nil && *meta.Posthog.ApiKey != "" {
			target.APIKey = *meta.Posthog.ApiKey
			if meta.Posthog.ApiHost != nil && *meta.Posthog.ApiHost != "" {
				target.Endpoint = *meta.Posthog.ApiHost
			}
		}
	}

	if target.APIKey == "" {
		return nil, nil, fmt.Errorf("feedback was not sent: no analytics key is available for this deployment (no build-time key and the engine reports none)")
	}

	event := FeedbackEvent{
		Category:       args.Category,
		Summary:        args.Summary,
		Detail:         args.Detail,
		Context:        args.Context,
		DeploymentType: deployment,
		CLIVersion:     s.deps.Version,
	}

	if err := s.deps.Feedback.Send(ctx, target, s.deps.AnonymousID, event); err != nil {
		return nil, nil, fmt.Errorf("feedback was not sent: %w", err)
	}

	payload := map[string]any{"sent": true, "category": args.Category, "deploymentType": deployment}

	if rp != nil {
		return textResult(rp, payload)
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(data)}},
	}, nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func waitRequested(wait *bool) bool {
	return wait == nil || *wait
}

func waitTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		return defaultWaitTimeout
	}

	timeout := time.Duration(seconds) * time.Second
	if timeout > maxWaitTimeout {
		return maxWaitTimeout
	}

	return timeout
}

func isTerminalStatus(status rest.V1TaskStatus) bool {
	switch status {
	case rest.V1TaskStatusCOMPLETED, rest.V1TaskStatusFAILED, rest.V1TaskStatusCANCELLED:
		return true
	default:
		return false
	}
}

// waitResult polls a run until it reaches a terminal status and renders its
// summary, merging extra fields into the payload.
func (s *Server) waitResult(ctx context.Context, rp *resolvedProfile, engine Engine, runID string, timeoutSeconds int, extra map[string]any) (*mcpsdk.CallToolResult, any, error) {
	details, timedOut, err := waitForRun(ctx, engine, runID, waitTimeout(timeoutSeconds))
	if err != nil {
		return nil, nil, err
	}
	if timedOut {
		return nil, nil, fmt.Errorf("run %s did not finish within the timeout; use get_run to check on it", runID)
	}

	summary := runSummary(details)
	for k, v := range extra {
		summary[k] = v
	}

	return textResult(rp, summary)
}

// waitForRun polls the run until it reaches a terminal status or the timeout
// elapses.
func waitForRun(ctx context.Context, engine Engine, runID string, timeout time.Duration) (*rest.V1WorkflowRunDetails, bool, error) {
	return pollRun(ctx, engine, runID, timeout, func(d *rest.V1WorkflowRunDetails) bool {
		return isTerminalStatus(d.Run.Status)
	})
}

// waitForReplay polls a replayed run until it reaches a terminal state from a
// new execution, i.e. its finishedAt differs from the pre-replay value.
func waitForReplay(ctx context.Context, engine Engine, runID string, priorFinishedAt *time.Time, timeout time.Duration) (*rest.V1WorkflowRunDetails, bool, error) {
	return pollRun(ctx, engine, runID, timeout, func(d *rest.V1WorkflowRunDetails) bool {
		if !isTerminalStatus(d.Run.Status) {
			return false
		}
		if priorFinishedAt == nil {
			return d.Run.FinishedAt != nil
		}
		return d.Run.FinishedAt != nil && !d.Run.FinishedAt.Equal(*priorFinishedAt)
	})
}

func pollRun(ctx context.Context, engine Engine, runID string, timeout time.Duration, done func(*rest.V1WorkflowRunDetails) bool) (*rest.V1WorkflowRunDetails, bool, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(runPollInterval)
	defer ticker.Stop()

	for {
		// Fetch errors before the deadline are treated as transient: a freshly
		// triggered run can 404 briefly until it is ingested.
		details, err := engine.GetRun(ctx, runID)
		if err == nil && done(details) {
			return details, false, nil
		}

		if time.Now().After(deadline) {
			if err != nil {
				return nil, false, err
			}
			return details, true, nil
		}

		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		case <-ticker.C:
		}
	}
}

// runSummary condenses run details into the fields an agent needs.
func runSummary(details *rest.V1WorkflowRunDetails) map[string]any {
	run := details.Run

	summary := map[string]any{
		"runId":       run.Metadata.Id,
		"displayName": run.DisplayName,
		"workflowId":  run.WorkflowId,
		"status":      run.Status,
	}

	if run.ErrorMessage != nil && *run.ErrorMessage != "" {
		summary["errorMessage"] = *run.ErrorMessage
	}
	if run.StartedAt != nil {
		summary["startedAt"] = *run.StartedAt
	}
	if run.FinishedAt != nil {
		summary["finishedAt"] = *run.FinishedAt
	}
	if run.Duration != nil {
		summary["durationMs"] = *run.Duration
	}
	if len(run.Output) > 0 {
		summary["output"] = run.Output
	}

	var failedTasks []map[string]any
	for _, task := range details.Tasks {
		if task.Status == rest.V1TaskStatusFAILED && task.ErrorMessage != nil && *task.ErrorMessage != "" {
			failedTasks = append(failedTasks, map[string]any{
				"task":         task.DisplayName,
				"errorMessage": *task.ErrorMessage,
			})
		}
	}
	if len(failedTasks) > 0 {
		summary["failedTasks"] = failedTasks
	}

	return summary
}

// grantedStatusRows lists granted profiles only; ungranted profile names must
// never reach the agent.
func grantedStatusRows(profiles map[string]cliconfig.Profile, grants *Grants, defaultName string) []map[string]any {
	rows := make([]map[string]any, 0, len(profiles))

	for _, name := range grantedProfileNames(profiles, grants) {
		rows = append(rows, map[string]any{
			"name":    name,
			"default": name == defaultName,
		})
	}

	return rows
}

func statusResult(status map[string]any) (*mcpsdk.CallToolResult, any, error) {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("could not encode result: %w", err)
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(data)}},
	}, nil, nil
}

func validFeedbackCategory(category string) bool {
	for _, c := range feedbackCategories {
		if c == category {
			return true
		}
	}

	return false
}

// classifyDeployment labels the deployment for feedback events: embedded,
// cloud, or self-hosted.
func classifyDeployment(rp *resolvedProfile, meta *rest.APIMeta) string {
	if rp.Embedded || (meta != nil && meta.Embedded != nil && *meta.Embedded) {
		return "embedded"
	}

	url := strings.ToLower(rp.Profile.ApiServerURL)
	if strings.Contains(url, ".onhatchet.run") || strings.Contains(url, "cloud.hatchet.run") {
		return "cloud"
	}

	return "self-hosted"
}
