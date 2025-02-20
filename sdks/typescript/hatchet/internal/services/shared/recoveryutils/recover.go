package recoveryutils

import (
	"fmt"
	"runtime/debug"

	"github.com/rs/zerolog"

	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
)

func RecoverWithAlert(l *zerolog.Logger, a *hatcheterrors.Wrapped, r any) error {
	var ok bool
	err, ok := r.(error)

	if !ok {
		err = fmt.Errorf("%v", r)
	}

	err = fmt.Errorf("recovered from panic: %w. Stack trace:\n%s", err, string(debug.Stack()))

	l.Error().Err(err).Msgf("recovered from panic")

	if a != nil {
		return a.WrapErr(err, nil)
	}

	return err
}
