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

	"github.com/hatchet-dev/hatchet/pkg/integrations/email"
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
	postmarkAPIURL      = "https://api.postmarkapp.com"
	postmarkEmailPath   = "/email/withTemplate"
	postmarkEmailMethod = "POST"
)

func (c *PostmarkClient) IsValid() bool {
	return true
}

func (c *PostmarkClient) SendTenantInviteEmail(ctx context.Context, to string, data email.TenantInviteEmailData) error {
	return c.SendTemplateEmail(ctx, to, email.UserInviteTemplate, data, false)
}

func (c *PostmarkClient) SendWorkflowRunFailedAlerts(ctx context.Context, emails []string, data email.WorkflowRunsFailedEmailData) error {
	return c.SendTemplateEmailBCC(ctx, strings.Join(emails, ","), email.WorkflowRunsFailedTemplate, data, false)
}

func (c *PostmarkClient) SendExpiringTokenEmail(ctx context.Context, emails []string, data email.ExpiringTokenEmailData) error {
	return c.SendTemplateEmailBCC(ctx, strings.Join(emails, ","), email.TokenAlertExpiringTemplate, data, false)
}

func (c *PostmarkClient) SendTenantResourceLimitAlert(ctx context.Context, emails []string, data email.ResourceLimitAlertData) error {
	return c.SendTemplateEmailBCC(ctx, strings.Join(emails, ","), email.ResourceLimitAlertTemplate, data, true)
}

func (c *PostmarkClient) SendTemplateEmail(ctx context.Context, to, templateAlias string, templateModelData interface{}, bccSupport bool) error {
	var bcc string

	if bccSupport {
		bcc = c.supportEmail
	}

	return c.sendPostmarkEmail(ctx, &email.SendEmailFromTemplateRequest{
		From:          fmt.Sprintf("%s <%s>", c.fromName, c.fromEmail),
		To:            to,
		Bcc:           bcc,
		TemplateAlias: templateAlias,
		TemplateModel: templateModelData,
	})
}

func (c *PostmarkClient) SendTemplateEmailBCC(ctx context.Context, bcc, templateAlias string, templateModelData interface{}, bccSupport bool) error {

	if bccSupport {
		bcc = fmt.Sprintf("%s,%s", bcc, c.supportEmail)
	}

	return c.sendPostmarkEmail(ctx, &email.SendEmailFromTemplateRequest{
		From:          fmt.Sprintf("%s <%s>", c.fromName, c.fromEmail),
		Bcc:           bcc,
		TemplateAlias: templateAlias,
		TemplateModel: templateModelData,
	})
}

func (c *PostmarkClient) sendPostmarkEmail(ctx context.Context, data *email.SendEmailFromTemplateRequest) error {
	reqURL, err := url.Parse(postmarkAPIURL)
	if err != nil {
		return nil
	}

	reqURL.Path = postmarkEmailPath

	strData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		postmarkEmailMethod,
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
			return fmt.Errorf("request failed with status code %d, but could not read body (%s)", res.StatusCode, err.Error())
		}

		return fmt.Errorf("request failed with status code %d: %s", res.StatusCode, string(resBytes))
	}

	return nil
}
