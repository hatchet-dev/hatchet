package worker

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/compute"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

type triggerConverter interface {
	ToWorkflowTriggers(wt *types.WorkflowTriggers, namespace string)
}

type cron string

func Cron(c string) cron {
	return cron(c)
}

func (c cron) ToWorkflowTriggers(wt *types.WorkflowTriggers, namespace string) {
	if wt.Cron == nil {
		wt.Cron = []string{}
	}

	wt.Cron = append(wt.Cron, string(c))
}

type cronArr []string

func Crons(c ...string) cronArr {
	return cronArr(c)
}

func (c cronArr) ToWorkflowTriggers(wt *types.WorkflowTriggers, namespace string) {
	if wt.Cron == nil {
		wt.Cron = []string{}
	}

	wt.Cron = append(wt.Cron, c...)
}

type noTrigger struct{}

func NoTrigger() noTrigger {
	return noTrigger{}
}

func (n noTrigger) ToWorkflowTriggers(wt *types.WorkflowTriggers, namespace string) {
	// do nothing
}

type scheduled []time.Time

func At(t ...time.Time) scheduled {
	return t
}

func (s scheduled) ToWorkflowTriggers(wt *types.WorkflowTriggers, namespace string) {
	if wt.Schedules == nil {
		wt.Schedules = []time.Time{}
	}

	wt.Schedules = append(wt.Schedules, s...)
}

func (w *Worker) Call(action string) *WorkflowStep {
	registeredAction, exists := w.actions[action]

	if !exists {
		panic(fmt.Sprintf("action %s does not exist", action))
	}

	parsedAction, err := types.ParseActionID(action)

	if err != nil {
		panic(err)
	}

	return &WorkflowStep{
		Function: registeredAction.MethodFn(),
		Name:     parsedAction.Verb,
	}
}

type event string

func Event(e string) event {
	return event(e)
}

func (e event) ToWorkflowTriggers(wt *types.WorkflowTriggers, namespace string) {
	if wt.Events == nil {
		wt.Events = []string{}
	}

	wt.Events = append(wt.Events, string(e))

	// Prepend the namespace to each event
	for i, event := range wt.Events {
		wt.Events[i] = namespace + event
	}
}

type eventsArr []string

func Events(events ...string) eventsArr {
	return events
}

func (e eventsArr) ToWorkflowTriggers(wt *types.WorkflowTriggers, namespace string) {
	if wt.Events == nil {
		wt.Events = []string{}
	}

	wt.Events = append(wt.Events, e...)

	// Prepend the namespace to each event
	for i, event := range wt.Events {
		wt.Events[i] = namespace + event
	}
}

type workflowConverter interface {
	ToWorkflow(svcName string, namespace string) types.Workflow
	ToActionMap(svcName string) ActionMap
	ToWorkflowTrigger() triggerConverter
}

type Workflow struct {
	Jobs []WorkflowJob
}

type GetWorkflowConcurrencyGroupFn func(ctx HatchetContext) (string, error)

type WorkflowJob struct {
	// The name of the job
	Name string

	Description string

	On triggerConverter

	Concurrency *WorkflowConcurrency

	// The steps that are run in the job
	Steps []*WorkflowStep

	OnFailure *WorkflowJob

	ScheduleTimeout string

	StickyStrategy *types.StickyStrategy
}

type WorkflowConcurrency struct {
	fn            GetWorkflowConcurrencyGroupFn
	expr          *string
	maxRuns       *int32
	limitStrategy *types.WorkflowConcurrencyLimitStrategy
}

func Expression(expr string) *WorkflowConcurrency {
	return &WorkflowConcurrency{
		expr: &expr,
	}
}

func Concurrency(fn GetWorkflowConcurrencyGroupFn) *WorkflowConcurrency {
	return &WorkflowConcurrency{
		fn: fn,
	}
}

func (c *WorkflowConcurrency) MaxRuns(maxRuns int32) *WorkflowConcurrency {
	c.maxRuns = &maxRuns
	return c
}

func (c *WorkflowConcurrency) LimitStrategy(limitStrategy types.WorkflowConcurrencyLimitStrategy) *WorkflowConcurrency {
	c.limitStrategy = &limitStrategy
	return c
}

