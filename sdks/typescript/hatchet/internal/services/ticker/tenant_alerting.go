package ticker

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *TickerImpl) runPollTenantAlerts(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()

		t.l.Debug().Msgf("ticker: polling tenant alerts")

		alerts, err := t.repo.Ticker().PollTenantAlerts(ctx, t.tickerId)

		if err != nil {
			t.l.Err(err).Msg("could not poll tenant alerts")
			return
		}

		for _, alert := range alerts {
			tenantId := sqlchelpers.UUIDToStr(alert.TenantId)

			t.l.Debug().Msgf("ticker: handling alert for tenant %s", tenantId)

			innerErr := t.ta.SendWorkflowRunAlert(tenantId, alert.PrevLastAlertedAt.Time)

			if innerErr != nil {
				err = multierror.Append(err, innerErr)
			}
		}

		if err != nil {
			t.l.Err(err).Msg("could not handle tenant alerts")
		}
	}
}

func (t *TickerImpl) runExpiringTokenAlerts(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 300*time.Second) // only runs once per day, so long context timeout
		defer cancel()

		t.l.Debug().Msg("ticker: polling expiring tokens")

		expiring_tokens, err := t.repo.Ticker().PollExpiringTokens(ctx)

		if err != nil {
			t.l.Err(err).Msg("could not poll expiring tokens")
			return
		}

		t.l.Debug().Msgf("ticker: alerting %d expiring tokens", len(expiring_tokens))

		for _, expiring_token := range expiring_tokens {
			tenantId := sqlchelpers.UUIDToStr(expiring_token.TenantId)

			t.l.Debug().Msgf("ticker: handling expiring token for tenant %s", tenantId)

			innerErr := t.ta.SendExpiringTokenAlert(tenantId, expiring_token)

			if innerErr != nil {
				err = multierror.Append(err, innerErr)
			}
		}

		if err != nil {
			t.l.Err(err).Msg("could not handle expiring tokens")
		}
	}
}

func (t *TickerImpl) runTenantResourceLimitAlerts(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		t.l.Debug().Msg("ticker: resolving tenant resource limits")

		err := t.entitlements.TenantLimit().ResolveAllTenantResourceLimits(ctx)

		if err != nil {
			t.l.Err(err).Msg("could not resolve tenant resource limits")
			return
		}

		t.l.Debug().Msg("ticker: polling tenant resource limit alerts")

		alerts, err := t.repo.Ticker().PollTenantResourceLimitAlerts(ctx)

		if err != nil {
			t.l.Err(err).Msg("could not poll tenant resource limit alerts")
			return
		}

		t.l.Debug().Msgf("ticker: alerting %d tenant resource limit alerts", len(alerts))

		for _, alert := range alerts {
			tenantId := sqlchelpers.UUIDToStr(alert.TenantId)

			t.l.Debug().Msgf("ticker: handling tenant resource limit alert for tenant %s", tenantId)

			innerErr := t.ta.SendTenantResourceLimitAlert(tenantId, alert)

			if innerErr != nil {
				err = multierror.Append(err, innerErr)
			}
		}

		if err != nil {
			t.l.Err(err).Msg("could not handle tenant resource limit alerts")
		}
	}
}
