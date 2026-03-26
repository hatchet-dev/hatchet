package opentelemetry

import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck // SA1019: needed for cross-workflow trace propagation via WithSourceInfo
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// NewMiddleware creates a Hatchet middleware that wraps each step run execution
// with an OpenTelemetry span. It:
//   - Extracts W3C traceparent from AdditionalMetadata for distributed trace propagation
//   - Creates a "hatchet.start_step_run" span with hatchet.* attributes
//   - Stores attributes in context so HatchetAttributeSpanProcessor can inject
//     them into all child spans
//
//nolint:staticcheck // SA1019: worker.MiddlewareFunc is deprecated but still used internally
func NewMiddleware(tracer trace.Tracer) worker.MiddlewareFunc {
	propagator := propagation.TraceContext{}

	return func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {
		// Build hatchet attributes from context
		attrs := hatchetAttributes(ctx)

		// Extract traceparent from additional metadata if present
		parentCtx := ctx.GetContext()
		if meta := ctx.AdditionalMetadata(); meta != nil {
			if tp, ok := meta["traceparent"]; ok && tp != "" {
				carrier := propagation.MapCarrier(map[string]string{
					"traceparent": tp,
				})
				parentCtx = propagator.Extract(parentCtx, carrier)
			}
		}

		// Store hatchet attributes in context for the SpanProcessor
		parentCtx = withHatchetAttributes(parentCtx, attrs)

		// Store source info so event Push/BulkPush can inject it into metadata
		parentCtx = client.WithSourceInfo(parentCtx, client.SourceInfo{
			WorkflowRunID: ctx.WorkflowRunId(),
			StepRunID:     ctx.StepRunId(),
		})

		// Start span
		spanCtx, span := tracer.Start(parentCtx, SpanStartStepRun,
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(attrs...),
		)
		defer span.End()

		// Update the HatchetContext with the OTel-enriched context
		ctx.SetContext(spanCtx)

		// Execute the next middleware/action
		err := next(ctx)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}
