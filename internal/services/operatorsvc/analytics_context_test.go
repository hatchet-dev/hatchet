package operatorsvc

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/analytics"
)

func TestContextWithAnalyticsTenantFillsMissingTenantID(t *testing.T) {
	tenantID := uuid.New()

	ctx := contextWithAnalyticsTenant(context.Background(), tenantID)

	got := analytics.TenantIDFromContext(ctx)
	if got == nil || *got != tenantID {
		t.Fatalf("tenant id = %v, want %s", got, tenantID)
	}
}

func TestContextWithAnalyticsTenantKeepsExistingTenantID(t *testing.T) {
	existing := uuid.New()
	ctx := context.WithValue(context.Background(), analytics.TenantIDKey, existing)

	ctx = contextWithAnalyticsTenant(ctx, uuid.New())

	got := analytics.TenantIDFromContext(ctx)
	if got == nil || *got != existing {
		t.Fatalf("tenant id = %v, want %s", got, existing)
	}
}

func TestContextWithAnalyticsTenantIgnoresNilTenantID(t *testing.T) {
	ctx := contextWithAnalyticsTenant(context.Background(), uuid.Nil)

	if got := analytics.TenantIDFromContext(ctx); got != nil {
		t.Fatalf("tenant id = %s, want none", *got)
	}
}
