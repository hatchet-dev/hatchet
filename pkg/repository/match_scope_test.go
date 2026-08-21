//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type userEventScopeTestRepositories struct {
	durable *durableEventsRepository
	matches *MatchRepositoryImpl
	shared  *sharedRepository
}

func newUserEventScopeTestRepositories(t *testing.T, pool *pgxpool.Pool) userEventScopeTestRepositories {
	t.Helper()

	logger := zerolog.Nop()
	shared, cleanup := newSharedRepository(
		pool,
		pool,
		validator.NewDefaultValidator(),
		&logger,
		PayloadStoreRepositoryOpts{},
		limits.LimitConfigFile{},
		false,
		time.Minute,
	)
	t.Cleanup(func() { _ = cleanup() })

	return userEventScopeTestRepositories{
		durable: &durableEventsRepository{sharedRepository: shared},
		matches: &MatchRepositoryImpl{sharedRepository: shared},
		shared:  shared,
	}
}

func createUserEventScopeTestTask(t *testing.T, ctx context.Context, repos userEventScopeTestRepositories, tenantID uuid.UUID, taskID int64) *sqlcv1.FlattenExternalIdsRow {
	t.Helper()

	insertedAt := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	task := &sqlcv1.FlattenExternalIdsRow{
		ID:         taskID,
		InsertedAt: insertedAt,
		ExternalID: uuid.New(),
	}

	_, err := repos.shared.queries.IncrementLogFileInvocationCounts(ctx, repos.shared.pool, sqlcv1.IncrementLogFileInvocationCountsParams{
		Durabletaskids:         []int64{task.ID},
		Durabletaskinsertedats: []pgtype.Timestamptz{task.InsertedAt},
		Tenantids:              []uuid.UUID{tenantID},
	})
	require.NoError(t, err)

	return task
}

func ingestUserEventScopeTestWaiter(
	t *testing.T,
	ctx context.Context,
	repo *durableEventsRepository,
	tenantID uuid.UUID,
	task *sqlcv1.FlattenExternalIdsRow,
	key string,
	scope *string,
	considerEventsSince *time.Time,
	expression string,
) *IngestWaitForResult {
	t.Helper()

	result, err := repo.IngestDurableTaskEvent(ctx, IngestDurableTaskEventOpts{
		BaseIngestEventOpts: &BaseIngestEventOpts{
			Task:            task,
			Kind:            sqlcv1.V1DurableEventLogKindWAITFOR,
			InvocationCount: 1,
			TenantId:        tenantID,
		},
		WaitFor: &IngestWaitForOpts{WaitForConditions: []CreateExternalSignalConditionOpt{{
			Kind:                         CreateExternalSignalConditionKindUSEREVENT,
			ReadableDataKey:              "payload",
			OrGroupId:                    uuid.New(),
			UserEventKey:                 &key,
			UserEventScope:               scope,
			UserEventConsiderEventsSince: considerEventsSince,
			Expression:                   expression,
		}}},
	})
	require.NoError(t, err)
	require.NotNil(t, result.WaitForResult)

	return result.WaitForResult
}

func insertUserEventScopeTestEvent(
	t *testing.T,
	ctx context.Context,
	repos userEventScopeTestRepositories,
	tenantID uuid.UUID,
	key string,
	scope *string,
	seenAt time.Time,
	payload []byte,
) {
	t.Helper()

	eventExternalID := uuid.New()
	eventSeenAt := pgtype.Timestamptz{Time: seenAt, Valid: true}

	createdEvents, err := repos.shared.queries.BulkCreateEvents(ctx, repos.shared.pool, sqlcv1.BulkCreateEventsParams{
		Tenantids:              []uuid.UUID{tenantID},
		Externalids:            []uuid.UUID{eventExternalID},
		Seenats:                []pgtype.Timestamptz{eventSeenAt},
		Keys:                   []string{key},
		Additionalmetadatas:    [][]byte{[]byte(`{}`)},
		Scopes:                 []pgtype.Text{sqlchelpers.TextFromMaybeStr(scope)},
		TriggeringWebhookNames: []pgtype.Text{{Valid: false}},
	})
	require.NoError(t, err)
	require.Len(t, createdEvents, 1)

	require.NoError(t, repos.shared.payloadStore.Store(ctx, repos.shared.pool, StorePayloadOpts{
		Id:         createdEvents[0].ID,
		InsertedAt: eventSeenAt,
		ExternalId: eventExternalID,
		Type:       sqlcv1.V1PayloadTypeUSEREVENTINPUT,
		Payload:    payload,
		TenantId:   tenantID,
	}))
}

