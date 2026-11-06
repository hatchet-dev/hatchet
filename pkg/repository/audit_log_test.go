//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/pgoutbox"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestAuditLogOutbox(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	outbox, err := pgoutbox.NewOutbox(ctx, pool, pgoutbox.WithAutoMigrate(false))
	require.NoError(t, err)

	auditLogs := NewAuditLogOutbox(outbox, pool)
	olap := createOLAPRepository(pool)

	outbox.AddFlusher(AuditLogOutboxTopic, &auditLogFlusher{sink: olap})

	t.Run("delivers queued entries with their original timestamps", func(t *testing.T) {
		tenantId := uuid.New()
		correlationId, ip, userAgent := "corr-1", "10.0.0.1", "test-agent"
		queuedAt := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)

		require.NoError(t, auditLogs.Enqueue(ctx, []AuditLogEntry{
			{
				InsertedAt:    queuedAt,
				TenantID:      tenantId,
				ActorType:     sqlcv1.V1AuditLogActorTypeUser,
				ActorID:       uuid.New(),
				Action:        "WorkflowRunCreate",
				CorrelationID: &correlationId,
				ResourceID:    "run-1",
				ResourceType:  "workflow-run",
				IPAddress:     &ip,
				UserAgent:     &userAgent,
			},
			testAuditLogEntry(tenantId, "run-2"),
		}))

		drainAuditLogOutbox(t, ctx, outbox)

		rows := listAuditLogs(t, ctx, pool, tenantId)
		require.Len(t, rows, 2)

		assert.Equal(t, "run-1", rows[0].resourceId)
		assert.True(t, rows[0].insertedAt.Equal(queuedAt))
		assert.Equal(t, "user", rows[0].actorType)
		assert.Equal(t, pgtype.Text{String: correlationId, Valid: true}, rows[0].correlationId)
		assert.Equal(t, pgtype.Text{String: ip, Valid: true}, rows[0].ipAddress)
		assert.Equal(t, pgtype.Text{String: userAgent, Valid: true}, rows[0].userAgent)

		assert.Equal(t, "run-2", rows[1].resourceId)
		assert.Equal(t, "api_key", rows[1].actorType)
		assert.False(t, rows[1].correlationId.Valid)
		assert.False(t, rows[1].ipAddress.Valid)
		assert.False(t, rows[1].userAgent.Valid)

		assert.Zero(t, countAuditLogOutboxMessages(t, ctx, pool))
	})

	t.Run("redelivered entries are inserted once", func(t *testing.T) {
		tenantId := uuid.New()
		entry := testAuditLogEntry(tenantId, "run-1")
		entry.ID = uuid.New()
		entry.InsertedAt = time.Now().UTC()

		require.NoError(t, olap.CreateAuditLogs(ctx, []AuditLogEntry{entry}))
		require.NoError(t, olap.CreateAuditLogs(ctx, []AuditLogEntry{entry}))

		assert.Len(t, listAuditLogs(t, ctx, pool, tenantId), 1)
	})

	t.Run("a batch that can't be written stays queued", func(t *testing.T) {
		tenantId := uuid.New()

		// v1_audit_log has no partition this far back, so the insert fails a check constraint.
		noPartition := testAuditLogEntry(tenantId, "no-partition")
		noPartition.InsertedAt = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

		require.NoError(t, auditLogs.Enqueue(ctx, []AuditLogEntry{testAuditLogEntry(tenantId, "run-1"), noPartition}))

		_, err := outbox.ProcessMessages(ctx, AuditLogOutboxTopic)
		require.Error(t, err)

		assert.Equal(t, 2, countAuditLogOutboxMessages(t, ctx, pool))
		assert.Empty(t, listAuditLogs(t, ctx, pool, tenantId))

		_, err = pool.Exec(ctx, `DELETE FROM outbox.messages WHERE topic = $1`, AuditLogOutboxTopic)
		require.NoError(t, err)
	})

	t.Run("an invalid entry fails the enqueue and queues nothing", func(t *testing.T) {
		invalid := testAuditLogEntry(uuid.New(), "run-1")
		invalid.Action = ""

		err := auditLogs.Enqueue(ctx, []AuditLogEntry{testAuditLogEntry(uuid.New(), "run-1"), invalid})
		require.Error(t, err)

		assert.Zero(t, countAuditLogOutboxMessages(t, ctx, pool))
	})

	t.Run("run copies entries until cancelled", func(t *testing.T) {
		tenantId := uuid.New()

		require.NoError(t, auditLogs.Enqueue(ctx, []AuditLogEntry{testAuditLogEntry(tenantId, "run-1")}))

		runCtx, stop := context.WithCancel(ctx)
		done := make(chan error, 1)

		go func() {
			done <- auditLogs.Run(runCtx, olap)
		}()

		require.Eventually(t, func() bool {
			return len(listAuditLogs(t, ctx, pool, tenantId)) == 1
		}, 10*time.Second, 100*time.Millisecond)

		stop()
		assert.ErrorIs(t, <-done, context.Canceled)
	})
}

func testAuditLogEntry(tenantId uuid.UUID, resourceId string) AuditLogEntry {
	return AuditLogEntry{
		TenantID:     tenantId,
		ActorType:    sqlcv1.V1AuditLogActorTypeApiKey,
		ActorID:      uuid.New(),
		Action:       "/v1.AdminService/TriggerWorkflowRun",
		ResourceID:   resourceId,
		ResourceType: "workflow-run",
	}
}

func drainAuditLogOutbox(t *testing.T, ctx context.Context, outbox pgoutbox.Outbox) {
	t.Helper()

	for {
		msgs, err := outbox.ProcessMessages(ctx, AuditLogOutboxTopic)
		require.NoError(t, err)

		if len(msgs) == 0 {
			return
		}
	}
}

type auditLogRow struct {
	resourceId    string
	actorType     string
	insertedAt    time.Time
	correlationId pgtype.Text
	ipAddress     pgtype.Text
	userAgent     pgtype.Text
}

func listAuditLogs(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantId uuid.UUID) []auditLogRow {
	t.Helper()

	rows, err := pool.Query(ctx, `
		SELECT resource_id, actor_type::TEXT, inserted_at, correlation_id, ip_address, user_agent
		FROM v1_audit_log
		WHERE tenant_id = $1
		ORDER BY resource_id`, tenantId)
	require.NoError(t, err)
	defer rows.Close()

	var res []auditLogRow

	for rows.Next() {
		var r auditLogRow
		require.NoError(t, rows.Scan(&r.resourceId, &r.actorType, &r.insertedAt, &r.correlationId, &r.ipAddress, &r.userAgent))
		res = append(res, r)
	}

	require.NoError(t, rows.Err())

	return res
}

func countAuditLogOutboxMessages(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int {
	t.Helper()

	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox.messages WHERE topic = $1`, AuditLogOutboxTopic).Scan(&n))

	return n
}
