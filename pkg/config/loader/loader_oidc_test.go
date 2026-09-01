package loader

import "testing"

func TestParseOIDCGroupMappingsAcceptsGlobalRoles(t *testing.T) {
	mappings, err := parseOIDCGroupMappings(`{"owners":"OWNER","platform-admins":"ADMIN","auditors":"VIEWER"}`)
	if err != nil {
		t.Fatalf("parse global mappings: %v", err)
	}
	if mappings["platform-admins"] != "ADMIN" {
		t.Fatalf("platform-admins role = %q, want ADMIN", mappings["platform-admins"])
	}
}

func TestParseOIDCGroupMappingsRejectsTenantMappings(t *testing.T) {
	if _, err := parseOIDCGroupMappings(`{"tenant-members":[{"tenantId":"11111111-1111-4111-8111-111111111111","role":"MEMBER"}]}`); err == nil {
		t.Fatal("expected tenant mapping to be rejected")
	}
}

func TestParseOIDCGroupMappingsRejectsMember(t *testing.T) {
	if _, err := parseOIDCGroupMappings(`{"members":"MEMBER"}`); err == nil {
		t.Fatal("expected global MEMBER mapping to be rejected")
	}
}
