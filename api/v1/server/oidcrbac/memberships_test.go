package oidcrbac

import (
	"testing"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestDesiredTenantMembershipsUsesHighestRole(t *testing.T) {
	tenantID := uuid.New()
	mappings := map[string][]tenantGroupMapping{
		"engineering": {{TenantID: tenantID, Role: "MEMBER"}},
		"admins":      {{TenantID: tenantID, Role: "ADMIN"}},
	}
	desired := desiredTenantMemberships([]string{"engineering", "admins"}, nil, mappings, nil)
	if desired[tenantID] != sqlcv1.TenantMemberRoleADMIN {
		t.Fatalf("role = %q, want ADMIN", desired[tenantID])
	}
}

func TestDesiredTenantMembershipsExpandsHighestGlobalRole(t *testing.T) {
	tenantID := uuid.New()
	mappings := map[string][]tenantGroupMapping{"admins": {{TenantID: tenantID, Role: "ADMIN"}}}
	desired := desiredTenantMemberships([]string{"owners", "admins"}, map[string]string{"owners": "OWNER"}, mappings, []uuid.UUID{tenantID})
	if desired[tenantID] != sqlcv1.TenantMemberRoleOWNER {
		t.Fatalf("role = %q, want OWNER", desired[tenantID])
	}
}
