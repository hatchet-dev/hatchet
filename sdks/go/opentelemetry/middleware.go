package opentelemetry

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck // SA1019: needed for cross-workflow trace propagation via WithSourceInfo
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// NewMiddleware creates a Hatchet middleware that wraps each step run execution
// with an OpenTelemetry span. It:
//   - Derives the engine's workflow_run span_id from the workflow run UUID so the
//     step_run span parents under the engine's workflow_run root span
//   - Creates a "hatchet.start_step_run" span with hatchet.* attributes
//   - Stores attributes in context so HatchetAttributeSpanProcessor can inject
//     them into all child spans
//
//nolint:staticcheck // SA1019: worker.MiddlewareFunc is deprecated but still used internally
func NewMiddleware(tracer trace.Tracer) worker.MiddlewareFunc {
	return func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {
		attrs := hatchetAttributes(ctx)

		parentCtx := ctx.GetContext()
		if meta := ctx.AdditionalMetadata(); meta != nil {
			if tp, ok := meta["traceparent"]; ok && tp != "" {
				if traceID, ok := parseTraceIDFromTraceparent(tp); ok {
					engineSpanID := deriveWorkflowRunSpanID(ctx.WorkflowRunId())
					if engineSpanID.IsValid() {
						sc := trace.NewSpanContext(trace.SpanContextConfig{
							TraceID:    traceID,
							SpanID:     engineSpanID,
							TraceFlags: trace.FlagsSampled,
							Remote:     true,
						})
						parentCtx = trace.ContextWithRemoteSpanContext(parentCtx, sc)
					}
				}
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

// parseTraceIDFromTraceparent extracts the trace_id from a W3C traceparent header.
// Format: "00-<32 hex trace_id>-<16 hex span_id>-<2 hex flags>"
func parseTraceIDFromTraceparent(tp string) (trace.TraceID, bool) {
	if len(tp) < 55 || tp[2] != '-' || tp[35] != '-' || tp[52] != '-' {
		return trace.TraceID{}, false
	}

	b, err := hex.DecodeString(tp[3:35])
	if err != nil || len(b) != 16 {
		return trace.TraceID{}, false
	}

	var traceID trace.TraceID
	copy(traceID[:], b)
	return traceID, true
}

// deriveWorkflowRunSpanID produces the same deterministic span_id that the
// engine uses for its workflow_run root span:
// SHA-256("hatchet-engine-wf-span:" + uuid_bytes)[:8].
func deriveWorkflowRunSpanID(workflowRunID string) trace.SpanID {
	id, err := uuid.Parse(workflowRunID)
	if err != nil {
		return trace.SpanID{}
	}

	h := sha256.New()
	h.Write([]byte("hatchet-engine-wf-span:"))
	h.Write(id[:])

	var spanID trace.SpanID
	copy(spanID[:], h.Sum(nil)[:8])
	return spanID
}
