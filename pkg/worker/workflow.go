package worker

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

type triggerConverter interface {
	ToWorkflowTriggers(*types.WorkflowTriggers)
}

type Cron string

func (c Cron) ToWorkflowTriggers(wt *types.WorkflowTriggers) {
	if wt.Cron == nil {
		wt.Cron = []string{}
	}

	wt.Cron = append(wt.Cron, string(c))
}

type Crons []string

func (c Crons) ToWorkflowTriggers(wt *types.WorkflowTriggers) {
	if wt.Cron == nil {
		wt.Cron = []string{}
	}

	wt.Cron = append(wt.Cron, c...)
}

type Event string

func (e Event) ToWorkflowTriggers(wt *types.WorkflowTriggers) {
	if wt.Events == nil {
		wt.Events = []string{}
	}

	wt.Events = append(wt.Events, string(e))
}

type Events []string

func (e Events) ToWorkflowTriggers(wt *types.WorkflowTriggers) {
	if wt.Events == nil {
		wt.Events = []string{}
	}

	wt.Events = append(wt.Events, e...)
}

type workflowConverter interface {
	ToWorkflow(svcName string) types.Workflow
	ToActionMap(svcName string) map[string]any
}

type Workflow struct {
	Jobs []WorkflowJob
}

type workflowFn struct {
	Function any

	name string
}

func Fn(f any) workflowFn {
	return workflowFn{
		Function: f,
	}
}

func (w workflowFn) SetName(name string) workflowFn {
	w.name = name
	return w
}

func (w workflowFn) ToWorkflow(svcName string) types.Workflow {
	jobName := w.name

	if jobName == "" {
		jobName = getFnName(w.Function)
	}
	workflowJob := &WorkflowJob{
		Name: jobName,
		Steps: []WorkflowStep{
			{
				Function: w.Function,
				ID:       w.name,
			},
		},
	}

	return workflowJob.ToWorkflow(svcName)
}

func (w workflowFn) ToActionMap(svcName string) map[string]any {
	step := &WorkflowStep{
		Function: w.Function,
		ID:       w.name,
	}

	return map[string]any{
		step.GetActionId(svcName, 0): w.Function,
	}
}

type WorkflowJob struct {
	// The name of the job
	Name string

	Description string

	Timeout string

	// The steps that are run in the job
	Steps []WorkflowStep
}

func (j *WorkflowJob) ToWorkflow(svcName string) types.Workflow {
	apiJob, err := j.ToWorkflowJob(svcName)

	if err != nil {
		panic(err)
	}

	jobs := map[string]types.WorkflowJob{
		j.Name: *apiJob,
	}

	return types.Workflow{
		Name: j.Name,
		Jobs: jobs,
	}
}

func (j *WorkflowJob) ToWorkflowJob(svcName string) (*types.WorkflowJob, error) {
	apiJob := &types.WorkflowJob{
		Description: j.Description,
		Timeout:     j.Timeout,
		Steps:       []types.WorkflowStep{},
	}

	var prevStep *step

	for i, step := range j.Steps {
		newStep, err := step.ToWorkflowStep(prevStep, svcName, i)

		if err != nil {
			return nil, err
		}

		apiJob.Steps = append(apiJob.Steps, newStep.APIStep)

		prevStep = newStep
	}

	return apiJob, nil
}

func (j *WorkflowJob) ToActionMap(svcName string) map[string]any {
	res := map[string]any{}

	for i, step := range j.Steps {
		actionId := step.GetActionId(svcName, i)

		res[actionId] = step.Function
	}

	return res
}

type WorkflowStep struct {
	// The step timeout
	Timeout string

	// The executed function
	Function any

	// The step id. If not set, one will be generated from the function name
	ID string
}

type step struct {
	Id string

	// non-ctx input is not optional
	NonCtxInput reflect.Type

	// non-err output is optional
	NonErrOutput *reflect.Type

	APIStep types.WorkflowStep
}

func (s *WorkflowStep) ToWorkflowStep(prevStep *step, svcName string, index int) (*step, error) {
	fnType := reflect.TypeOf(s.Function)

	res := &step{}

	res.Id = s.GetStepId(index)

	res.APIStep = types.WorkflowStep{
		Name:     res.Id,
		ID:       s.GetStepId(index),
		Timeout:  s.Timeout,
		ActionID: s.GetActionId(svcName, index),
	}

	inputs, err := decodeFnArgTypes(fnType)

	if err != nil {
		return nil, err
	}

	res.NonCtxInput = inputs[1]

	outputs, err := decodeFnReturnTypes(fnType)

	if err != nil {
		return nil, err
	}

	if len(outputs) > 1 {
		res.NonErrOutput = &outputs[0]
	}

	// if the previous step's first output matches the last input of this step, then the data
	// is passed through
	if prevStep != nil && prevStep.NonErrOutput != nil {
		if inputs[1] == *prevStep.NonErrOutput {
			res.APIStep.With = map[string]interface{}{
				"object": "{{ index .steps \"" + prevStep.Id + "\" \"json\" }}",
			}
		}
	} else {
		res.APIStep.With = map[string]interface{}{
			"object": "{{ .input.json }}",
		}
	}

	return res, nil
}

func (s *WorkflowStep) GetStepId(index int) string {
	if s.ID != "" {
		return s.ID
	}

	stepId := getFnName(s.Function)

	// this can happen if the function is anonymous
	if stepId == "" {
		stepId = fmt.Sprintf("step%d", index)
	}

	return stepId
}

func (s *WorkflowStep) GetActionId(svcName string, index int) string {
	stepId := s.GetStepId(index)

	return fmt.Sprintf("%s:%s", svcName, stepId)
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
