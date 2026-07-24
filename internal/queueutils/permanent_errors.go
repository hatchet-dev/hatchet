package queueutils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

// PermanentConsumerErrorReason returns a stable reason for consumer errors that should be shed instead of retried.
func PermanentConsumerErrorReason(err error) string {
	if err == nil {
		return ""
	}

	if errors.Is(err, repository.ErrExternalPayloadNotFound) {
		return "external_payload_not_found"
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.InvalidTextRepresentation {
		return "invalid_json"
	}

	errStr := err.Error()
	if strings.Contains(errStr, fmt.Sprintf("SQLSTATE %s", pgerrcode.InvalidTextRepresentation)) {
		return "invalid_json"
	}
	if strings.Contains(errStr, "invalid input syntax for type json") {
		return "invalid_json"
	}

	return ""
}

// IsPermanentConsumerError returns true when a consumer can drop the offending payload from a buffered batch.
func IsPermanentConsumerError(err error) bool {
	return PermanentConsumerErrorReason(err) != ""
}
