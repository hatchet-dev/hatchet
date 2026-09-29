//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestIsPartitionLockConflict(t *testing.T) {
	pgErr := func(code string) error {
		return &pgconn.PgError{Code: code, Message: code}
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "lock not available", err: pgErr(pgerrcode.LockNotAvailable), want: true},
		{name: "deadlock detected", err: pgErr(pgerrcode.DeadlockDetected), want: true},
		{name: "wrapped deadlock", err: fmt.Errorf("failed to attach index: %w", pgErr(pgerrcode.DeadlockDetected)), want: true},
		{name: "other pg error", err: pgErr(pgerrcode.UndefinedTable), want: false},
		{name: "non-pg error", err: errors.New("boom"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isPartitionLockConflict(tt.err))
		})
	}
}
