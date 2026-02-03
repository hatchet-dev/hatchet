package olap

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (o *OLAPControllerImpl) runTenantProcessAlerts(ctx context.Context) func() {
	return func() {
		o.l.Debug().Msgf("partition: processing tenant alerts")

		// list all tenants
		tenants, err := o.p.ListTenantsForController(ctx, sqlcv1.TenantMajorEngineVersionV1)

		if err != nil {
			o.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		o.processTenantAlertOperations.SetTenants(tenants)

		for i := range tenants {
			tenantId := tenants[i].ID.String()

			o.processTenantAlertOperations.RunOrContinue(tenantId)
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

	if lastAlerted.IsZero() || lastAlerted.Before(time.Now().Add(-24*time.Hour)) {
		lastAlerted = time.Now().Add(-24 * time.Hour).UTC()
	}

	failedRuns, _, err := o.repo.OLAP().ListWorkflowRuns(ctx, uuid.MustParse(tenantId), v1.ListWorkflowRunOpts{
		Statuses: []sqlcv1.V1ReadableStatusOlap{
			sqlcv1.V1ReadableStatusOlapFAILED,
		},
		CreatedAfter:    lastAlerted,
		Limit:           1000,
		IncludePayloads: false,
	})

	if err != nil {
		return false, fmt.Errorf("could not list workflow runs: %w", err)
	}

	err = o.ta.SendWorkflowRunAlertV1(uuid.MustParse(tenantId), failedRuns)

	if err != nil {
		return false, fmt.Errorf("could not send alert: %w", err)
	}

	return false, nil
}
