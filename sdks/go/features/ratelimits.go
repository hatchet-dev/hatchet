// package features provides functionality for interacting with hatchet features.
package features

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

// CreateRatelimitOpts contains options for creating or updating a rate limit.
type CreateRatelimitOpts struct {
	// key is the unique identifier for the rate limit
	Key string
	// limit is the maximum number of requests allowed within the duration
	Limit int
	// duration specifies the time period for the rate limit
	Duration types.RateLimitDuration
}

// RateLimitsClient provides methods for interacting with rate limits
type RateLimitsClient struct {
	api      *rest.ClientWithResponses
	admin    v0Client.AdminClient
	tenantId uuid.UUID
}

// NewRateLimitsClient creates a new RateLimitsClient with the provided api client, tenant id, and admin client.
func NewRateLimitsClient(
	api *rest.ClientWithResponses,
	tenantId uuid.UUID,
	admin v0Client.AdminClient,
) *RateLimitsClient {
	tenantIdUUid := tenantId

	return &RateLimitsClient{
		api:      api,
		tenantId: tenantIdUUid,
		admin:    admin,
	}
}

// Upsert creates or updates a rate limit with the provided options.
func (c *RateLimitsClient) Upsert(opts CreateRatelimitOpts) error {
	if err := c.admin.PutRateLimit(opts.Key, &types.RateLimitOpts{
		Max:      opts.Limit,
		Duration: opts.Duration,
	}); err != nil {
		return errors.Wrap(err, "failed to upsert rate limit")
	}

	return nil
}

// List retrieves rate limits based on the provided parameters (optional).
func (c *RateLimitsClient) List(ctx context.Context, opts *rest.RateLimitListParams) (*rest.RateLimitList, error) {
	resp, err := c.api.RateLimitListWithResponse(
		ctx,
		c.tenantId,
		opts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list rate limits")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
