package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type User struct {
	key      string
	name     string
	email    string
	password string
	role     string
	id       string
}

type Tenant struct {
	name string
	slug string
	id   string
}

func SeedDatabaseForCypress(dc *database.Layer) error {
	ctx := context.Background()
	logger := log.New(os.Stdout, "seed-cypress: ", log.LstdFlags)
	start := time.Now()

	// Use slices (not maps) so ordering is deterministic and we can persist IDs on the structs.
	users := []User{
		{
			key:      "owner",
			name:     "Owner Owen",
			email:    "owner@example.com",
			password: "OwnerPassword123!",
			role:     "OWNER",
		},
		{
			key:      "admin",
			name:     "Admin Adam",
			email:    "admin@example.com",
			password: "AdminPassword123!",
			role:     "ADMIN",
		},
		{
			key:      "member",
			name:     "Member Mike",
			email:    "member@example.com",
			password: "MemberPassword123!",
			role:     "MEMBER",
		},
	}

	tenants := []Tenant{
		{
			name: "Tenant 1",
			slug: "tenant1",
		},
		{
			name: "Tenant 2",
			slug: "tenant2",
		},
	}

	logger.Printf("starting seed (users=%d tenants=%d)", len(users), len(tenants))

	var createdUsers, existingUsers int
	for i := range users {
		user := &users[i]

		hashedPw, err := repository.HashPassword(user.password)
		if err != nil {
			return err
		}

		insertedUser, err := dc.APIRepository.User().GetUserByEmail(ctx, user.email)
		action := "exists"

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				insertedUser, err = dc.APIRepository.User().CreateUser(ctx, &repository.CreateUserOpts{
					Email:         user.email,
					Name:          repository.StringPtr(user.name),
					EmailVerified: repository.BoolPtr(true),
					Password:      hashedPw,
				})

				if err != nil {
					return err
				}

				action = "created"
				createdUsers++
			} else {
				return err
			}
		} else {
			existingUsers++
		}

		user.id = sqlchelpers.UUIDToStr(insertedUser.ID)
		logger.Printf("user %s: name=%q email=%q role=%s user_id=%s", action, user.name, user.email, user.role, user.id)
	}

	var createdTenants, existingTenants int
	for i := range tenants {
		tenant := &tenants[i]
		action := "exists"

		insertedTenant, err := dc.APIRepository.Tenant().GetTenantBySlug(ctx, tenant.slug)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				insertedTenant, err = dc.APIRepository.Tenant().CreateTenant(ctx, &repository.CreateTenantOpts{
					Name: tenant.name,
					Slug: tenant.slug,
				})

				if err != nil {
					return err
				}

				action = "created"
				createdTenants++
			} else {
				return err
			}
		} else {
			existingTenants++
		}

		tenant.id = sqlchelpers.UUIDToStr(insertedTenant.ID)
		logger.Printf("tenant %s: name=%q slug=%q tenant_id=%s", action, tenant.name, tenant.slug, tenant.id)
	}

	var createdMembers, existingMembers int
	for i := range tenants {
		tenant := &tenants[i]
		for j := range users {
			user := &users[j]

			// Idempotent: check membership first so reruns are clean.
			member, err := dc.APIRepository.Tenant().GetTenantMemberByUserID(ctx, tenant.id, user.id)
			if err == nil {
				existingMembers++
				logger.Printf(
					"tenant_member exists: tenant_slug=%q tenant_id=%s user_email=%q user_id=%s role=%s member_id=%s",
					tenant.slug,
					tenant.id,
					user.email,
					user.id,
					user.role,
					sqlchelpers.UUIDToStr(member.ID),
				)
				continue
			}

			if !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf(
					"checking tenant member (tenant_slug=%s tenant_id=%s user_email=%s user_id=%s): %w",
					tenant.slug,
					tenant.id,
					user.email,
					user.id,
					err,
				)
			}

			createdMember, err := dc.APIRepository.Tenant().CreateTenantMember(ctx, tenant.id, &repository.CreateTenantMemberOpts{
				Role:   user.role,
				UserId: user.id,
			})

			if err != nil {
				return fmt.Errorf(
					"creating tenant member (tenant_slug=%s tenant_id=%s user_email=%s user_id=%s role=%s): %w",
					tenant.slug,
					tenant.id,
					user.email,
					user.id,
					user.role,
					err,
				)
			}

			createdMembers++
			logger.Printf(
				"tenant_member created: tenant_slug=%q tenant_id=%s user_email=%q user_id=%s role=%s member_id=%s",
				tenant.slug,
				tenant.id,
				user.email,
				user.id,
				user.role,
				sqlchelpers.UUIDToStr(createdMember.ID),
			)
		}
	}

	// Write the seeded users to Cypress support as a TS module so tests can import stable credentials.
	if err := writeCypressSeededUsersModule(users); err != nil {
		return err
	}

	logger.Printf(
		"seed complete in %s (users created=%d exists=%d; tenants created=%d exists=%d; tenant_members created=%d exists=%d)",
		time.Since(start).Truncate(time.Millisecond),
		createdUsers,
		existingUsers,
		createdTenants,
		existingTenants,
		createdMembers,
		existingMembers,
	)

	return nil
}

type cypressSeededUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	UserID   string `json:"userId"`
}

type cypressEnvUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func writeCypressSeededUsersModule(users []User) error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}

	targetFile := filepath.Join(repoRoot, "frontend", "app", "cypress", "support", "seeded-users.generated.ts")

	byKey := make(map[string]cypressSeededUser, len(users))
	list := make([]cypressSeededUser, 0, len(users))
	envList := make([]cypressEnvUser, 0, len(users))

	for _, u := range users {
		seeded := cypressSeededUser{
			Name:     u.name,
			Email:    u.email,
			Password: u.password,
			Role:     u.role,
			UserID:   u.id,
		}

		byKey[u.key] = seeded
		list = append(list, seeded)
		envList = append(envList, cypressEnvUser{Email: u.email, Password: u.password})
	}

	byKeyJSON, err := json.MarshalIndent(byKey, "", "  ")
	if err != nil {
		return err
	}

	listJSON, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	envListJSON, err := json.MarshalIndent(envList, "", "  ")
	if err != nil {
		return err
	}

	generatedAt := time.Now().UTC().Format(time.RFC3339)
	contents := fmt.Sprintf(`// Code generated by hatchet-admin seed-cypress. DO NOT EDIT.
// Generated at %s

export const seededUsers = %s as const;

export const seededUsersList = %s as const;

// Convenience: matches Cypress.env('seedUsers') shape used by loginSession().
export const seedUsersForEnv = %s as const;

export type SeededUserKey = keyof typeof seededUsers;
export type SeededUser = (typeof seededUsers)[SeededUserKey];
`, generatedAt, string(byKeyJSON), string(listJSON), string(envListJSON))

	if err := os.MkdirAll(filepath.Dir(targetFile), 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(targetFile, []byte(contents), 0o644); err != nil {
		return err
	}

	return nil
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repo root (no go.mod found walking up from %s)", wd)
		}

		dir = parent
	}
}
