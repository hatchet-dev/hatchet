package authz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestCanViewPayloads(t *testing.T) {
	e := echo.New()
	newCtx := func() echo.Context {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		return e.NewContext(req, rec)
	}

	t.Run("api tokens unrestricted", func(t *testing.T) {
		if !CanViewPayloads(newCtx()) {
			t.Fatal("expected unrestricted access when no tenant-member is in context")
		}
	})

	t.Run("member flag false", func(t *testing.T) {
		c := newCtx()
		c.Set("tenant-member", &sqlcv1.PopulateTenantMembersRow{
			Role:            sqlcv1.TenantMemberRoleMEMBER,
			CanViewPayloads: false,
		})
		if CanViewPayloads(c) {
			t.Fatal("expected restricted MEMBER to be denied payloads")
		}
	})

	t.Run("member flag true", func(t *testing.T) {
		c := newCtx()
		c.Set("tenant-member", &sqlcv1.PopulateTenantMembersRow{
			Role:            sqlcv1.TenantMemberRoleMEMBER,
			CanViewPayloads: true,
		})
		if !CanViewPayloads(c) {
			t.Fatal("expected MEMBER with flag set to see payloads")
		}
	})

	t.Run("admin always", func(t *testing.T) {
		c := newCtx()
		c.Set("tenant-member", &sqlcv1.PopulateTenantMembersRow{
			Role:            sqlcv1.TenantMemberRoleADMIN,
			CanViewPayloads: false,
		})
		if !CanViewPayloads(c) {
			t.Fatal("expected ADMIN to see payloads regardless of flag")
		}
	})

	t.Run("owner always", func(t *testing.T) {
		c := newCtx()
		c.Set("tenant-member", &sqlcv1.PopulateTenantMembersRow{
			Role:            sqlcv1.TenantMemberRoleOWNER,
			CanViewPayloads: false,
		})
		if !CanViewPayloads(c) {
			t.Fatal("expected OWNER to see payloads regardless of flag")
		}
	})

	t.Run("viewer flag false", func(t *testing.T) {
		c := newCtx()
		c.Set("tenant-member", &sqlcv1.PopulateTenantMembersRow{
			Role:            sqlcv1.TenantMemberRoleVIEWER,
			CanViewPayloads: false,
		})
		if CanViewPayloads(c) {
			t.Fatal("expected restricted VIEWER to be denied payloads")
		}
	})
}
