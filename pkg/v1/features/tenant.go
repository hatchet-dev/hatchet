package features

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// TenantClient provides methods for interacting with your Tenant
type TenantClient interface {
	// Get the details of the current tenant
	Get(ctx context.Context) (*rest.Tenant, error)
}

type tenantClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

func NewTenantCliet(
	api *rest.ClientWithResponses,
	tenantId *string,
) TenantClient {
	tenantIdUUID := uuid.MustParse(*tenantId)

	return &tenantClientImpl{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

func (t *tenantClientImpl) Get(ctx context.Context) (*rest.Tenant, error) {
	resp, err := t.api.TenantGetWithResponse(ctx, t.tenantId)

	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
