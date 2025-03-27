package condition

import (
	"time"

	"github.com/google/uuid"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// Condition represents a set of conditions to either trigger a task or satisfy a wait condition.
// Callers should not use Condition directly. Instead, you should use a condition wrapper, such
// as Conditions, SleepCondition, UserEventCondition, ParentCondition, Or.
type Condition interface {
	ToPB(action contracts.Action) *ConditionMulti
}

type ConditionMulti struct {
	SleepConditions     []*contracts.SleepMatchCondition
	UserEventConditions []*contracts.UserEventMatchCondition
	ParentConditions    []*contracts.ParentOverrideMatchCondition
}

type baseCondition struct {
	readableDataKey string
	orGroupID       uuid.UUID
	expression      string
}

func (b *baseCondition) baseCondition(action contracts.Action) *contracts.BaseMatchCondition {
	return &contracts.BaseMatchCondition{
		ReadableDataKey: b.readableDataKey,
		Action:          action,
		OrGroupId:       b.orGroupID.String(),
		Expression:      b.expression,
	}
}

func (b *baseCondition) ToPB(action contracts.Action) *ConditionMulti {
	return &ConditionMulti{
		SleepConditions:     nil,
		UserEventConditions: nil,
		ParentConditions:    nil,
	}
}

type sleepCondition struct {
	*baseCondition

	duration time.Duration
}

// SleepCondition creates a new condition that waits for a specified duration.
func SleepCondition(duration time.Duration) *sleepCondition {
	return &sleepCondition{
		baseCondition: &baseCondition{
			readableDataKey: "sleep:" + duration.String(),
		},
		duration: duration,
	}
}

func (s *sleepCondition) Key() string {
	return s.readableDataKey
}

func (s *sleepCondition) ToPB(action contracts.Action) *ConditionMulti {
	sleep := &contracts.SleepMatchCondition{
		Base:     s.baseCondition.baseCondition(action),
		SleepFor: s.duration.String(),
	}

	return &ConditionMulti{
		SleepConditions:     []*contracts.SleepMatchCondition{sleep},
		UserEventConditions: nil,
		ParentConditions:    nil,
	}
}

type userEventCondition struct {
	baseCondition

	eventKey string
}

// UserEventCondition creates a new condition that waits for a user event to occur.
// The eventKey is the key of the user event that the condition is waiting for.
// The expression is an optional CEL expression that can be used to filter the user event,
// such as `event.data.key == "value"`.
func UserEventCondition(eventKey string, expression string) *userEventCondition {
	return &userEventCondition{
		baseCondition: baseCondition{
			readableDataKey: eventKey,
			expression:      expression,
		},
		eventKey: eventKey,
	}
}

func (u *userEventCondition) ToPB(action contracts.Action) *ConditionMulti {
	userEvent := &contracts.UserEventMatchCondition{
		Base:         u.baseCondition.baseCondition(action),
		UserEventKey: u.eventKey,
	}

	return &ConditionMulti{
		SleepConditions:     nil,
		UserEventConditions: []*contracts.UserEventMatchCondition{userEvent},
		ParentConditions:    nil,
	}
}

type parentCondition struct {
	baseCondition

	parentReadableId string
}

// ParentCondition creates a new condition that is satisfied based on the output of the parent task
// in a DAG.
func ParentCondition(parent string, expression string) *parentCondition {
	return &parentCondition{
		baseCondition: baseCondition{
			readableDataKey: parent,
			expression:      expression,
		},
		parentReadableId: parent,
	}
}

type orGroup struct {
	orGroupID  uuid.UUID
	conditions []Condition
}

// Or creates a new condition that is satisfied if any of the conditions in the group are satisfied.
// By default, conditions are evaluated as an intersection (AND). For more complex conditions, you can
// use Or to create a group of conditions that are evaluated as a union (OR).
func Or(conditions ...Condition) *orGroup {
	return &orGroup{
		orGroupID:  uuid.New(),
		conditions: conditions,
	}
}

func (o *orGroup) ToPB(action contracts.Action) *ConditionMulti {
	sleepConditions := make([]*contracts.SleepMatchCondition, 0)
	userEventConditions := make([]*contracts.UserEventMatchCondition, 0)
	parentConditions := make([]*contracts.ParentOverrideMatchCondition, 0)

	for _, condition := range o.conditions {
		c := condition.ToPB(action)

		for _, sleepCondition := range c.SleepConditions {
			sleepCondition.Base.OrGroupId = o.orGroupID.String()
			sleepConditions = append(sleepConditions, sleepCondition)
		}

		for _, userEventCondition := range c.UserEventConditions {
			userEventCondition.Base.OrGroupId = o.orGroupID.String()
			userEventConditions = append(userEventConditions, userEventCondition)
		}

		for _, parentCondition := range c.ParentConditions {
			parentCondition.Base.OrGroupId = o.orGroupID.String()
			parentConditions = append(parentConditions, parentCondition)
		}
	}

	return &ConditionMulti{
		SleepConditions:     sleepConditions,
		UserEventConditions: userEventConditions,
		ParentConditions:    parentConditions,
	}
}

type conditions struct {
	conditions []Condition
}

// Conditions creates a new condition that is satisfied if all of the conditions are satisfied.
func Conditions(cs ...Condition) *conditions {
	return &conditions{
		conditions: cs,
	}
}

func (c *conditions) ToPB(action contracts.Action) *ConditionMulti {
	sleepConditions := make([]*contracts.SleepMatchCondition, 0)
	userEventConditions := make([]*contracts.UserEventMatchCondition, 0)
	parentConditions := make([]*contracts.ParentOverrideMatchCondition, 0)

	for _, condition := range c.conditions {
		c := condition.ToPB(action)
		sleepConditions = append(sleepConditions, c.SleepConditions...)
		userEventConditions = append(userEventConditions, c.UserEventConditions...)
		parentConditions = append(parentConditions, c.ParentConditions...)
	}

	return &ConditionMulti{
		SleepConditions:     sleepConditions,
		UserEventConditions: userEventConditions,
		ParentConditions:    parentConditions,
	}
}
