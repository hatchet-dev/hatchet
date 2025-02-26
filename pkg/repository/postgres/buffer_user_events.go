package postgres

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func newUserEventBuffer(shared *sharedRepository, conf buffer.ConfigFileBuffer) (*buffer.TenantBufferManager[*repository.CreateEventOpts, dbsqlc.Event], error) {
	userEventBufOpts := buffer.TenantBufManagerOpts[*repository.CreateEventOpts, dbsqlc.Event]{
		Name:       "create_user_events",
		OutputFunc: shared.bulkWriteUserEvents,
		SizeFunc:   sizeOfEvent,
		L:          shared.l,
		V:          shared.v,
		Config:     conf,
	}

	manager, err := buffer.NewTenantBufManager(userEventBufOpts)

	if err != nil {
		shared.l.Err(err).Msg("could not create tenant buffer manager")
		return nil, err
	}

	return manager, nil
}

func sizeOfEvent(item *repository.CreateEventOpts) int {
	return len(item.Data) + len(item.AdditionalMetadata)
}

func (r *sharedRepository) bulkWriteUserEvents(ctx context.Context, opts []*repository.CreateEventOpts) ([]*dbsqlc.Event, error) {
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

	return events, nil
}
