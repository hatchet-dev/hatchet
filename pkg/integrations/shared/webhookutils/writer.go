package webhookutils

import (
	"encoding/json"
	"errors"
	"net/http"
	"syscall"

	"github.com/rs/zerolog"

	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
)

type ResultWriter interface {
	WriteResult(w http.ResponseWriter, r *http.Request, v interface{})
}

// default generalizes response codes for common operations
// (http.StatusOK, http.StatusCreated, etc)
type DefaultResultWriter struct {
	logger  *zerolog.Logger
	alerter hatcheterrors.Alerter
}

func NewDefaultResultWriter(
	logger *zerolog.Logger,
	alerter hatcheterrors.Alerter,
) ResultWriter {
	return &DefaultResultWriter{logger, alerter}
}

func (j *DefaultResultWriter) WriteResult(w http.ResponseWriter, r *http.Request, v interface{}) {
	err := json.NewEncoder(w).Encode(v)

	if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
		// either a broken pipe error or econnreset, ignore. This means the client closed the connection while
		// the server was sending bytes.
		return
	} else if err != nil {
		HandleAPIError(j.logger, j.alerter, w, r, hatcheterrors.NewErrInternal(err), true)
	}
}
