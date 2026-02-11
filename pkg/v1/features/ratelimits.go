// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package features

import (
	"context"

	"github.com/google/uuid"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

// Deprecated: CreateRatelimitOpts is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type CreateRatelimitOpts struct {
	Key      string
	Limit    int
	Duration types.RateLimitDuration
}

// Deprecated: RateLimitsClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type RateLimitsClient interface {
	Upsert(opts CreateRatelimitOpts) error

	List(ctx context.Context, opts *rest.RateLimitListParams) (*rest.RateLimitListResponse, error)
}

// rlClientImpl implements the rateLimitsClient interface.
type rlClientImpl struct {
	api      *rest.ClientWithResponses
	admin    *v0Client.AdminClient
	tenantId uuid.UUID
}

// Deprecated: NewRateLimitsClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// newRateLimitsClient creates a new rateLimitsClient with the provided api client, tenant id, and admin client.
func NewRateLimitsClient(
	api *rest.ClientWithResponses,
	tenantId *string,
	admin *v0Client.AdminClient,
) RateLimitsClient {
	tenantIdUUid := uuid.MustParse(*tenantId)

	return &rlClientImpl{
		api:      api,
		tenantId: tenantIdUUid,
		admin:    admin,
	}
}

// upsert creates or updates a rate limit with the provided options.
func (c *rlClientImpl) Upsert(opts CreateRatelimitOpts) error {
	return (*c.admin).PutRateLimit(opts.Key, &types.RateLimitOpts{
		Max:      opts.Limit,
		Duration: opts.Duration,
	})
}

// list retrieves rate limits based on the provided parameters (optional).
func (c *rlClientImpl) List(ctx context.Context, opts *rest.RateLimitListParams) (*rest.RateLimitListResponse, error) {
	return c.api.RateLimitListWithResponse(
		ctx,
		c.tenantId,
		opts,
	)
}
