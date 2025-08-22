package tenants

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantCreate(ctx echo.Context, request gen.TenantCreateRequestObject) (gen.TenantCreateResponseObject, error) {
	user := ctx.Get("user").(*dbsqlc.User)

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
	_, err := t.config.APIRepository.Tenant().GetTenantBySlug(ctx.Request().Context(), request.Body.Slug)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if err == nil {
		// just return bad request
		return gen.TenantCreate400JSONResponse(
			apierrors.NewAPIErrors("Tenant with that slug already exists."),
		), nil
	}

	createOpts := &repository.CreateTenantOpts{
		Slug: request.Body.Slug,
		Name: request.Body.Name,
	}

	if t.config.Runtime.Limits.DefaultTenantRetentionPeriod != "" {
		createOpts.DataRetentionPeriod = &t.config.Runtime.Limits.DefaultTenantRetentionPeriod
	}

	uiVersion := dbsqlc.TenantMajorUIVersionV0

	if request.Body.UiVersion != nil {
		ver := *request.Body.UiVersion
		uiVersion = dbsqlc.TenantMajorUIVersion(ver)
	}

	createOpts.UIVersion = &uiVersion

	var engineVersion *dbsqlc.TenantMajorEngineVersion

	if request.Body.EngineVersion != nil {
		ver := dbsqlc.TenantMajorEngineVersion(*request.Body.EngineVersion)
		engineVersion = &ver
	}

	createOpts.EngineVersion = engineVersion

	// write the user to the db
	tenant, err := t.config.APIRepository.Tenant().CreateTenant(ctx.Request().Context(), createOpts)

	if err != nil {
		return nil, err
	}

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	err = t.config.EntitlementRepository.TenantLimit().SelectOrInsertTenantLimits(context.Background(), tenantId, nil)

	if err != nil {
		return nil, err
	}

	// add the user as an owner of the tenant
	_, err = t.config.APIRepository.Tenant().CreateTenantMember(ctx.Request().Context(), tenantId, &repository.CreateTenantMemberOpts{
		UserId: sqlchelpers.UUIDToStr(user.ID),
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
		sqlchelpers.UUIDToStr(user.ID),
		&tenantId,
		map[string]interface{}{
			"tenant_created": true,
		},
		map[string]interface{}{
			"name": tenant.Name,
			"slug": tenant.Slug,
		},
	)

	return gen.TenantCreate200JSONResponse(
		*transformers.ToTenant(tenant),
	), nil
}
