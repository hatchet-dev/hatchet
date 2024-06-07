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

type ExpiringTokenEmailData struct {
	ExpiresAtAbsoluteDate string `json:"expires_at_absolute_date"`
	ExpiresAtRelativeDate string `json:"expires_at_relative_date"`
	TokenName             string `json:"token_name"`
	Subject               string `json:"subject"`
	TenantName            string `json:"tenant_name"`
	TokenSettings         string `json:"token_settings"`
	SettingsLink          string `json:"settings_link"`
}

type ResourceLimitAlertData struct {
	Subject      string `json:"subject"`
	Summary      string `json:"summary"`
	Summary2     string `json:"summary2"`
	TenantName   string `json:"tenant_name"`
	Link         string `json:"link"`
	Resource     string `json:"resource"`
	AlertType    string `json:"alert_type"`
	CurrentValue int    `json:"current_value"`
	LimitValue   int    `json:"limit_value"`
	Percentage   int    `json:"percentage"`
	SettingsLink string `json:"settings_link"`
}

type EmailService interface {
	// for clients to show email settings
	IsValid() bool

	SendTenantInviteEmail(ctx context.Context, email string, data TenantInviteEmailData) error
	SendWorkflowRunFailedAlerts(ctx context.Context, emails []string, data WorkflowRunsFailedEmailData) error
	SendExpiringTokenEmail(ctx context.Context, emails []string, data ExpiringTokenEmailData) error
	SendTenantResourceLimitAlert(ctx context.Context, emails []string, data ResourceLimitAlertData) error
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

func (s *NoOpService) SendExpiringTokenEmail(ctx context.Context, emails []string, data ExpiringTokenEmailData) error {
	return nil
}

func (s *NoOpService) SendTenantResourceLimitAlert(ctx context.Context, emails []string, data ResourceLimitAlertData) error {
	return nil
}
