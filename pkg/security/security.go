package security

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"

	"github.com/rs/zerolog"
)

const checkInterval = time.Hour

type SecurityCheck interface {
	Check()
	Start(ctx context.Context)
	Shutdown()
}

type DefaultSecurityCheck struct {
	Enabled  bool
	Endpoint string
	Logger   *zerolog.Logger
	Version  string
	Repo     v1.SecurityCheckRepository

	MQKind         string
	OAuthProviders []string

	startTime time.Time
}

func NewSecurityCheck(opts *DefaultSecurityCheck, repo v1.SecurityCheckRepository) SecurityCheck {
	return DefaultSecurityCheck{
		Enabled:        opts.Enabled,
		Endpoint:       opts.Endpoint,
		Logger:         opts.Logger,
		Version:        opts.Version,
		Repo:           repo,
		MQKind:         opts.MQKind,
		OAuthProviders: opts.OAuthProviders,
		startTime:      time.Now(),
	}
}

func detectEnvironment() string {
	switch {
	case os.Getenv("GITHUB_ACTIONS") == "true":
		return "github_actions"
	case os.Getenv("CI") != "":
		return "ci"
	case os.Getenv("KUBERNETES_SERVICE_HOST") != "":
		return "kubernetes"
	case dockerEnv():
		return "docker"
	default:
		return "unknown"
	}
}

func dockerEnv() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func (a DefaultSecurityCheck) Start(ctx context.Context) {
	if !a.Enabled {
		return
	}

	a.Check()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.Check()
		}
	}
}

func (a DefaultSecurityCheck) Check() {
	a.report(5 * time.Second)
}

func (a DefaultSecurityCheck) Shutdown() {
	a.report(1 * time.Second)
}

func (a DefaultSecurityCheck) report(timeout time.Duration) {
	if !a.Enabled {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic in check: %v", r)
		}
	}()

	a.Logger.Debug().Msgf("Fetching security alerts for version %s", a.Version)

	ident, err := a.Repo.GetIdent()
	if err != nil {
		a.Logger.Debug().Msgf("Error fetching security alerts: %s", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	params := url.Values{}
	params.Set("version", a.Version)
	params.Set("tag", ident)
	params.Set("uptime_seconds", strconv.FormatInt(int64(time.Since(a.startTime).Seconds()), 10))
	params.Set("environment", detectEnvironment())
	if a.MQKind != "" {
		params.Set("mq_kind", a.MQKind)
	}
	if len(a.OAuthProviders) > 0 {
		params.Set("oauth_providers", strings.Join(a.OAuthProviders, ","))
	}

	reqURL := fmt.Sprintf("%s/check?%s", a.Endpoint, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		a.Logger.Debug().Msgf("Error creating security check request: %s", err)
		return
	}

	resp, err := http.DefaultClient.Do(req) // #nosec
	if err != nil {
		a.Logger.Debug().Msgf("Error making request to security endpoint: %s", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.Logger.Debug().Msgf("Unexpected status code from security endpoint: %d", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		a.Logger.Debug().Msgf("Error reading response body: %s", err)
		return
	}

	if len(body) == 0 {
		a.Logger.Debug().Msg("No security alerts found")
		return
	}

	a.Logger.Error().Msgf("Security Alert:\n\n%s\n******************\n", body)
}
