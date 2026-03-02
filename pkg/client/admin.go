// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type ChildWorkflowOpts struct {
	ParentId           string
	ParentTaskRunId    string
	ChildIndex         int
	ChildKey           *string
	DesiredWorkerId    *string
	AdditionalMetadata *map[string]string
	Priority           *int32
}

type WorkflowRun struct {
	Name    string
	Input   interface{}
	Options []RunOptFunc
}

func taskStatusFromProto(s v1contracts.RunStatus) rest.V1TaskStatus {
	switch s {
	case v1contracts.RunStatus_COMPLETED:
		return rest.V1TaskStatusCOMPLETED
	case v1contracts.RunStatus_CANCELLED:
		return rest.V1TaskStatusCANCELLED
	case v1contracts.RunStatus_FAILED:
		return rest.V1TaskStatusFAILED
	case v1contracts.RunStatus_RUNNING:
		return rest.V1TaskStatusRUNNING
	default:
		return rest.V1TaskStatusQUEUED
	}
}

type TaskRunDetails struct {
	ExternalId uuid.UUID
	ReadableId string
	Status     rest.V1TaskStatus
	Output     json.RawMessage
	Error      *string
}

type RunDetails struct {
	ExternalId         uuid.UUID
	Status             rest.V1TaskStatus
	Input              json.RawMessage
	AdditionalMetadata json.RawMessage
	TaskRuns           map[string]*TaskRunDetails
	Done               bool
}

type AdminClient interface {
	// Deprecated: PutWorkflow is part of the legacy v0 workflow definition system.
	// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
	PutWorkflow(workflow *types.Workflow, opts ...PutOptFunc) error
	// Deprecated: PutWorkflowV1 is an internal method used by the new Go SDK.
	// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
	PutWorkflowV1(workflow *v1contracts.CreateWorkflowVersionRequest, opts ...PutOptFunc) error

	ScheduleWorkflow(workflowName string, opts ...ScheduleOptFunc) error

	// RunWorkflow triggers a workflow run and returns the run id
	RunWorkflow(workflowName string, input interface{}, opts ...RunOptFunc) (*Workflow, error)

	BulkRunWorkflow(workflows []*WorkflowRun) ([]string, error)

	RunChildWorkflow(workflowName string, input interface{}, opts *ChildWorkflowOpts) (string, error)
	RunChildWorkflows(workflows []*RunChildWorkflowsOpts) ([]string, error)

	PutRateLimit(key string, opts *types.RateLimitOpts) error

	GetRunDetails(ctx context.Context, externalId uuid.UUID) (*RunDetails, error)
}

type DedupeViolationErr struct {
	details string
}

func (d *DedupeViolationErr) Error() string {
	return fmt.Sprintf("DedupeViolationErr: %s", d.details)
}

type adminClientImpl struct {
	client   admincontracts.WorkflowServiceClient
	v1Client v1contracts.AdminServiceClient

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader

	namespace string

	subscriber SubscribeClient

	sharedMeta map[string]string

	listenerMu sync.Mutex
	listener   *WorkflowRunsListener
}

func newAdmin(conn *grpc.ClientConn, opts *sharedClientOpts, subscriber SubscribeClient) AdminClient {
	return &adminClientImpl{
		client:     admincontracts.NewWorkflowServiceClient(conn),
		v1Client:   v1contracts.NewAdminServiceClient(conn),
		l:          opts.l,
		v:          opts.v,
		ctx:        opts.ctxLoader,
		namespace:  opts.namespace,
		subscriber: subscriber,
		sharedMeta: opts.sharedMeta,
	}
}

type putOpts struct {
}

type PutOptFunc func(*putOpts)

func defaultPutOpts() *putOpts {
	return &putOpts{}
}

