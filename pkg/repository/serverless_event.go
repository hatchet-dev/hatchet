package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	defaultServerlessEventListLimit int64 = 100
	maxServerlessEventListLimit     int64 = 500
)

// ServerlessEndpointEventOpts is one event to record. Details is marshalled to JSON as is; nil
// stores {}.
type ServerlessEndpointEventOpts struct {
	CreatedAt  time.Time
	Details    any
	Kind       sqlcv1.V1ServerlessEndpointEventKind
	Message    string
	TenantId   uuid.UUID
	EndpointId uuid.UUID
}

// ServerlessEndpointEventRepository holds what the operator observed about endpoints, for the
// dashboard. The operator writes it on failure-streak starts, status transitions, catalog
// changes and rate-limited delivery failures, never once per poll. The table is partitioned by
// day and its partitions are dropped with the other core tables' after the core retention
// period.
type ServerlessEndpointEventRepository interface {
	// Insert records the events in one statement. One event the database refuses fails them
	// all, with an error IsServerlessEventRowError recognizes.
	Insert(ctx context.Context, events []ServerlessEndpointEventOpts) error
	// List returns the endpoint's most recent events, newest first. Events created before
	// retentionStart, the start of the tenant's retention window, are never returned; since,
	// when set, keeps only later events; limit is clamped to 1..500, 0 meaning 100.
	List(ctx context.Context, tenantId, endpointId uuid.UUID, retentionStart time.Time, since *time.Time, limit int64) ([]*sqlcv1.V1ServerlessEndpointEvent, error)
}

// errServerlessEventEncoding is an event whose details cannot be encoded to JSON.
var errServerlessEventEncoding = errors.New("could not encode serverless endpoint event details")

// IsServerlessEventRowError reports whether an Insert failed on the values of an event rather
// than on the database: details that do not encode, or a value Postgres refuses (a data
// exception, such as a JSONB string it cannot store). Inserting the batch's events one by one
// then loses only the events at fault.
func IsServerlessEventRowError(err error) bool {
	if errors.Is(err, errServerlessEventEncoding) {
		return true
	}

	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgerrcode.IsDataException(pgErr.Code)
}

type serverlessEventRepository struct {
	*sharedRepository
}

func (r *serverlessEventRepository) Insert(ctx context.Context, events []ServerlessEndpointEventOpts) error {
	if len(events) == 0 {
		return nil
	}

	params := sqlcv1.InsertServerlessEndpointEventsParams{
		Tenantids:   make([]uuid.UUID, len(events)),
		Endpointids: make([]uuid.UUID, len(events)),
		Createdats:  make([]pgtype.Timestamptz, len(events)),
		Kinds:       make([]string, len(events)),
		Messages:    make([]string, len(events)),
		Details:     make([][]byte, len(events)),
	}

	for i, ev := range events {
		details := []byte(`{}`)

		if ev.Details != nil {
			encoded, err := json.Marshal(ev.Details)

			if err != nil {
				return fmt.Errorf("%w: %w", errServerlessEventEncoding, err)
			}

			details = encoded
		}

		createdAt := ev.CreatedAt

		if createdAt.IsZero() {
			createdAt = time.Now()
		}

		params.Tenantids[i] = ev.TenantId
		params.Endpointids[i] = ev.EndpointId
		params.Createdats[i] = pgtype.Timestamptz{Time: createdAt, Valid: true}
		params.Kinds[i] = string(ev.Kind)
		params.Messages[i] = ev.Message
		params.Details[i] = details
	}

	return r.queries.InsertServerlessEndpointEvents(ctx, r.pool, params)
}

func (r *serverlessEventRepository) List(ctx context.Context, tenantId, endpointId uuid.UUID, retentionStart time.Time, since *time.Time, limit int64) ([]*sqlcv1.V1ServerlessEndpointEvent, error) {
	switch {
	case limit <= 0:
		limit = defaultServerlessEventListLimit
	case limit > maxServerlessEventListLimit:
		limit = maxServerlessEventListLimit
	}

	params := sqlcv1.ListServerlessEndpointEventsParams{
		Tenantid:       tenantId,
		Endpointid:     endpointId,
		Retentionstart: pgtype.Timestamptz{Time: retentionStart, Valid: true},
		Eventlimit:     limit,
	}

	if since != nil {
		params.Since = pgtype.Timestamptz{Time: *since, Valid: true}
	}

	return r.queries.ListServerlessEndpointEvents(ctx, r.pool, params)
}
