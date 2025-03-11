package olap

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
)

func (oc *OLAPControllerImpl) runOLAPTablePartition(ctx context.Context) func() {
	return func() {
		oc.l.Debug().Msgf("partition: running task table partition")

		// list all tenants
		tenant, err := oc.p.GetInternalTenantForController(ctx)

		if err != nil {
			oc.l.Error().Err(err).Msg("could not get internal tenant")
			return
		}

		if tenant == nil {
			return
		}

		err = oc.createTablePartition(ctx)

		if err != nil {
			oc.l.Error().Err(err).Msg("could not create table partition")
		}
	}
}

func (oc *OLAPControllerImpl) createTablePartition(ctx context.Context) error {
	ctx, span := telemetry.NewSpan(ctx, "create-table-partition")
	defer span.End()

	err := oc.repo.OLAP().UpdateTablePartitions(ctx)

	if err != nil {
		return fmt.Errorf("could not create table partition: %w", err)
	}

	return nil
}
