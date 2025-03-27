package types

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/worker/condition"

	"gopkg.in/yaml.v3"
)

type StickyStrategy int32

const (
	StickyStrategy_SOFT StickyStrategy = 0
	StickyStrategy_HARD StickyStrategy = 1
)

func StickyStrategyPtr(v StickyStrategy) *StickyStrategy {
	return &v
}

type Concurrency struct {
	Expression    string                            `yaml:"expression,omitempty"`
	MaxRuns       *int32                            `yaml:"maxRuns,omitempty"`
	LimitStrategy *WorkflowConcurrencyLimitStrategy `yaml:"limitStrategy,omitempty"`
}

type TaskConditions struct {
	// TODO: DOCS
	WaitFor  condition.Condition
	SkipIf   condition.Condition
	CancelIf condition.Condition
}

type Workflow struct {
	Name string `yaml:"name,omitempty"`

	ScheduleTimeout string `yaml:"scheduleTimeout,omitempty"`

	Concurrency *WorkflowConcurrency `yaml:"concurrency,omitempty"`

	Version string `yaml:"version,omitempty"`

	Description string `yaml:"description,omitempty"`

	Triggers WorkflowTriggers `yaml:"triggers"`

	Jobs map[string]WorkflowJob `yaml:"jobs"`

	OnFailureJob *WorkflowJob `yaml:"onFailureJob,omitempty"`

	StickyStrategy *StickyStrategy `yaml:"sticky,omitempty"`
}

type WorkflowConcurrencyLimitStrategy string

const (
	CancelInProgress WorkflowConcurrencyLimitStrategy = "CANCEL_IN_PROGRESS"
	CancelNewest     WorkflowConcurrencyLimitStrategy = "CANCEL_NEWEST"
	GroupRoundRobin  WorkflowConcurrencyLimitStrategy = "GROUP_ROUND_ROBIN"
	DropNewest       WorkflowConcurrencyLimitStrategy = "DROP_NEWEST"
	QueueNewest      WorkflowConcurrencyLimitStrategy = "QUEUE_NEWEST"
)

type WorkflowConcurrency struct {
	Expression *string `yaml:"expression,omitempty"`

	MaxRuns int32 `yaml:"maxRuns,omitempty"`

	LimitStrategy WorkflowConcurrencyLimitStrategy `yaml:"limitStrategy,omitempty"`

	ActionID *string `yaml:"action,omitempty"`
}

type WorkflowTriggers struct {
	Events    []string    `yaml:"events,omitempty"`
	Cron      []string    `yaml:"crons,omitempty"`
	Schedules []time.Time `yaml:"schedules,omitempty"`
}

type RandomScheduleOpt string

const (
	Random15Min  RandomScheduleOpt = "random_15_min"
	RandomHourly RandomScheduleOpt = "random_hourly"
	RandomDaily  RandomScheduleOpt = "random_daily"
)

type WorkflowOnCron struct {
	Schedule string `yaml:"schedule,omitempty"`
}

type WorkflowEvent struct {
	Name string `yaml:"name,omitempty"`
}

type WorkflowJob struct {
	Description string `yaml:"description,omitempty"`

	Steps []WorkflowStep `yaml:"steps"`
}

type WorkerLabelComparator int32

const (
	WorkerLabelComparator_EQUAL                 WorkerLabelComparator = 0
	WorkerLabelComparator_NOT_EQUAL             WorkerLabelComparator = 1
	WorkerLabelComparator_GREATER_THAN          WorkerLabelComparator = 2
	WorkerLabelComparator_GREATER_THAN_OR_EQUAL WorkerLabelComparator = 3
	WorkerLabelComparator_LESS_THAN             WorkerLabelComparator = 4
	WorkerLabelComparator_LESS_THAN_OR_EQUAL    WorkerLabelComparator = 5
)

func ComparatorPtr(v WorkerLabelComparator) *WorkerLabelComparator {
	return &v
}

type DesiredWorkerLabel struct {
	Value      any                    `yaml:"value,omitempty"`
	Required   bool                   `yaml:"required,omitempty"`
	Weight     int32                  `yaml:"weight,omitempty"`
	Comparator *WorkerLabelComparator `yaml:"comparator,omitempty"`
}

type WorkflowStep struct {
	Name     string `yaml:"name,omitempty"`
	ID       string `yaml:"id,omitempty"`
	ActionID string `yaml:"action"`
	Timeout  string `yaml:"timeout,omitempty"`

	// Deprecated: this field has no effect and will be removed in a future release.
	With map[string]interface{} `yaml:"with,omitempty"`

	UserData               map[string]interface{}         `yaml:"userData,omitempty"`
	Parents                []string                       `yaml:"parents,omitempty"`
	Retries                int                            `yaml:"retries"`
	RateLimits             []RateLimit                    `yaml:"rateLimits,omitempty"`
	DesiredLabels          map[string]*DesiredWorkerLabel `yaml:"desiredLabels,omitempty"`
	RetryBackoffFactor     *float32                       `yaml:"retryBackoffFactor,omitempty"`
	RetryMaxBackoffSeconds *int32                         `yaml:"retryMaxBackoffSeconds,omitempty"`
}

type RateLimit struct {
	Key            string             `yaml:"key,omitempty"`
	KeyExpr        *string            `yaml:"keyExpr,omitempty"`
	Units          *int               `yaml:"units,omitempty"`
	UnitsExpr      *string            `yaml:"unitsExpr,omitempty"`
	LimitValueExpr *string            `yaml:"limitValueExpr,omitempty"`
	Duration       *RateLimitDuration `yaml:"duration,omitempty"`
}

func ParseYAML(ctx context.Context, yamlBytes []byte) (Workflow, error) {
	var workflowFile Workflow

	if yamlBytes == nil {
		return workflowFile, fmt.Errorf("workflow yaml input is nil")
	}

	err := yaml.Unmarshal(yamlBytes, &workflowFile)
	if err != nil {
		return workflowFile, fmt.Errorf("error unmarshalling workflow yaml: %w", err)
	}

	return workflowFile, nil
}

func ToYAML(ctx context.Context, workflow *Workflow) ([]byte, error) {
	var b bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2)
	err := yamlEncoder.Encode(&workflow)

	if err != nil {
		return nil, fmt.Errorf("error marshaling workflow yaml: %w", err)
	}

	return b.Bytes(), nil
}
