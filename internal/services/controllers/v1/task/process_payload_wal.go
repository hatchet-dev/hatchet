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
	return tc.repov1.Payloads().ProcessPayloadWAL(ctx, partitionNumber, tc.pubBuffer)
}

func (tc *TasksControllerImpl) runProcessPayloadExternalCutovers(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("processing payload external cutovers")

		partitions := []int64{0, 1, 2, 3}

		tc.processPayloadExternalCutoversOperations.SetPartitions(partitions)

		for _, partitionId := range partitions {
			tc.processPayloadExternalCutoversOperations.RunOrContinue(partitionId)
		}
	}
}

func (tc *TasksControllerImpl) processPayloadExternalCutovers(ctx context.Context, partitionNumber int64) (bool, error) {
	return tc.repov1.Payloads().CopyOffloadedPayloadsIntoTempTable(ctx)
}
