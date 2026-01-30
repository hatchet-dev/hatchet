package transformers

import (
	"github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToUser(user *sqlcv1.User, hasPassword bool, hashedEmail *string) *gen.User {
	var name *string

	if user.Name.Valid {
		name = &user.Name.String
	}

	return &gen.User{
		Metadata:      *toAPIMetadata(user.ID.String(), user.CreatedAt.Time, user.UpdatedAt.Time),
		Email:         types.Email(user.Email),
		EmailHash:     hashedEmail,
		EmailVerified: user.EmailVerified,
		Name:          name,
		HasPassword:   &hasPassword,
	}
}

func ToTenantMember(tenantMember *sqlcv1.PopulateTenantMembersRow) *gen.TenantMember {
	var environment *gen.TenantEnvironment
	if tenantMember.TenantEnvironment.Valid {
		env := gen.TenantEnvironment(tenantMember.TenantEnvironment.TenantEnvironment)
		environment = &env
	}

	res := &gen.TenantMember{
		Metadata: *toAPIMetadata(tenantMember.ID.String(), tenantMember.CreatedAt.Time, tenantMember.UpdatedAt.Time),
		User: gen.UserTenantPublic{
			Email: types.Email(tenantMember.Email),
			Name:  v1.StringPtr(tenantMember.Name.String),
		},
		Tenant: &gen.Tenant{
			Metadata:          *toAPIMetadata(tenantMember.TenantId.String(), tenantMember.TenantCreatedAt.Time, tenantMember.TenantUpdatedAt.Time),
			Name:              tenantMember.TenantName,
			Slug:              tenantMember.TenantSlug,
			AnalyticsOptOut:   &tenantMember.AnalyticsOptOut,
			AlertMemberEmails: &tenantMember.AlertMemberEmails,
			Version:           gen.TenantVersion(tenantMember.TenantVersion),
			Environment:       environment,
		},
		Role: gen.TenantMemberRole(tenantMember.Role),
	}

	return res
}
