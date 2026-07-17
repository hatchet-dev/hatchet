// Package telemetry sends anonymous CLI usage telemetry. It collects no personal
// data: only the CLI version, OS and architecture, keyed to a randomly generated
// install ID. It is enabled by default and can be disabled by setting
// HATCHET_CLI_TELEMETRY_ENABLED=false or telemetry.enabled: false in
// ~/.hatchet/config.yaml.
package telemetry

import (
	"context"
	"net/http"
	"net/url"
	"runtime"
	"time"

	cliconfig "github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
)

const sendTimeout = 2 * time.Second

// Send fires a single best-effort telemetry ping for the given CLI version. It
// blocks until the request completes or the context passed by Report is done,
// and silently swallows any error so telemetry never affects CLI behavior.
func send(ctx context.Context, version string) {
	if !cliconfig.TelemetryEnabled() {
		return
	}

	anonymousID := cliconfig.EnsureAnonymousID()
	if anonymousID == "" {
		return
	}

	params := url.Values{}
	params.Set("version", version)
	params.Set("tag", anonymousID)
	params.Set("os", runtime.GOOS)
	params.Set("arch", runtime.GOARCH)
	params.Set("source", "cli")

	reqURL := cliconfig.TelemetryEndpoint() + "/check?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return
	}

	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		return
	}
	resp.Body.Close()
}

// Report sends the telemetry ping in the background and returns a function that
// waits for it to finish (bounded by sendTimeout). Callers run the returned
// function after their command completes so a slow network never delays output.
func Report(version string) func() {
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	done := make(chan struct{})

	go func() {
		defer close(done)
		send(ctx, version)
	}()

	return func() {
		defer cancel()
		select {
		case <-done:
		case <-ctx.Done():
		}
	}
}
