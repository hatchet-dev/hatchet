package task

import (
	"fmt"
	"strings"
	"time"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
)

type NamedTaskImpl struct {
	Name string
}

type TaskBase interface {
	Dump(workflowName string, taskDefaults *create.TaskDefaults) *contracts.CreateTaskOpts
}

type TaskShared struct {
	Fn                     interface{}
	ExecutionTimeout       *time.Duration
	ScheduleTimeout        *time.Duration
	Retries                *int32
	RetryBackoffFactor     *float32
	RetryMaxBackoffSeconds *int32
	WorkerLabels           map[string]*types.DesiredWorkerLabel
	RateLimits             []*types.RateLimit
	Concurrency            []*types.Concurrency
}

// TaskDeclaration represents a standard (non-durable) task configuration that can be added to a workflow.
type TaskDeclaration[I any] struct {
	TaskShared
	TaskBase
	WaitFor  condition.Condition
	SkipIf   condition.Condition
	CancelIf condition.Condition
	Fn       interface{}
	NamedTaskImpl
	Name    string
	Parents []string
}

// DurableTaskDeclaration represents a durable task configuration that can be added to a workflow.
// Durable tasks can use the DurableHatchetContext for operations that persist across worker restarts.
type DurableTaskDeclaration[I any] struct {
	TaskShared
	TaskBase
	WaitFor  condition.Condition
	SkipIf   condition.Condition
	CancelIf condition.Condition
	Fn       interface{}
	NamedTaskImpl
	Name        string
	Parents     []string
	Concurrency []*types.Concurrency
}

// OnFailureTaskDeclaration represents a task that will be executed if
// any tasks in the workflow fail.
type OnFailureTaskDeclaration[I any] struct {
	TaskBase
	TaskShared

	// The function to execute when any tasks in the workflow have failed
	Fn interface{}
}

func makeContractTaskOpts(t *TaskShared, taskDefaults *create.TaskDefaults) *contracts.CreateTaskOpts {
	taskOpts := &contracts.CreateTaskOpts{
		RateLimits:  make([]*contracts.CreateTaskRateLimit, len(t.RateLimits)),
		Concurrency: make([]*contracts.Concurrency, len(t.Concurrency)),
	}

	for j, rateLimit := range t.RateLimits {
		rlContract := &contracts.CreateTaskRateLimit{
			Key:             rateLimit.Key,
			KeyExpr:         rateLimit.KeyExpr,
			UnitsExpr:       rateLimit.UnitsExpr,
			LimitValuesExpr: rateLimit.LimitValueExpr,
		}

		if rateLimit.LimitValueExpr == nil {
			negOne := "-1"
			rlContract.LimitValuesExpr = &negOne
		}

		if rateLimit.Units != nil {
			units32 := int32(*rateLimit.Units) // nolint: gosec
			rlContract.Units = &units32
		}

		if rateLimit.Duration != nil {
			var duration contracts.RateLimitDuration

			switch *rateLimit.Duration {
			case types.Year:
				duration = contracts.RateLimitDuration_YEAR
			case types.Month:
				duration = contracts.RateLimitDuration_MONTH
			case types.Day:
				duration = contracts.RateLimitDuration_DAY
			case types.Hour:
				duration = contracts.RateLimitDuration_HOUR
			case types.Minute:
				duration = contracts.RateLimitDuration_MINUTE
			case types.Second:
				duration = contracts.RateLimitDuration_SECOND
			default:
				duration = contracts.RateLimitDuration_MINUTE
			}

			rlContract.Duration = &duration
		}

		taskOpts.RateLimits[j] = rlContract
	}

	for j, concurrency := range t.Concurrency {
		concurrencyOpts := &contracts.Concurrency{
			Expression: concurrency.Expression,
			MaxRuns:    concurrency.MaxRuns,
		}

		if concurrency.LimitStrategy != nil {
			strategy := *concurrency.LimitStrategy
			strategyInt := contracts.ConcurrencyLimitStrategy_value[string(strategy)]
			strategyEnum := contracts.ConcurrencyLimitStrategy(strategyInt)
			concurrencyOpts.LimitStrategy = &strategyEnum
		}

		taskOpts.Concurrency[j] = concurrencyOpts
	}

	if t.ExecutionTimeout != nil {
		taskOpts.Timeout = durationToSeconds(*t.ExecutionTimeout)
	}

	if t.ScheduleTimeout != nil {
		scheduleTimeout := durationToSeconds(*t.ScheduleTimeout)
		taskOpts.ScheduleTimeout = &scheduleTimeout
	}

	// Only set Retries if it's not nil
	if t.Retries != nil {
		taskOpts.Retries = *t.Retries
	}

	if t.RetryBackoffFactor != nil {
		taskOpts.BackoffFactor = t.RetryBackoffFactor
	}

	if t.RetryMaxBackoffSeconds != nil {
		taskOpts.BackoffMaxSeconds = t.RetryMaxBackoffSeconds
	}

	if len(t.WorkerLabels) > 0 {
		taskOpts.WorkerLabels = make(map[string]*contracts.DesiredWorkerLabels)

		for key, value := range t.WorkerLabels {
			taskOpts.WorkerLabels[key] = &contracts.DesiredWorkerLabels{}

			switch v := value.Value.(type) {
			case string:
				strValue := v
				taskOpts.WorkerLabels[key].StrValue = &strValue
			case int:
				intValue := int32(v) // nolint: gosec
				taskOpts.WorkerLabels[key].IntValue = &intValue
			case int32:
				taskOpts.WorkerLabels[key].IntValue = &v
			case int64:
				intValue := int32(v) // nolint: gosec
				taskOpts.WorkerLabels[key].IntValue = &intValue
			default:
				// For any other type, convert to string
				strValue := fmt.Sprintf("%v", v)
				taskOpts.WorkerLabels[key].StrValue = &strValue
			}

			if value.Required {
				taskOpts.WorkerLabels[key].Required = &value.Required
			}

			if value.Weight != 0 {
				taskOpts.WorkerLabels[key].Weight = &value.Weight
			}

			if value.Comparator != nil {
				c := contracts.WorkerLabelComparator(*value.Comparator)
				taskOpts.WorkerLabels[key].Comparator = &c
			}
		}
	}

	// Apply workflow task defaults if they are not set
	if taskDefaults != nil {
		if t.Retries == nil && taskDefaults.Retries != 0 {
			taskOpts.Retries = taskDefaults.Retries
		}

		if t.ExecutionTimeout == nil && taskDefaults.ExecutionTimeout != 0 {
			taskOpts.Timeout = durationToSeconds(taskDefaults.ExecutionTimeout)
		}

		if t.ScheduleTimeout == nil && taskDefaults.ScheduleTimeout != 0 {
			scheduleTimeout := durationToSeconds(taskDefaults.ScheduleTimeout)
			taskOpts.ScheduleTimeout = &scheduleTimeout
		}

		if t.RetryBackoffFactor == nil && taskDefaults.RetryBackoffFactor != 0 {
			taskOpts.BackoffFactor = &taskDefaults.RetryBackoffFactor
		}

		if t.RetryMaxBackoffSeconds == nil && taskDefaults.RetryMaxBackoffSeconds != 0 {
			taskOpts.BackoffMaxSeconds = &taskDefaults.RetryMaxBackoffSeconds
		}
	}

	return taskOpts
}

