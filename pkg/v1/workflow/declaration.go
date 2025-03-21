package workflow

import (
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
)

type CreateOpts struct {
	Name string
}

type WorkflowDeclaration interface {
	Task(opts task.CreateOpts) *task.TaskDeclaration

	// TODO bind runs and things
}

type workflowDeclarationImpl struct {
	v0 *v0Client.Client

	name  string
	tasks []*task.TaskDeclaration
}

func NewWorkflowDeclaration(opts CreateOpts, v0 *v0Client.Client) WorkflowDeclaration {
	return &workflowDeclarationImpl{
		v0:    v0,
		name:  opts.Name,
		tasks: []*task.TaskDeclaration{},
	}
}

func (w *workflowDeclarationImpl) Task(opts task.CreateOpts) *task.TaskDeclaration {
	task := task.NewTaskDeclaration(opts)
	w.tasks = append(w.tasks, task)
	return task
}
