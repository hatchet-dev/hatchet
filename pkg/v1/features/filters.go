// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package features

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// Deprecated: FiltersClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type FiltersClient interface {
	List(ctx context.Context, opts *rest.V1FilterListParams) (*rest.V1FilterList, error)

	Get(ctx context.Context, filterID string) (*rest.V1Filter, error)

	Create(ctx context.Context, opts rest.V1CreateFilterRequest) (*rest.V1Filter, error)

	Delete(ctx context.Context, filterID string) (*rest.V1Filter, error)

	Update(ctx context.Context, filterID string, opts rest.V1FilterUpdateJSONRequestBody) (*rest.V1Filter, error)
}

type filtersClientImpl struct {
	api      *rest.ClientWithResponses
	tenantID uuid.UUID
}

// Deprecated: NewFiltersClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func NewFiltersClient(
	api *rest.ClientWithResponses,
	tenantID *string,
) FiltersClient {
	return &filtersClientImpl{
		api:      api,
		tenantID: uuid.MustParse(*tenantID),
	}
}

// Deprecated: List is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *filtersClientImpl) List(ctx context.Context, opts *rest.V1FilterListParams) (*rest.V1FilterList, error) {
	resp, err := c.api.V1FilterListWithResponse(
		ctx,
		c.tenantID,
		opts,
	)

	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Deprecated: Get is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *filtersClientImpl) Get(ctx context.Context, filterID string) (*rest.V1Filter, error) {
	resp, err := c.api.V1FilterGetWithResponse(
		ctx,
		c.tenantID,
		uuid.MustParse(filterID),
	)

	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Deprecated: Create is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *filtersClientImpl) Create(ctx context.Context, opts rest.V1CreateFilterRequest) (*rest.V1Filter, error) {
	resp, err := c.api.V1FilterCreateWithResponse(
		ctx,
		c.tenantID,
		opts,
	)

	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Deprecated: Delete is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *filtersClientImpl) Delete(ctx context.Context, filterID string) (*rest.V1Filter, error) {
	resp, err := c.api.V1FilterDeleteWithResponse(
		ctx,
		c.tenantID,
		uuid.MustParse(filterID),
	)

	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Deprecated: Update is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *filtersClientImpl) Update(ctx context.Context, filterID string, opts rest.V1FilterUpdateJSONRequestBody) (*rest.V1Filter, error) {
	resp, err := c.api.V1FilterUpdateWithResponse(
		ctx,
		c.tenantID,
		uuid.MustParse(filterID),
		opts,
	)

	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
