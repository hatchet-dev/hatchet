package task

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (tc *TasksControllerImpl) runTaskTablePartition(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("partition: running task table partition")

		// list all tenants
		tenants, err := tc.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		for _, tenant := range tenants {
			if tenant.Name != "internal" {
				continue
			}

			err := tc.createTablePartition(ctx)

			if err != nil {
				tc.l.Error().Err(err).Msg("could not create table partition")
			}
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
