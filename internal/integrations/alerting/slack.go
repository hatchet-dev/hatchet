package alerting

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/slack-go/slack"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting/alerttypes"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// postSlackWebhook decrypts the webhook URL and posts the message. The decrypted
// URL carries a bearer secret in its path, so any error returned from here names
// the webhook by its row ID and never includes the URL.
func (t *TenantAlertManager) postSlackWebhook(slackWebhook *sqlcv1.SlackAppWebhook, msg *slack.WebhookMessage) error {
	whDecrypted, err := t.enc.Decrypt(slackWebhook.WebhookURL, "incoming_webhook_url")

	if err != nil {
		return fmt.Errorf("slack webhook %s: could not decrypt webhook url: %w", slackWebhook.ID, err)
	}

	webhookURL := string(whDecrypted)

	if err := slack.PostWebhookCustomHTTP(webhookURL, slackWebhookClient, msg); err != nil {
		return redactSlackWebhookError(slackWebhook.ID, webhookURL, err)
	}

	return nil
}

// slackWebhookClient never follows redirects, so a 3xx from the webhook host
// surfaces as slack-go's status error instead of a redirect chain that could
// repeat the secret-bearing path in a new request or error.
var slackWebhookClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

const redactedWebhookPlaceholder = "[redacted]"

// redactSlackWebhookError rebuilds a PostWebhook error without the webhook URL.
// Go's *url.Error embeds the full request URL in its message, so it is replaced
// by its operation and underlying cause. The resulting text is then scrubbed of
// every form of the URL (full, host plus path, path, raw path) because causes
// such as a malformed Location header quote the path on its own. The returned
// error is built from the scrubbed string only, with no wrapped cause, so no
// other formatting path can reintroduce the URL.
func redactSlackWebhookError(id uuid.UUID, webhookURL string, err error) error {
	var uerr *url.Error

	if errors.As(err, &uerr) && uerr.Err != nil {
		err = fmt.Errorf("%s: %w", uerr.Op, uerr.Err)
	}

	msg := err.Error()

	for _, secret := range webhookSecretForms(webhookURL) {
		msg = strings.ReplaceAll(msg, secret, redactedWebhookPlaceholder)
	}

	return fmt.Errorf("slack webhook %s: %s", id, msg)
}

// webhookSecretForms lists the substrings of the webhook URL that carry its
// secret path, longest first so partial replacements never leave a tail behind.
func webhookSecretForms(webhookURL string) []string {
	forms := []string{webhookURL}

	u, err := url.Parse(webhookURL)

	if err != nil {
		return forms
	}

	for _, path := range []string{u.RawPath, u.Path} {
		if path == "" || path == "/" {
			continue
		}

		forms = append(forms, u.Host+path, path)
	}

	return forms
}

func (t *TenantAlertManager) sendSlackWorkflowRunAlert(slackWebhook *sqlcv1.SlackAppWebhook, numFailed int, failedRuns []alerttypes.WorkflowRunFailedItem) error {
	headerText, blocks := t.getSlackWorkflowRunTextAndBlocks(numFailed, failedRuns)

	return t.postSlackWebhook(slackWebhook, &slack.WebhookMessage{
		Text:   headerText,
		Blocks: blocks,
	})
}

func (t *TenantAlertManager) getSlackWorkflowRunTextAndBlocks(numFailed int, failedRuns []alerttypes.WorkflowRunFailedItem) (string, *slack.Blocks) {
	res := make([]slack.Block, 0)

	headerText := fmt.Sprintf("%d Hatchet workflows failed:", numFailed)

	if numFailed <= 1 {
		headerText = fmt.Sprintf("%d Hatchet workflow failed:", numFailed)
	}

	res = append(res, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, headerText, false, false),
		nil,
		nil,
	))

	for i, workflowRun := range failedRuns {
		// don't add more than 5 failed workflow runs
		if i >= 5 {
			break
		}

		buttonAccessory := slack.NewAccessory(
			slack.NewButtonBlockElement(
				"View",
				workflowRun.WorkflowRunReadableId,
				slack.NewTextBlockObject(slack.PlainTextType, "View", true, false),
			),
		)

		buttonAccessory.ButtonElement.URL = workflowRun.Link
		buttonAccessory.ButtonElement.ActionID = "button-action"

		res = append(res, slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf(":warning: %s failed %s", workflowRun.WorkflowName, workflowRun.RelativeDate),
				false,
				false,
			),
			nil,
			buttonAccessory,
		))
	}

	return headerText, &slack.Blocks{
		BlockSet: res,
	}
}

