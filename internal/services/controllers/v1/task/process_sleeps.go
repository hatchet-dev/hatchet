package task

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (tc *TasksControllerImpl) runTenantSleepEmitter(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("partition: running sleep emitter for tasks")

		// list all tenants
		tenants, err := tc.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		tc.emitSleepOperations.SetTenants(tenants)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			tc.emitSleepOperations.RunOrContinue(tenantId)
		}
	}
}

func (tc *TasksControllerImpl) processSleeps(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-sleep")
	defer span.End()

	matchResult, shouldContinue, err := tc.repov1.Tasks().ProcessDurableSleeps(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not list step runs to timeout for tenant %s: %w", tenantId, err)
	}

	if len(matchResult.CreatedTasks) > 0 {
		err = tc.signalTasksCreated(ctx, tenantId, matchResult.CreatedTasks)

		if err != nil {
			return false, fmt.Errorf("could not signal created tasks: %w", err)
		}
	}

	return shouldContinue, nil
}
