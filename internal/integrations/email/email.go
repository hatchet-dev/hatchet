package email

import "context"

type TenantInviteEmailData struct {
	InviteSenderName string `json:"invite_sender_name"`
	TenantName       string `json:"tenant_name"`
	ActionURL        string `json:"action_url"`
}

type EmailService interface {
	SendTenantInviteEmail(ctx context.Context, email string, data TenantInviteEmailData) error
}

type NoOpService struct{}

func (s *NoOpService) SendTenantInviteEmail(ctx context.Context, email string, data TenantInviteEmailData) error {
	return nil
}
