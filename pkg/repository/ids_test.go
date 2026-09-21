package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_childSpawnKeyNamespacesDagStepsFromUserSpawns(t *testing.T) {
	parent := uuid.New()
	childIndex := int64(0)

	userSpawn := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			ParentExternalId: &parent,
			ChildIndex:       &childIndex,
		},
	}

	dagStep := &WorkflowNameTriggerOpts{
		IsDagStepTrigger: true,
		TriggerTaskData: &TriggerTaskData{
			ParentExternalId: &parent,
			ChildIndex:       &childIndex,
		},
	}

	assert.NotEqual(t, userSpawn.childSpawnKey(), dagStep.childSpawnKey())
}

func Test_childSpawnKeyIsStableForTheSameStep(t *testing.T) {
	parent := uuid.New()
	childIndex := int64(2)

	opt := func() *WorkflowNameTriggerOpts {
		return &WorkflowNameTriggerOpts{
			IsDagStepTrigger: true,
			TriggerTaskData: &TriggerTaskData{
				ParentExternalId: &parent,
				ChildIndex:       &childIndex,
			},
		}
	}

	assert.Equal(t, opt().childSpawnKey(), opt().childSpawnKey())
}
