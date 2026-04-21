package retention

import (
	"context"
	"time"
)

func (rc *RetentionControllerImpl) runCleanupOldWorkers(ctx context.Context) func() {
	return func() {
		rc.l.Debug().Ctx(ctx).Msg("retention controller: cleaning up old workers")

		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		shouldContinue := true
		var err error

		for shouldContinue {
			shouldContinue, err = rc.repo.Workers().CleanupOldWorkers(ctx)
			if err != nil {
				rc.l.Err(err).Ctx(ctx).Msg("could not cleanup old workers")
				return
			}
		}
	}
}