// Dump converts the task declaration into a protobuf request.
func (t *TaskDeclaration[I]) Dump(workflowName string, taskDefaults *create.TaskDefaults) *contracts.CreateTaskOpts {
	base := makeContractTaskOpts(&t.TaskShared, taskDefaults)
	base.ReadableId = t.Name
	base.Action = getActionID(workflowName, t.Name)
	base.Parents = make([]string, len(t.Parents))
	copy(base.Parents, t.Parents)

	sleepConditions := make([]*contracts.SleepMatchCondition, 0)
	userEventConditions := make([]*contracts.UserEventMatchCondition, 0)
	parentOverrideConditions := make([]*contracts.ParentOverrideMatchCondition, 0)

	if t.WaitFor != nil {
		cs := t.WaitFor.ToPB(contracts.Action_QUEUE)

		sleepConditions = append(sleepConditions, cs.SleepConditions...)
		userEventConditions = append(userEventConditions, cs.UserEventConditions...)
		parentOverrideConditions = append(parentOverrideConditions, cs.ParentConditions...)
	}

	if t.SkipIf != nil {
		cs := t.SkipIf.ToPB(contracts.Action_SKIP)

		sleepConditions = append(sleepConditions, cs.SleepConditions...)
		userEventConditions = append(userEventConditions, cs.UserEventConditions...)
		parentOverrideConditions = append(parentOverrideConditions, cs.ParentConditions...)
	}

	if t.CancelIf != nil {
		cs := t.CancelIf.ToPB(contracts.Action_CANCEL)

		sleepConditions = append(sleepConditions, cs.SleepConditions...)
		userEventConditions = append(userEventConditions, cs.UserEventConditions...)
		parentOverrideConditions = append(parentOverrideConditions, cs.ParentConditions...)
	}

	base.Conditions = &contracts.TaskConditions{
		SleepConditions:          sleepConditions,
		UserEventConditions:      userEventConditions,
		ParentOverrideConditions: parentOverrideConditions,
	}

	return base
}

func durationToSeconds(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	return fmt.Sprintf("%ds", int(d.Seconds()))
}

func (t *DurableTaskDeclaration[I]) Dump(workflowName string, taskDefaults *create.TaskDefaults) *contracts.CreateTaskOpts {
	base := makeContractTaskOpts(&t.TaskShared, taskDefaults)
	base.ReadableId = t.Name
	base.Action = getActionID(workflowName, t.Name)
	base.Parents = make([]string, len(t.Parents))
	copy(base.Parents, t.Parents)
	return base
}

// Dump converts the on failure task declaration into a protobuf request.
func (t *OnFailureTaskDeclaration[I]) Dump(workflowName string, taskDefaults *create.TaskDefaults) *contracts.CreateTaskOpts {
	base := makeContractTaskOpts(&t.TaskShared, taskDefaults)

	base.ReadableId = "on-failure"
	base.Action = getActionID(workflowName, "on-failure")

	return base
}

// Implement GetName for TaskDeclaration
func (t *TaskDeclaration[I]) GetName() string {
	return t.Name
}

// Implement GetName for DurableTaskDeclaration
func (t *DurableTaskDeclaration[I]) GetName() string {
	return t.Name
}

// Implement GetName for NamedTask
func (t *NamedTaskImpl) GetName() string {
	return t.Name
}

func getActionID(workflowName, taskName string) string {
	return strings.ToLower(fmt.Sprintf("%s:%s", workflowName, taskName))
}
