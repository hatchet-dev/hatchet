package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func ToTenantInviteLink(invite *db.TenantInviteLinkModel) *gen.TenantInvite {
	res := &gen.TenantInvite{
		Metadata: *toAPIMetadata(invite.ID, invite.CreatedAt, invite.UpdatedAt),
		Email:    invite.InviteeEmail,
		Expires:  invite.Expires,
		Role:     gen.TenantMemberRole(invite.Role),
		TenantId: invite.TenantID,
	}

	if invite.RelationsTenantInviteLink.Tenant != nil {
		res.TenantName = &invite.Tenant().Name
	}

	return res
}
