package olap

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (o *OLAPControllerImpl) runTenantDAGStatusUpdates(ctx context.Context) func() {
	return func() {
		o.l.Debug().Msgf("partition: running status updates for dags")

		// list all tenants
		tenants, err := o.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			o.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		o.updateDAGStatusOperations.SetTenants(tenants)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			o.updateDAGStatusOperations.RunOrContinue(tenantId)
		}
	}
}

func (o *OLAPControllerImpl) updateDAGStatuses(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-dag-statuses")
	defer span.End()

	return o.repo.OLAP().UpdateDAGStatuses(ctx, tenantId)
}
