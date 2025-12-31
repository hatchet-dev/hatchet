package seed

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
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
					Name:          repository.StringPtr(dc.Seed.AdminName),
					EmailVerified: repository.BoolPtr(true),
					Password:      hashedPw,
				})

				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		userID = sqlchelpers.UUIDToStr(user.ID)
	}

	_, err := dc.V1.Tenant().GetTenantBySlug(context.Background(), dc.Seed.DefaultTenantSlug)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// seed an example tenant
			// initialize a tenant
			sqlcTenant, err := dc.V1.Tenant().CreateTenant(context.Background(), &v1.CreateTenantOpts{
				ID:   &dc.Seed.DefaultTenantID,
				Name: dc.Seed.DefaultTenantName,
				Slug: dc.Seed.DefaultTenantSlug,
			})

			if err != nil {
				return err
			}

			tenant, err := dc.V1.Tenant().GetTenantByID(context.Background(), sqlchelpers.UUIDToStr(sqlcTenant.ID))

			if err != nil {
				return err
			}

			fmt.Println("created tenant", sqlchelpers.UUIDToStr(tenant.ID))

			// add the user to the tenant
			_, err = dc.V1.Tenant().CreateTenantMember(context.Background(), sqlchelpers.UUIDToStr(tenant.ID), &v1.CreateTenantMemberOpts{
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
