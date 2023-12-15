package prisma

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type dispatcherRepository struct {
	client *db.PrismaClient
	v      validator.Validator
}

func NewDispatcherRepository(client *db.PrismaClient, v validator.Validator) repository.DispatcherRepository {
	return &dispatcherRepository{
		client: client,
		v:      v,
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
