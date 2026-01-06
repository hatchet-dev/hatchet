package task

import (
	"context"
)

func (tc *TasksControllerImpl) runAnalyze(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("analyze: running analyze on partitioned tables")

		err := tc.repov1.Tasks().AnalyzeTaskTables(ctx)

		if err != nil {
			tc.l.Error().Err(err).Msg("could not analyze task tables")
		}
	}
}
