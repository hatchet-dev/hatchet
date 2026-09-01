package oidcrbac

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// ReconcileTenantMemberships changes only memberships managed by the verified issuer.
func ReconcileTenantMemberships(ctx context.Context, config *server.ServerConfig, userID uuid.UUID, issuer string, groups []string) error {
	if config.Layer == nil || config.Layer.Pool == nil || issuer == "" {
		return fmt.Errorf("OIDC group mapping requires a database and issuer URL")
	}

	tx, err := config.Layer.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlcv1.New()
	databaseMappings, err := queries.ListOIDCGroupMappings(ctx, tx, issuer)
	if err != nil {
		return err
	}
	mappings := make(map[string][]tenantGroupMapping)
	for _, mapping := range databaseMappings {
		mappings[mapping.Group] = append(mappings[mapping.Group], tenantGroupMapping{
			TenantID: mapping.TenantId,
			Role:     string(mapping.Role),
		})
	}

	allTenantIDs := []uuid.UUID(nil)
	if hasGlobalGroupMapping(groups, config.Auth.OIDCGroupMappings) {
		tenants, err := queries.ListTenants(ctx, tx)
		if err != nil {
			return err
		}
		for _, tenant := range tenants {
			if tenant.Slug != "internal" {
				allTenantIDs = append(allTenantIDs, tenant.ID)
			}
		}
	}

	desired := desiredTenantMemberships(groups, config.Auth.OIDCGroupMappings, mappings, allTenantIDs)
	current, err := queries.ListOIDCTenantMembers(ctx, tx, sqlcv1.ListOIDCTenantMembersParams{
		Userid: userID, Oidcissuer: issuer,
	})
	if err != nil {
		return err
	}

	for tenantID, role := range desired {
		if err := queries.UpsertOIDCTenantMember(ctx, tx, sqlcv1.UpsertOIDCTenantMemberParams{
			Tenantid: tenantID, Userid: userID, Role: role, Oidcissuer: issuer,
		}); err != nil {
			return err
		}
	}
	for _, member := range current {
		if _, ok := desired[member.TenantId]; ok {
			continue
		}
		if err := queries.DeleteOIDCTenantMember(ctx, tx, sqlcv1.DeleteOIDCTenantMemberParams{
			Tenantid: member.TenantId, Userid: userID, Oidcissuer: issuer,
		}); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

type tenantGroupMapping struct {
	TenantID uuid.UUID
	Role     string
}

func hasGlobalGroupMapping(groups []string, mappings map[string]string) bool {
	for _, group := range groups {
		if _, ok := mappings[group]; ok {
			return true
		}
	}
	return false
}

func desiredTenantMemberships(groups []string, globalMappings map[string]string, tenantMappings map[string][]tenantGroupMapping, allTenantIDs []uuid.UUID) map[uuid.UUID]sqlcv1.TenantMemberRole {
	desired := make(map[uuid.UUID]sqlcv1.TenantMemberRole)
	for _, group := range groups {
		if role, ok := globalMappings[group]; ok {
			for _, tenantID := range allTenantIDs {
				addDesiredRole(desired, tenantID, sqlcv1.TenantMemberRole(role))
			}
		}
		for _, mapping := range tenantMappings[group] {
			addDesiredRole(desired, mapping.TenantID, sqlcv1.TenantMemberRole(mapping.Role))
		}
	}
	return desired
}

func addDesiredRole(desired map[uuid.UUID]sqlcv1.TenantMemberRole, tenantID uuid.UUID, role sqlcv1.TenantMemberRole) {
	if current, ok := desired[tenantID]; !ok || roleRank(role) > roleRank(current) {
		desired[tenantID] = role
	}
}

func roleRank(role sqlcv1.TenantMemberRole) int {
	switch role {
	case sqlcv1.TenantMemberRoleOWNER:
		return 4
	case sqlcv1.TenantMemberRoleADMIN:
		return 3
	case sqlcv1.TenantMemberRoleMEMBER:
		return 2
	case sqlcv1.TenantMemberRoleVIEWER:
		return 1
	default:
		return 0
	}
}
