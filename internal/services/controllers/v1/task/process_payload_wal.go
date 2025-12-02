package task

import (
	"context"
)

func (tc *TasksControllerImpl) runProcessPayloadExternalCutovers(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("processing payload external cutovers")

		tc.processPayloadExternalCutoversOperations.RunOrContinue(0)
	}
}

func (tc *TasksControllerImpl) processPayloadExternalCutovers(ctx context.Context, partitionNumber int64) (bool, error) {
	return false, tc.repov1.Payloads().CopyOffloadedPayloadsIntoTempTable(ctx)
}
