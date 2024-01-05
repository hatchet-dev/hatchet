package types

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

type Workflow struct {
	Name string `yaml:"name,omitempty"`

	Version string `yaml:"version,omitempty"`

	Description string `yaml:"description,omitempty"`

	Triggers WorkflowTriggers `yaml:"triggers"`

	Jobs map[string]WorkflowJob `yaml:"jobs"`
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

	Timeout string `yaml:"timeout,omitempty"`

	Steps []WorkflowStep `yaml:"steps"`
}

type WorkflowStep struct {
	Name     string                 `yaml:"name,omitempty"`
	ID       string                 `yaml:"id,omitempty"`
	ActionID string                 `yaml:"action"`
	Timeout  string                 `yaml:"timeout,omitempty"`
	With     map[string]interface{} `yaml:"with,omitempty"`
}

func ParseYAML(ctx context.Context, yamlBytes []byte) (Workflow, error) {
	var workflowFile Workflow

	if yamlBytes == nil {
		return workflowFile, fmt.Errorf("workflow yaml input is nil")
	}

	err := yaml.Unmarshal(yamlBytes, &workflowFile)
	if err != nil {
		return workflowFile, fmt.Errorf("error unmarshaling workflow yaml: %w", err)
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
