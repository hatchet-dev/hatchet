package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToTenantInviteLink(invite *sqlcv1.TenantInviteLink) *gen.TenantInvite {
	res := &gen.TenantInvite{
		Metadata: *toAPIMetadata(invite.ID, invite.CreatedAt.Time, invite.UpdatedAt.Time),
		Email:    invite.InviteeEmail,
		Expires:  invite.Expires.Time,
		Role:     gen.TenantMemberRole(invite.Role),
		TenantId: invite.TenantId.String(),
	}

	return res
}

func ToUserTenantInviteLink(invite *sqlcv1.ListTenantInvitesByEmailRow) *gen.TenantInvite {
	res := &gen.TenantInvite{
		Metadata:   *toAPIMetadata(invite.ID, invite.CreatedAt.Time, invite.UpdatedAt.Time),
		Email:      invite.InviteeEmail,
		Expires:    invite.Expires.Time,
		Role:       gen.TenantMemberRole(invite.Role),
		TenantId:   invite.TenantId.String(),
		TenantName: &invite.TenantName,
	}

	return res
}
