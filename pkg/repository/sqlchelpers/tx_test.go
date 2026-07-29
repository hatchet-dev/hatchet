package sqlchelpers

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

func TestDeferRollbackRunsWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	l := zerolog.Nop()

	called := false
	rollback := func(rollbackCtx context.Context) error {
		called = true
		return rollbackCtx.Err()
	}

	DeferRollback(ctx, &l, rollback)

	if !called {
		t.Fatal("rollback was not called for a cancelled context; this leaks the pooled connection since pgxpool only releases connections via Rollback/Commit")
	}
}

func TestDeferRollbackRunsWithLiveContext(t *testing.T) {
	l := zerolog.Nop()

	called := false
	rollback := func(rollbackCtx context.Context) error {
		called = true
		return nil
	}

	DeferRollback(context.Background(), &l, rollback)

	if !called {
		t.Fatal("rollback was not called")
	}
}
