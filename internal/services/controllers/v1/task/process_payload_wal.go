package task

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/telemetry"
	"go.opentelemetry.io/otel/codes"
)

func (tc *TasksControllerImpl) processPayloadExternalCutovers(ctx context.Context) func() {
	return func() {
		ctx, span := telemetry.NewSpan(ctx, "TasksControllerImpl.processPayloadExternalCutovers")
		defer span.End()

		tc.l.Debug().Msgf("payload external cutover: processing external cutover payloads")

		err := tc.repov1.Payloads().CopyOffloadedPayloadsIntoTempTable(ctx)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not process external cutover payloads")
			tc.l.Error().Err(err).Msg("could not process external cutover payloads")
		}
	}
}
