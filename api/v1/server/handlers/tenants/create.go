package tenants

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (t *TenantService) TenantCreate(ctx echo.Context, request gen.TenantCreateRequestObject) (gen.TenantCreateResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)

	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantCreate400JSONResponse(*apiErrors), nil
	}

	// determine if a tenant with the slug already exists
	existingTenant, err := t.config.Repository.Tenant().GetTenantBySlug(request.Body.Slug)

	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}

	if existingTenant != nil {
		// just return bad request
		return gen.TenantCreate400JSONResponse(
			apierrors.NewAPIErrors("Tenant with the slug already exists."),
		), nil
	}

	createOpts := &repository.CreateTenantOpts{
		Slug: request.Body.Slug,
		Name: request.Body.Name,
	}

	// write the user to the db
	tenant, err := t.config.Repository.Tenant().CreateTenant(createOpts)

	if err != nil {
		return nil, err
	}

	// add the user as an owner of the tenant
	_, err = t.config.Repository.Tenant().CreateTenantMember(tenant.ID, &repository.CreateTenantMemberOpts{
		UserId: user.ID,
		Role:   "OWNER",
	})

	if err != nil {
		return nil, err
	}

	return gen.TenantCreate200JSONResponse(
		*transformers.ToTenant(tenant),
	), nil
}
