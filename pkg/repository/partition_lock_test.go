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

func TestReattachValidatesParent(t *testing.T) {
	tests := []struct {
		serverVersionNum int
		want             bool
	}{
		{serverVersionNum: 130020, want: false},
		{serverVersionNum: 140022, want: false},
		{serverVersionNum: 140023, want: true},
		{serverVersionNum: 150006, want: false},
		{serverVersionNum: 150017, want: false},
		{serverVersionNum: 150018, want: true},
		{serverVersionNum: 160013, want: false},
		{serverVersionNum: 160014, want: true},
		{serverVersionNum: 170009, want: false},
		{serverVersionNum: 170010, want: true},
		{serverVersionNum: 180003, want: false},
		{serverVersionNum: 180004, want: true},
		{serverVersionNum: 190000, want: true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.serverVersionNum), func(t *testing.T) {
			assert.Equal(t, tt.want, reattachValidatesParent(tt.serverVersionNum))
		})
	}
}
