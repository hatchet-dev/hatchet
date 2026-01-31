package seed

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/config/database"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func SeedDatabase(dc *database.Layer) error {
	shouldSeedUser := dc.Seed.AdminEmail != "" && dc.Seed.AdminPassword != ""
	var userID string

	if shouldSeedUser {
		// seed an example user
		hashedPw, err := v1.HashPassword(dc.Seed.AdminPassword)

		if err != nil {
			return err
		}

		user, err := dc.V1.User().GetUserByEmail(context.Background(), dc.Seed.AdminEmail)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				user, err = dc.V1.User().CreateUser(context.Background(), &v1.CreateUserOpts{
					Email:         dc.Seed.AdminEmail,
					Name:          v1.StringPtr(dc.Seed.AdminName),
					EmailVerified: v1.BoolPtr(true),
					Password:      hashedPw,
				})

				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		userID = user.ID.String()
	}

	_, err := dc.V1.Tenant().GetTenantBySlug(context.Background(), dc.Seed.DefaultTenantSlug)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// seed an example tenant
			// initialize a tenant
			tenantID, err := uuid.Parse(dc.Seed.DefaultTenantID)
			if err != nil {
				return fmt.Errorf("invalid default tenant ID: %w", err)
			}

			sqlcTenant, err := dc.V1.Tenant().CreateTenant(context.Background(), &v1.CreateTenantOpts{
				ID:   &tenantID,
				Name: dc.Seed.DefaultTenantName,
				Slug: dc.Seed.DefaultTenantSlug,
			})

			if err != nil {
				return err
			}

			tenant, err := dc.V1.Tenant().GetTenantByID(context.Background(), sqlcTenant.ID)

			if err != nil {
				return err
			}

			fmt.Println("created tenant", tenant.ID.String())

			// add the user to the tenant
			_, err = dc.V1.Tenant().CreateTenantMember(context.Background(), tenant.ID, &v1.CreateTenantMemberOpts{
				Role:   "OWNER",
				UserId: userID,
			})

			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
