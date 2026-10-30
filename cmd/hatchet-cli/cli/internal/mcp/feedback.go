package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

var (
	// PosthogAPIKey is the public capture-only project key for cli_mcp_feedback
	// events (safe to commit: it can ingest events but never read data, like the
	// keys shipped in browser bundles). Overridable at build time via
	// ldflags: -X .../cli/internal/mcp.PosthogAPIKey=phc_xxx. The key and
	// endpoint are pinned at build time: if the key is somehow empty,
	// submit_feedback reports that feedback cannot be sent. It never falls
	// back to a key or host chosen by a connected engine.
	PosthogAPIKey = "phc_Nd6kn74LHMatXkF0OHVJMiq1qp2iu7xUIzRaipZAZY1" // #nosec G101 -- public capture-only project key, not a secret; same class as keys shipped in browser bundles

	// PosthogEndpoint is the PostHog ingestion host used with PosthogAPIKey.
	PosthogEndpoint = "https://us.i.posthog.com"
)

// feedbackEventName is the PostHog event emitted by the submit_feedback tool.
const feedbackEventName = "cli_mcp_feedback"

// FeedbackEvent is the payload of a single submit_feedback invocation. It must
// never contain tokens or workflow payloads.
type FeedbackEvent struct {
	Category       string
	Summary        string
	Detail         string
	Context        string
	DeploymentType string
	CLIVersion     string
}

// FeedbackTarget identifies the PostHog project a feedback event is sent to.
type FeedbackTarget struct {
	APIKey   string
	Endpoint string
}

// FeedbackSender delivers feedback events. Interfaced so tool handlers can be
// tested without network access.
type FeedbackSender interface {
	Send(ctx context.Context, target FeedbackTarget, distinctID string, event FeedbackEvent) error
}

// buildCapturePayload constructs the PostHog /capture/ request body for a
// feedback event.
func buildCapturePayload(apiKey, distinctID string, event FeedbackEvent, now time.Time) map[string]any {
	return map[string]any{
		"api_key":     apiKey,
		"event":       feedbackEventName,
		"distinct_id": distinctID,
		"timestamp":   now.UTC().Format(time.RFC3339),
		"properties": map[string]any{
			"category":        event.Category,
			"summary":         event.Summary,
			"detail":          event.Detail,
			"context":         event.Context,
			"deployment_type": event.DeploymentType,
			"cli_version":     event.CLIVersion,
			"os":              runtime.GOOS,
			"arch":            runtime.GOARCH,
			"source":          "hatchet-mcp",
		},
	}
}

// httpFeedbackSender posts feedback events to PostHog's public capture API.
type httpFeedbackSender struct {
	client *http.Client
}

// NewHTTPFeedbackSender returns the production feedback sender. Its HTTP
// client never follows redirects: the capture endpoint is pinned, and a
// redirect answer must not re-post the event to a different destination (it
// surfaces as a non-success status instead).
func NewHTTPFeedbackSender() FeedbackSender {
	return &httpFeedbackSender{client: &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

func (s *httpFeedbackSender) Send(ctx context.Context, target FeedbackTarget, distinctID string, event FeedbackEvent) error {
	if target.APIKey == "" || target.Endpoint == "" {
		return fmt.Errorf("no analytics key is configured for this deployment")
	}

	body, err := json.Marshal(buildCapturePayload(target.APIKey, distinctID, event, time.Now()))
	if err != nil {
		return fmt.Errorf("could not encode feedback event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.Endpoint+"/capture/", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("could not build feedback request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("could not send feedback: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("feedback endpoint returned status %d", resp.StatusCode)
	}

	return nil
}
