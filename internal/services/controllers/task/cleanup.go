package task

import (
	"context"
	"time"
)

func (tc *TasksControllerImpl) runCleanup(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Ctx(ctx).Msgf("task controller: running cleanup")

		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()

		shouldContinue := true
		var err error

		for shouldContinue {
			shouldContinue, err = tc.repov1.Tasks().Cleanup(ctx)

			if err != nil {
				tc.l.Error().Ctx(ctx).Err(err).Msg("could not run cleanup")
				break
			}
		}
	}
}
