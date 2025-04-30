package seed

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func SeedDatabase(dc *database.Layer) error {
	shouldSeedUser := dc.Seed.AdminEmail != "" && dc.Seed.AdminPassword != ""
	var userID string

	if shouldSeedUser {
		// seed an example user
		hashedPw, err := repository.HashPassword(dc.Seed.AdminPassword)

		if err != nil {
			return err
		}

		user, err := dc.APIRepository.User().GetUserByEmail(context.Background(), dc.Seed.AdminEmail)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				user, err = dc.APIRepository.User().CreateUser(context.Background(), &repository.CreateUserOpts{
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

	_, err := dc.APIRepository.Tenant().GetTenantBySlug(context.Background(), "default")

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// seed an example tenant
			// initialize a tenant
			sqlcTenant, err := dc.APIRepository.Tenant().CreateTenant(context.Background(), &repository.CreateTenantOpts{
				ID:   &dc.Seed.DefaultTenantID,
				Name: dc.Seed.DefaultTenantName,
				Slug: dc.Seed.DefaultTenantSlug,
			})

			if err != nil {
				return err
			}

			tenant, err := dc.APIRepository.Tenant().GetTenantByID(context.Background(), sqlchelpers.UUIDToStr(sqlcTenant.ID))

			if err != nil {
				return err
			}

			fmt.Println("created tenant", sqlchelpers.UUIDToStr(tenant.ID))

			// add the user to the tenant
			_, err = dc.APIRepository.Tenant().CreateTenantMember(context.Background(), sqlchelpers.UUIDToStr(tenant.ID), &repository.CreateTenantMemberOpts{
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
