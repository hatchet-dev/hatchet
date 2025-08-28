package task

import (
	"context"
	"strconv"
)

func (tc *TasksControllerImpl) runProcessPayloadWAL(ctx context.Context) func() {
	return func() {
		tc.l.Debug().Msgf("processing payload WAL")

		partitions := []int32{0, 1, 2, 3}

		tc.processPayloadWALOperations.SetPartitions(partitions)

		for _, partitionId := range partitions {
			partitionIdString := strconv.Itoa(int(partitionId))
			tc.processPayloadWALOperations.RunOrContinue(partitionIdString)
		}
	}
}

func (tc *TasksControllerImpl) processPayloadWAL(ctx context.Context, partitionNumberString string) (bool, error) {
	parsedPartitionNumber, err := strconv.Atoi(partitionNumberString)

	if err != nil {
		return false, err
	}

	return tc.repov1.Payloads().ProcessPayloadWAL(ctx, int32(parsedPartitionNumber))
}
