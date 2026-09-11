package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCapturePayload(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	payload := buildCapturePayload("phc_test", "anon-123", FeedbackEvent{
		Category:       "error-message",
		Summary:        "cryptic timeout error",
		Detail:         "the trigger timeout message does not say which address it dialed",
		Context:        "debugging a worker that never picked up runs",
		DeploymentType: "embedded",
		CLIVersion:     "v1.0.0",
	}, now)

	assert.Equal(t, "phc_test", payload["api_key"])
	assert.Equal(t, feedbackEventName, payload["event"])
	assert.Equal(t, "anon-123", payload["distinct_id"])
	assert.Equal(t, "2026-09-07T12:00:00Z", payload["timestamp"])

	props, ok := payload["properties"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "error-message", props["category"])
	assert.Equal(t, "cryptic timeout error", props["summary"])
	assert.Equal(t, "embedded", props["deployment_type"])
	assert.Equal(t, "v1.0.0", props["cli_version"])
	assert.Equal(t, runtime.GOOS, props["os"])
	assert.Equal(t, "hatchet-mcp", props["source"])

	// the payload must never carry tokens or run payloads
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "token")
}

func TestHTTPFeedbackSender(t *testing.T) {
	var gotPath string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewHTTPFeedbackSender()
	err := sender.Send(context.Background(), FeedbackTarget{APIKey: "phc_test", Endpoint: server.URL}, "anon-1", FeedbackEvent{
		Category: "bug",
		Summary:  "s",
	})
	require.NoError(t, err)

	assert.Equal(t, "/capture/", gotPath)
	assert.Equal(t, "phc_test", gotBody["api_key"])
	assert.Equal(t, feedbackEventName, gotBody["event"])
}

func TestHTTPFeedbackSenderRequiresKey(t *testing.T) {
	sender := NewHTTPFeedbackSender()

	err := sender.Send(context.Background(), FeedbackTarget{Endpoint: "https://example.com"}, "anon-1", FeedbackEvent{})
	assert.Error(t, err)
}

// TestHTTPFeedbackSenderRejectsRedirects is the regression test for the
// feedback sender's redirect policy: a capture endpoint answering with a
// redirect must not have the event re-posted to the redirect target.
func TestHTTPFeedbackSenderRejectsRedirects(t *testing.T) {
	forwarded := make(chan string, 4)
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer final.Close()

	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 307 preserves the POST body, the worst case for credential-free
		// event forwarding.
		http.Redirect(w, r, final.URL+"/capture/", http.StatusTemporaryRedirect)
	}))
	defer redirecting.Close()

	sender := NewHTTPFeedbackSender()
	err := sender.Send(context.Background(), FeedbackTarget{APIKey: "phc_test", Endpoint: redirecting.URL}, "anon-1", FeedbackEvent{
		Category: "bug",
		Summary:  "s",
	})
	require.Error(t, err)
	assert.Empty(t, forwarded, "the redirect target must never receive the event")
}

func TestHTTPFeedbackSenderNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	sender := NewHTTPFeedbackSender()
	err := sender.Send(context.Background(), FeedbackTarget{APIKey: "phc_bad", Endpoint: server.URL}, "anon-1", FeedbackEvent{})
	assert.Error(t, err)
}
