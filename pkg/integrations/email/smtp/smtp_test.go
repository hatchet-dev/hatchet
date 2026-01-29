//go:build !e2e && !load && !rampup && !integration

package smtp

import (
	"context"
	"fmt"
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/integrations/email"
	"github.com/stretchr/testify/require"
)

const (
	testFromEmail    = "from@example.com"
	testFromName     = "Test"
	testSupportEmail = "support@example.com"
	testUsername     = "testuser"
	testPassword     = "testpass"
)

func TestSMTPServiceSendMail(t *testing.T) {
	tests := []struct {
		name        string
		sendFunc    func(*SMTPService) error
		wantRcpts   []string
		wantSubject string
	}{
		{
			name: "tenant invite",
			sendFunc: func(s *SMTPService) error {
				return s.SendTenantInviteEmail(context.Background(), "user@example.com", email.TenantInviteEmailData{
					TenantName:       "Acme Corp",
					InviteSenderName: "Alice",
					ActionURL:        "https://app.example.com/join/abc123",
				})
			},
			wantRcpts:   []string{"user@example.com"},
			wantSubject: "Alice invited you to join Acme Corp on Hatchet",
		},
		{
			name: "workflow failed alert",
			sendFunc: func(s *SMTPService) error {
				return s.SendWorkflowRunFailedAlerts(context.Background(), []string{"admin1@example.com", "admin2@example.com"}, email.WorkflowRunsFailedEmailData{
					TenantName:   "Acme Corp",
					Subject:      "3 workflow runs failed in the last hour",
					Summary:      "3 workflow runs failed",
					SettingsLink: "https://app.example.com/settings",
				})
			},
			wantRcpts:   []string{"admin1@example.com", "admin2@example.com"},
			wantSubject: "[Acme Corp] 3 workflow runs failed in the last hour",
		},
		{
			name: "expiring token alert",
			sendFunc: func(s *SMTPService) error {
				return s.SendExpiringTokenEmail(context.Background(), []string{"admin@example.com"}, email.ExpiringTokenEmailData{
					TenantName:            "Acme Corp",
					Subject:               "API token 'production-api-key' expires soon",
					TokenName:             "production-api-key",
					ExpiresAtAbsoluteDate: "2026-02-01",
					ExpiresAtRelativeDate: "5 days",
					SettingsLink:          "https://app.example.com/settings/tokens",
				})
			},
			wantRcpts:   []string{"admin@example.com"},
			wantSubject: "[Acme Corp] API token 'production-api-key' expires soon",
		},
		{
			name: "resource limit alert",
			sendFunc: func(s *SMTPService) error {
				return s.SendTenantResourceLimitAlert(context.Background(), []string{"admin@example.com"}, email.ResourceLimitAlertData{
					TenantName:   "Acme Corp",
					Subject:      "Workflow runs limit reached",
					Summary:      "Workflow runs at 90%",
					Resource:     "workflow runs",
					CurrentValue: 900,
					LimitValue:   1000,
					Percentage:   90,
					Link:         "https://app.example.com/billing",
				})
			},
			wantRcpts:   []string{"admin@example.com", testSupportEmail},
			wantSubject: "[Acme Corp] Workflow runs limit reached",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured, service := setupTestServer(t)

			err := tt.sendFunc(service)
			require.NoError(t, err)

			require.Len(t, captured.Messages, 1)
			require.Len(t, captured.Froms, 1)

			require.ElementsMatch(t, tt.wantRcpts, captured.Rcpts)
			require.Contains(t, captured.Froms[0], testFromEmail)

			msg := captured.Messages[0]
			require.Contains(t, msg.Header.Get("From"), testFromEmail)
			require.Contains(t, msg.Header.Get("From"), testFromName)
			require.Contains(t, msg.Header.Get("Content-Type"), "text/html")
			require.Empty(t, msg.Header.Get("Bcc"), "Bcc header should not be visible")
			require.Equal(t, tt.wantSubject, msg.Header.Get("Subject"))
		})
	}
}

func TestSMTPBasicAuth(t *testing.T) {
	captured, service := setupTestServer(t)

	err := service.SendTenantInviteEmail(context.Background(), "user@example.com", email.TenantInviteEmailData{
		TenantName:       "Test",
		InviteSenderName: "Bob",
		ActionURL:        "https://example.com",
	})
	require.NoError(t, err)

	require.Len(t, captured.Usernames, 1)
	require.Equal(t, testUsername, captured.Usernames[0])

	require.Len(t, captured.Passwords, 1)
	require.Equal(t, testPassword, captured.Passwords[0])
}

func setupTestServer(t *testing.T) (*SMTPCapture, *SMTPService) {
	t.Helper()

	port, captured, cancel, err := StartMockSMTPServer()
	require.NoError(t, err)
	t.Cleanup(cancel)

	serverAddr := fmt.Sprintf("127.0.0.1:%d", port)
	service, err := NewSMTPService(
		serverAddr,
		testUsername,
		testPassword,
		testFromEmail,
		testFromName,
		testSupportEmail,
	)
	require.NoError(t, err)

	return captured, service
}
