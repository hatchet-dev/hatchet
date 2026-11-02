package olap

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (o *OLAPControllerImpl) runTenantProcessAlerts(ctx context.Context) func() {
	return func() {
		o.l.Debug().Ctx(ctx).Msgf("partition: processing tenant alerts")

		// list all tenants
		tenants, err := o.p.ListTenantsForController(ctx)

		if err != nil {
			o.l.Error().Ctx(ctx).Err(err).Msg("could not list tenants")
			return
		}

		o.processTenantAlertOperations.SetTenants(tenants)

		for _, tenantId := range tenants {
			o.processTenantAlertOperations.RunOrContinue(tenantId.String())
		}
	}
}

func (o *OLAPControllerImpl) processTenantAlerts(ctx context.Context, tenantId string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	ctx, span := telemetry.NewSpan(ctx, "process-tenant-alerts")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

	isActive, _, err := o.repo.Ticker().IsTenantAlertActive(ctx, uuid.MustParse(tenantId))

	if err != nil {
		return false, fmt.Errorf("could not check if tenant is active: %w", err)
	}

	if !isActive {
		return false, nil
	}

	failedRuns, failedThrough, err := o.repo.OLAP().ListPendingFailedRunAlerts(ctx, uuid.MustParse(tenantId))

	if err != nil {
		return false, fmt.Errorf("could not list failed workflow runs: %w", err)
	}

	err = o.ta.SendWorkflowRunAlertV1(uuid.MustParse(tenantId), failedRuns)

	if err != nil {
		return false, fmt.Errorf("could not send alert: %w", err)
	}

	if failedThrough.Valid {
		if err = o.repo.OLAP().DeleteSentFailedRunAlerts(ctx, uuid.MustParse(tenantId), failedThrough); err != nil {
			return false, fmt.Errorf("could not delete sent failed run alerts: %w", err)
		}
	}

	return false, nil
}

func (o *OLAPControllerImpl) recordFailedRunAlerts(ctx context.Context, tenantIdToFailedRuns map[uuid.UUID][]v1.FailedRunAlert) {
	for tenantId, runs := range tenantIdToFailedRuns {
		enabled, err := o.repo.TenantAlertingSettings().IsWorkflowRunFailureAlertEnabled(ctx, tenantId)

		if err != nil {
			o.l.Error().Ctx(ctx).Err(err).Msgf("could not check workflow run failure alerts for tenant %s", tenantId.String())
			continue
		}

		if !enabled {
			continue
		}

		if err := o.repo.OLAP().CreateFailedRunAlerts(ctx, tenantId, runs); err != nil {
			o.l.Error().Ctx(ctx).Err(err).Msgf("could not record failed runs for alerting for tenant %s", tenantId.String())
			continue
		}

		o.processTenantAlertOperations.RunOrContinue(tenantId.String())
	}
}
