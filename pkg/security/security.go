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
}

func NewSecurityCheck(opts *DefaultSecurityCheck) SecurityCheck {
	return DefaultSecurityCheck{
		Enabled:  opts.Enabled,
		Endpoint: opts.Endpoint,
		L:        opts.L,
	}
}

func (a DefaultSecurityCheck) Check() {
	if !a.Enabled {
		return
	}

	req := fmt.Sprintf("%s/check?version=%s", a.Endpoint, "0.1.0")

	resp, err := http.Get(req) // #nosec

	if err != nil {
		return // Do nothing if there's an error
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return // Do nothing if the response status is not OK
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return // Do nothing if there's an error reading the body
	}

	if len(body) > 0 {
		a.L.Warn().Msgf("Security Alert:\n\n%s******************\n", body)
	}
}
