package webhookutils

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog"

	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
)

type ErrorOpts struct {
	Code uint
}

func HandleAPIError(
	l *zerolog.Logger,
	alerter hatcheterrors.Alerter,
	w http.ResponseWriter,
	r *http.Request,
	err error,
	writeErr bool,
) {
	// if the error is of type detailed error, get the code from that
	detailedErr := hatcheterrors.DetailedError{}

	if ok := errors.As(err, &detailedErr); ok {
		if detailedErr.Code == 0 || detailedErr.Code >= http.StatusInternalServerError {
			handleInternalError(
				l,
				alerter,
				w,
				r,
				detailedErr,
				writeErr,
			)
		} else {
			w.WriteHeader(int(detailedErr.Code))
			writerErr := json.NewEncoder(w).Encode(detailedErr)

			if writerErr != nil {
				handleInternalError(
					l,
					alerter,
					w,
					r,
					writerErr,
					false,
				)
			}
		}
	}
}

func handleInternalError(l *zerolog.Logger,
	alerter hatcheterrors.Alerter,
	w http.ResponseWriter,
	r *http.Request,
	err error,
	writeErr bool) {
	event := l.Warn().
		Str("internal_error", err.Error())

	event.Send()

	data := make(map[string]interface{})

	data["method"] = r.Method
	data["url"] = r.URL.String()

	alerter.SendAlert(r.Context(), err, data)

	w.WriteHeader(http.StatusInternalServerError)
}
