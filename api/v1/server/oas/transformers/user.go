package transformers

import (
	"github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func ToUser(user *dbsqlc.User, hasPassword bool, hashedEmail *string) *gen.User {
	var name *string

	if user.Name.Valid {
		name = &user.Name.String
	}

	return &gen.User{
		Metadata:      *toAPIMetadata(sqlchelpers.UUIDToStr(user.ID), user.CreatedAt.Time, user.UpdatedAt.Time),
		Email:         types.Email(user.Email),
		EmailHash:     hashedEmail,
		EmailVerified: user.EmailVerified,
		Name:          name,
		HasPassword:   &hasPassword,
	}
}

func ToTenantMember(tenantMember *dbsqlc.PopulateTenantMembersRow) *gen.TenantMember {
	res := &gen.TenantMember{
		Metadata: *toAPIMetadata(sqlchelpers.UUIDToStr(tenantMember.ID), tenantMember.CreatedAt.Time, tenantMember.UpdatedAt.Time),
		User: gen.UserTenantPublic{
			Email: types.Email(tenantMember.Email),
			Name:  repository.StringPtr(tenantMember.Name.String),
		},
		Tenant: &gen.Tenant{
			Metadata:          *toAPIMetadata(sqlchelpers.UUIDToStr(tenantMember.TenantId), tenantMember.TenantCreatedAt.Time, tenantMember.TenantUpdatedAt.Time),
			Name:              tenantMember.TenantName,
			Slug:              tenantMember.TenantSlug,
			AnalyticsOptOut:   &tenantMember.AnalyticsOptOut,
			AlertMemberEmails: &tenantMember.AlertMemberEmails,
			Version:           gen.TenantVersion(tenantMember.TenantVersion),
			UiVersion:         gen.TenantUIVersion(tenantMember.TenantUiVersion),
		},
		Role: gen.TenantMemberRole(tenantMember.Role),
	}

	return res
}