// Deprecated: PutWorkflow is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (a *adminClientImpl) PutWorkflow(workflow *types.Workflow, fs ...PutOptFunc) error {
	opts := defaultPutOpts()

	for _, f := range fs {
		f(opts)
	}

	req, err := a.getPutRequest(workflow)

	if err != nil {
		return fmt.Errorf("could not get put opts: %w", err)
	}

	_, err = a.client.PutWorkflow(a.ctx.newContext(context.Background()), req)

	if err != nil {
		return fmt.Errorf("could not create workflow %s: %w", workflow.Name, err)
	}

	return nil
}

// Deprecated: PutWorkflowV1 is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (a *adminClientImpl) PutWorkflowV1(workflow *v1contracts.CreateWorkflowVersionRequest, fs ...PutOptFunc) error {
	opts := defaultPutOpts()

	for _, f := range fs {
		f(opts)
	}

	_, err := a.v1Client.PutWorkflow(a.ctx.newContext(context.Background()), workflow)

	if err != nil {
		return fmt.Errorf("could not create workflow %s: %w", workflow.Name, err)
	}

	return nil
}

type scheduleOpts struct {
	schedules []time.Time
	input     any
	priority  *int32
}

type ScheduleOptFunc func(*scheduleOpts)

func WithInput(input any) ScheduleOptFunc {
	return func(opts *scheduleOpts) {
		opts.input = input
	}
}

func WithSchedules(schedules ...time.Time) ScheduleOptFunc {
	return func(opts *scheduleOpts) {
		opts.schedules = schedules
	}
}

func defaultScheduleOpts() *scheduleOpts {
	return &scheduleOpts{}
}

func (a *adminClientImpl) ScheduleWorkflow(workflowName string, fs ...ScheduleOptFunc) error {
	opts := defaultScheduleOpts()

	for _, f := range fs {
		f(opts)
	}

	if len(opts.schedules) == 0 {
		return fmt.Errorf("ScheduleWorkflow error: schedules are required")
	}

	pbSchedules := make([]*timestamppb.Timestamp, len(opts.schedules))

	for i, scheduled := range opts.schedules {
		pbSchedules[i] = timestamppb.New(scheduled)
	}

	inputBytes, err := json.Marshal(opts.input)

	if err != nil {
		return err
	}

	workflowName = client.ApplyNamespace(workflowName, &a.namespace)

	_, err = a.client.ScheduleWorkflow(a.ctx.newContext(context.Background()), &admincontracts.ScheduleWorkflowRequest{
		Name:      workflowName,
		Schedules: pbSchedules,
		Input:     string(inputBytes),
		Priority:  opts.priority,
	})

	if err != nil {
		return fmt.Errorf("could not schedule workflow: %w", err)
	}

	return nil
}

type RunOptFunc func(*admincontracts.TriggerWorkflowRequest) error

func WithRunMetadata(metadata interface{}) RunOptFunc {
	return func(r *admincontracts.TriggerWorkflowRequest) error {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return err
		}

		metadataString := string(metadataBytes)

		r.AdditionalMetadata = &metadataString

		return nil
	}
}

func WithPriority(priority int32) RunOptFunc {
	return func(r *admincontracts.TriggerWorkflowRequest) error {
		r.Priority = &priority

		return nil
	}
}

// func WithSticky(sticky bool) RunOptFunc {
// 	return func(r *admincontracts.TriggerWorkflowRequest) error {
// 		r.Sticky = &sticky

// 		return nil
// 	}
// }

func (a *adminClientImpl) RunWorkflow(workflowName string, input interface{}, options ...RunOptFunc) (*Workflow, error) {
	inputBytes, err := json.Marshal(input)

	if err != nil {
		return nil, fmt.Errorf("could not marshal input: %w", err)
	}

	workflowName = client.ApplyNamespace(workflowName, &a.namespace)

	request := &admincontracts.TriggerWorkflowRequest{
		Name:  workflowName,
		Input: string(inputBytes),
	}

	for _, optionFunc := range options {
		err = optionFunc(request)
		if err != nil {
			return nil, fmt.Errorf("could not apply run option: %w", err)
		}
	}

	res, err := a.client.TriggerWorkflow(a.ctx.newContext(context.Background()), request)

	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, &DedupeViolationErr{
				details: fmt.Sprintf("could not trigger workflow: %s", err.Error()),
			}
		}

		return nil, fmt.Errorf("could not trigger workflow: %w", err)
	}

	listener, err := a.saveOrLoadListener()

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to workflow run events: %w", err)
	}

	return &Workflow{
		workflowRunId: res.WorkflowRunId,
		listener:      listener,
	}, nil
}

