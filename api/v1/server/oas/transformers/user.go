package transformers

import (
	"github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func ToUser(user *db.UserModel, hasPassword bool, hashedEmail *string) *gen.User {
	var name *string

	if dbName, ok := user.Name(); ok {
		name = &dbName
	}

	return &gen.User{
		Metadata:      *toAPIMetadata(user.ID, user.CreatedAt, user.UpdatedAt),
		Email:         types.Email(user.Email),
		EmailHash:     hashedEmail,
		EmailVerified: user.EmailVerified,
		Name:          name,
		HasPassword:   &hasPassword,
	}
}

func ToUserTenantPublic(user *db.UserModel) *gen.UserTenantPublic {
	var name *string

	if dbName, ok := user.Name(); ok {
		name = &dbName
	}

	return &gen.UserTenantPublic{
		Email: types.Email(user.Email),
		Name:  name,
	}
}

func ToTenantMember(tenantMember *db.TenantMemberModel) *gen.TenantMember {
	res := &gen.TenantMember{
		Metadata: *toAPIMetadata(tenantMember.ID, tenantMember.CreatedAt, tenantMember.UpdatedAt),
		User:     *ToUserTenantPublic(tenantMember.User()),
		Role:     gen.TenantMemberRole(tenantMember.Role),
	}

	if tenantMember.Tenant() != nil {
		res.Tenant = ToTenant(tenantMember.Tenant())
	}

	return res
}