func requireUserEventScopeTestWaiterState(t *testing.T, ctx context.Context, repos userEventScopeTestRepositories, task *sqlcv1.FlattenExternalIdsRow, nodeID, branchID int64, expected bool) {
	t.Helper()

	entry, err := repos.shared.queries.GetDurableEventLogEntry(ctx, repos.shared.pool, sqlcv1.GetDurableEventLogEntryParams{
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Nodeid:                nodeID,
		Branchid:              branchID,
	})
	require.NoError(t, err)
	require.Equal(t, expected, entry.IsSatisfied)
}

func TestDurableUserEventScopesIsolateLiveMatches(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	repos := newUserEventScopeTestRepositories(t, pool)
	key := "shared-user-event-key"
	scopeA := "scope-a"
	scopeB := "scope-b"
	taskA := createUserEventScopeTestTask(t, ctx, repos, tenantID, 101)
	taskB := createUserEventScopeTestTask(t, ctx, repos, tenantID, 102)
	unscopedTask := createUserEventScopeTestTask(t, ctx, repos, tenantID, 103)

	waitA := ingestUserEventScopeTestWaiter(t, ctx, repos.durable, tenantID, taskA, key, &scopeA, nil, "true")
	require.False(t, waitA.IsSatisfied)
	waitB := ingestUserEventScopeTestWaiter(t, ctx, repos.durable, tenantID, taskB, key, &scopeB, nil, "true")
	require.False(t, waitB.IsSatisfied)
	waitUnscoped := ingestUserEventScopeTestWaiter(t, ctx, repos.durable, tenantID, unscopedTask, key, nil, nil, "true")
	require.False(t, waitUnscoped.IsSatisfied)

	results, err := repos.matches.ProcessUserEventMatches(ctx, tenantID, []CandidateEventMatch{{
		ID:             uuid.New(),
		EventTimestamp: time.Now().UTC(),
		Key:            key,
		ResourceHint:   &scopeA,
		Data:           []byte(`{"scope":"a"}`),
	}})
	require.NoError(t, err)
	require.Len(t, results.SatisfiedDurableEventLogEntries, 2)
	require.ElementsMatch(t, []uuid.UUID{taskA.ExternalID, unscopedTask.ExternalID}, []uuid.UUID{
		results.SatisfiedDurableEventLogEntries[0].DurableTaskExternalId,
		results.SatisfiedDurableEventLogEntries[1].DurableTaskExternalId,
	})
	requireUserEventScopeTestWaiterState(t, ctx, repos, taskA, waitA.NodeId, waitA.BranchId, true)
	requireUserEventScopeTestWaiterState(t, ctx, repos, taskB, waitB.NodeId, waitB.BranchId, false)
	requireUserEventScopeTestWaiterState(t, ctx, repos, unscopedTask, waitUnscoped.NodeId, waitUnscoped.BranchId, true)

	results, err = repos.matches.ProcessUserEventMatches(ctx, tenantID, []CandidateEventMatch{{
		ID:             uuid.New(),
		EventTimestamp: time.Now().UTC(),
		Key:            key,
		Data:           []byte(`{"scope":"unscoped"}`),
	}})
	require.NoError(t, err)
	require.Empty(t, results.SatisfiedDurableEventLogEntries)
	requireUserEventScopeTestWaiterState(t, ctx, repos, taskB, waitB.NodeId, waitB.BranchId, false)
}

func TestDurableUserEventScopesMatchMixedLiveBatch(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	repos := newUserEventScopeTestRepositories(t, pool)
	key := "shared-user-event-key"
	scopeA := "scope-a"
	scopeB := "scope-b"
	taskA := createUserEventScopeTestTask(t, ctx, repos, tenantID, 201)
	taskB := createUserEventScopeTestTask(t, ctx, repos, tenantID, 202)
	unscopedTask := createUserEventScopeTestTask(t, ctx, repos, tenantID, 203)

	require.False(t, ingestUserEventScopeTestWaiter(t, ctx, repos.durable, tenantID, taskA, key, &scopeA, nil, `input.scope == "a"`).IsSatisfied)
	require.False(t, ingestUserEventScopeTestWaiter(t, ctx, repos.durable, tenantID, taskB, key, &scopeB, nil, `input.scope == "b"`).IsSatisfied)
	require.False(t, ingestUserEventScopeTestWaiter(t, ctx, repos.durable, tenantID, unscopedTask, key, nil, nil, `input.scope == "unscoped"`).IsSatisfied)

	now := time.Now().UTC()
	results, err := repos.matches.ProcessUserEventMatches(ctx, tenantID, []CandidateEventMatch{
		{
			ID:             uuid.New(),
			EventTimestamp: now.Add(2 * time.Second),
			Key:            key,
			ResourceHint:   &scopeA,
			Data:           []byte(`{"scope":"a"}`),
		},
		{
			ID:             uuid.New(),
			EventTimestamp: now.Add(time.Second),
			Key:            key,
			ResourceHint:   &scopeB,
			Data:           []byte(`{"scope":"b"}`),
		},
		{
			ID:             uuid.New(),
			EventTimestamp: now,
			Key:            key,
			Data:           []byte(`{"scope":"unscoped"}`),
		},
	})
	require.NoError(t, err)
	require.Len(t, results.SatisfiedDurableEventLogEntries, 3)

	payloadsByTask := make(map[uuid.UUID][]byte, len(results.SatisfiedDurableEventLogEntries))
	for _, entry := range results.SatisfiedDurableEventLogEntries {
		payloadsByTask[entry.DurableTaskExternalId] = entry.Data
	}

	require.JSONEq(t, `{"CREATE":{"payload":[{"scope":"a"}]}}`, string(payloadsByTask[taskA.ExternalID]))
	require.JSONEq(t, `{"CREATE":{"payload":[{"scope":"b"}]}}`, string(payloadsByTask[taskB.ExternalID]))
	require.JSONEq(t, `{"CREATE":{"payload":[{"scope":"unscoped"}]}}`, string(payloadsByTask[unscopedTask.ExternalID]))
}

