package olap

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

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

	isActive, lastAlerted, err := o.repo.Ticker().IsTenantAlertActive(ctx, uuid.MustParse(tenantId))

	if err != nil {
		return false, fmt.Errorf("could not check if tenant is active: %w", err)
	}

	if !isActive {
		return false, nil
	}

	now := time.Now().UTC()
	// finished_at is stamped before its transaction commits; querying behind now avoids skipping late commits.
	lag := time.Minute

	lastAlerted = lastAlerted.UTC()

	if lastAlerted.Before(now.Add(-24 * time.Hour)) {
		lastAlerted = now.Add(-24 * time.Hour)
	}

	failedRuns, err := o.repo.OLAP().ListFailedWorkflowRunsByFinishedAt(ctx, uuid.MustParse(tenantId), lastAlerted.Add(-lag), now.Add(-lag), 1000)

	if err != nil {
		return false, fmt.Errorf("could not list workflow runs: %w", err)
	}

	err = o.ta.SendWorkflowRunAlertV1(uuid.MustParse(tenantId), failedRuns, now)

	if err != nil {
		return false, fmt.Errorf("could not send alert: %w", err)
	}

	return false, nil
}
