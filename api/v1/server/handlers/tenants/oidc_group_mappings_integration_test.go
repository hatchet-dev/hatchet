//go:build integration

package tenants

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func TestTenantOIDCGroupMappingPersistence(t *testing.T) {
	testutils.RunTestWithDatabase(t, func(db *database.Layer) error {
		ctx := context.Background()
		repo := db.V1.Tenant()

		if _, err := repo.CreateTenant(ctx, &v1.CreateTenantOpts{
			Name: "Invalid owner mapping",
			Slug: "invalid-owner-" + uuid.NewString(),
			OIDCGroupMapping: &v1.UpsertTenantOIDCGroupMappingOpts{
				Group: "owners",
				Role:  "OWNER",
			},
		}); err == nil {
			t.Fatal("expected OWNER mapping to be rejected")
		}

		tenant, err := repo.CreateTenant(ctx, &v1.CreateTenantOpts{
			Name: "Mapped tenant",
			Slug: "mapped-tenant-" + uuid.NewString(),
			OIDCGroupMapping: &v1.UpsertTenantOIDCGroupMappingOpts{
				Group: "engineering",
				Role:  "MEMBER",
			},
		})
		if err != nil {
			return err
		}
		defer db.Pool.Exec(ctx, `DELETE FROM "Tenant" WHERE "id" = $1`, tenant.ID) //nolint:errcheck

		mappings, err := repo.ListTenantOIDCGroupMappings(ctx, tenant.ID)
		if err != nil {
			return err
		}
		if len(mappings) != 1 || mappings[0].Group != "engineering" || mappings[0].Role != "MEMBER" {
			t.Fatalf("unexpected mappings: %+v", mappings)
		}

		deleted, err := repo.DeleteTenantOIDCGroupMapping(ctx, uuid.New(), mappings[0].ID)
		if err != nil {
			return err
		}
		if deleted {
			t.Fatal("mapping was deleted through a different tenant")
		}

		deleted, err = repo.DeleteTenantOIDCGroupMapping(ctx, tenant.ID, mappings[0].ID)
		if err != nil {
			return err
		}
		if !deleted {
			t.Fatal("expected mapping to be deleted")
		}
		return nil
	})
}
