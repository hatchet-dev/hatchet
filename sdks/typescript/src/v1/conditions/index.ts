/* eslint-disable no-param-reassign */
/* eslint-disable no-shadow */
/* eslint-disable max-classes-per-file */

import { Condition, Action } from './base';
import { Parent, ParentCondition } from './parent-condition';
import { Sleep, SleepCondition } from './sleep-condition';
import { UserEvent, UserEventCondition } from './user-event-condition';

export { Sleep, SleepCondition, UserEvent, UserEventCondition };

export type IConditions = Sleep | UserEvent | Parent;

export type Conditions = Condition | OrCondition | IConditions;

export class OrCondition {
  conditions: Condition[];

  constructor(conditions: Condition[]) {
    this.conditions = conditions;
  }
}

/**
 * Creates a condition that waits for all provided conditions to be met (AND logic)
 * use Or() to create a condition that waits for any of the provided conditions to be met (OR logic)
 * @param conditions - Conditions or OrConditions to be rendered
 * @returns A flattened array of Conditions
 *
 * @example
 * const conditions = Render(
 *   Or({ sleepFor: 5 }, { eventKey: 'user:update' }),
 *   { eventKey: 'user:create' },
 *   Or({ eventKey: 'user:update' }, { eventKey: 'user:delete' })
 * );
 */
export function Render(action?: Action, conditions?: Conditions | Conditions[]): Condition[] {
  if (!conditions) return [];

  if (!Array.isArray(conditions)) {
    return Render(action, [conditions]);
  }

  const renderedConditions = conditions.reduce<Condition[]>((acc, conditionOrObj) => {
    if (conditionOrObj instanceof Condition) {
      return [...acc, conditionOrObj];
    }

    if (conditionOrObj instanceof OrCondition) {
      return [...acc, ...Render(action, conditionOrObj.conditions)];
    }

    // Handle object syntax
    if ('sleepFor' in conditionOrObj) {
      return [...acc, new SleepCondition(conditionOrObj.sleepFor, action)];
    }
    if ('eventKey' in conditionOrObj) {
      return [
        ...acc,
        new UserEventCondition(conditionOrObj.eventKey, conditionOrObj.expression || '', action),
      ];
    }
    if ('parent' in conditionOrObj) {
      return [
        ...acc,
        new ParentCondition(conditionOrObj.parent, conditionOrObj.expression || '', action),
      ];
    }

    throw new Error(`Unknown condition object: ${JSON.stringify(conditionOrObj)}`);
  }, []);

  // set the action for each condition
  return renderedConditions.filter((condition) => {
    if (condition instanceof Condition) {
      condition.base.action = action;
    }
    return condition;
  });
}

/**
 * Creates a condition group with OR logic
 * Flattens nested Or conditions to ensure proper grouping
 */
export function Or(...conditionsOrObjs: (IConditions | Condition)[]): OrCondition {
  // must be Condition[] because OrCondition is not a Condition
  const conditions = Render(undefined, conditionsOrObjs);

  const orGroupId = generateGroupId();
  conditions.forEach((condition) => {
    if (condition instanceof Condition) {
      condition.base.orGroupId = orGroupId;
    }
  });

  return new OrCondition(conditions);
}

export function generateGroupId(): string {
  return crypto.randomUUID ? crypto.randomUUID() : `or-${Date.now()}-${Math.random()}`;
}
