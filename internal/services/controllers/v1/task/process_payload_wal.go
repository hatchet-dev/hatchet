package task

import (
	"context"
)

func (tc *TasksControllerImpl) processPayloadExternalCutovers(ctx context.Context) (bool, error) {
	return false, tc.repov1.Payloads().CopyOffloadedPayloadsIntoTempTable(ctx)
}
