package task

import "github.com/hatchet-dev/hatchet/pkg/worker"

type CreateOpts[I any, O any] struct {
	Name    string
	Parents []*TaskDeclaration[I, O]
	Fn      func(input I, ctx worker.HatchetContext) (*O, error)
}

type TaskBase interface{}

type TaskDeclaration[I any, O any] struct {
	TaskBase
	Name string
	Fn   func(input I, ctx worker.HatchetContext) (*O, error)
}

func NewTaskDeclaration[I any, O any](opts CreateOpts[I, O]) *TaskDeclaration[I, O] {
	return &TaskDeclaration[I, O]{
		Name: opts.Name,
		Fn:   opts.Fn,
	}
}
