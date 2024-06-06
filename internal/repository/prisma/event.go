package prisma

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/metered"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type eventAPIRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewEventAPIRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.EventAPIRepository {
	queries := dbsqlc.New()

	return &eventAPIRepository{
		client:  client,
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

func (r *eventAPIRepository) ListEvents(tenantId string, opts *repository.ListEventOpts) (*repository.ListEventResult, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	res := &repository.ListEventResult{}

	pgTenantId := &pgtype.UUID{}

	if err := pgTenantId.Scan(tenantId); err != nil {
		return nil, err
	}

	queryParams := dbsqlc.ListEventsParams{
		TenantId: *pgTenantId,
	}

	countParams := dbsqlc.CountEventsParams{
		TenantId: *pgTenantId,
	}

	if opts.Search != nil {
		queryParams.Search = sqlchelpers.TextFromStr(*opts.Search)
		countParams.Search = sqlchelpers.TextFromStr(*opts.Search)
	}

	if opts.Offset != nil {
		queryParams.Offset = *opts.Offset
	}

	if opts.Limit != nil {
		queryParams.Limit = *opts.Limit
	}

	if opts.Keys != nil {
		queryParams.Keys = opts.Keys
		countParams.Keys = opts.Keys
	}

	if opts.Workflows != nil {
		queryParams.Workflows = opts.Workflows
		countParams.Workflows = opts.Workflows
	}

	if opts.WorkflowRunStatus != nil {
		statuses := make([]string, 0)

		for _, status := range opts.WorkflowRunStatus {
			statuses = append(statuses, string(status))
		}

		queryParams.Statuses = statuses
		countParams.Statuses = statuses
	}

	if opts.AdditionalMetadata != nil {
		queryParams.AdditionalMetadata = opts.AdditionalMetadata
	}

	orderByField := "createdAt"
	orderByDirection := "DESC"

	if opts.OrderBy != nil {
		orderByField = *opts.OrderBy
	}

	if opts.OrderDirection != nil {
		orderByDirection = *opts.OrderDirection
	}

	queryParams.Orderby = orderByField + " " + orderByDirection

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), r.l, tx.Rollback)

	events, err := r.queries.ListEvents(context.Background(), tx, queryParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			events = make([]*dbsqlc.ListEventsRow, 0)
		} else {
			return nil, fmt.Errorf("could not list events: %w", err)
		}
	}

	count, err := r.queries.CountEvents(context.Background(), tx, countParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			count = 0
		} else {
			return nil, fmt.Errorf("could not count events: %w", err)
		}
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	res.Rows = events
	res.Count = int(count)

	return res, nil
}

func (r *eventAPIRepository) ListEventKeys(tenantId string) ([]string, error) {
	var rows []struct {
		Key string `json:"key"`
	}

	err := r.client.Prisma.QueryRaw(
		`
		SELECT DISTINCT ON("Event"."key") "Event"."key"
		FROM "Event"
		WHERE
		"Event"."tenantId"::text = $1
		`,
		tenantId,
	).Exec(context.Background(), &rows)

	if err != nil {
		return nil, err
	}

	keys := make([]string, len(rows))

	for i, row := range rows {
		keys[i] = row.Key
	}

	return keys, nil
}

func (r *eventAPIRepository) GetEventById(id string) (*db.EventModel, error) {
	return r.client.Event.FindUnique(
		db.Event.ID.Equals(id),
	).Exec(context.Background())
}

func (r *eventAPIRepository) ListEventsById(tenantId string, ids []string) ([]db.EventModel, error) {
	return r.client.Event.FindMany(
		db.Event.ID.In(ids),
		db.Event.TenantID.Equals(tenantId),
	).Exec(context.Background())
}

type eventEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
	m       *metered.Metered
}

func NewEventEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, m *metered.Metered) repository.EventEngineRepository {
	queries := dbsqlc.New()

	return &eventEngineRepository{
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
		m:       m,
	}
}

func (r *eventEngineRepository) GetEventForEngine(ctx context.Context, tenantId, id string) (*dbsqlc.Event, error) {
	return r.queries.GetEventForEngine(ctx, r.pool, sqlchelpers.UUIDFromStr(id))
}

func (r *eventEngineRepository) CreateEvent(ctx context.Context, opts *repository.CreateEventOpts) (*dbsqlc.Event, error) {
	return metered.MakeMetered(ctx, r.m, dbsqlc.LimitResourceEVENT, opts.TenantId, func() (*dbsqlc.Event, error) {

		ctx, span := telemetry.NewSpan(ctx, "db-create-event")
		defer span.End()

		if err := r.v.Validate(opts); err != nil {
			return nil, err
		}

		createParams := dbsqlc.CreateEventParams{
			ID:                 sqlchelpers.UUIDFromStr(uuid.New().String()),
			Key:                opts.Key,
			Tenantid:           sqlchelpers.UUIDFromStr(opts.TenantId),
			Data:               opts.Data,
			Additionalmetadata: opts.AdditionalMetadata,
		}

		if opts.ReplayedEvent != nil {
			createParams.ReplayedFromId = sqlchelpers.UUIDFromStr(*opts.ReplayedEvent)
		}

		e, err := r.queries.CreateEvent(
			ctx,
			r.pool,
			createParams,
		)

		if err != nil {
			return nil, fmt.Errorf("could not create event: %w", err)
		}

		return e, nil
	})
}

func (r *eventEngineRepository) ListEventsByIds(ctx context.Context, tenantId string, ids []string) ([]*dbsqlc.Event, error) {
	pgIds := make([]pgtype.UUID, len(ids))

	for i, id := range ids {
		if err := pgIds[i].Scan(id); err != nil {
			return nil, err
		}
	}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	return r.queries.ListEventsByIDs(ctx, r.pool, dbsqlc.ListEventsByIDsParams{
		Tenantid: pgTenantId,
		Ids:      pgIds,
	})
}
