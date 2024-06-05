package alerting

import (
	"fmt"

	"github.com/slack-go/slack"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting/alerttypes"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

func (t *TenantAlertManager) sendSlackWorkflowRunAlert(slackWebhook *dbsqlc.SlackAppWebhook, numFailed int, failedRuns []alerttypes.WorkflowRunFailedItem) error {
	headerText, blocks := t.getSlackWorkflowRunTextAndBlocks(numFailed, failedRuns)

	// decrypt the webhook url
	whDecrypted, err := t.enc.Decrypt(slackWebhook.WebhookURL, "incoming_webhook_url")

	if err != nil {
		return err
	}

	err = slack.PostWebhook(string(whDecrypted), &slack.WebhookMessage{
		Text:   headerText,
		Blocks: blocks,
	})

	if err != nil {
		return err
	}

	return nil
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

func (t *TenantAlertManager) sendSlackExpiringTokenAlert(slackWebhook *dbsqlc.SlackAppWebhook, payload *alerttypes.ExpiringTokenItem) error {
	headerText, blocks := t.getSlackExpiringTokenTextAndBlocks(payload)

	// decrypt the webhook url
	whDecrypted, err := t.enc.Decrypt(slackWebhook.WebhookURL, "incoming_webhook_url")

	if err != nil {
		return err
	}

	err = slack.PostWebhook(string(whDecrypted), &slack.WebhookMessage{
		Text:   headerText,
		Blocks: blocks,
	})

	if err != nil {
		return err
	}

	return nil
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