func (a *adminClientImpl) BulkRunWorkflow(workflows []*WorkflowRun) ([]string, error) {

	triggerWorkflowRequests := make([]*admincontracts.TriggerWorkflowRequest, len(workflows))

	for i, workflow := range workflows {
		inputBytes, err := json.Marshal(workflow.Input)
		if err != nil {
			return nil, fmt.Errorf("could not marshal input: %w", err)
		}

		workflowName := client.ApplyNamespace(workflow.Name, &a.namespace)
		triggerWorkflowRequests[i] = &admincontracts.TriggerWorkflowRequest{
			Name:  workflowName,
			Input: string(inputBytes),
		}

		for _, optionFunc := range workflow.Options {
			err = optionFunc(triggerWorkflowRequests[i])
			if err != nil {
				return nil, fmt.Errorf("could not apply run option: %w", err)
			}
		}
	}

	r := admincontracts.BulkTriggerWorkflowRequest{
		Workflows: triggerWorkflowRequests,
	}

	res, err := a.client.BulkTriggerWorkflow(a.ctx.newContext(context.Background()), &r)

	if err != nil {
		return nil, fmt.Errorf("could not bulk trigger workflows: %w", err)
	}

	return res.WorkflowRunIds, nil

}

func (a *adminClientImpl) RunChildWorkflow(workflowName string, input interface{}, opts *ChildWorkflowOpts) (string, error) {
	inputBytes, err := json.Marshal(input)

	if err != nil {
		return "", fmt.Errorf("could not marshal input: %w", err)
	}

	workflowName = client.ApplyNamespace(workflowName, &a.namespace)

	childIndex := int32(opts.ChildIndex) // nolint: gosec

	metadataBytes, err := a.getAdditionalMetaBytes(opts.AdditionalMetadata)

	if err != nil {
		return "", fmt.Errorf("could not get additional metadata: %w", err)
	}

	metadata := string(metadataBytes)

	res, err := a.client.TriggerWorkflow(a.ctx.newContext(context.Background()), &admincontracts.TriggerWorkflowRequest{
		Name:                    workflowName,
		Input:                   string(inputBytes),
		ParentId:                &opts.ParentId,
		ParentTaskRunExternalId: &opts.ParentTaskRunId,
		ChildIndex:              &childIndex,
		ChildKey:                opts.ChildKey,
		DesiredWorkerId:         opts.DesiredWorkerId,
		AdditionalMetadata:      &metadata,
		Priority:                opts.Priority,
	})

	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return "", &DedupeViolationErr{
				details: fmt.Sprintf("could not trigger child workflow: %s", err.Error()),
			}
		}

		return "", fmt.Errorf("could not trigger child workflow: %w", err)
	}

	return res.WorkflowRunId, nil

}

type RunChildWorkflowsOpts struct {
	WorkflowName string
	Input        interface{}
	Opts         *ChildWorkflowOpts
}

