package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type tenantRoleResolver struct {
	repo repository.Repository
}

func (r *tenantRoleResolver) ResolveRole(ctx context.Context, p pkgauthz.Principal, a pkgauthz.Action) (string, bool, error) {
	action, ok := a.(pkgauthz.RESTAction)
	if !ok {
		return "", false, fmt.Errorf("authz: tenant roles only apply to REST actions, got %s", a.Type())
	}

	tenantID := action.TenantID

	switch p := p.(type) {
	case pkgauthz.UserPrincipal:
		if tenantID == nil {
			return pkgauthz.NoRole, true, nil
		}

		member, err := r.member(ctx, *tenantID, p.UserID)
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
	}

	return "", false, fmt.Errorf("authz: unhandled principal %T", p)
}

// member returns nil, nil when the user is not in the tenant; the middleware reuses it for handlers.
func (r *tenantRoleResolver) member(ctx context.Context, tenantID, userID uuid.UUID) (*sqlcv1.PopulateTenantMembersRow, error) {
	key := pkgauthz.Key{Kind: "tenant-member", ID: tenantID.String() + "/" + userID.String()}

	// load runs only on a cache miss; errors and missing members are never cached, so a retry reloads them
	rows, err := pkgauthz.Fetch(ctx, []pkgauthz.Key{key}, func(ctx context.Context, _ []pkgauthz.Key) (map[pkgauthz.Key]*sqlcv1.PopulateTenantMembersRow, error) {
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
