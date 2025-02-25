package task

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
)

func (tc *TasksControllerImpl) runTaskTablePartition(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("partition: running task table partition")

		// get internal tenant
		tenant, err := tc.p.GetInternalTenantForController(ctx)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not get internal tenant")
			return
		}

		if tenant == nil {
			return
		}

		err = tc.createTablePartition(ctx)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not create table partition")
		}
	}
}

func (tc *TasksControllerImpl) createTablePartition(ctx context.Context) error {
	ctx, span := telemetry.NewSpan(ctx, "create-table-partition")
	defer span.End()

	err := tc.repov1.Tasks().UpdateTablePartitions(ctx)

	if err != nil {
		return fmt.Errorf("could not create table partition: %w", err)
	}

	return nil
}