func (a *adminClientImpl) RunChildWorkflows(workflows []*RunChildWorkflowsOpts) ([]string, error) {

	triggerWorkflowRequests := make([]*admincontracts.TriggerWorkflowRequest, len(workflows))

	for i, workflow := range workflows {
		if workflow.Opts == nil {
			workflow.Opts = &ChildWorkflowOpts{}
		}

		inputBytes, err := json.Marshal(workflow.Input)

		if err != nil {
			return nil, fmt.Errorf("could not marshal input: %w", err)
		}

		workflowName := client.ApplyNamespace(workflow.WorkflowName, &a.namespace)

		if workflow.Opts.ChildIndex < math.MinInt32 || workflow.Opts.ChildIndex > math.MaxInt32 {
			return nil, fmt.Errorf("child index out of range")
		}
		childIndex := int32(workflow.Opts.ChildIndex) // nolint: gosec

		metadataBytes, err := a.getAdditionalMetaBytes(workflow.Opts.AdditionalMetadata)

		if err != nil {
			return nil, fmt.Errorf("could not get additional metadata: %w", err)
		}

		metadata := string(metadataBytes)

		triggerWorkflowRequests[i] = &admincontracts.TriggerWorkflowRequest{
			Name:                    workflowName,
			Input:                   string(inputBytes),
			ParentId:                &workflow.Opts.ParentId,
			ParentTaskRunExternalId: &workflow.Opts.ParentTaskRunId,
			ChildIndex:              &childIndex,
			ChildKey:                workflow.Opts.ChildKey,
			DesiredWorkerId:         workflow.Opts.DesiredWorkerId,
			AdditionalMetadata:      &metadata,
			Priority:                workflow.Opts.Priority,
		}

	}

	res, err := a.client.BulkTriggerWorkflow(a.ctx.newContext(context.Background()), &admincontracts.BulkTriggerWorkflowRequest{
		Workflows: triggerWorkflowRequests,
	})

	if err != nil {

		return nil, fmt.Errorf("could not trigger child workflow: %w", err)
	}

	return res.WorkflowRunIds, nil
}

func (a *adminClientImpl) PutRateLimit(key string, opts *types.RateLimitOpts) error {
	if err := a.v.Validate(opts); err != nil {
		return fmt.Errorf("could not validate rate limit opts: %w", err)
	}

	putParams := &admincontracts.PutRateLimitRequest{
		Key:   key,
		Limit: int32(opts.Max), // nolint: gosec
	}

	switch opts.Duration {
	case types.Second:
		putParams.Duration = admincontracts.RateLimitDuration_SECOND
	case types.Minute:
		putParams.Duration = admincontracts.RateLimitDuration_MINUTE
	case types.Hour:
		putParams.Duration = admincontracts.RateLimitDuration_HOUR
	default:
		putParams.Duration = admincontracts.RateLimitDuration_SECOND
	}

	_, err := a.client.PutRateLimit(a.ctx.newContext(context.Background()), putParams)

	if err != nil {
		return fmt.Errorf("could not upsert rate limit: %w", err)
	}

	return nil
}

