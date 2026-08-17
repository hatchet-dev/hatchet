//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestUserEventScopeIsolatedForLiveMatches(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	key := "shared-user-event-key"
	scopeA := "scope-a"
	scopeB := "scope-b"
	logger := zerolog.Nop()
	repo := &MatchRepositoryImpl{sharedRepository: &sharedRepository{
		pool:      pool,
		l:         &logger,
		queries:   sqlcv1.New(),
		celParser: cel.NewCELParser(),
	}}
	insertedAt := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	register := func(scope string, taskID int64) uuid.UUID {
		groupID := uuid.New()
		err := repo.RegisterSignalMatchConditions(ctx, tenantID, []ExternalCreateSignalMatchOpts{{
			Conditions: []CreateExternalSignalConditionOpt{{
				Kind:            CreateExternalSignalConditionKindUSEREVENT,
				ReadableDataKey: "payload",
				OrGroupId:       groupID,
				UserEventKey:    &key,
				UserEventScope:  &scope,
				Expression:      "true",
			}},
			SignalTaskId:         taskID,
			SignalTaskInsertedAt: insertedAt,
			SignalExternalId:     uuid.New(),
			SignalTaskExternalId: uuid.New(),
			SignalKey:            "signal",
		}})
		require.NoError(t, err)
		return groupID
	}
	groupA := register(scopeA, 1)
	groupB := register(scopeB, 2)

	_, err := repo.ProcessUserEventMatches(ctx, tenantID, []CandidateEventMatch{{
		ID:             uuid.New(),
		EventTimestamp: time.Now(),
		Key:            key,
		ResourceHint:   &scopeA,
		Data:           []byte(`{"value":"scope-a"}`),
	}})
	require.NoError(t, err)

	var satisfiedA, satisfiedB bool
	err = pool.QueryRow(ctx, `
		SELECT a.is_satisfied, b.is_satisfied
		FROM v1_match_condition a
		JOIN v1_match_condition b ON a.event_key = b.event_key
		WHERE a.or_group_id = $1 AND b.or_group_id = $2
		`, groupA, groupB).Scan(&satisfiedA, &satisfiedB)
	require.NoError(t, err)
	require.True(t, satisfiedA)
	require.False(t, satisfiedB)
}

func TestUserEventLookbackPairsKeyAndScope(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	key := "shared-user-event-key"
	now := time.Now().UTC()

	_, err := pool.Exec(ctx, `
		INSERT INTO v1_event (tenant_id, external_id, key, scope, seen_at, additional_metadata)
		VALUES
			($1, $2, $3, 'scope-a', $4, '{}'),
			($1, $5, $3, 'scope-b', $4, '{}')
		`, tenantID, uuid.New(), key, now, uuid.New())
	require.NoError(t, err)

	rows, err := sqlcv1.New().GetPreviousMatchingEventsByKeysWithScopeHint(ctx, pool, sqlcv1.GetPreviousMatchingEventsByKeysWithScopeHintParams{
		Tenantid:   tenantID,
		Keys:       []string{key},
		Seensinces: []pgtype.Timestamptz{{Time: now.Add(-time.Minute), Valid: true}},
		Scopes:     []string{"scope-a"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "scope-a", rows[0].Scope.String)

	var resourceHint *string
	if rows[0].Scope.Valid {
		resourceHint = &rows[0].Scope.String
	}
	candidate := CandidateEventMatch{Key: rows[0].Key, ResourceHint: resourceHint}
	require.NotNil(t, candidate.ResourceHint)
	require.Equal(t, "scope-a", *candidate.ResourceHint)
}
