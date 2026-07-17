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

	reqURL := cliconfig.TelemetryEndpoint() + "/cli-check?" + params.Encode()

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
