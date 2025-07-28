package task

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
)

func (tc *TasksControllerImpl) runTaskTablePartition(ctx context.Context) func() {
	return func() {
		ctx, span := telemetry.NewSpan(ctx, "TasksControllerImpl.runTaskTablePartition")
		defer span.End()

		tc.l.Debug().Msgf("partition: running task table partition")

		// get internal tenant
		tenant, err := tc.p.GetInternalTenantForController(ctx)

		if err != nil {
			err := fmt.Errorf("could not get internal tenant: %w", err)

			span.RecordError(err)
			tc.l.Error().Err(err).Msg("could not get internal tenant")

			return
		}

		if tenant == nil {
			return
		}

		err = tc.createTablePartition(ctx)

		if err != nil {
			err := fmt.Errorf("could not create table partition: %w", err)

			span.RecordError(err)
			tc.l.Error().Err(err).Msg("could not create table partition")
		}
	}
}

func (tc *TasksControllerImpl) createTablePartition(ctx context.Context) error {
	ctx, span := telemetry.NewSpan(ctx, "TasksControllerImpl.createTablePartition")
	defer span.End()

	qCtx, qCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer qCancel()

	err := tc.repov1.Tasks().UpdateTablePartitions(qCtx)

	if err != nil {
		err := fmt.Errorf("could not create table partition: %w", err)

		span.RecordError(err)
		return err
	}

	return nil
}
