package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/enums"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	roleAPIToken = "API_TOKEN"
	roleNoAuth   = "NOAUTH"
)

type tenantRoleResolver struct {
	repo repository.Repository
}

func (r *tenantRoleResolver) ResolveRole(ctx context.Context, p pkgauthz.Principal, scope pkgauthz.Scope) (string, bool, error) {
	switch p := p.(type) {
	case pkgauthz.UserPrincipal:
		if scope.TenantID == nil {
			return pkgauthz.NoRole, true, nil
		}

		member, err := r.member(ctx, *scope.TenantID, p.UserID)
		if err != nil {
			return "", false, err
		}

		if member == nil {
			return "", false, nil
		}

		return string(member.Role), true, nil

	case pkgauthz.TokenPrincipal:
		// Management tokens hold no tenant role; the extension's Authorize hook decides for them.
		if p.Kind == pkgauthz.ManagementToken {
			return pkgauthz.NoRole, true, nil
		}

		if p.Kind != pkgauthz.TenantAPIToken || scope.TenantID == nil || p.TenantID == nil || *p.TenantID != *scope.TenantID {
			return "", false, nil
		}

		if p.ReadOnly {
			return string(enums.TenantMemberRoleVIEWER), true, nil
		}

		return roleAPIToken, true, nil
	}

	return "", false, fmt.Errorf("authz: unhandled principal %T", p)
}

// member returns nil, nil when the user is not in the tenant; the middleware reuses it for handlers.
func (r *tenantRoleResolver) member(ctx context.Context, tenantID, userID uuid.UUID) (*sqlcv1.PopulateTenantMembersRow, error) {
	key := pkgauthz.Key{Kind: pkgauthz.KindTenantMember, ID: "user:" + userID.String()}

	rows, err := pkgauthz.Fetch(ctx, pkgauthz.TenantScope(tenantID), []pkgauthz.Key{key}, func(ctx context.Context, _ []pkgauthz.Key) (map[pkgauthz.Key]*sqlcv1.PopulateTenantMembersRow, error) {
		m, err := r.repo.Tenant().GetTenantMemberByUserID(ctx, tenantID, userID)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && m == nil) {
			return map[pkgauthz.Key]*sqlcv1.PopulateTenantMembersRow{}, nil
		}

		if err != nil {
			return nil, err
		}

		return map[pkgauthz.Key]*sqlcv1.PopulateTenantMembersRow{key: m}, nil
	})
	if err != nil {
		return nil, err
	}

	return rows[key], nil
}
