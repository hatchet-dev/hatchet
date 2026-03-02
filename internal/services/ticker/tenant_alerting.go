package ticker

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"
)

func (t *TickerImpl) runExpiringTokenAlerts(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 300*time.Second) // only runs once per day, so long context timeout
		defer cancel()

		t.l.Debug().Msg("ticker: polling expiring tokens")

		expiring_tokens, err := t.repov1.Ticker().PollExpiringTokens(ctx)

		if err != nil {
			t.l.Err(err).Msg("could not poll expiring tokens")
			return
		}

		t.l.Debug().Msgf("ticker: alerting %d expiring tokens", len(expiring_tokens))

		for _, expiring_token := range expiring_tokens {
			tenantId := expiring_token.TenantId

			if tenantId == nil {
				continue
			}

			t.l.Debug().Msgf("ticker: handling expiring token for tenant %s", tenantId)

			innerErr := t.ta.SendExpiringTokenAlert(*tenantId, expiring_token)

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

		err := t.repov1.TenantLimit().ResolveAllTenantResourceLimits(ctx)

		if err != nil {
			t.l.Err(err).Msg("could not resolve tenant resource limits")
			return
		}

		t.l.Debug().Msg("ticker: polling tenant resource limit alerts")

		alerts, err := t.repov1.Ticker().PollTenantResourceLimitAlerts(ctx)

		if err != nil {
			t.l.Err(err).Msg("could not poll tenant resource limit alerts")
			return
		}

		t.l.Debug().Msgf("ticker: alerting %d tenant resource limit alerts", len(alerts))

		for _, alert := range alerts {
			tenantId := alert.TenantId

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
