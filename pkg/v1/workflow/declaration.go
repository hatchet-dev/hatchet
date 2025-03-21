package workflow

import (
	"reflect"

	"github.com/hatchet-dev/hatchet/pkg/client"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
)

type CreateOpts struct {
	Name string

	InputType  reflect.Type
	OutputType reflect.Type
}

type WorkflowBase interface {
	Name() string
}

type WorkflowDeclaration[I any, O any] interface {
	WorkflowBase
	Task(opts task.CreateOpts[I, O]) *task.TaskDeclaration[I, O]
	WithParents(parents ...*task.TaskDeclaration[I, O]) []*task.TaskDeclaration[I, O]

	// TODO bind runs and things
	Run(input I) (*O, error)
}

type workflowDeclarationImpl[I any, O any] struct {
	v0 *v0Client.Client

	name  string
	tasks []*task.TaskDeclaration[I, O]
}

func NewWorkflowDeclaration[I any, O any](opts CreateOpts, v0 *client.Client) WorkflowDeclaration[I, O] {
	return &workflowDeclarationImpl[I, O]{
		v0:    v0,
		name:  opts.Name,
		tasks: []*task.TaskDeclaration[I, O]{},
	}
}

func (w *workflowDeclarationImpl[I, O]) Task(opts task.CreateOpts[I, O]) *task.TaskDeclaration[I, O] {
	task := task.NewTaskDeclaration(opts)
	w.tasks = append(w.tasks, task)
	return task
}

func (w *workflowDeclarationImpl[I, O]) WithParents(parents ...*task.TaskDeclaration[I, O]) []*task.TaskDeclaration[I, O] {
	w.tasks = append(w.tasks, parents...)
	return w.tasks
}

func (w *workflowDeclarationImpl[I, O]) Run(input I) (*O, error) {
	return nil, nil
}

func (w *workflowDeclarationImpl[I, O]) Name() string {
	return w.name
}
