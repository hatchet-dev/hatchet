package prisma

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlctoprisma"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type eventRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewEventRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.EventRepository {
	queries := dbsqlc.New()

	return &eventRepository{
		client:  client,
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

func (r *eventRepository) ListEvents(tenantId string, opts *repository.ListEventOpts) (*repository.ListEventResult, error) {
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

func (r *eventRepository) ListEventKeys(tenantId string) ([]string, error) {
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

func (r *eventRepository) GetEventById(id string) (*db.EventModel, error) {
	return r.client.Event.FindUnique(
		db.Event.ID.Equals(id),
	).Exec(context.Background())
}

func (r *eventRepository) ListEventsById(tenantId string, ids []string) ([]db.EventModel, error) {
	return r.client.Event.FindMany(
		db.Event.ID.In(ids),
		db.Event.TenantID.Equals(tenantId),
	).Exec(context.Background())
}

func (r *eventRepository) CreateEvent(ctx context.Context, opts *repository.CreateEventOpts) (*db.EventModel, error) {
	ctx, span := telemetry.NewSpan(ctx, "db-create-event")
	defer span.End()

	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	// dataBytes, err := opts.Data.MarshalJSON()

	// if err != nil {
	// 	return nil, err
	// }

	createParams := dbsqlc.CreateEventParams{
		ID:       sqlchelpers.UUIDFromStr(uuid.New().String()),
		Key:      opts.Key,
		Tenantid: sqlchelpers.UUIDFromStr(opts.TenantId),
		Data:     []byte(json.RawMessage(*opts.Data)),
	}

	if opts.ReplayedEvent != nil {
		createParams.ReplayedFromId = sqlchelpers.UUIDFromStr(*opts.ReplayedEvent)
	}

	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), r.l, tx.Rollback)

	e, err := r.queries.CreateEvent(
		ctx,
		tx,
		createParams,
	)

	if err != nil {
		return nil, fmt.Errorf("could not create event: %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	// params := []db.EventSetParam{
	// 	db.Event.Data.SetIfPresent(opts.Data),
	// }

	// if opts.ReplayedEvent != nil {
	// 	params = append(params, db.Event.ReplayedFrom.Link(
	// 		db.Event.ID.Equals(*opts.ReplayedEvent),
	// 	))
	// }

	// return r.client.Event.CreateOne(
	// 	db.Event.Key.Set(opts.Key),
	// 	db.Event.Tenant.Link(
	// 		db.Tenant.ID.Equals(opts.TenantId),
	// 	),
	// 	params...,
	// ).Exec(ctx)

	return sqlctoprisma.NewConverter[dbsqlc.Event, db.EventModel]().ToPrisma(e), nil
}
