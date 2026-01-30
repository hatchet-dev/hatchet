//go:build !e2e && !load && !rampup && !integration

package smtp

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"mime/quotedprintable"
	"strings"
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
		name string

		expectedTemplate   *template.Template
		expectedSubject    string
		expectedRecipients []string

		templateData interface{}

		sendFunc func(s *SMTPService, ctx context.Context, d interface{}) error
	}{
		{
			name:               "tenant invite",
			expectedTemplate:   templateRegistry[email.UserInviteTemplate].bodyTmpl,
			expectedSubject:    "Alice invited you to join Acme Corp on Hatchet",
			expectedRecipients: []string{"user@example.com"},
			templateData: email.TenantInviteEmailData{
				TenantName:       "Acme Corp",
				InviteSenderName: "Alice",
				ActionURL:        "https://app.example.com/join/abc123",
			},
			sendFunc: func(s *SMTPService, ctx context.Context, d interface{}) error {
				return s.SendTenantInviteEmail(
					ctx, "user@example.com", d.(email.TenantInviteEmailData),
				)
			},
		},
		{
			name:               "workflow failed alert",
			expectedTemplate:   templateRegistry[email.WorkflowRunsFailedTemplate].bodyTmpl,
			expectedSubject:    "[Acme Corp] 3 workflow runs failed in the last hour",
			expectedRecipients: []string{"admin1@example.com", "admin2@example.com"},
			templateData: email.WorkflowRunsFailedEmailData{
				TenantName:   "Acme Corp",
				Subject:      "3 workflow runs failed in the last hour",
				Summary:      "3 workflow runs failed",
				SettingsLink: "https://app.example.com/settings",
			},
			sendFunc: func(s *SMTPService, ctx context.Context, d interface{}) error {
				return s.SendWorkflowRunFailedAlerts(
					ctx, []string{"admin1@example.com", "admin2@example.com"}, d.(email.WorkflowRunsFailedEmailData),
				)
			},
		},
		{
			name:               "expiring token alert",
			expectedTemplate:   templateRegistry[email.TokenAlertExpiringTemplate].bodyTmpl,
			expectedSubject:    "[Acme Corp] API token 'production-api-key' expires soon",
			expectedRecipients: []string{"admin@example.com"},
			templateData: email.ExpiringTokenEmailData{
				TenantName:            "Acme Corp",
				Subject:               "API token 'production-api-key' expires soon",
				TokenName:             "production-api-key",
				ExpiresAtAbsoluteDate: "2026-02-01",
				ExpiresAtRelativeDate: "5 days",
				SettingsLink:          "https://app.example.com/settings/tokens",
			},
			sendFunc: func(s *SMTPService, ctx context.Context, d interface{}) error {
				return s.SendExpiringTokenEmail(
					ctx, []string{"admin@example.com"}, d.(email.ExpiringTokenEmailData),
				)
			},
		},
		{
			name:             "resource limit alert",
			expectedTemplate: templateRegistry[email.ResourceLimitAlertTemplate].bodyTmpl,
			expectedSubject:  "[Acme Corp] Workflow runs limit reached",
			// note: a support email is required for ResourceLimitAlert types
			expectedRecipients: []string{"admin@example.com", testSupportEmail},
			templateData: email.ResourceLimitAlertData{
				TenantName:   "Acme Corp",
				Subject:      "Workflow runs limit reached",
				Summary:      "Workflow runs at 90%",
				Resource:     "workflow runs",
				CurrentValue: 900,
				LimitValue:   1000,
				Percentage:   90,
				Link:         "https://app.example.com/billing",
			},
			sendFunc: func(s *SMTPService, ctx context.Context, d interface{}) error {
				return s.SendTenantResourceLimitAlert(
					ctx, []string{"admin@example.com"}, d.(email.ResourceLimitAlertData),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured, service := setupTestServer(t)

			err := tt.sendFunc(service, t.Context(), tt.templateData)
			require.NoError(t, err)

			require.Len(t, captured.Messages, 1, "Expected exactly one email sent")
			require.Len(t, captured.Froms, 1)

			msg := captured.Messages[0]

			require.ElementsMatch(t, tt.expectedRecipients, captured.Rcpts, "Recipients list mismatch")

			require.Equal(t, tt.expectedSubject, msg.Header.Get("Subject"), "Subject mismatch")
			require.Contains(t, captured.Froms[0], testFromEmail)
			require.Contains(t, msg.Header.Get("From"), testFromEmail)
			require.Contains(t, msg.Header.Get("From"), testFromName)
			require.Contains(t, msg.Header.Get("Content-Type"), "text/html")
			require.Empty(t, msg.Header.Get("Bcc"), "Bcc header should not be visible")

			// HACK: validate that the template renders correctly by directly accessing
			// from the template registry, reversing the encoding from the actual response,
			// and checking that the recieved message body equals the expected.
			var buf bytes.Buffer
			require.NoError(t, tt.expectedTemplate.Execute(&buf, tt.templateData), "failed to render template")
			expectedBody := strings.ReplaceAll(buf.String(), "\n", "\r\n")

			qpReader := quotedprintable.NewReader(msg.Body)
			bodyBytes, err := io.ReadAll(qpReader)
			require.NoError(t, err)
			actualBody := string(bodyBytes)

			require.Equal(t, expectedBody, actualBody, "Rendered email body mismatch")
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
