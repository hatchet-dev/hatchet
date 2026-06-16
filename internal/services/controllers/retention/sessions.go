package retention

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (rc *RetentionControllerImpl) runCleanupExpiredUserSessions(ctx context.Context) func() {
	return func() {
		rc.l.Debug().Ctx(ctx).Msg("retention controller: cleaning up expired user sessions")

		ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()

		if err := rc.runCleanupExpiredUserSessionsQueries(ctx); err != nil {
			rc.l.Err(err).Ctx(ctx).Msg("could not cleanup expired user sessions")
		}
	}
}

func (rc *RetentionControllerImpl) runCleanupExpiredUserSessionsQueries(ctx context.Context) error {
	ctx, span := telemetry.NewSpan(ctx, "cleanup-expired-user-sessions")
	defer span.End()

	// batchSize bounds each delete so a large backlog of expired sessions drains
	// across iterations instead of one long-running transaction.
	const batchSize int32 = 10000

	shouldContinue := true

	for shouldContinue {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		var err error
		shouldContinue, err = rc.repo.UserSession().DeleteExpired(ctx, batchSize)
		if err != nil {
			return err
		}
	}

	return nil
}
