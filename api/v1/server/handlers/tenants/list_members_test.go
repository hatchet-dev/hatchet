package tenants

import "testing"

func TestHasGlobalConfigMapping(t *testing.T) {
	mappings := map[string]string{"platform-admins": "ADMIN"}
	if !hasGlobalConfigMapping("platform-admins", mappings) {
		t.Fatal("expected configured group to be globally managed")
	}
	if hasGlobalConfigMapping("engineering", mappings) {
		t.Fatal("expected unconfigured group to remain tenant-managed")
	}
}

func TestTenantCreationAllowed(t *testing.T) {
	if tenantCreationAllowed(true, true, false) {
		t.Fatal("expected non-global admin to be denied")
	}
	if !tenantCreationAllowed(true, true, true) {
		t.Fatal("expected global admin to be allowed")
	}
	if tenantCreationAllowed(false, false, true) {
		t.Fatal("expected disabled tenant creation to deny global admin")
	}
}
