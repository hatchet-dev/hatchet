package postgres

import "github.com/hatchet-dev/hatchet/pkg/repository"

type schedulerRepository struct {
	lease        repository.LeaseRepository
	queueFactory repository.QueueFactoryRepository
	rateLimit    repository.RateLimitRepository
	assignment   repository.AssignmentRepository
}

func newSchedulerRepository(shared *sharedRepository) *schedulerRepository {
	return &schedulerRepository{
		lease:        newLeaseRepository(shared),
		queueFactory: newQueueFactoryRepository(shared),
		rateLimit:    newRateLimitRepository(shared),
		assignment:   newAssignmentRepository(shared),
	}
}

func (d *schedulerRepository) Lease() repository.LeaseRepository {
	return d.lease
}

func (d *schedulerRepository) QueueFactory() repository.QueueFactoryRepository {
	return d.queueFactory
}

func (d *schedulerRepository) RateLimit() repository.RateLimitRepository {
	return d.rateLimit
}

func (d *schedulerRepository) Assignment() repository.AssignmentRepository {
	return d.assignment
}
