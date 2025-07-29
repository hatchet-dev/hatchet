package task

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (tc *TasksControllerImpl) runProcessPayloadWAL(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("processing payload WAL")

		tenants, err := tc.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		tc.processPayloadWALOperations.SetTenants(tenants)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			tc.processPayloadWALOperations.RunOrContinue(tenantId)
		}
	}
}

func (tc *TasksControllerImpl) processPayloadWAL(ctx context.Context, tenantId string) (bool, error) {
	return tc.repov1.Payloads().ProcessPayloadWAL(ctx, tenantId)
}
