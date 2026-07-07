package retention

import (
	"context"
	"time"
)

func (rc *RetentionControllerImpl) runCleanupUserSessions(ctx context.Context) func() {
	return func() {
		rc.l.Debug().Ctx(ctx).Msg("retention controller: cleaning up user sessions")

		ctx, cancel := context.WithTimeout(ctx, time.Second*20) //nolint
		defer cancel()

		err := rc.repo.UserSession().CleanupUserSessions(ctx)
		if err != nil {
			rc.l.Err(err).Ctx(ctx).Msg("user sessions cleanup failed")
		}
	}
}
