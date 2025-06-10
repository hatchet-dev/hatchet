package task

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (tc *TasksControllerImpl) runTenantSleepEmitter(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("partition(%s): running sleep emitter for tasks", tc.p.GetControllerPartitionId())

		// list all tenants
		tenants, err := tc.p.V1ListTenantsForController(ctx, repository.TenantControllerFilter{
			WithFilter:        tc.opsTenantFilters,
			WithExpiredSleeps: true,
		})

		if err != nil {
			tc.l.Error().Err(err).Msg("could not list tenants with expired sleeps")
			return
		}

		tc.l.Debug().Msgf("partition(%s): tenants with expired sleeps: %d", tc.p.GetControllerPartitionId(), len(tenants))

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
		return false, fmt.Errorf("could not list process durable sleeps for tenant %s: %w", tenantId, err)
	}

	if len(matchResult.CreatedTasks) > 0 {
		err = tc.signalTasksCreated(ctx, tenantId, matchResult.CreatedTasks)

		if err != nil {
			return false, fmt.Errorf("could not signal created tasks: %w", err)
		}
	}

	return shouldContinue, nil
}
