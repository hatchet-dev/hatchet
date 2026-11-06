package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/pgoutbox"
	outboxsqlc "github.com/hatchet-dev/pgoutbox/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// AuditLogOutboxTopic holds audit log entries in the core database until the OLAP controller
// copies them into v1_audit_log. The outbox's default expiration must not exceed the archiver's
// delay after a partition ends, so no queued entry outlives its partition.
const AuditLogOutboxTopic = "v1_audit_log"

// auditLogFlushTimeout bounds the OLAP write of one batch. The outbox transaction on the core
// database, and its lock on the topic row, stay open until it finishes.
const auditLogFlushTimeout = 10 * time.Second

// AuditLogEntry is one v1_audit_log row; its JSON field names are the table's column names.
// Enqueue fills in a zero ID or InsertedAt before queueing, so redelivery keeps an entry's key.
type AuditLogEntry struct {
	ID         uuid.UUID `json:"id"`
	InsertedAt time.Time `json:"inserted_at"`

	TenantID      uuid.UUID                  `json:"tenant_id"`
	ActorType     sqlcv1.V1AuditLogActorType `json:"actor_type"`
	ActorID       uuid.UUID                  `json:"actor_id"`
	Action        string                     `json:"action"`
	CorrelationID *string                    `json:"correlation_id,omitempty"`
	ResourceID    string                     `json:"resource_id"`
	ResourceType  string                     `json:"resource_type"`
	IPAddress     *string                    `json:"ip_address,omitempty"`
	UserAgent     *string                    `json:"user_agent,omitempty"`
}

func (e *AuditLogEntry) validate() error {
	switch {
	case e.TenantID == uuid.Nil:
		return errors.New("tenant id is required")
	case e.ActorType != sqlcv1.V1AuditLogActorTypeUser && e.ActorType != sqlcv1.V1AuditLogActorTypeApiKey:
		return fmt.Errorf("unknown actor type %q", e.ActorType)
	case e.ActorID == uuid.Nil:
		return errors.New("actor id is required")
	case e.Action == "":
		return errors.New("action is required")
	case e.ResourceID == "":
		return errors.New("resource id is required")
	case e.ResourceType == "":
		return errors.New("resource type is required")
	}

	return nil
}

// AuditLogOutbox queues audit log entries in the core database, so recording them doesn't
// depend on the OLAP database being reachable.
type AuditLogOutbox struct {
	outbox pgoutbox.Outbox
	pool   *pgxpool.Pool
}

// NewAuditLogOutbox needs an outbox whose tables live in the database behind pool.
func NewAuditLogOutbox(outbox pgoutbox.Outbox, pool *pgxpool.Pool) *AuditLogOutbox {
	return &AuditLogOutbox{
		outbox: outbox,
		pool:   pool,
	}
}

// Enqueue writes the entries to the outbox in their own transaction on the core database.
func (o *AuditLogOutbox) Enqueue(ctx context.Context, entries []AuditLogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	now := time.Now().UTC()
	msgs := make([]pgoutbox.MessageOpts, len(entries))

	for i, entry := range entries {
		if entry.ID == uuid.Nil {
			entry.ID = uuid.New()
		}

		if entry.InsertedAt.IsZero() {
			entry.InsertedAt = now
		}

		if err := entry.validate(); err != nil {
			return fmt.Errorf("invalid audit log entry: %w", err)
		}

		payload, err := json.Marshal(entry)

		if err != nil {
			return fmt.Errorf("could not marshal audit log entry: %w", err)
		}

		msgs[i] = pgoutbox.MessageOpts{Payload: payload}
	}

	return pgx.BeginFunc(ctx, o.pool, func(tx pgx.Tx) error {
		return o.outbox.AddMessages(ctx, tx, AuditLogOutboxTopic, msgs)
	})
}

// Run copies queued entries into v1_audit_log until ctx is cancelled.
func (o *AuditLogOutbox) Run(ctx context.Context, olap OLAPRepository) error {
	o.outbox.AddFlusher(AuditLogOutboxTopic, &auditLogFlusher{olap: olap})

	return o.outbox.Subscribe(ctx, AuditLogOutboxTopic)
}

type auditLogFlusher struct {
	olap OLAPRepository
}

// Flush fails the whole batch on any error, so the outbox keeps every entry and retries it. An
// entry that can never be written holds up the topic until it expires.
func (f *auditLogFlusher) Flush(ctx pgoutbox.FlushContext, msgs []*outboxsqlc.Message) error {
	entries := make([]AuditLogEntry, len(msgs))

	for i, msg := range msgs {
		if err := json.Unmarshal(msg.Payload, &entries[i]); err != nil {
			return fmt.Errorf("could not decode audit log message %d: %w", msg.ID, err)
		}
	}

	writeCtx, cancel := context.WithTimeout(ctx, auditLogFlushTimeout)
	defer cancel()

	return f.olap.CreateAuditLogs(writeCtx, entries)
}

func (r *OLAPRepositoryImpl) CreateAuditLogs(ctx context.Context, entries []AuditLogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	payload, err := json.Marshal(entries)

	if err != nil {
		return fmt.Errorf("could not marshal audit log entries: %w", err)
	}

	return r.queries.CreateAuditLogs(ctx, r.pool.ForShared(), payload)
}
