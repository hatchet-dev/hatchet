package task

import "github.com/hatchet-dev/hatchet/pkg/worker"

type CreateOpts struct {
	Name string
	Fn   func(ctx worker.HatchetContext) error
}

type TaskDeclaration struct {
	Name string
	Fn   func(ctx worker.HatchetContext) error
}

func NewTaskDeclaration(opts CreateOpts) *TaskDeclaration {
	return &TaskDeclaration{
		Name: opts.Name,
		Fn:   opts.Fn,
	}
}