func (a *adminClientImpl) GetRunDetails(ctx context.Context, externalId uuid.UUID) (*RunDetails, error) {
	resp, err := a.v1Client.GetRunDetails(a.ctx.newContext(ctx), &v1contracts.GetRunDetailsRequest{
		ExternalId: externalId.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not get run details: %w", err)
	}

	taskRuns := make(map[string]*TaskRunDetails, len(resp.GetTaskRuns()))
	for readableId, detail := range resp.GetTaskRuns() {
		var errStr *string
		if detail.Error != nil {
			errStr = detail.Error
		}

		externalId, err := uuid.Parse(detail.ExternalId)

		if err != nil {
			return nil, fmt.Errorf("could not parse task run external id: %w", err)
		}

		taskRuns[readableId] = &TaskRunDetails{
			ExternalId: externalId,
			ReadableId: detail.GetReadableId(),
			Status:     taskStatusFromProto(detail.GetStatus()),
			Output:     detail.GetOutput(),
			Error:      errStr,
		}
	}

	return &RunDetails{
		ExternalId:         externalId,
		Status:             taskStatusFromProto(resp.GetStatus()),
		Input:              resp.GetInput(),
		AdditionalMetadata: resp.GetAdditionalMetadata(),
		TaskRuns:           taskRuns,
		Done:               resp.GetDone(),
	}, nil
}

func (a *adminClientImpl) getPutRequest(workflow *types.Workflow) (*admincontracts.PutWorkflowRequest, error) {
	opts := &admincontracts.CreateWorkflowVersionOpts{
		Name:          workflow.Name,
		Version:       workflow.Version,
		Description:   workflow.Description,
		EventTriggers: workflow.Triggers.Events,
		CronTriggers:  workflow.Triggers.Cron,
	}

	if workflow.StickyStrategy != nil {
		s := admincontracts.StickyStrategy(*workflow.StickyStrategy)
		opts.Sticky = &s
	}

	if workflow.Concurrency != nil {
		opts.Concurrency = &admincontracts.WorkflowConcurrencyOpts{
			Action:     workflow.Concurrency.ActionID,
			Expression: workflow.Concurrency.Expression,
		}

		var limitStrat admincontracts.ConcurrencyLimitStrategy

		switch workflow.Concurrency.LimitStrategy {
		case types.CancelInProgress:
			limitStrat = admincontracts.ConcurrencyLimitStrategy_CANCEL_IN_PROGRESS
		case types.GroupRoundRobin:
			limitStrat = admincontracts.ConcurrencyLimitStrategy_GROUP_ROUND_ROBIN
		case types.CancelNewest:
			limitStrat = admincontracts.ConcurrencyLimitStrategy_CANCEL_NEWEST
		default:
			limitStrat = admincontracts.ConcurrencyLimitStrategy_CANCEL_IN_PROGRESS
		}

		opts.Concurrency.LimitStrategy = &limitStrat

		if workflow.Concurrency.MaxRuns != 0 {
			maxRuns := workflow.Concurrency.MaxRuns
			opts.Concurrency.MaxRuns = &maxRuns
		}
	}

	if workflow.ScheduleTimeout != "" {
		opts.ScheduleTimeout = &workflow.ScheduleTimeout
	}

	if workflow.OnFailureJob != nil {
		onFailureJob, err := a.getJobOpts("on-failure", workflow.OnFailureJob)

		if err != nil {
			return nil, fmt.Errorf("could not get on failure job opts: %w", err)
		}

		opts.OnFailureJob = onFailureJob
	}

	jobOpts := make([]*admincontracts.CreateWorkflowJobOpts, 0)

	for jobName, job := range workflow.Jobs {
		jobCp := job

		res, err := a.getJobOpts(jobName, &jobCp)

		if err != nil {
			return nil, fmt.Errorf("could not get job opts: %w", err)
		}

		jobOpts = append(jobOpts, res)
	}

	opts.ScheduledTriggers = make([]*timestamppb.Timestamp, len(workflow.Triggers.Schedules))

	for i, scheduled := range workflow.Triggers.Schedules {
		opts.ScheduledTriggers[i] = timestamppb.New(scheduled)
	}

	opts.Jobs = jobOpts

	return &admincontracts.PutWorkflowRequest{
		Opts: opts,
	}, nil
}

func (a *adminClientImpl) getJobOpts(jobName string, job *types.WorkflowJob) (*admincontracts.CreateWorkflowJobOpts, error) {
	jobOpt := &admincontracts.CreateWorkflowJobOpts{
		Name:        jobName,
		Description: job.Description,
	}

	stepOpts := make([]*admincontracts.CreateWorkflowStepOpts, len(job.Steps))

	for i, step := range job.Steps {
		inputBytes, err := json.Marshal(step.With)

		if err != nil {
			return nil, fmt.Errorf("could not marshal step inputs: %w", err)
		}

		userDataBytes, err := json.Marshal(step.UserData)

		if err != nil {
			return nil, fmt.Errorf("could not marshal step user data: %w", err)
		}

		stepOpt := &admincontracts.CreateWorkflowStepOpts{
			ReadableId:        step.ID,
			Action:            step.ActionID,
			Timeout:           step.Timeout,
			Inputs:            string(inputBytes),
			UserData:          string(userDataBytes),
			Parents:           step.Parents,
			Retries:           int32(step.Retries), // nolint: gosec
			BackoffFactor:     step.RetryBackoffFactor,
			BackoffMaxSeconds: step.RetryMaxBackoffSeconds,
		}

		for _, rateLimit := range step.RateLimits {
			opt := &admincontracts.CreateStepRateLimit{
				Key:             rateLimit.Key,
				KeyExpr:         rateLimit.KeyExpr,
				UnitsExpr:       rateLimit.UnitsExpr,
				LimitValuesExpr: rateLimit.LimitValueExpr,
			}

			if rateLimit.Units != nil {
				units := int32(*rateLimit.Units) // nolint: gosec
				opt.Units = &units
			}

			if rateLimit.Duration != nil {
				var duration admincontracts.RateLimitDuration

				switch *rateLimit.Duration {
				case types.Year:
					duration = admincontracts.RateLimitDuration_YEAR
				case types.Month:
					duration = admincontracts.RateLimitDuration_MONTH
				case types.Day:
					duration = admincontracts.RateLimitDuration_DAY
				case types.Hour:
					duration = admincontracts.RateLimitDuration_HOUR
				case types.Minute:
					duration = admincontracts.RateLimitDuration_MINUTE
				case types.Second:
					duration = admincontracts.RateLimitDuration_SECOND
				default:
					duration = admincontracts.RateLimitDuration_MINUTE
				}

				opt.Duration = &duration
			}

			stepOpt.RateLimits = append(stepOpt.RateLimits, opt)
		}

		if step.DesiredLabels != nil {
			stepOpt.WorkerLabels = make(map[string]*admincontracts.DesiredWorkerLabels, len(step.DesiredLabels))
			for key, desiredLabel := range step.DesiredLabels {
				stepOpt.WorkerLabels[key] = &admincontracts.DesiredWorkerLabels{
					Required: &desiredLabel.Required,
					Weight:   &desiredLabel.Weight,
				}

				switch value := desiredLabel.Value.(type) {
				case string:
					strValue := value
					stepOpt.WorkerLabels[key].StrValue = &strValue
				case int:
					intValue := int32(value) // nolint: gosec
					stepOpt.WorkerLabels[key].IntValue = &intValue
				case int32:
					stepOpt.WorkerLabels[key].IntValue = &value
				case int64:
					intValue := int32(value) // nolint: gosec
					stepOpt.WorkerLabels[key].IntValue = &intValue
				default:
					// For any other type, convert to string
					strValue := fmt.Sprintf("%v", value)
					stepOpt.WorkerLabels[key].StrValue = &strValue
				}

				if desiredLabel.Comparator != nil {
					c := admincontracts.WorkerLabelComparator(*desiredLabel.Comparator)
					stepOpt.WorkerLabels[key].Comparator = &c
				}
			}
		}

		stepOpts[i] = stepOpt
	}

	jobOpt.Steps = stepOpts

	return jobOpt, nil
}

func (a *adminClientImpl) getAdditionalMetaBytes(opt *map[string]string) ([]byte, error) {
	additionalMeta := make(map[string]string)

	for key, value := range a.sharedMeta {
		additionalMeta[key] = value
	}

	if opt != nil {
		for key, value := range *opt {
			additionalMeta[key] = value
		}
	}

	metadataBytes, err := json.Marshal(additionalMeta)

	if err != nil {
		return nil, fmt.Errorf("could not marshal additional metadata: %w", err)
	}

	return metadataBytes, nil
}

func (h *adminClientImpl) saveOrLoadListener() (*WorkflowRunsListener, error) {
	h.listenerMu.Lock()
	defer h.listenerMu.Unlock()

	if h.listener != nil {
		return h.listener, nil
	}

	listener, err := h.subscriber.SubscribeToWorkflowRunEvents(context.Background())

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to workflow run events: %w", err)
	}

	h.listener = listener

	return listener, nil
}
