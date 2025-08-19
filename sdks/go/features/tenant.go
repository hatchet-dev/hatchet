package features

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// TenantClient provides methods for interacting with your Tenant
type TenantClient struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// NewTenantCliet creates a new TenantClient
func NewTenantCliet(
	api *rest.ClientWithResponses,
	tenantId string,
) *TenantClient {
	tenantIdUUID := uuid.MustParse(tenantId)

	return &TenantClient{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

// Get retrieves the details of the current tenant
func (t *TenantClient) Get(ctx context.Context) (*rest.Tenant, error) {
	resp, err := t.api.TenantGetWithResponse(ctx, t.tenantId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tenant")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
