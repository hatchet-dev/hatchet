package task

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/codes"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) runTaskTablePartition(ctx context.Context) func() {
	return func() {
		ctx, span := telemetry.NewSpan(ctx, "TasksControllerImpl.runTaskTablePartition")
		defer span.End()

		tc.l.Debug().Ctx(ctx).Msgf("partition: running task table partition")

		err := tc.createTablePartition(ctx)

		if err != nil {
			if errors.Is(err, v1.ErrPartitionLockConflict) {
				tc.l.Warn().Ctx(ctx).Msg("partition: lock conflict with concurrent table operation (e.g. ANALYZE), will retry at next interval")
				return
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not create table partition")
			tc.l.Error().Ctx(ctx).Err(err).Msg("could not create table partition")
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
		if errors.Is(err, v1.ErrPartitionLockConflict) {
			return err
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not create table partition")
		return err
	}

	return nil
}
