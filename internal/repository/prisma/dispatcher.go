package prisma

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type dispatcherRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewDispatcherRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.DispatcherRepository {
	queries := dbsqlc.New()

	return &dispatcherRepository{
		client:  client,
		pool:    pool,
		queries: queries,
		v:       v,
		l:       l,
	}
}

func (d *dispatcherRepository) GetDispatcherForWorker(workerId string) (*db.DispatcherModel, error) {
	return d.client.Dispatcher.FindFirst(
		db.Dispatcher.Workers.Some(
			db.Worker.ID.Equals(workerId),
		),
	).Exec(context.Background())
}

func (d *dispatcherRepository) CreateNewDispatcher(opts *repository.CreateDispatcherOpts) (*db.DispatcherModel, error) {
	return d.client.Dispatcher.CreateOne(
		db.Dispatcher.ID.Set(opts.ID),
	).Exec(context.Background())
}

func (d *dispatcherRepository) UpdateDispatcher(dispatcherId string, opts *repository.UpdateDispatcherOpts) (*db.DispatcherModel, error) {
	if err := d.v.Validate(opts); err != nil {
		return nil, err
	}

	return d.client.Dispatcher.FindUnique(
		db.Dispatcher.ID.Equals(dispatcherId),
	).Update(
		db.Dispatcher.LastHeartbeatAt.SetIfPresent(opts.LastHeartbeatAt),
	).Exec(context.Background())
}

func (d *dispatcherRepository) AddWorker(dispatcherId, workerId string) (*db.DispatcherModel, error) {
	return d.client.Dispatcher.FindUnique(
		db.Dispatcher.ID.Equals(dispatcherId),
	).Update(
		db.Dispatcher.Workers.Link(
			db.Worker.ID.Equals(workerId),
		),
	).Exec(context.Background())
}

func (d *dispatcherRepository) Delete(dispatcherId string) error {
	_, err := d.client.Dispatcher.FindUnique(
		db.Dispatcher.ID.Equals(dispatcherId),
	).Delete().Exec(context.Background())

	return err
}

func (d *dispatcherRepository) UpdateStaleDispatchers(onStale func(dispatcherId string, getValidDispatcherId func() string) error) error {
	tx, err := d.pool.Begin(context.Background())

	if err != nil {
		return err
	}

	defer deferRollback(context.Background(), d.l, tx.Rollback)

	staleDispatchers, err := d.queries.ListStaleDispatchers(context.Background(), tx)

	if err != nil {
		return err
	}

	activeDispatchers, err := d.queries.ListActiveDispatchers(context.Background(), tx)

	if err != nil {
		return err
	}

	dispatchersToDelete := make([]pgtype.UUID, 0)

	for i, dispatcher := range staleDispatchers {
		err := onStale(sqlchelpers.UUIDToStr(dispatcher.Dispatcher.ID), func() string {
			// assign tickers in round-robin fashion
			return sqlchelpers.UUIDToStr(activeDispatchers[i%len(activeDispatchers)].Dispatcher.ID)
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
