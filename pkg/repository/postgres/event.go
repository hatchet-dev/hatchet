package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type eventAPIRepository struct {
	*sharedRepository
}

func NewEventAPIRepository(shared *sharedRepository) repository.EventAPIRepository {
	return &eventAPIRepository{
		sharedRepository: shared,
	}
}

func (r *eventAPIRepository) ListEvents(ctx context.Context, tenantId string, opts *repository.ListEventOpts) (*repository.ListEventResult, error) {
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

	if opts.Ids != nil {
		queryParams.EventIds = make([]pgtype.UUID, len(opts.Ids))
		countParams.EventIds = make([]pgtype.UUID, len(opts.Ids))

		for i := range opts.Ids {
			queryParams.EventIds[i] = sqlchelpers.UUIDFromStr(opts.Ids[i])
			countParams.EventIds[i] = sqlchelpers.UUIDFromStr(opts.Ids[i])
		}
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
		countParams.AdditionalMetadata = opts.AdditionalMetadata
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
	countParams.Orderby = orderByField + " " + orderByDirection

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(context.Background(), r.l, tx.Rollback)

	events, err := r.queries.ListEvents(ctx, tx, queryParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			events = make([]*dbsqlc.ListEventsRow, 0)
		} else {
			return nil, fmt.Errorf("could not list events: %w", err)
		}
	}

	count, err := r.queries.CountEvents(ctx, tx, countParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			count = 0
		} else {
			return nil, fmt.Errorf("could not count events: %w", err)
		}
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	res.Rows = events
	res.Count = int(count)

	return res, nil
}

