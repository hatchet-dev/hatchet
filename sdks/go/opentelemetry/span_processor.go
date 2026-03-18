package hatchetotel

import (
	"context"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// HatchetAttributeSpanProcessor wraps an inner SpanProcessor and injects
// hatchet.* attributes into every span created within a step run context.
// This ensures child spans are queryable by the same attributes (e.g.
// hatchet.step_run_id) as the parent span.
type HatchetAttributeSpanProcessor struct {
	inner sdktrace.SpanProcessor
}

// NewHatchetAttributeSpanProcessor creates a new HatchetAttributeSpanProcessor
// that wraps the given inner processor.
func NewHatchetAttributeSpanProcessor(inner sdktrace.SpanProcessor) *HatchetAttributeSpanProcessor {
	return &HatchetAttributeSpanProcessor{inner: inner}
}

func (p *HatchetAttributeSpanProcessor) OnStart(ctx context.Context, span sdktrace.ReadWriteSpan) {
	attrs := getHatchetAttributes(ctx)
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	p.inner.OnStart(ctx, span)
}

func (p *HatchetAttributeSpanProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	p.inner.OnEnd(span)
}

func (p *HatchetAttributeSpanProcessor) Shutdown(ctx context.Context) error {
	return p.inner.Shutdown(ctx)
}

func (p *HatchetAttributeSpanProcessor) ForceFlush(ctx context.Context) error {
	return p.inner.ForceFlush(ctx)
}
