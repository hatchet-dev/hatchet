package task

import (
	"context"

	"go.opentelemetry.io/otel/codes"

	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (tc *TasksControllerImpl) processPayloadExternalCutovers(ctx context.Context) func() {
	return func() {
		ctx, span := telemetry.NewSpan(ctx, "TasksControllerImpl.processPayloadExternalCutovers")
		defer span.End()

		tc.l.Debug().Msgf("payload external cutover: processing external cutover payloads")

		err := tc.repov1.Payloads().ProcessPayloadCutovers(ctx)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not process external cutover payloads")
			tc.l.Error().Err(err).Msg("could not process external cutover payloads")
		}
	}
}
