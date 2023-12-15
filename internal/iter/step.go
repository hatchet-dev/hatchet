package iter

import (
	"fmt"
	"sync"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

type StepIterator struct {
	root    *db.StepModel
	curr    *db.StepModel
	stepMap map[string]*db.StepModel
	mu      sync.Mutex
}

func New(steps []db.StepModel) (*StepIterator, error) {
	// construct a list of steps by their next id
	backStepMap := make(map[string]string)
	stepMap := make(map[string]*db.StepModel)
	var lastStep *db.StepModel

	for _, step := range steps {
		stepCp := step
		stepMap[step.ID] = &stepCp

		if nextId, _ := step.NextID(); nextId != "" {
			backStepMap[nextId] = step.ID
		} else {
			lastStep = &stepCp
		}
	}

	if lastStep == nil {
		return nil, fmt.Errorf("could not find last step")
	}

	// find the root step by traversing backwards
	var root *db.StepModel
	currStepId := lastStep.ID

	for _, step := range steps {
		if backStepMap[step.ID] == "" {
			stepCp := step
			root = &stepCp
			break
		}

		currStepId = backStepMap[currStepId]
	}

	if root == nil {
		return nil, fmt.Errorf("could not find root step")
	}

	return &StepIterator{
		root:    root,
		curr:    nil,
		stepMap: stepMap,
	}, nil
}

func (si *StepIterator) Next() (*db.StepModel, bool) {
	si.mu.Lock()
	defer si.mu.Unlock()

	if si.curr == nil {
		si.curr = si.root
		return si.curr, true
	}

	nextId, _ := si.curr.NextID()

	if nextId == "" {
		return nil, false
	}

	nextStep, ok := si.stepMap[nextId]

	if !ok {
		return nil, false
	}

	si.curr = nextStep
	return si.curr, true
}
