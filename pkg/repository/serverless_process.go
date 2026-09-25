package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type UpsertServerlessProcessOpts struct {
	ProcessId uuid.UUID `validate:"required"`

	// TTL is how long the row stays live after this heartbeat. Expiry is computed in the
	// database from its own clock.
	TTL time.Duration `validate:"required,min=1ms"`

	UnitCount     int32 `validate:"min=0"`
	EndpointCount int32 `validate:"min=0"`

	Hostname *string
	Version  *string
}

type serverlessProcessRepository struct {
	*sharedRepository
}

func (r *serverlessProcessRepository) Upsert(ctx context.Context, opts UpsertServerlessProcessOpts) error {
	if err := r.v.Validate(opts); err != nil {
		return err
	}

	return r.queries.UpsertServerlessProcess(ctx, r.pool, sqlcv1.UpsertServerlessProcessParams{
		Processid:     opts.ProcessId,
		Ttl:           sqlchelpers.DurationToPgInterval(opts.TTL),
		Unitcount:     opts.UnitCount,
		Endpointcount: opts.EndpointCount,
		Hostname:      sqlchelpers.TextFromMaybeStr(opts.Hostname),
		Version:       sqlchelpers.TextFromMaybeStr(opts.Version),
	})
}

func (r *serverlessProcessRepository) ListLive(ctx context.Context) ([]*sqlcv1.V1ServerlessProcess, []*sqlcv1.V1ServerlessProcess, error) {
	rows, err := r.queries.ListServerlessProcesses(ctx, r.pool)

	if err != nil {
		return nil, nil, err
	}

	live := make([]*sqlcv1.V1ServerlessProcess, 0, len(rows))
	dead := make([]*sqlcv1.V1ServerlessProcess, 0)

	for _, row := range rows {
		process := row.V1ServerlessProcess

		if row.Expired {
			dead = append(dead, &process)
			continue
		}

		live = append(live, &process)
	}

	return live, dead, nil
}

func (r *serverlessProcessRepository) DeleteExpired(ctx context.Context, cutoff time.Time) (int64, int64, error) {
	row, err := r.queries.DeleteExpiredServerlessProcesses(ctx, r.pool, sqlchelpers.TimestamptzFromTime(cutoff))

	if err != nil {
		return 0, 0, err
	}

	return row.DeletedProcesses, row.ReleasedUnits, nil
}

func (r *serverlessProcessRepository) Delete(ctx context.Context, processId uuid.UUID) error {
	return r.queries.DeleteServerlessProcess(ctx, r.pool, processId)
}
