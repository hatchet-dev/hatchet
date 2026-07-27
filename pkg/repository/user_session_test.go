//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createUserSessionRepository(pool *pgxpool.Pool) *userSessionRepository {
	logger := zerolog.Nop()
	shared := &sharedRepository{
		pool:    pool,
		ddlPool: pool,
		l:       &logger,
		queries: sqlcv1.New(),
	}
	return &userSessionRepository{
		sharedRepository: shared,
	}
}

func createTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	userId := uuid.New()
	_, err := pool.Exec(ctx(t), `
		INSERT INTO "User" ("id", "email", "emailVerified", "name", "createdAt", "updatedAt")
		VALUES ($1, $2, false, $3, NOW(), NOW())
	`, userId, userId.String()+"@test.com", "Test User")
	require.NoError(t, err)
	return userId
}

func sessionExists(t *testing.T, pool *pgxpool.Pool, sessionId uuid.UUID) bool {
	t.Helper()
	var exists bool
	err := pool.QueryRow(ctx(t), `SELECT EXISTS(SELECT 1 FROM "UserSession" WHERE "id" = $1)`, sessionId).Scan(&exists)
	require.NoError(t, err)
	return exists
}

func countExistingSessions(t *testing.T, pool *pgxpool.Pool, sessionIds []uuid.UUID) int {
	t.Helper()
	if len(sessionIds) == 0 {
		return 0
	}

	var count int
	err := pool.QueryRow(ctx(t), `SELECT COUNT(*) FROM "UserSession" WHERE "id" = ANY($1::uuid[])`, sessionIds).Scan(&count)
	require.NoError(t, err)

	return count
}

func TestCleanupUserSessions(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	testUserId := createTestUser(t, pool)
	repo := createUserSessionRepository(pool)

	err := repo.CleanupUserSessions(ctx(t))
	require.NoError(t, err)

	testCases := []struct {
		name         string
		expiresAt    time.Time
		userId       *uuid.UUID
		ageHours     int
		shouldDelete bool
	}{
		{
			name:         "expired-session-with-user",
			expiresAt:    time.Now().UTC().Add(-1 * time.Hour),
			userId:       &testUserId,
			ageHours:     0,
			shouldDelete: true,
		},
		{
			name:         "just-expired-session",
			expiresAt:    time.Now().UTC().Add(-1 * time.Second),
			userId:       &testUserId,
			ageHours:     0,
			shouldDelete: true,
		},
		{
			name:         "unauthenticated-old-session",
			expiresAt:    time.Now().UTC().Add(48 * time.Hour),
			userId:       nil,
			ageHours:     25,
			shouldDelete: true,
		},
		{
			name:         "valid-session-with-user",
			expiresAt:    time.Now().UTC().Add(24 * time.Hour),
			userId:       &testUserId,
			ageHours:     0,
			shouldDelete: false,
		},
		{
			name:         "unauthenticated-recent-session",
			expiresAt:    time.Now().UTC().Add(48 * time.Hour),
			userId:       nil,
			ageHours:     1,
			shouldDelete: false,
		},
		{
			name:         "session-expires-in-1-second",
			expiresAt:    time.Now().UTC().Add(1 * time.Second),
			userId:       &testUserId,
			ageHours:     0,
			shouldDelete: false,
		},
	}

	sessionIds := make(map[string]uuid.UUID)
	for _, tc := range testCases {
		sessionId := uuid.New()
		sessionIds[tc.name] = sessionId

		var err error
		if tc.ageHours > 0 {
			_, err = pool.Exec(ctx(t), `
				INSERT INTO "UserSession" ("id", "expiresAt", "userId", "data", "createdAt", "updatedAt")
				VALUES ($1, $2 AT TIME ZONE 'UTC', $3, '{}', NOW() - INTERVAL '1 hour' * $4, NOW() - INTERVAL '1 hour' * $4)
			`, sessionId, tc.expiresAt, tc.userId, tc.ageHours)
		} else {
			_, err = pool.Exec(ctx(t), `
				INSERT INTO "UserSession" ("id", "expiresAt", "userId", "data", "createdAt", "updatedAt")
				VALUES ($1, $2 AT TIME ZONE 'UTC', $3, '{}', NOW(), NOW())
			`, sessionId, tc.expiresAt, tc.userId)
		}
		require.NoError(t, err, "failed to insert session: %s", tc.name)
	}

	err = repo.CleanupUserSessions(ctx(t))
	require.NoError(t, err)

	for _, tc := range testCases {
		sessionId := sessionIds[tc.name]
		exists := sessionExists(t, pool, sessionId)

		if tc.shouldDelete {
			assert.False(t, exists, "session '%s' should be deleted from DB", tc.name)
		} else {
			assert.True(t, exists, "session '%s' should exist in DB", tc.name)
		}
	}
}

func TestCleanupUserSessions_MultiBatch(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	testUserId := createTestUser(t, pool)
	repo := createUserSessionRepository(pool)

	const totalSessions = 1500
	sessionIds := make([]uuid.UUID, totalSessions)

	for i := 0; i < totalSessions; i++ {
		sessionIds[i] = uuid.New()
		_, err := pool.Exec(ctx(t), `
			INSERT INTO "UserSession" ("id", "expiresAt", "userId", "data", "createdAt", "updatedAt")
			VALUES ($1, $2 AT TIME ZONE 'UTC', $3, '{}', NOW(), NOW())
		`, sessionIds[i], time.Now().UTC().Add(-1*time.Hour), testUserId)
		require.NoError(t, err)
	}

	err := repo.CleanupUserSessions(ctx(t))
	require.NoError(t, err)

	existingCount := countExistingSessions(t, pool, sessionIds)
	assert.Equal(t, 0, existingCount, "all %d sessions should be deleted", totalSessions)
}

func TestCleanupUserSessions_ExactBatchBoundary(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	testUserId := createTestUser(t, pool)
	repo := createUserSessionRepository(pool)

	const totalSessions = 1000
	sessionIds := make([]uuid.UUID, totalSessions)

	for i := 0; i < totalSessions; i++ {
		sessionIds[i] = uuid.New()
		_, err := pool.Exec(ctx(t), `
			INSERT INTO "UserSession" ("id", "expiresAt", "userId", "data", "createdAt", "updatedAt")
			VALUES ($1, $2 AT TIME ZONE 'UTC', $3, '{}', NOW(), NOW())
		`, sessionIds[i], time.Now().UTC().Add(-1*time.Hour), testUserId)
		require.NoError(t, err)
	}

	err := repo.CleanupUserSessions(ctx(t))
	require.NoError(t, err)

	existingCount := countExistingSessions(t, pool, sessionIds)
	assert.Equal(t, 0, existingCount, "all %d sessions should be deleted", totalSessions)
}

func ctx(t *testing.T) context.Context {
	t.Helper()
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}
