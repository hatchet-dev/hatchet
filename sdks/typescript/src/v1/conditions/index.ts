/* eslint-disable no-param-reassign */
/* eslint-disable no-shadow */
/* eslint-disable max-classes-per-file */

export enum Action {
  CREATE = 0,
  QUEUE = 1,
  CANCEL = 2,
  SKIP = 3,
  UNRECOGNIZED = -1,
}

export interface BaseCondition {
  eventKey?: string; // remove
  readableDataKey?: string;
  action?: Action;
  /** a UUID defining the OR group for this condition */
  orGroupId?: string;
  expression?: string; // options
}

export abstract class Condition {
  base: BaseCondition;

  constructor(base: BaseCondition) {
    this.base = base;
  }
}

// TODO export from root?
export interface Sleep {
  sleepFor: number; // seconds
}

export class SleepCondition extends Condition {
  // TODO duration consistency
  sleepFor: number; // seconds

  constructor(sleepFor: number, action?: Action) {
    super({
      readableDataKey: '',
      action,
      orGroupId: '',
      expression: '',
    });
    this.sleepFor = sleepFor;
  }
}

export interface UserEvent {
  eventKey: string;
  expression?: string;
}

export class UserEventCondition extends Condition {
  eventKey: string;
  expression: string;

  constructor(eventKey: string, expression: string, action?: Action) {
    super({
      readableDataKey: '',
      action,
      orGroupId: '',
      expression: '',
    });
    this.eventKey = eventKey;
    this.expression = expression;
  }
}

export type IConditions = Sleep | UserEvent;

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
