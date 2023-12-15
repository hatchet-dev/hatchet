package prisma

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type eventRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
}

func NewEventRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator) repository.EventRepository {
	queries := dbsqlc.New()

	return &eventRepository{
		client:  client,
		pool:    pool,
		v:       v,
		queries: queries,
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

	defer tx.Rollback(context.Background())

	events, err := r.queries.ListEvents(context.Background(), tx, queryParams)

	if err != nil {
		return nil, err
	}

	count, err := r.queries.CountEvents(context.Background(), tx, countParams)

	if err != nil {
		return nil, err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
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

func (r *eventRepository) CreateEvent(opts *repository.CreateEventOpts) (*db.EventModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.EventSetParam{
		db.Event.Data.SetIfPresent(opts.Data),
	}

	if opts.ReplayedEvent != nil {
		params = append(params, db.Event.ReplayedFrom.Link(
			db.Event.ID.Equals(*opts.ReplayedEvent),
		))
	}

	return r.client.Event.CreateOne(
		db.Event.Key.Set(opts.Key),
		db.Event.Tenant.Link(
			db.Tenant.ID.Equals(opts.TenantId),
		),
		params...,
	).Exec(context.Background())
}
