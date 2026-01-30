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
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type User struct {
	key         string
	name        string
	email       string
	password    string
	role        string
	id          string
	tenantSlugs []string
}

type Tenant struct {
	name string
	slug string
	id   string
}

const (
	tenant1Slug = "tenant1"
	tenant2Slug = "tenant2"
)

func SeedDatabaseForCypress(dc *database.Layer) error {
	ctx := context.Background()
	logger := log.New(os.Stdout, "seed-cypress: ", log.LstdFlags)
	start := time.Now()

	// Use slices (not maps) so ordering is deterministic and we can persist IDs on the structs.
	users := []User{
		{
			key:         "owner",
			name:        "Owner Owen",
			email:       "owner@example.com",
			password:    "OwnerPassword123!",
			role:        "OWNER",
			tenantSlugs: []string{tenant1Slug, tenant2Slug},
		},
		{
			key:         "admin",
			name:        "Admin Adam",
			email:       "admin@example.com",
			password:    "AdminPassword123!",
			role:        "ADMIN",
			tenantSlugs: []string{tenant1Slug},
		},
		{
			key:         "member",
			name:        "Member Mike",
			email:       "member@example.com",
			password:    "MemberPassword123!",
			role:        "MEMBER",
			tenantSlugs: []string{tenant1Slug},
		},
	}

	tenants := []Tenant{
		{
			name: "Tenant 1",
			slug: tenant1Slug,
		},
		{
			name: "Tenant 2",
			slug: tenant2Slug,
		},
	}

	logger.Printf("starting seed (users=%d tenants=%d)", len(users), len(tenants))

	var createdUsers, existingUsers int
	for i := range users {
		user := &users[i]

		hashedPw, err := v1.HashPassword(user.password)
		if err != nil {
			return err
		}

		insertedUser, err := dc.V1.User().GetUserByEmail(ctx, user.email)
		action := "exists"

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				insertedUser, err = dc.V1.User().CreateUser(ctx, &v1.CreateUserOpts{
					Email:         user.email,
					Name:          v1.StringPtr(user.name),
					EmailVerified: v1.BoolPtr(true),
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

		user.id = insertedUser.ID.String()
		logger.Printf("user %s: name=%q email=%q role=%s user_id=%s", action, user.name, user.email, user.role, user.id)
	}

	var createdTenants, existingTenants int
	for i := range tenants {
		tenant := &tenants[i]
		action := "exists"

		insertedTenant, err := dc.V1.Tenant().GetTenantBySlug(ctx, tenant.slug)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				insertedTenant, err = dc.V1.Tenant().CreateTenant(ctx, &v1.CreateTenantOpts{
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

		tenant.id = insertedTenant.ID.String()
		logger.Printf("tenant %s: name=%q slug=%q tenant_id=%s", action, tenant.name, tenant.slug, tenant.id)
	}

	var createdMembers, existingMembers int
	var deletedDisallowedMembers int
	for i := range tenants {
		tenant := &tenants[i]
		for j := range users {
			user := &users[j]

			allowed := userHasTenantSlug(*user, tenant.slug)

			// Idempotent: check membership first so reruns are clean.
			member, err := dc.V1.Tenant().GetTenantMemberByUserID(ctx, tenant.id, user.id)
			if err == nil {
				if !allowed {
					if err := dc.V1.Tenant().DeleteTenantMember(ctx, member.ID.String()); err != nil {
						return fmt.Errorf(
							"deleting disallowed tenant member (tenant_slug=%s tenant_id=%s user_email=%s user_id=%s member_id=%s): %w",
							tenant.slug,
							tenant.id,
							user.email,
							user.id,
							member.ID.String(),
							err,
						)
					}

					deletedDisallowedMembers++
					logger.Printf(
						"tenant_member deleted (disallowed): tenant_slug=%q tenant_id=%s user_email=%q user_id=%s role=%s member_id=%s",
						tenant.slug,
						tenant.id,
						user.email,
						user.id,
						user.role,
						member.ID.String(),
					)
					continue
				}

				existingMembers++
				logger.Printf(
					"tenant_member exists: tenant_slug=%q tenant_id=%s user_email=%q user_id=%s role=%s member_id=%s",
					tenant.slug,
					tenant.id,
					user.email,
					user.id,
					user.role,
					member.ID.String(),
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

			// No row exists.
			if !allowed {
				// Intentionally skip creating disallowed memberships.
				continue
			}

			createdMember, err := dc.V1.Tenant().CreateTenantMember(ctx, tenant.id, &v1.CreateTenantMemberOpts{
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
				createdMember.ID.String(),
			)
		}
	}

	// Write the seeded users to Cypress support as a TS module so tests can import stable credentials.
	if err := writeCypressSeededUsersModule(users); err != nil {
		return err
	}

	logger.Printf(
		"seed complete in %s (users created=%d exists=%d; tenants created=%d exists=%d; tenant_members created=%d exists=%d deleted_disallowed=%d)",
		time.Since(start).Truncate(time.Millisecond),
		createdUsers,
		existingUsers,
		createdTenants,
		existingTenants,
		createdMembers,
		existingMembers,
		deletedDisallowedMembers,
	)

	return nil
}

func userHasTenantSlug(u User, tenantSlug string) bool {
	for _, slug := range u.tenantSlugs {
		if slug == tenantSlug {
			return true
		}
	}

	return false
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

	if err := os.WriteFile(targetFile, []byte(contents), 0o600); err != nil {
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
