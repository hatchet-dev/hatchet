package postmark

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/internal/integrations/email"
)

type PostmarkClient struct {
	serverKey    string
	fromEmail    string
	fromName     string
	supportEmail string

	httpClient *http.Client
}

// NewPostmarkClient creates a new client which sends emails through Postmark
func NewPostmarkClient(serverKey, fromEmail, fromName, supportEmail string) *PostmarkClient {
	httpClient := &http.Client{
		Timeout: time.Minute,
	}

	return &PostmarkClient{serverKey, fromEmail, fromName, supportEmail, httpClient}
}

const (
	postmarkAPIURL             = "https://api.postmarkapp.com"
	userInviteTemplate         = "user-invitation"
	workflowRunsFailedTemplate = "workflow-runs-failed"
	tokenAlertExpiringTemplate = "token-expiring" // nolint: gosec
	resourceLimitAlertTemplate = "resource-limit-alert"
)

type sendEmailFromTemplateRequest struct {
	From          string      `json:"From"`
	To            string      `json:"To,omitempty"`
	Bcc           string      `json:"Bcc,omitempty"`
	TemplateAlias string      `json:"TemplateAlias"`
	TemplateModel interface{} `json:"TemplateModel"`
}

type VerifyEmailData struct {
	ActionURL string `json:"link" mapstructure:"action_url"`
}

func (s *PostmarkClient) IsValid() bool {
	return true
}

func (c *PostmarkClient) SendTenantInviteEmail(ctx context.Context, to string, data email.TenantInviteEmailData) error {
	return c.sendTemplateEmail(ctx, to, userInviteTemplate, data, false)
}

func (c *PostmarkClient) SendWorkflowRunFailedAlerts(ctx context.Context, emails []string, data email.WorkflowRunsFailedEmailData) error {
	return c.sendTemplateEmailBCC(ctx, strings.Join(emails, ","), workflowRunsFailedTemplate, data, false)
}

func (c *PostmarkClient) SendExpiringTokenEmail(ctx context.Context, emails []string, data email.ExpiringTokenEmailData) error {
	return c.sendTemplateEmailBCC(ctx, strings.Join(emails, ","), tokenAlertExpiringTemplate, data, false)
}

func (c *PostmarkClient) SendTenantResourceLimitAlert(ctx context.Context, emails []string, data email.ResourceLimitAlertData) error {
	return c.sendTemplateEmailBCC(ctx, strings.Join(emails, ","), resourceLimitAlertTemplate, data, true)
}

func (c *PostmarkClient) sendTemplateEmail(ctx context.Context, to, templateAlias string, templateModelData interface{}, bccSupport bool) error {
	var bcc string

	if bccSupport {
		bcc = c.supportEmail
	}

	return c.sendRequest(ctx, "/email/withTemplate", "POST", &sendEmailFromTemplateRequest{
		From:          fmt.Sprintf("%s <%s>", c.fromName, c.fromEmail),
		To:            to,
		Bcc:           bcc,
		TemplateAlias: templateAlias,
		TemplateModel: templateModelData,
	})
}

func (c *PostmarkClient) sendTemplateEmailBCC(ctx context.Context, bcc, templateAlias string, templateModelData interface{}, bccSupport bool) error {

	if bccSupport {
		bcc = fmt.Sprintf("%s,%s", bcc, c.supportEmail)
	}

	return c.sendRequest(ctx, "/email/withTemplate", "POST", &sendEmailFromTemplateRequest{
		From:          fmt.Sprintf("%s <%s>", c.fromName, c.fromEmail),
		Bcc:           bcc,
		TemplateAlias: templateAlias,
		TemplateModel: templateModelData,
	})
}

func (c *PostmarkClient) sendRequest(ctx context.Context, path, method string, data interface{}) error {
	reqURL, err := url.Parse(postmarkAPIURL)
	if err != nil {
		return nil
	}

	reqURL.Path = path

	strData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		reqURL.String(),
		strings.NewReader(string(strData)),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Postmark-Server-Token", c.serverKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		resBytes, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("request failed with status code %d, but could not read body (%s)\n", res.StatusCode, err.Error())
		}

		return fmt.Errorf("request failed with status code %d: %s\n", res.StatusCode, string(resBytes))
	}

	return nil
}
