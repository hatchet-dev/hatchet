package tenants

import (
	"context"
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *TenantService) TenantCreate(ctx echo.Context, request gen.TenantCreateRequestObject) (gen.TenantCreateResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)

	if !t.config.Runtime.AllowCreateTenant {
		return gen.TenantCreate400JSONResponse(
			apierrors.NewAPIErrors("tenant signups are disabled"),
		), nil
	}

	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantCreate400JSONResponse(*apiErrors), nil
	}

	// determine if a tenant with the slug already exists
	existingTenant, err := t.config.APIRepository.Tenant().GetTenantBySlug(request.Body.Slug)

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

	if t.config.Runtime.Limits.DefaultTenantRetentionPeriod != "" {
		createOpts.DataRetentionPeriod = &t.config.Runtime.Limits.DefaultTenantRetentionPeriod
	}

	// write the user to the db
	tenant, err := t.config.APIRepository.Tenant().CreateTenant(createOpts)

	if err != nil {
		return nil, err
	}

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	err = t.config.EntitlementRepository.TenantLimit().SelectOrInsertTenantLimits(context.Background(), tenantId, nil)

	if err != nil {
		return nil, err
	}

	// add the user as an owner of the tenant
	_, err = t.config.APIRepository.Tenant().CreateTenantMember(tenantId, &repository.CreateTenantMemberOpts{
		UserId: user.ID,
		Role:   "OWNER",
	})

	if err != nil {
		return nil, err
	}

	t.config.Analytics.Tenant(tenantId, map[string]interface{}{
		"name": tenant.Name,
		"slug": tenant.Slug,
	})

	t.config.Analytics.Enqueue(
		"tenant:create",
		user.ID,
		&tenantId,
		nil,
	)

	return gen.TenantCreate200JSONResponse(
		*transformers.ToTenantSqlc(tenant),
	), nil
}
