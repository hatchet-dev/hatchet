package ticker

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
)

func (t *TickerImpl) runPollTenantAlerts(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
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

			innerErr := t.ta.SendAlert(tenantId, alert.PrevLastAlertedAt.Time)

			if innerErr != nil {
				err = multierror.Append(err, innerErr)
			}
		}

		if err != nil {
			t.l.Err(err).Msg("could not handle tenant alerts")
		}
	}
}
