package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/hatchet-dev/hatchet/pkg/limitalerts"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TickerImpl) runExpiringTokenAlerts(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 300*time.Second) // only runs once per day, so long context timeout
		defer cancel()

		t.l.Debug().Ctx(ctx).Msg("ticker: polling expiring tokens")

		expiring_tokens, err := t.repov1.Ticker().PollExpiringTokens(ctx)

		if err != nil {
			t.l.Err(err).Ctx(ctx).Msg("could not poll expiring tokens")
			return
		}

		t.l.Debug().Ctx(ctx).Msgf("ticker: alerting %d expiring tokens", len(expiring_tokens))

		for _, expiring_token := range expiring_tokens {
			tenantId := expiring_token.TenantId

			if tenantId == nil {
				continue
			}

			t.l.Debug().Ctx(ctx).Msgf("ticker: handling expiring token for tenant %s", tenantId)

			innerErr := t.ta.SendExpiringTokenAlert(*tenantId, expiring_token)

			if innerErr != nil {
				err = multierror.Append(err, innerErr)
			}
		}

		if err != nil {
			t.l.Err(err).Ctx(ctx).Msg("could not handle expiring tokens")
		}
	}
}

func (t *TickerImpl) runTenantResourceLimitAlerts(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		t.l.Debug().Ctx(ctx).Msg("ticker: resolving tenant resource limits")

		err := t.repov1.TenantLimit().ResolveAllTenantResourceLimits(ctx)

		if err != nil {
			t.l.Err(err).Ctx(ctx).Msg("could not resolve tenant resource limits")
			return
		}

		t.l.Debug().Ctx(ctx).Msg("ticker: polling tenant resource limit alerts")

		alerts, err := t.repov1.Ticker().PollTenantResourceLimitAlerts(ctx)

		if err != nil {
			t.l.Err(err).Ctx(ctx).Msg("could not poll tenant resource limit alerts")
			return
		}

		t.l.Debug().Ctx(ctx).Msgf("ticker: alerting %d tenant resource limit alerts", len(alerts))

		for _, alert := range alerts {
			t.l.Debug().Ctx(ctx).Msgf("ticker: handling tenant resource limit alert for tenant %s", alert.TenantId)

			if innerErr := t.dispatchTenantResourceLimitAlert(ctx, alert); innerErr != nil {
				err = multierror.Append(err, innerErr)
			}
		}

		if err != nil {
			t.l.Err(err).Ctx(ctx).Msg("could not handle tenant resource limit alerts")
		}
	}
}

// dispatchTenantResourceLimitAlert delivers one new alert to the tenant-facing
// alerter and then to the optional hook. Both run even if the other fails; the
// errors are combined so the caller can log them.
func (t *TickerImpl) dispatchTenantResourceLimitAlert(ctx context.Context, alert *sqlcv1.TenantResourceLimitAlert) error {
	var err error

	if innerErr := t.ta.SendTenantResourceLimitAlert(alert.TenantId, alert); innerErr != nil {
		err = multierror.Append(err, innerErr)
	}

	if t.limitAlertHook != nil {
		if innerErr := t.limitAlertHook(ctx, limitAlertFromRow(alert)); innerErr != nil {
			err = multierror.Append(err, fmt.Errorf("tenant resource limit alert hook: %w", innerErr))
		}
	}

	return err
}

func limitAlertFromRow(row *sqlcv1.TenantResourceLimitAlert) limitalerts.Alert {
	return limitalerts.Alert{
		ID:        row.ID,
		TenantID:  row.TenantId,
		Resource:  string(row.Resource),
		AlertType: limitalerts.AlertType(row.AlertType),
		Value:     row.Value,
		Limit:     row.Limit,
		CreatedAt: row.CreatedAt.Time,
	}
}