func (t *TenantAlertManager) sendSlackExpiringTokenAlert(slackWebhook *sqlcv1.SlackAppWebhook, payload *alerttypes.ExpiringTokenItem) error {
	headerText, blocks := t.getSlackExpiringTokenTextAndBlocks(payload)

	return t.postSlackWebhook(slackWebhook, &slack.WebhookMessage{
		Text:   headerText,
		Blocks: blocks,
	})
}

func (t *TenantAlertManager) getSlackExpiringTokenTextAndBlocks(payload *alerttypes.ExpiringTokenItem) (string, *slack.Blocks) {
	res := make([]slack.Block, 0)

	headerText := fmt.Sprintf(":lock: Heads up! Your `%s` hatchet token will expire `%s`", payload.TokenName, payload.ExpiresAtRelativeDate)

	res = append(res, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, headerText, false, false),
		nil,
		nil,
	))

	buttonAccessory := slack.NewAccessory(
		slack.NewButtonBlockElement(
			"Manage Tokens",
			payload.TokenName,
			slack.NewTextBlockObject(slack.PlainTextType, "Manage Tokens", true, false),
		),
	)

	buttonAccessory.ButtonElement.URL = payload.Link
	buttonAccessory.ButtonElement.ActionID = "button-action"

	res = append(res, slack.NewSectionBlock(
		slack.NewTextBlockObject(
			slack.MarkdownType,
			"Once expired, any workers or clients using this token will no longer be able to connect to Hatchet.",
			false,
			false,
		),
		nil,
		buttonAccessory,
	))

	return headerText, &slack.Blocks{
		BlockSet: res,
	}
}

func (t *TenantAlertManager) sendSlackTenantResourceLimitAlert(slackWebhook *sqlcv1.SlackAppWebhook, payload *alerttypes.ResourceLimitAlert) error {
	headerText, blocks := t.getSlackTenantResourceLimitTextAndBlocks(payload)

	return t.postSlackWebhook(slackWebhook, &slack.WebhookMessage{
		Text:   headerText,
		Blocks: blocks,
	})
}

func (t *TenantAlertManager) getSlackTenantResourceLimitTextAndBlocks(payload *alerttypes.ResourceLimitAlert) (string, *slack.Blocks) {
	res := make([]slack.Block, 0)

	var headerText string

	if payload.AlertType == string(sqlcv1.TenantResourceLimitAlertTypeAlarm) {
		headerText = fmt.Sprintf(":warning: Limit Alarm! `%s` resource is at %d%% of its limit (%d/%d)", payload.Resource, payload.Percentage, payload.CurrentValue, payload.LimitValue)
	}

	if payload.AlertType == string(sqlcv1.TenantResourceLimitAlertTypeExhausted) {
		headerText = fmt.Sprintf(":no_entry: Limit Exhausted! `%s` resource is at 100%% of its limit (%d/%d)", payload.Resource, payload.CurrentValue, payload.LimitValue)
	}

	res = append(res, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, headerText, false, false),
		nil,
		nil,
	))

	buttonAccessory := slack.NewAccessory(
		slack.NewButtonBlockElement(
			"View Limits",
			payload.Resource,
			slack.NewTextBlockObject(slack.PlainTextType, "View Limits", true, false),
		),
	)

	buttonAccessory.ButtonElement.URL = payload.Link
	buttonAccessory.ButtonElement.ActionID = "button-action"

	if payload.AlertType == string(sqlcv1.TenantResourceLimitAlertTypeExhausted) {
		res = append(res, slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				"Any further resource usage will be denied until the limit is increased.",
				false,
				false,
			),
			nil,
			nil,
		))

	}

	res = append(res, slack.NewSectionBlock(
		slack.NewTextBlockObject(
			slack.MarkdownType,
			"Please review your resource usage and consider upgrading your plan.",
			false,
			false,
		),
		nil,
		buttonAccessory,
	))

	return headerText, &slack.Blocks{
		BlockSet: res,
	}
}
