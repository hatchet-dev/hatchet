package tenants

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type enqueueRecorder struct {
	analytics.NoOpAnalytics
	ctx context.Context
}

func (r *enqueueRecorder) Enqueue(ctx context.Context, _ analytics.Resource, _ analytics.Action, _ string, _ analytics.Properties) {
	r.ctx = ctx
}

func TestTenantCreateAnalyticsCarriesTenantID(t *testing.T) {
	t.Parallel()

	rec := &enqueueRecorder{}
	svc := &TenantService{config: &server.ServerConfig{Analytics: rec}}

	tenant := &sqlcv1.Tenant{ID: uuid.New(), Name: "acme", Slug: "acme"}

	svc.dispatchTenantCreateAnalyticsEvent(context.Background(), tenant)

	if rec.ctx == nil {
		t.Fatal("expected an Enqueue call")
	}

	got := analytics.TenantIDFromContext(rec.ctx)
	if got == nil || *got != tenant.ID {
		t.Fatalf("expected tenant id %s on the analytics context, got %v", tenant.ID, got)
	}
}
