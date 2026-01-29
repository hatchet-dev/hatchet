package rabbitmq

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsPermanentPreAckError_PgError22P02(t *testing.T) {
	pgErr := &pgconn.PgError{Code: pgerrcode.InvalidTextRepresentation, Message: "invalid input syntax for type json"}
	wrapped := fmt.Errorf("wrap: %w", pgErr)

	if !isPermanentPreAckError(wrapped) {
		t.Fatalf("expected true for wrapped pg error 22P02")
	}
}

func TestIsPermanentPreAckError_StringFallback(t *testing.T) {
	err := errors.New("ERROR: invalid input syntax for type json (SQLSTATE 22P02)")
	if !isPermanentPreAckError(err) {
		t.Fatalf("expected true for sqlstate 22P02 string fallback")
	}
}

func TestIsPermanentPreAckError_OtherError(t *testing.T) {
	err := errors.New("some transient error")
	if isPermanentPreAckError(err) {
		t.Fatalf("expected false for non-permanent error")
	}
}