func (r *eventAPIRepository) ListEventKeys(tenantId string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	keys, err := r.queries.ListEventKeys(ctx, r.pool, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (r *eventAPIRepository) GetEventById(ctx context.Context, id string) (*dbsqlc.Event, error) {
	return r.queries.GetEventForEngine(ctx, r.pool, sqlchelpers.UUIDFromStr(id))
}

func (r *eventAPIRepository) ListEventsById(ctx context.Context, tenantId string, ids []string) ([]*dbsqlc.Event, error) {
	pgIds := make([]pgtype.UUID, len(ids))

	for i, id := range ids {
		pgIds[i] = sqlchelpers.UUIDFromStr(id)
	}

	return r.queries.ListEventsByIDs(
		ctx,
		r.pool,
		dbsqlc.ListEventsByIDsParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			Ids:      pgIds,
		},
	)
}

type eventEngineRepository struct {
	*sharedRepository

	m                   *metered.Metered
	callbacks           []repository.TenantScopedCallback[*dbsqlc.Event]
	createEventKeyCache *lru.Cache[string, bool]
}

func NewEventEngineRepository(shared *sharedRepository, m *metered.Metered, bufferConf buffer.ConfigFileBuffer) repository.EventEngineRepository {
	createEventKeyCache, _ := lru.New[string, bool](2000) // nolint: errcheck - this only returns an error if the size is less than 0

	return &eventEngineRepository{
		sharedRepository:    shared,
		m:                   m,
		createEventKeyCache: createEventKeyCache,
	}
}

func (r *eventEngineRepository) RegisterCreateCallback(callback repository.TenantScopedCallback[*dbsqlc.Event]) {
	if r.callbacks == nil {
		r.callbacks = make([]repository.TenantScopedCallback[*dbsqlc.Event], 0)
	}

	r.callbacks = append(r.callbacks, callback)
}

func (r *eventEngineRepository) GetEventForEngine(ctx context.Context, tenantId, id string) (*dbsqlc.Event, error) {
	return r.queries.GetEventForEngine(ctx, r.pool, sqlchelpers.UUIDFromStr(id))
}

func (r *eventEngineRepository) createEventKeys(ctx context.Context, tx pgx.Tx, keys map[string]struct {
	key      string
	tenantId string
}) error {

	eventKeys := make([]string, 0)
	eventKeysTenantIds := make([]pgtype.UUID, 0)

	for _, eventKey := range keys {
		cacheKey := fmt.Sprintf("%s-%s", eventKey.tenantId, eventKey.key)

		// if the key is already in the cache, skip it
		if _, ok := r.createEventKeyCache.Get(cacheKey); ok {
			continue
		}

		r.l.Debug().Msgf("creating event key %s for tenant %s", eventKey.key, eventKey.tenantId)
		eventKeys = append(eventKeys, eventKey.key)
		eventKeysTenantIds = append(eventKeysTenantIds, sqlchelpers.UUIDFromStr(eventKey.tenantId))
	}

	err := r.queries.CreateEventKeys(ctx, tx, dbsqlc.CreateEventKeysParams{
		Tenantids: eventKeysTenantIds,
		Keys:      eventKeys,
	})

	if err != nil {
		return err
	}

	// add to cache
	for i := range eventKeys {
		r.createEventKeyCache.Add(fmt.Sprintf("%s-%s", sqlchelpers.UUIDToStr(eventKeysTenantIds[i]), eventKeys[i]), true)
	}

	return nil
}

func (r *eventEngineRepository) CreateEvent(ctx context.Context, opts *repository.CreateEventOpts) (*dbsqlc.Event, error) {
	return metered.MakeMetered(ctx, r.m, dbsqlc.LimitResourceEVENT, opts.TenantId, 1, func() (*string, *dbsqlc.Event, error) {

		_, span := telemetry.NewSpan(ctx, "db-create-event")
		defer span.End()

		if err := r.v.Validate(opts); err != nil {
			return nil, nil, err
		}

		createOpts := repository.CreateEventOpts{
			TenantId:           opts.TenantId,
			Key:                opts.Key,
			Data:               opts.Data,
			AdditionalMetadata: opts.AdditionalMetadata,
			ReplayedEvent:      opts.ReplayedEvent,
			Priority:           opts.Priority,
			ResourceHint:       opts.ResourceHint,
		}

		event, err := r.bulkUserEventBuffer.FireAndWait(ctx, opts.TenantId, &createOpts)

		if err != nil {
			return nil, nil, fmt.Errorf("could not buffer event: %w", err)
		}

		for _, cb := range r.callbacks {
			cb.Do(r.l, opts.TenantId, event)
		}

		id := sqlchelpers.UUIDToStr(event.ID)

		if event.TenantId != sqlchelpers.UUIDFromStr(opts.TenantId) {
			panic("tenant id mismatch")
		}

		return &id, event, nil
	})
}

func (r *eventEngineRepository) BulkCreateEvent(ctx context.Context, opts *repository.BulkCreateEventOpts) (*repository.BulkCreateEventResult, error) {

	numberOfResources := len(opts.Events)
	if numberOfResources < math.MinInt32 || numberOfResources > math.MaxInt32 {
		return nil, fmt.Errorf("number of resources is out of range")
	}

	return metered.MakeMetered(ctx, r.m, dbsqlc.LimitResourceEVENT, opts.TenantId, int32(numberOfResources), func() (*string, *repository.BulkCreateEventResult, error) {

		ctx, span := telemetry.NewSpan(ctx, "db-bulk-create-event")
		defer span.End()

		if err := r.v.Validate(opts); err != nil {
			return nil, nil, err
		}
		params := make([]dbsqlc.CreateEventsParams, len(opts.Events))
		ids := make([]pgtype.UUID, len(opts.Events))

		uniqueEventKeys := make(map[string]struct {
			key      string
			tenantId string
		})

		for i, event := range opts.Events {
			eventId := uuid.New().String()

			params[i] = dbsqlc.CreateEventsParams{
				ID:                 sqlchelpers.UUIDFromStr(eventId),
				Key:                event.Key,
				TenantId:           sqlchelpers.UUIDFromStr(event.TenantId),
				Data:               event.Data,
				AdditionalMetadata: event.AdditionalMetadata,
			}

			if event.ReplayedEvent != nil {
				params[i].ReplayedFromId = sqlchelpers.UUIDFromStr(*event.ReplayedEvent)
			}

			uniqueEventKeys[fmt.Sprintf("%s-%s", event.TenantId, event.Key)] = struct {
				key      string
				tenantId string
			}{
				key:      event.Key,
				tenantId: event.TenantId,
			}

			ids[i] = sqlchelpers.UUIDFromStr(eventId)
		}

		// start a transaction
		tx, err := r.pool.Begin(ctx)

		if err != nil {
			return nil, nil, err
		}

		defer sqlchelpers.DeferRollback(ctx, r.l, tx.Rollback)

		err = r.createEventKeys(ctx, tx, uniqueEventKeys)

		if err != nil {
			return nil, nil, fmt.Errorf("could not create event keys: %w", err)
		}
		// create events
		insertCount, err := r.queries.CreateEvents(
			ctx,
			tx,
			params,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("could not create events: %w", err)
		}

		r.l.Info().Msgf("inserted %d events", insertCount)

		events, err := r.queries.GetInsertedEvents(ctx, tx, ids)

		if err != nil {
			return nil, nil, fmt.Errorf("could not retrieve inserted events: %w", err)
		}
		err = tx.Commit(ctx)

		if err != nil {
			return nil, nil, fmt.Errorf("could not commit transaction: %w", err)
		}

		var returnString string

		for _, e := range events {

			for _, cb := range r.callbacks {
				cb.Do(r.l, opts.TenantId, e)
			}

		}

		if len(events) > 0 {

			returnString = sqlchelpers.UUIDToStr(events[0].ID)
		}

		// TODO is this return string important?
		return &returnString, &repository.BulkCreateEventResult{Events: events}, nil
	})
}
func (r *eventEngineRepository) BulkCreateEventSharedTenant(ctx context.Context, opts []*repository.CreateEventOpts) ([]*dbsqlc.Event, error) {

	// need to do the metering beforehand
	numberOfResources := len(opts)
	if numberOfResources < math.MinInt32 || numberOfResources > math.MaxInt32 {
		return nil, fmt.Errorf("number of resources is out of range")
	}

	ctx, span := telemetry.NewSpan(ctx, "db-bulk-create-event-shared-tenant")
	defer span.End()

	for _, opt := range opts {

		if err := r.v.Validate(opt); err != nil {
			return nil, err
		}
	}
	params := make([]dbsqlc.CreateEventsParams, len(opts))
	ids := make([]pgtype.UUID, len(opts))

	for i, event := range opts {

		if i > math.MaxInt32 || i < math.MinInt32 {
			return nil, fmt.Errorf("number of resources is out of range for int 32")
		}

		eventId := uuid.New().String()

		params[i] = dbsqlc.CreateEventsParams{
			ID:                 sqlchelpers.UUIDFromStr(eventId),
			Key:                event.Key,
			TenantId:           sqlchelpers.UUIDFromStr(event.TenantId),
			Data:               event.Data,
			AdditionalMetadata: event.AdditionalMetadata,
			InsertOrder:        sqlchelpers.ToInt(int32(i)),
		}

		if event.ReplayedEvent != nil {
			params[i].ReplayedFromId = sqlchelpers.UUIDFromStr(*event.ReplayedEvent)
		}

		ids[i] = sqlchelpers.UUIDFromStr(eventId)
	}

	// start a transaction
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, r.l, tx.Rollback)

	insertCount, err := r.queries.CreateEvents(
		ctx,
		tx,
		params,
	)

	if err != nil {
		return nil, fmt.Errorf("could not create events: %w", err)
	}

	r.l.Info().Msgf("inserted %d events", insertCount)

	events, err := r.queries.GetInsertedEvents(ctx, tx, ids)

	if err != nil {
		return nil, fmt.Errorf("could not retrieve inserted events: %w", err)
	}
	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	for _, e := range events {

		tenantId := sqlchelpers.UUIDToStr(e.TenantId)

		for _, cb := range r.callbacks {
			cb.Do(r.l, tenantId, e)
		}

	}

	// TODO is this return string important?
	return events, nil

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

func (r *eventEngineRepository) SoftDeleteExpiredEvents(ctx context.Context, tenantId string, before time.Time) (bool, error) {
	hasMore, err := r.queries.SoftDeleteExpiredEvents(ctx, r.pool, dbsqlc.SoftDeleteExpiredEventsParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Createdbefore: sqlchelpers.TimestampFromTime(before),
		Limit:         1000,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return hasMore, nil
}

func (r *eventEngineRepository) ClearEventPayloadData(ctx context.Context, tenantId string) (bool, error) {
	hasMore, err := r.queries.ClearEventPayloadData(ctx, r.pool, dbsqlc.ClearEventPayloadDataParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Limit:    1000,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return hasMore, nil
}
