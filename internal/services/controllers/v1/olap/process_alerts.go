package olap

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (o *OLAPControllerImpl) runTenantProcessAlerts(ctx context.Context) func() {
	return func() {
		o.l.Debug().Msgf("partition: processing tenant alerts")

		// list all tenants
		tenants, err := o.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV1)

		if err != nil {
			o.l.Error().Err(err).Msg("could not list tenants")
			return
		}

		o.processTenantAlertOperations.SetTenants(tenants)

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			o.processTenantAlertOperations.RunOrContinue(tenantId)
		}
	}
}

func (o *OLAPControllerImpl) processTenantAlerts(ctx context.Context, tenantId string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	ctx, span := telemetry.NewSpan(ctx, "process-tenant-alerts")
	defer span.End()

	isActive, lastAlerted, err := o.repo.Ticker().IsTenantAlertActive(ctx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not check if tenant is active: %w", err)
	}

	if !isActive {
		return false, nil
	}

	if lastAlerted.IsZero() || lastAlerted.Before(time.Now().Add(-24*time.Hour)) {
		lastAlerted = time.Now().Add(-24 * time.Hour).UTC()
	}

	failedRuns, _, err := o.repo.OLAP().ListWorkflowRuns(ctx, tenantId, v1.ListWorkflowRunOpts{
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

	err = o.ta.SendWorkflowRunAlertV1(tenantId, failedRuns)

	if err != nil {
		return false, fmt.Errorf("could not send alert: %w", err)
	}

	return false, nil
}
