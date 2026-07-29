package seed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/authmode"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

func SeedDatabase(dc *database.Layer) error {
	shouldSeedUser := dc.Seed.AdminEmail != "" && dc.Seed.AdminPassword != ""
	var userID uuid.UUID

	if shouldSeedUser {
		// validate the password meets complexity requirements before hashing
		v := validator.NewDefaultValidator()
		opts := struct {
			Password string `validate:"password"`
		}{dc.Seed.AdminPassword}

		if err := v.Validate(opts); err != nil {
			return fmt.Errorf("ADMIN_PASSWORD does not meet requirements: must be between 8 and 64 characters and contain at least one uppercase letter, one lowercase letter, and one number")
		}

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

		userID = user.ID
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

	if authmode.IsDisabled {
		if err := seedAuthDisabledToken(dc); err != nil {
			return err
		}
	}

	return nil
}

func seedAuthDisabledToken(dc *database.Layer) error {
	ctx := context.Background()

	if dc.Seed.DefaultTenantID != authmode.EmbeddedTokenTenantID {
		return fmt.Errorf("authdisabled mode requires the default tenant ID (%s) to match the embedded token tenant (%s)", dc.Seed.DefaultTenantID, authmode.EmbeddedTokenTenantID)
	}

	tokenID, err := uuid.Parse(authmode.EmbeddedTokenID)
	if err != nil {
		return fmt.Errorf("invalid embedded token ID: %w", err)
	}

	if _, getErr := dc.V1.APIToken().GetAPITokenById(ctx, tokenID); getErr == nil {
		return nil
	} else if !errors.Is(getErr, pgx.ErrNoRows) {
		return fmt.Errorf("checking for embedded token: %w", getErr)
	}

	tenantID, err := uuid.Parse(authmode.EmbeddedTokenTenantID)
	if err != nil {
		return fmt.Errorf("invalid embedded token tenant ID: %w", err)
	}

	name := "authdisabled-default"

	_, err = dc.V1.APIToken().CreateAPIToken(ctx, &v1.CreateAPITokenOpts{
		ID:        tokenID,
		ExpiresAt: time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
		TenantId:  &tenantID,
		Name:      &name,
		Internal:  true,
	})

	return err
}