func (j *WorkflowJob) ToWorkflow(svcName string, namespace string) types.Workflow {
	apiJob, err := j.ToWorkflowJob(svcName, namespace)

	if err != nil {
		panic(err)
	}

	var onFailureJob *types.WorkflowJob

	if j.OnFailure != nil {
		onFailureJob, err = j.OnFailure.ToWorkflowJob(svcName, namespace)

		if err != nil {
			panic(err)
		}
	}

	jobs := map[string]types.WorkflowJob{
		j.Name: *apiJob,
	}

	w := types.Workflow{
		Name:            namespace + j.Name,
		Jobs:            jobs,
		OnFailureJob:    onFailureJob,
		ScheduleTimeout: j.ScheduleTimeout,
	}

	if j.Concurrency != nil {
		w.Concurrency = &types.WorkflowConcurrency{}

		if j.Concurrency.fn != nil {
			actionId := "concurrency:" + getFnName(j.Concurrency.fn)
			w.Concurrency.ActionID = &actionId
		}

		if j.Concurrency.expr != nil {
			w.Concurrency.Expression = j.Concurrency.expr
		}

		if j.Concurrency.maxRuns != nil {
			w.Concurrency.MaxRuns = *j.Concurrency.maxRuns
		}

		if j.Concurrency.limitStrategy != nil {
			w.Concurrency.LimitStrategy = *j.Concurrency.limitStrategy
		}
	}

	if j.StickyStrategy != nil {
		w.StickyStrategy = j.StickyStrategy
	}

	return w
}

func (j *WorkflowJob) ToWorkflowJob(svcName string, namespace string) (*types.WorkflowJob, error) {
	apiJob := &types.WorkflowJob{
		Description: j.Description,
		Steps:       []types.WorkflowStep{},
	}

	for i := range j.Steps {

		newStep, err := j.Steps[i].ToWorkflowStep(svcName, i, namespace)

		if err != nil {
			return nil, err
		}

		apiJob.Steps = append(apiJob.Steps, newStep.APIStep)
	}

	return apiJob, nil
}

func (j *WorkflowJob) ToWorkflowTrigger() triggerConverter {
	return j.On
}

type ActionWithCompute struct {
	fn      any
	compute *compute.Compute
}

type ActionMap map[string]ActionWithCompute

func (j *WorkflowJob) ToActionMap(svcName string) ActionMap {
	res := ActionMap{}

	for i, step := range j.Steps {
		actionId := step.GetActionId(svcName, i)

		res[actionId] = ActionWithCompute{
			fn:      step.Function,
			compute: step.Compute,
		}
	}

	if j.Concurrency != nil && j.Concurrency.fn != nil {
		res["concurrency:"+getFnName(j.Concurrency.fn)] = ActionWithCompute{
			fn:      j.Concurrency.fn,
			compute: nil, // FIXME add compute to concurrency
		}
	}

	if j.OnFailure != nil {
		onFailureActionMap := j.OnFailure.ToActionMap(svcName)

		for k, v := range onFailureActionMap {
			res[k] = v
		}
	}

	return res
}

type WorkflowStep struct {
	// The step timeout
	Timeout string

	// The executed function
	Function any

	// The step id/name. If not set, one will be generated from the function name
	Name string

	// The ids of the parents
	Parents []string

	Retries int

	RetryBackoffFactor *float32

	RetryMaxBackoffSeconds *int32

	RateLimit []RateLimit

	DesiredLabels map[string]*types.DesiredWorkerLabel

	Compute *compute.Compute
}

type RateLimit struct {
	// Key is the rate limit key
	Key     string  `yaml:"key,omitempty"`
	KeyExpr *string `yaml:"keyExpr,omitempty"`

	// Units is the amount of units this step consumes
	Units          *int    `yaml:"units,omitempty"`
	UnitsExpr      *string `yaml:"unitsExpr,omitempty"`
	LimitValueExpr *string `yaml:"limitValueExpr,omitempty"`

	// Duration is the duration of the rate limit
	Duration *types.RateLimitDuration `yaml:"duration,omitempty"`
}

func Fn(f any) *WorkflowStep {
	return &WorkflowStep{
		Function:  f,
		Parents:   []string{},
		RateLimit: []RateLimit{},
	}
}

func (w *WorkflowStep) SetName(name string) *WorkflowStep {
	w.Name = name
	return w
}

func (w *WorkflowStep) SetCompute(compute *compute.Compute) *WorkflowStep {
	w.Compute = compute
	return w
}

func (w *WorkflowStep) SetDesiredLabels(labels map[string]*types.DesiredWorkerLabel) *WorkflowStep {
	w.DesiredLabels = labels
	return w
}

func (w *WorkflowStep) SetRateLimit(rateLimit RateLimit) *WorkflowStep {
	w.RateLimit = append(w.RateLimit, rateLimit)
	return w
}

func (w *WorkflowStep) SetTimeout(timeout string) *WorkflowStep {
	w.Timeout = timeout
	return w
}

func (w *WorkflowStep) SetRetries(retries int) *WorkflowStep {
	w.Retries = retries
	return w
}

