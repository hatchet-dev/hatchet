// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package features

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// Deprecated: TenantClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// TenantClient provides methods for interacting with your Tenant
type TenantClient interface {
	// Get the details of the current tenant
	Get(ctx context.Context) (*rest.Tenant, error)
}

type tenantClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// Deprecated: NewTenantCliet is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
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

// Deprecated: Get is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (t *tenantClientImpl) Get(ctx context.Context) (*rest.Tenant, error) {
	resp, err := t.api.TenantGetWithResponse(ctx, t.tenantId)

	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
