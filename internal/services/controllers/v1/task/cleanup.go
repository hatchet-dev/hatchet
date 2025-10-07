package task

import (
	"context"
	"time"
)

func (tc *TasksControllerImpl) runCleanup(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("task controller: running cleanup")

		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		for ctx.Err() == nil {
			shouldContinue, err := tc.repov1.Tasks().Cleanup(ctx)

			if err != nil {
				tc.l.Error().Err(err).Msg("could not run cleanup")
				return
			}

			if !shouldContinue {
				tc.l.Debug().Msgf("cleanup completed")
				return
			}

			tc.l.Debug().Msgf("cleanup has more work, continuing...")
		}
	}
}
