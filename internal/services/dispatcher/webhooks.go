package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/signature"
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

func (w *webhookController) Start(ctx context.Context, action *contracts.AssignedAction) error {
	log.Printf("sending webhook for action %s", action.ActionId)

	tenant, err := w.repo.Tenant().GetTenantByID(ctx, action.TenantId)
	if err != nil {
		return err
	}

	if !tenant.WebhookSecret.Valid {
		return fmt.Errorf("no webhook secret found for tenant %s", action.TenantId)
	}

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

	body, err := json.Marshal(action)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", webhookUrl, bytes.NewReader(body))
	if err != nil {
		return err
	}

	sig, err := signature.Sign(string(body), tenant.WebhookSecret.String)
	if err != nil {
		return err
	}
	req.Header.Set("X-Hatchet-Signature", sig)

	// nolint:gosec
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO!!! handle error
		return fmt.Errorf("webhook failed with status code %d", resp.StatusCode)
	}

	return nil
}
