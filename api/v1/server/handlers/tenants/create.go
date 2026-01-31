package tenants

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantCreate(ctx echo.Context, request gen.TenantCreateRequestObject) (gen.TenantCreateResponseObject, error) {
	user := ctx.Get("user").(*sqlcv1.User)

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
	_, err := t.config.V1.Tenant().GetTenantBySlug(ctx.Request().Context(), request.Body.Slug)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if err == nil {
		// just return bad request
		return gen.TenantCreate400JSONResponse(
			apierrors.NewAPIErrors("Tenant with that slug already exists."),
		), nil
	}

	createOpts := &v1.CreateTenantOpts{
		Slug: request.Body.Slug,
		Name: request.Body.Name,
	}

	if request.Body.Environment != nil {
		environment := string(*request.Body.Environment)
		createOpts.Environment = &environment
	}

	if t.config.Runtime.Limits.DefaultTenantRetentionPeriod != "" {
		createOpts.DataRetentionPeriod = &t.config.Runtime.Limits.DefaultTenantRetentionPeriod
	}

	var engineVersion *sqlcv1.TenantMajorEngineVersion

	if request.Body.EngineVersion != nil {
		ver := sqlcv1.TenantMajorEngineVersion(*request.Body.EngineVersion)
		engineVersion = &ver
	}

	createOpts.EngineVersion = engineVersion

	if request.Body.OnboardingData != nil {
		createOpts.OnboardingData = *request.Body.OnboardingData
	}

	// write the user to the db
	tenant, err := t.config.V1.Tenant().CreateTenant(ctx.Request().Context(), createOpts)

	if err != nil {
		return nil, err
	}

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// add the user as an owner of the tenant
	_, err = t.config.V1.Tenant().CreateTenantMember(ctx.Request().Context(), tenantId, &v1.CreateTenantMemberOpts{
		UserId: sqlchelpers.UUIDToStr(user.ID),
		Role:   "OWNER",
	})

	if err != nil {
		return nil, err
	}

	ctx.Set("tenant", tenant)

	return gen.TenantCreate200JSONResponse(
		*transformers.ToTenant(tenant),
	), nil
}
