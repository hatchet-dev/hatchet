package email

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting/alerttypes"
)

type TenantInviteEmailData struct {
	InviteSenderName string `json:"invite_sender_name"`
	TenantName       string `json:"tenant_name"`
	ActionURL        string `json:"action_url"`
}

type WorkflowRunsFailedEmailData struct {
	Items        []alerttypes.WorkflowRunFailedItem `json:"items"`
	Subject      string                             `json:"subject"`
	Summary      string                             `json:"summary"`
	TenantName   string                             `json:"tenant_name"`
	SettingsLink string                             `json:"settings_link"`
}

type EmailService interface {
	// for clients to show email settings
	IsValid() bool

	SendTenantInviteEmail(ctx context.Context, email string, data TenantInviteEmailData) error
	SendWorkflowRunFailedAlerts(ctx context.Context, emails []string, data WorkflowRunsFailedEmailData) error
}

type NoOpService struct{}

func (s *NoOpService) IsValid() bool {
	return false
}

func (s *NoOpService) SendTenantInviteEmail(ctx context.Context, email string, data TenantInviteEmailData) error {
	return nil
}

func (s *NoOpService) SendWorkflowRunFailedAlerts(ctx context.Context, emails []string, data WorkflowRunsFailedEmailData) error {
	return nil
}
