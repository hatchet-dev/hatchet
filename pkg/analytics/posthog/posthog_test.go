package posthog

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/analytics"
)

var testLogger = zerolog.Nop()

type countCall struct {
	Resource   analytics.Resource
	Action     analytics.Action
	TenantID   uuid.UUID
	TokenID    *uuid.UUID
	Count      int64
	Properties analytics.Properties
}

type countRecorder struct {
	mu    sync.Mutex
	calls []countCall
}

func (r *countRecorder) record(resource analytics.Resource, action analytics.Action, tenantID uuid.UUID, tokenID *uuid.UUID, count int64, properties analytics.Properties) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, countCall{
		Resource:   resource,
		Action:     action,
		TenantID:   tenantID,
		TokenID:    tokenID,
		Count:      count,
		Properties: properties,
	})
}

func (r *countRecorder) getCalls() []countCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]countCall, len(r.calls))
	copy(cp, r.calls)
	return cp
}

func TestCount_DefaultToOne(t *testing.T) {
	rec := &countRecorder{}
	agg := analytics.NewAggregator(&testLogger, true, 50, 0, rec.record)
	agg.Start()
	p := &PosthogAnalytics{
		aggregator: agg,
	}

	ctx := context.Background()
	p.Count(ctx, analytics.Event, analytics.Create)

	agg.Shutdown()

	calls := rec.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Count != 1 {
		t.Errorf("expected count 1, got %d", calls[0].Count)
	}
}

func TestCount_SinglePropertyFallbackToOne(t *testing.T) {
	rec := &countRecorder{}
	agg := analytics.NewAggregator(&testLogger, true, 50, 0, rec.record)
	agg.Start()
	p := &PosthogAnalytics{
		aggregator: agg,
	}

	ctx := context.Background()
	p.Count(ctx, analytics.Event, analytics.Create, analytics.Props(
		"has_priority", true,
	))

	agg.Shutdown()

	calls := rec.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Count != 1 {
		t.Errorf("expected count 1, got %d", calls[0].Count)
	}
}

func TestCount_ExplicitCountFromLastProperties(t *testing.T) {
	rec := &countRecorder{}
	agg := analytics.NewAggregator(&testLogger, true, 50, 0, rec.record)
	agg.Start()
	p := &PosthogAnalytics{
		aggregator: agg,
	}

	ctx := context.Background()
	p.Count(ctx, analytics.Event, analytics.Create,
		analytics.Props("has_priority", true),
		analytics.Props("count", int64(5)),
	)

	agg.Shutdown()

	calls := rec.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Count != 5 {
		t.Errorf("expected count 5, got %d", calls[0].Count)
	}
}

func TestCount_SumMultipleInt64Values(t *testing.T) {
	rec := &countRecorder{}
	agg := analytics.NewAggregator(&testLogger, true, 50, 0, rec.record)
	agg.Start()
	p := &PosthogAnalytics{
		aggregator: agg,
	}

	ctx := context.Background()
	p.Count(ctx, analytics.Event, analytics.Create,
		analytics.Props("has_priority", true),
		analytics.Props("count", int64(2), "other_count", int64(3)),
	)

	agg.Shutdown()

	calls := rec.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Count != 5 {
		t.Errorf("expected count 5 (sum of 2+3), got %d", calls[0].Count)
	}
}

func TestCount_IgnoresNonInt64Values(t *testing.T) {
	rec := &countRecorder{}
	agg := analytics.NewAggregator(&testLogger, true, 50, 0, rec.record)
	agg.Start()
	p := &PosthogAnalytics{
		aggregator: agg,
	}

	ctx := context.Background()
	p.Count(ctx, analytics.Event, analytics.Create,
		analytics.Props("has_priority", true),
		analytics.Props("count", int64(2), "name", "test", "active", true),
	)

	agg.Shutdown()

	calls := rec.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Count != 2 {
		t.Errorf("expected count 2, got %d", calls[0].Count)
	}
}

func TestCount_NilLastPropertiesFallbackToOne(t *testing.T) {
	rec := &countRecorder{}
	agg := analytics.NewAggregator(&testLogger, true, 50, 0, rec.record)
	agg.Start()
	p := &PosthogAnalytics{
		aggregator: agg,
	}

	ctx := context.Background()
	p.Count(ctx, analytics.Event, analytics.Create,
		analytics.Props("has_priority", true),
		nil,
	)

	agg.Shutdown()

	calls := rec.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Count != 1 {
		t.Errorf("expected count 1, got %d", calls[0].Count)
	}
}

func TestCount_FirstPropertyPassedToAggregator(t *testing.T) {
	rec := &countRecorder{}
	agg := analytics.NewAggregator(&testLogger, true, 50, 0, rec.record)
	agg.Start()
	p := &PosthogAnalytics{
		aggregator: agg,
	}

	ctx := context.Background()
	p.Count(ctx, analytics.Event, analytics.Create,
		analytics.Props("has_priority", true, "source", "api"),
		analytics.Props("count", int64(10)),
	)

	agg.Shutdown()

	calls := rec.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Properties["has_priority"] != true {
		t.Error("expected has_priority=true in properties")
	}
	if calls[0].Properties["source"] != "api" {
		t.Error("expected source=api in properties")
	}
}
