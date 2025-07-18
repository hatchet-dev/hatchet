package task

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
)

func (tc *TasksControllerImpl) runTaskTablePartition(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("partition: running task table partition")

		err := tc.createTablePartition(ctx)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not create table partition")
		}
	}
}

func (tc *TasksControllerImpl) createTablePartition(ctx context.Context) error {
	ctx, span := telemetry.NewSpan(ctx, "create-table-partition")
	defer span.End()

	qCtx, qCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer qCancel()

	err := tc.repov1.Tasks().UpdateTablePartitions(qCtx)

	if err != nil {
		return fmt.Errorf("could not create table partition: %w", err)
	}

	return nil
}
