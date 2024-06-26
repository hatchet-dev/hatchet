package security

import (
	"fmt"
	"io"
	"net/http"

	"github.com/hatchet-dev/hatchet/pkg/repository"

	"github.com/rs/zerolog"
)

type SecurityCheck interface {
	Check()
}

type DefaultSecurityCheck struct {
	Enabled  bool
	Endpoint string
	Logger   *zerolog.Logger
	Version  string
	Repo     repository.SecurityCheckRepository
}

func NewSecurityCheck(opts *DefaultSecurityCheck, repo repository.SecurityCheckRepository) SecurityCheck {
	return DefaultSecurityCheck{
		Enabled:  opts.Enabled,
		Endpoint: opts.Endpoint,
		Logger:   opts.Logger,
		Version:  opts.Version,
		Repo:     repo,
	}
}

func (a DefaultSecurityCheck) Check() {
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

	req := fmt.Sprintf("%s/check?version=%s&tag=%s", a.Endpoint, a.Version, ident)
	resp, err := http.Get(req) // #nosec
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
