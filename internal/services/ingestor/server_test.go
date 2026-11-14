package ingestor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// refusingRepo refuses every admission and records what it was asked for.
type refusingRepo struct {
	v1.Repository
	v1.TenantLimitRepository

	resource sqlcv1.LimitResource
	items    int
}

func (r *refusingRepo) TenantLimit() v1.TenantLimitRepository { return r }

func (r *refusingRepo) CanCreate(_ context.Context, resource sqlcv1.LimitResource, _ uuid.UUID, n int32) (bool, int, error) {
	r.resource, r.items = resource, int(n)
	return false, 0, &v1.TenantRateLimitedError{RetryAfter: time.Second}
}

func TestEventPushesAreRefusedByTheTenantRateLimit(t *testing.T) {
	ctx := context.WithValue(context.Background(), "tenant", &sqlcv1.Tenant{ID: uuid.New()}) // nolint: staticcheck

	repo := &refusingRepo{}
	i := &IngestorImpl{repov1: repo, v: validator.NewDefaultValidator(), analytics: analytics.NoOpAnalytics{}}

	var refusal *v1.TenantRateLimitedError

	_, err := i.Push(ctx, &contracts.PushEventRequest{Key: "k", Payload: "{}"})
	require.ErrorAs(t, err, &refusal)
	require.Equal(t, sqlcv1.LimitResourceEVENT, repo.resource)
	require.Equal(t, 1, repo.items)

	_, err = i.BulkPush(ctx, &contracts.BulkPushEventRequest{Events: []*contracts.PushEventRequest{
		{Key: "a", Payload: "{}"}, {Key: "b", Payload: "{}"}, {Key: "c", Payload: "{}"}, {Key: "d", Payload: "{}"},
	}})
	require.ErrorAs(t, err, &refusal)
	require.Equal(t, sqlcv1.LimitResourceEVENT, repo.resource)
	require.Equal(t, 4, repo.items)
}
