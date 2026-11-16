package tenants

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantGet(ctx echo.Context, request gen.TenantGetRequestObject) (gen.TenantGetResponseObject, error) {
	maybeTenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	t.trackClientConnect(ctx, maybeTenant)

	tenant := transformers.ToTenant(maybeTenant, t.config.Runtime.ServerURL)

	return gen.TenantGet200JSONResponse(
		*tenant,
	), nil
}

// trackClientConnect emits a tenant:connect event when a client identifies itself
// as the CLI, which happens when a profile is created or updated against this
// tenant. This is what links a CLI install to a tenant in analytics.
//
// The UI polls this endpoint, so requests without the header are ignored: the
// event is meant to mark a client attaching itself, not every tenant read.
func (t *TenantService) trackClientConnect(ctx echo.Context, tenant *sqlcv1.Tenant) {
	source := analytics.Source(ctx.Request().Header.Get(analytics.SourceMetadataKey))

	if source != analytics.SourceCLI {
		return
	}

	// The REST auth middleware hardcodes SourceAPI for token auth, so the real
	// source is set on the analytics context here rather than plumbed through authn.
	analyticsCtx := context.WithValue(ctx.Request().Context(), analytics.SourceKey, source)

	props := analytics.Properties{}

	if version := ctx.Request().Header.Get(analytics.CLIVersionMetadataKey); version != "" {
		props["cli_version"] = version
	}

	if command := ctx.Request().Header.Get(analytics.CLICommandMetadataKey); command != "" {
		props["command"] = command
	}

	t.config.Analytics.Enqueue(
		analyticsCtx,
		analytics.Tenant, analytics.Connect,
		tenant.ID.String(),
		props,
	)
}
