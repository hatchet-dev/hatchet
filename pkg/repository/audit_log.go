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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// AuditLogOutboxTopic is the pgoutbox topic that holds audit log entries in the core database
// until the OLAP controller copies them into v1_audit_log. Entries expire after the outbox's
// default expiration (24h). That must not exceed the archiver's delay after a partition ends
// (also 24h), so a queued entry expires before its partition can be archived.
const AuditLogOutboxTopic = "v1_audit_log"

// auditLogFlushTimeout bounds the OLAP writes of one batch. The outbox transaction on the core
// database, and its lock on the topic row, stay open until they finish.
const auditLogFlushTimeout = 10 * time.Second

// AuditLogEntry is one v1_audit_log row. Enqueue fills in a zero ID or InsertedAt, and both are
// written as queued: with TenantID they form the table's key, so a batch delivered twice inserts
// each row once.
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
	case e.ID == uuid.Nil:
		return errors.New("id is required")
	case e.InsertedAt.IsZero():
		return errors.New("inserted at is required")
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

// AuditLogSink writes audit log entries to v1_audit_log. A batch can be delivered more than
// once, so the write must ignore entries that already exist.
type AuditLogSink interface {
	CreateAuditLogs(ctx context.Context, entries []AuditLogEntry) error
}

// AuditLogOutbox queues audit log entries in the core database, so recording them doesn't
// depend on the OLAP database being reachable.
type AuditLogOutbox interface {
	// Enqueue writes the entries to the outbox in their own transaction on the core database.
	Enqueue(ctx context.Context, entries []AuditLogEntry) error

	// Run copies queued entries to sink until ctx is cancelled.
	Run(ctx context.Context, sink AuditLogSink) error
}

type auditLogOutbox struct {
	outbox pgoutbox.Outbox
	pool   *pgxpool.Pool
}

// NewAuditLogOutbox needs an outbox whose tables live in the database behind pool.
func NewAuditLogOutbox(outbox pgoutbox.Outbox, pool *pgxpool.Pool) AuditLogOutbox {
	return &auditLogOutbox{
		outbox: outbox,
		pool:   pool,
	}
}

func (o *auditLogOutbox) Enqueue(ctx context.Context, entries []AuditLogEntry) error {
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

func (o *auditLogOutbox) Run(ctx context.Context, sink AuditLogSink) error {
	o.outbox.AddFlusher(AuditLogOutboxTopic, &auditLogFlusher{sink: sink})

	return o.outbox.Subscribe(ctx, AuditLogOutboxTopic)
}

type auditLogFlusher struct {
	sink AuditLogSink
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

	return f.sink.CreateAuditLogs(writeCtx, entries)
}

func (r *OLAPRepositoryImpl) CreateAuditLogs(ctx context.Context, entries []AuditLogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	params := sqlcv1.CreateAuditLogsParams{
		Ids:            make([]uuid.UUID, len(entries)),
		Insertedats:    make([]pgtype.Timestamptz, len(entries)),
		Tenantids:      make([]uuid.UUID, len(entries)),
		Actortypes:     make([]string, len(entries)),
		Actorids:       make([]uuid.UUID, len(entries)),
		Actions:        make([]string, len(entries)),
		Correlationids: make([]pgtype.Text, len(entries)),
		Resourceids:    make([]string, len(entries)),
		Resourcetypes:  make([]string, len(entries)),
		Ipaddresses:    make([]pgtype.Text, len(entries)),
		Useragents:     make([]pgtype.Text, len(entries)),
	}

	for i, entry := range entries {
		params.Ids[i] = entry.ID
		params.Insertedats[i] = pgtype.Timestamptz{Time: entry.InsertedAt, Valid: true}
		params.Tenantids[i] = entry.TenantID
		params.Actortypes[i] = string(entry.ActorType)
		params.Actorids[i] = entry.ActorID
		params.Actions[i] = entry.Action
		params.Correlationids[i] = optionalText(entry.CorrelationID)
		params.Resourceids[i] = entry.ResourceID
		params.Resourcetypes[i] = entry.ResourceType
		params.Ipaddresses[i] = optionalText(entry.IPAddress)
		params.Useragents[i] = optionalText(entry.UserAgent)
	}

	return r.queries.CreateAuditLogs(ctx, r.pool.ForShared(), params)
}

func optionalText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *s, Valid: true}
}
