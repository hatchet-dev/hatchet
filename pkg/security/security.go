package security

import (
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
)

type SecurityCheck interface {
	Check()
}

type DefaultSecurityCheck struct {
	Enabled  bool
	Endpoint string
	L        *zerolog.Logger
	Version  string
}

func NewSecurityCheck(opts *DefaultSecurityCheck) SecurityCheck {
	return DefaultSecurityCheck{
		Enabled:  opts.Enabled,
		Endpoint: opts.Endpoint,
		L:        opts.L,
		Version:  opts.Version,
	}
}

func (a DefaultSecurityCheck) Check() {
	if !a.Enabled {
		return
	}

	req := fmt.Sprintf("%s/check?version=%s&tag=%s", a.Endpoint, a.Version, "helloworld")

	resp, err := http.Get(req) // #nosec

	if err != nil {
		return // Do nothing if there's an error
	}
	defer resp.Body.Close()

	a.L.Debug().Msgf("Fetching Security Alerts for %s", a.Version)

	if resp.StatusCode != http.StatusOK {
		a.L.Debug().Msgf("Error Fetching Security Alerts: %d", resp.StatusCode)
		return // Do nothing if the response status is not OK
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		a.L.Debug().Msg("No Security Alerts!")
		return // Do nothing if there's an error reading the body
	}

	if len(body) > 0 {
		a.L.Error().Msgf("Security Alert:\n\n%s\n******************\n", body)
	}
}
