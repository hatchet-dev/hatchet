package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

type webhookController struct {
	repo       repository.EngineRepository
	dispatcher *DispatcherImpl
}

type WebhookEvent struct {
	// the tenant id
	TenantId string `json:"tenantId,omitempty"`
	// the workflow run id (optional)
	WorkflowRunId string `json:"workflowRunId,omitempty"`
	// the get group key run id (optional)
	GetGroupKeyRunId string `json:"getGroupKeyRunId,omitempty"`
	// the job id
	JobId string `json:"jobId,omitempty"`
	// the job name
	JobName string `json:"jobName,omitempty"`
	// the job run id
	JobRunId string `json:"jobRunId,omitempty"`
	// the step id
	StepId string `json:"stepId,omitempty"`
	// the step run id
	StepRunId string `json:"stepRunId,omitempty"`
	// the action id
	ActionId string `json:"actionId,omitempty"`
	// the action payload
	ActionPayload string `json:"actionPayload,omitempty"`
	// the step name
	StepName string `json:"stepName,omitempty"`
}

func (w *webhookController) Send(ctx context.Context, tenantId string, action *contracts.AssignedAction) error {
	log.Printf("sending webhook for action %s", action.ActionId)

	// get webhook url from workflow version
	workflowRun, err := w.repo.WorkflowRun().GetWorkflowRunById(ctx, action.TenantId, action.WorkflowRunId)
	if err != nil {
		return fmt.Errorf("could not get workflow run: %w", err)
	}

	wfId := sqlchelpers.UUIDToStr(workflowRun.WorkflowVersion.WorkflowId)
	workflowVersion, err := w.repo.Workflow().GetLatestWorkflowVersion(ctx, action.TenantId, wfId)
	if err != nil {
		return fmt.Errorf("could not get workflow version: %w", err)
	}

	webhookUrl := workflowVersion.WorkflowVersion.Webhook.String

	if webhookUrl == "" {
		return fmt.Errorf("no webhook url found for workflow version %s", wfId)
	}

	log.Printf("sending webhook to %s", webhookUrl)

	// TODO!!!! notify
	// _, err = w.dispatcher.SendStepActionEvent(
	//	ctx,
	//	getActionEvent(assignedAction, client.ActionEventTypeStarted),
	//)
	// if err != nil {
	//	return fmt.Errorf("could not send action event: %w", err)
	//}

	body, err := json.Marshal(action)
	if err != nil {
		return err
	}
	// nolint:gosec
	resp, err := http.Post(webhookUrl, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO!!! handle error
		return fmt.Errorf("webhook failed with status code %d", resp.StatusCode)
	}

	log.Printf("setting step run status to completed")

	if err := w.dispatcher.handleStepRunCompletedImpl(ctx, tenantId, time.Now(), action.StepRunId, "{}"); err != nil {
		return fmt.Errorf("could not handle step run completed: %w", err)
	}

	return nil
}

// func getActionEvent(action *client.Action, eventType client.ActionEventType) *client.ActionEvent {
//	timestamp := time.Now().UTC()
//
//	return &client.ActionEvent{
//		Action:         action,
//		EventTimestamp: &timestamp,
//		EventType:      eventType,
//	}
//}
