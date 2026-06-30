package queueutils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

func TestIsPermanentConsumerError_PgError22P02(t *testing.T) {
	pgErr := &pgconn.PgError{Code: pgerrcode.InvalidTextRepresentation, Message: "invalid input syntax for type json"}
	wrapped := fmt.Errorf("wrap: %w", pgErr)

	if !IsPermanentConsumerError(wrapped) {
		t.Fatalf("expected true for wrapped pg error 22P02")
	}
}

func TestIsPermanentConsumerError_StringFallback(t *testing.T) {
	err := errors.New("ERROR: invalid input syntax for type json (SQLSTATE 22P02)")
	if !IsPermanentConsumerError(err) {
		t.Fatalf("expected true for sqlstate 22P02 string fallback")
	}
}

func TestIsPermanentConsumerError_ExternalPayloadNotFound(t *testing.T) {
	err := fmt.Errorf("wrap: %w", &repository.ExternalPayloadNotFoundError{
		Kind: repository.ExternalPayloadNotFoundKindIndexFile,
		Key:  "index/2026-06-11/example.index",
		Err:  errors.New("key not found"),
	})

	if !IsPermanentConsumerError(err) {
		t.Fatalf("expected true for external payload not found error")
	}

	if got := PermanentConsumerErrorReason(err); got != "external_payload_not_found" {
		t.Fatalf("expected external payload reason, got %q", got)
	}
}

func TestIsPermanentConsumerError_OtherError(t *testing.T) {
	err := errors.New("some transient error")
	if IsPermanentConsumerError(err) {
		t.Fatalf("expected false for non-permanent error")
	}
}
