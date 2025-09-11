package features

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// FiltersClient provides methods for interacting with filters
type FiltersClient struct {
	api      *rest.ClientWithResponses
	tenantID uuid.UUID
}

// NewFiltersClient creates a new FiltersClient
func NewFiltersClient(
	api *rest.ClientWithResponses,
	tenantID string,
) *FiltersClient {
	return &FiltersClient{
		api:      api,
		tenantID: uuid.MustParse(tenantID),
	}
}

// List lists filters for a given tenant.
func (c *FiltersClient) List(ctx context.Context, opts *rest.V1FilterListParams) (*rest.V1FilterList, error) {
	resp, err := c.api.V1FilterListWithResponse(
		ctx,
		c.tenantID,
		opts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list filters")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Get gets a filter by its ID.
func (c *FiltersClient) Get(ctx context.Context, filterID string) (*rest.V1Filter, error) {
	resp, err := c.api.V1FilterGetWithResponse(
		ctx,
		c.tenantID,
		uuid.MustParse(filterID),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get filter")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Create creates a new filter.
func (c *FiltersClient) Create(ctx context.Context, opts rest.V1CreateFilterRequest) (*rest.V1Filter, error) {
	resp, err := c.api.V1FilterCreateWithResponse(
		ctx,
		c.tenantID,
		opts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create filter")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Delete deletes a filter by its ID.
func (c *FiltersClient) Delete(ctx context.Context, filterID string) (*rest.V1Filter, error) {
	resp, err := c.api.V1FilterDeleteWithResponse(
		ctx,
		c.tenantID,
		uuid.MustParse(filterID),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to delete filter")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Update updates a filter by its ID.
func (c *FiltersClient) Update(ctx context.Context, filterID string, opts rest.V1FilterUpdateJSONRequestBody) (*rest.V1Filter, error) {
	resp, err := c.api.V1FilterUpdateWithResponse(
		ctx,
		c.tenantID,
		uuid.MustParse(filterID),
		opts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update filter")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
