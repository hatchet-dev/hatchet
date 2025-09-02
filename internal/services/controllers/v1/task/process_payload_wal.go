package task

import (
	"context"
)

func (tc *TasksControllerImpl) runProcessPayloadWAL(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("processing payload WAL")

		partitions := []int64{0, 1, 2, 3}

		tc.processPayloadWALOperations.SetPartitions(partitions)

		for _, partitionId := range partitions {
			tc.processPayloadWALOperations.RunOrContinue(partitionId)
		}
	}
}

func (tc *TasksControllerImpl) processPayloadWAL(ctx context.Context, partitionNumber int64) (bool, error) {
	return tc.repov1.Payloads().ProcessPayloadWAL(ctx, partitionNumber)
}
