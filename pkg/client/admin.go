package client

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type ChildWorkflowOpts struct {
	ParentId           string
	ParentStepRunId    string
	ChildIndex         int
	ChildKey           *string
	DesiredWorkerId    *string
	AdditionalMetadata *map[string]string
}

type WorkflowRun struct {
	Name    string
	Input   interface{}
	Options []RunOptFunc
}

type AdminClient interface {
	PutWorkflow(workflow *types.Workflow, opts ...PutOptFunc) error
	ScheduleWorkflow(workflowName string, opts ...ScheduleOptFunc) error

	// RunWorkflow triggers a workflow run and returns the run id
	RunWorkflow(workflowName string, input interface{}, opts ...RunOptFunc) (*Workflow, error)

	BulkRunWorkflow(workflows []*WorkflowRun) ([]string, error)

	RunChildWorkflow(workflowName string, input interface{}, opts *ChildWorkflowOpts) (string, error)
	RunChildWorkflows(workflows []*RunChildWorkflowsOpts) ([]string, error)

	PutRateLimit(key string, opts *types.RateLimitOpts) error
}

type DedupeViolationErr struct {
	details string
}

func (d *DedupeViolationErr) Error() string {
	return fmt.Sprintf("DedupeViolationErr: %s", d.details)
}

type adminClientImpl struct {
	client admincontracts.WorkflowServiceClient

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader

	namespace string

	subscriber SubscribeClient

	sharedMeta map[string]string
}

func newAdmin(conn *grpc.ClientConn, opts *sharedClientOpts, subscriber SubscribeClient) AdminClient {
	return &adminClientImpl{
		client:     admincontracts.NewWorkflowServiceClient(conn),
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

type scheduleOpts struct {
	schedules []time.Time
	input     any
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

	_, err = a.client.ScheduleWorkflow(a.ctx.newContext(context.Background()), &admincontracts.ScheduleWorkflowRequest{
		Name:      workflowName,
		Schedules: pbSchedules,
		Input:     string(inputBytes),
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

func (a *adminClientImpl) RunWorkflow(workflowName string, input interface{}, options ...RunOptFunc) (*Workflow, error) {
	inputBytes, err := json.Marshal(input)

	if err != nil {
		return nil, fmt.Errorf("could not marshal input: %w", err)
	}

	if a.namespace != "" && !strings.HasPrefix(workflowName, a.namespace) {
		workflowName = fmt.Sprintf("%s%s", a.namespace, workflowName)
	}

	request := admincontracts.TriggerWorkflowRequest{
		Name:  workflowName,
		Input: string(inputBytes),
	}

	for _, optionFunc := range options {
		err = optionFunc(&request)
		if err != nil {
			return nil, fmt.Errorf("could not apply run option: %w", err)
		}
	}

	res, err := a.client.TriggerWorkflow(a.ctx.newContext(context.Background()), &request)

	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, &DedupeViolationErr{
				details: fmt.Sprintf("could not trigger workflow: %s", err.Error()),
			}
		}

		return nil, fmt.Errorf("could not trigger workflow: %w", err)
	}

	listener, err := a.subscriber.SubscribeToWorkflowRunEvents(context.Background())

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

		triggerWorkflowRequests[i] = &admincontracts.TriggerWorkflowRequest{
			Name:  workflow.Name,
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

	if a.namespace != "" && !strings.HasPrefix(workflowName, a.namespace) {
		workflowName = fmt.Sprintf("%s%s", a.namespace, workflowName)
	}

	childIndex := int32(opts.ChildIndex) // nolint: gosec

	metadataBytes, err := a.getAdditionalMetaBytes(opts.AdditionalMetadata)

	if err != nil {
		return "", fmt.Errorf("could not get additional metadata: %w", err)
	}

	metadata := string(metadataBytes)

	res, err := a.client.TriggerWorkflow(a.ctx.newContext(context.Background()), &admincontracts.TriggerWorkflowRequest{
		Name:               workflowName,
		Input:              string(inputBytes),
		ParentId:           &opts.ParentId,
		ParentStepRunId:    &opts.ParentStepRunId,
		ChildIndex:         &childIndex,
		ChildKey:           opts.ChildKey,
		DesiredWorkerId:    opts.DesiredWorkerId,
		AdditionalMetadata: &metadata,
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

		var workflowName = workflow.WorkflowName

		if a.namespace != "" && !strings.HasPrefix(workflow.WorkflowName, a.namespace) {
			workflowName = fmt.Sprintf("%s%s", a.namespace, workflow.WorkflowName)
		}

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
			Name:               workflowName,
			Input:              string(inputBytes),
			ParentId:           &workflow.Opts.ParentId,
			ParentStepRunId:    &workflow.Opts.ParentStepRunId,
			ChildIndex:         &childIndex,
			ChildKey:           workflow.Opts.ChildKey,
			DesiredWorkerId:    workflow.Opts.DesiredWorkerId,
			AdditionalMetadata: &metadata,
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

		// TODO: should be a pointer because users might want to set maxRuns temporarily for disabling
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
