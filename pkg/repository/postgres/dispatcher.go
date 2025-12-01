package postgres

import (
	"github.com/google/uuid"

	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type dispatcherRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewDispatcherRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.DispatcherEngineRepository {
	queries := dbsqlc.New()

	return &dispatcherRepository{
		pool:    pool,
		queries: queries,
		v:       v,
		l:       l,
	}
}

func (d *dispatcherRepository) CreateNewDispatcher(ctx context.Context, opts *repository.CreateDispatcherOpts) (*dbsqlc.Dispatcher, error) {
	if err := d.v.Validate(opts); err != nil {
		return nil, err
	}

	return d.queries.CreateDispatcher(ctx, d.pool, uuid.MustParse(opts.ID))
}

func (d *dispatcherRepository) UpdateDispatcher(ctx context.Context, dispatcherId string, opts *repository.UpdateDispatcherOpts) (*dbsqlc.Dispatcher, error) {
	if err := d.v.Validate(opts); err != nil {
		return nil, err
	}

	return d.queries.UpdateDispatcher(ctx, d.pool, dbsqlc.UpdateDispatcherParams{
		ID:              uuid.MustParse(dispatcherId),
		LastHeartbeatAt: sqlchelpers.TimestampFromTime(opts.LastHeartbeatAt.UTC()),
	})
}

func (d *dispatcherRepository) Delete(ctx context.Context, dispatcherId string) error {
	_, err := d.queries.DeleteDispatcher(ctx, d.pool, uuid.MustParse(dispatcherId))

	return err
}

func (d *dispatcherRepository) UpdateStaleDispatchers(ctx context.Context, onStale func(dispatcherId string, getValidDispatcherId func() string) error) error {
	tx, err := d.pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer sqlchelpers.DeferRollback(context.Background(), d.l, tx.Rollback)

	staleDispatchers, err := d.queries.ListStaleDispatchers(context.Background(), tx)

	if err != nil {
		return err
	}

	activeDispatchers, err := d.queries.ListActiveDispatchers(context.Background(), tx)

	if err != nil {
		return err
	}

	dispatchersToDelete := make([]uuid.UUID, 0)

	for i, dispatcher := range staleDispatchers {
		err := onStale(dispatcher.Dispatcher.ID.String(), func() string {
			// assign tickers in round-robin fashion
			return activeDispatchers[i%len(activeDispatchers)].Dispatcher.ID.String()
		})

		if err != nil {
			return err
		}

		dispatchersToDelete = append(dispatchersToDelete, dispatcher.Dispatcher.ID)
	}

	_, err = d.queries.SetDispatchersInactive(context.Background(), tx, dispatchersToDelete)

	if err != nil {
		return err
	}

	return tx.Commit(context.Background())
}