func TestDurableUserEventScopesIsolateHistoricalLookback(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	repos := newUserEventScopeTestRepositories(t, pool)
	key := "shared-user-event-key"
	scopeA := "scope-a"
	scopeB := "scope-b"
	now := time.Now().UTC()
	insertUserEventScopeTestEvent(t, ctx, repos, tenantID, key, &scopeA, now.Add(-2*time.Minute), []byte(`{"scope":"a"}`))
	insertUserEventScopeTestEvent(t, ctx, repos, tenantID, key, &scopeB, now.Add(-time.Minute), []byte(`{"scope":"b"}`))

	excludedSince := now.Add(time.Minute)
	considerEventsSince := now.Add(-3 * time.Minute)
	excludedTaskA := createUserEventScopeTestTask(t, ctx, repos, tenantID, 301)
	excludedTaskB := createUserEventScopeTestTask(t, ctx, repos, tenantID, 302)
	taskA := createUserEventScopeTestTask(t, ctx, repos, tenantID, 303)
	taskB := createUserEventScopeTestTask(t, ctx, repos, tenantID, 304)
	excludedUnscopedTask := createUserEventScopeTestTask(t, ctx, repos, tenantID, 305)

	waitExcludedA := ingestUserEventScopeTestWaiter(t, ctx, repos.durable, tenantID, excludedTaskA, key, &scopeA, &excludedSince, "true")
	require.False(t, waitExcludedA.IsSatisfied)
	waitExcludedB := ingestUserEventScopeTestWaiter(t, ctx, repos.durable, tenantID, excludedTaskB, key, &scopeB, &excludedSince, "true")
	require.False(t, waitExcludedB.IsSatisfied)
	waitExcludedUnscoped := ingestUserEventScopeTestWaiter(t, ctx, repos.durable, tenantID, excludedUnscopedTask, key, nil, nil, "true")
	require.False(t, waitExcludedUnscoped.IsSatisfied)

	resultA := ingestUserEventScopeTestWaiter(t, ctx, repos.durable, tenantID, taskA, key, &scopeA, &considerEventsSince, "true")
	require.True(t, resultA.IsSatisfied)
	require.JSONEq(t, `{"CREATE":{"payload":[{"scope":"a"}]}}`, string(resultA.ResultPayload))
	requireUserEventScopeTestWaiterState(t, ctx, repos, excludedTaskA, waitExcludedA.NodeId, waitExcludedA.BranchId, false)
	requireUserEventScopeTestWaiterState(t, ctx, repos, excludedTaskB, waitExcludedB.NodeId, waitExcludedB.BranchId, false)
	requireUserEventScopeTestWaiterState(t, ctx, repos, excludedUnscopedTask, waitExcludedUnscoped.NodeId, waitExcludedUnscoped.BranchId, false)

	resultB := ingestUserEventScopeTestWaiter(t, ctx, repos.durable, tenantID, taskB, key, &scopeB, &considerEventsSince, "true")
	require.True(t, resultB.IsSatisfied)
	require.JSONEq(t, `{"CREATE":{"payload":[{"scope":"b"}]}}`, string(resultB.ResultPayload))
	requireUserEventScopeTestWaiterState(t, ctx, repos, taskA, resultA.NodeId, resultA.BranchId, true)
	requireUserEventScopeTestWaiterState(t, ctx, repos, taskB, resultB.NodeId, resultB.BranchId, true)
	requireUserEventScopeTestWaiterState(t, ctx, repos, excludedTaskA, waitExcludedA.NodeId, waitExcludedA.BranchId, false)
	requireUserEventScopeTestWaiterState(t, ctx, repos, excludedTaskB, waitExcludedB.NodeId, waitExcludedB.BranchId, false)
	requireUserEventScopeTestWaiterState(t, ctx, repos, excludedUnscopedTask, waitExcludedUnscoped.NodeId, waitExcludedUnscoped.BranchId, false)
}