func (w *WorkflowStep) SetRetryBackoffFactor(retryBackoffFactor float32) *WorkflowStep {
	w.RetryBackoffFactor = &retryBackoffFactor
	return w
}

func (w *WorkflowStep) SetRetryMaxBackoffSeconds(retryMaxBackoffSeconds int32) *WorkflowStep {
	w.RetryMaxBackoffSeconds = &retryMaxBackoffSeconds
	return w
}

func (w *WorkflowStep) AddParents(parents ...string) *WorkflowStep {
	w.Parents = append(w.Parents, parents...)
	return w
}

func (w *WorkflowStep) ToWorkflowTrigger() triggerConverter {
	return NoTrigger()
}

func (w *WorkflowStep) ToWorkflow(svcName string, namespace string) types.Workflow {
	jobName := w.Name

	if jobName == "" {
		jobName = getFnName(w.Function)
	}
	workflowJob := &WorkflowJob{
		Name: jobName,
		Steps: []*WorkflowStep{
			w,
		},
	}

	return workflowJob.ToWorkflow(svcName, namespace)
}

func (w *WorkflowStep) ToActionMap(svcName string) ActionMap {
	step := *w

	return ActionMap{
		step.GetActionId(svcName, 0): ActionWithCompute{
			fn:      w.Function,
			compute: w.Compute,
		},
	}
}

type Step struct {
	Id string

	// non-ctx input is not optional
	NonCtxInput reflect.Type

	// non-err output is optional
	NonErrOutput *reflect.Type

	APIStep types.WorkflowStep
}

func (w *WorkflowStep) ToWorkflowStep(svcName string, index int, namespace string) (*Step, error) {
	fnType := reflect.TypeOf(w.Function)

	res := &Step{}

	res.Id = w.GetStepId(index)

	res.APIStep = types.WorkflowStep{
		Name:                   res.Id,
		ID:                     w.GetStepId(index),
		Timeout:                w.Timeout,
		ActionID:               w.GetActionId(svcName, index),
		Parents:                []string{},
		Retries:                w.Retries,
		DesiredLabels:          w.DesiredLabels,
		RetryBackoffFactor:     w.RetryBackoffFactor,
		RetryMaxBackoffSeconds: w.RetryMaxBackoffSeconds,
	}

	for _, rateLimit := range w.RateLimit {
		res.APIStep.RateLimits = append(res.APIStep.RateLimits, types.RateLimit{
			Key:            rateLimit.Key,
			KeyExpr:        rateLimit.KeyExpr,
			Units:          rateLimit.Units,
			UnitsExpr:      rateLimit.UnitsExpr,
			LimitValueExpr: rateLimit.LimitValueExpr,
			Duration:       rateLimit.Duration,
		})
	}

	inputs, err := decodeFnArgTypes(fnType)

	if err != nil {
		return nil, err
	}

	if len(inputs) > 1 {
		res.NonCtxInput = inputs[1]
	}

	outputs, err := decodeFnReturnTypes(fnType)

	if err != nil {
		return nil, err
	}

	if len(outputs) > 1 {
		res.NonErrOutput = &outputs[0]
	}

	for _, parent := range w.Parents {
		if res.APIStep.With == nil {
			res.APIStep.With = map[string]interface{}{
				parent: "{{ index .steps \"" + parent + "\" \"json\" }}",
			}
		} else {
			res.APIStep.With[parent] = "{{ index .steps \"" + parent + "\" \"json\" }}"
		}

		res.APIStep.Parents = append(res.APIStep.Parents, parent)
	}

	if res.APIStep.With == nil {
		res.APIStep.With = map[string]interface{}{
			"object": "{{ .input.json }}",
		}
	}

	return res, nil
}

func (w *WorkflowStep) GetStepId(index int) string {
	if w.Name != "" {
		return w.Name
	}

	stepId := getFnName(w.Function)

	// this can happen if the function is anonymous
	if stepId == "" {
		stepId = fmt.Sprintf("step%d", index)
	}

	return stepId
}

func (w *WorkflowStep) GetActionId(svcName string, index int) string {
	stepId := w.GetStepId(index)

	return strings.ToLower(fmt.Sprintf("%s:%s", svcName, stepId))
}

func getFnName(fn any) string {
	fnInfo := runtime.FuncForPC(reflect.ValueOf(fn).Pointer())
	fnName := fnInfo.Name()

	// get after the last /
	if strings.LastIndex(fnName, "/") != -1 {
		fnName = fnName[strings.LastIndex(fnName, "/")+1:]
	}

	// get after the first .
	if firstDotIndex := strings.Index(fnName, "."); firstDotIndex != -1 {
		fnName = fnName[firstDotIndex+1:]
	}

	return strings.ReplaceAll(fnName, ".", "-")
}
