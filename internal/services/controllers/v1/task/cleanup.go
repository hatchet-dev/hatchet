package task

import (
	"context"
)

func (tc *TasksControllerImpl) runCleanup(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("task controller: running cleanup")

		err := tc.repov1.Tasks().Cleanup(ctx)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not run cleanup")
		}
	}
}
