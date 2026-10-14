//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// An owned trigger runs its preflight queries and its write transaction for one tenant. With a
// per-tenant limit of one connection, it only succeeds if the preflight releases its slot
// before the transaction takes one.
func TestOwnedTriggerStaysWithinTenantConnectionLimit(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	cfg := pool.Config().Copy()
	cfg.MaxConns = 10

	gated, err := fairpool.NewWithConfig(ctx, cfg, fairpool.Options{
		MaxPercent: 10,
		MaxWait:    300 * time.Millisecond,
	})
	require.NoError(t, err)
	defer gated.Close()

	require.EqualValues(t, 1, gated.ConnectionLimit())

	logger := zerolog.Nop()
	repo, stop := newSharedRepository(
		gated,
		pool,
		nil,
		&logger,
		PayloadStoreRepositoryOpts{},
		defaultLimitTestConfig(),
		true,
		time.Minute,
	)
	defer func() { _ = stop() }()

	tenantID := createLimitTestTenant(t, pool)

	const concurrent = 6

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for i := 0; i < concurrent; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			tuple := triggerTuple{
				externalId:        uuid.New(),
				workflowVersionId: uuid.New(),
				workflowId:        uuid.New(),
				workflowName:      "owned-trigger-limit",
				idempotency: &IdempotencyConfig{
					Expression: `"owned-trigger-` + uuid.NewString() + `"`,
					TTLMs:      60_000,
					Method:     sqlcv1.IdempotencyMethodTTL,
				},
			}

			_, _, _, _, triggerErr := repo.triggerWorkflows(ctx, nil, tenantID, []triggerTuple{tuple}, nil)

			mu.Lock()
			errs = append(errs, triggerErr)
			mu.Unlock()
		}()
	}

	wg.Wait()

	for _, triggerErr := range errs {
		var limitErr *fairpool.LimitError
		require.False(t, errors.As(triggerErr, &limitErr), "owned trigger hit the tenant connection limit: %v", triggerErr)
		require.NoError(t, triggerErr)
	}
}
