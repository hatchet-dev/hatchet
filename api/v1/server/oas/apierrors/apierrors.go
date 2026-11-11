package apierrors

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
)

// DriverError reports a database failure. Its text names SQL state, relations, and pool
// internals, so a handler returns it for the server error handler to replace.
func DriverError(err error) bool {
	var pgErr *pgconn.PgError
	var connectErr *pgconn.ConnectError
	var limitErr *fairpool.LimitError

	return errors.As(err, &pgErr) || errors.As(err, &connectErr) || errors.As(err, &limitErr) || errors.Is(err, pgx.ErrNoRows)
}

func NewAPIErrors(description string, field ...string) gen.APIErrors {
	apiError := gen.APIError{
		Description: description,
	}
	if len(field) > 0 {
		apiError.Field = &field[0]
	}
	return gen.APIErrors{
		Errors: []gen.APIError{apiError},
	}
}
