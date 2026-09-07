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

// REVIEW: no PostHog project write key is committed anywhere in this repo (the
// docs MCP reads NEXT_PUBLIC_POSTHOG_KEY from its deployment environment), so
// this build-time key is left empty. Release builds can inject it via
// ldflags: -X .../cli/internal/mcp.PosthogAPIKey=phc_xxx. When it is empty,
// submit_feedback falls back to the public frontend PostHog key served by the
// connected engine's /api/v1/meta endpoint (set on Hatchet Cloud), and reports
// clearly when no key is available at all.
var (
	// PosthogAPIKey is the PostHog project write key used for cli_mcp_feedback
	// events. Intentionally empty in source; see the note above.
	PosthogAPIKey = ""

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

// NewHTTPFeedbackSender returns the production feedback sender.
func NewHTTPFeedbackSender() FeedbackSender {
	return &httpFeedbackSender{client: &http.Client{Timeout: 10 * time.Second}}
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
