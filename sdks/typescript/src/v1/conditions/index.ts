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

export interface BaseMatchCondition {
  eventKey?: string; // remove
  readableDataKey?: string;
  action?: Action;
  /** a UUID defining the OR group for this condition */
  orGroupId?: string;
  expression?: string; // options
}

export abstract class Condition {
  base: BaseMatchCondition;

  constructor(base: BaseMatchCondition) {
    this.base = base;
  }
}

export interface Sleep {
  sleepFor: number; // seconds
}

export class SleepCondition extends Condition {
  // TODO duration consistency
  sleepFor: number; // seconds

  constructor(sleepFor: number) {
    super({
      readableDataKey: '',
      action: Action.CREATE,
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

  constructor(eventKey: string, expression: string) {
    super({
      readableDataKey: '',
      action: Action.CREATE,
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

export function render(condition: Condition | OrCondition): string {
  if (condition instanceof SleepCondition) {
    return `sleepFor: ${condition.sleepFor}`;
  }
  if (condition instanceof UserEventCondition) {
    return `event: ${condition.eventKey}${condition.expression ? `, expression: ${condition.expression}` : ''}`;
  }
  if (condition instanceof OrCondition) {
    return `OR(${condition.conditions.map((c) => render(c)).join(' || ')})`;
  }
  return 'Unknown condition';
}

/**
 * Creates a condition that waits for all provided conditions to be met (AND logic)
 */
export function Render(...conditionsOrObjs: Conditions[]): Condition[] {
  return conditionsOrObjs.reduce<Condition[]>((acc, conditionOrObj) => {
    if (conditionOrObj instanceof Condition) {
      return [...acc, conditionOrObj];
    }

    if (conditionOrObj instanceof OrCondition) {
      return [...acc, ...Render(...conditionOrObj.conditions)];
    }

    // Handle object syntax
    if ('sleepFor' in conditionOrObj) {
      return [...acc, new SleepCondition(conditionOrObj.sleepFor)];
    }
    if ('eventKey' in conditionOrObj) {
      return [
        ...acc,
        new UserEventCondition(conditionOrObj.eventKey, conditionOrObj.expression || ''),
      ];
    }

    throw new Error(`Unknown condition object: ${JSON.stringify(conditionOrObj)}`);
  }, []);
}

/**
 * Creates a condition group with OR logic
 * Flattens nested Or conditions to ensure proper grouping
 */
export function Or(...conditionsOrObjs: (IConditions | Condition)[]): OrCondition {
  // must be Condition[] because OrCondition is not a Condition
  const conditions = Render(...conditionsOrObjs);

  const orGroupId = crypto.randomUUID ? crypto.randomUUID() : `or-${Date.now()}-${Math.random()}`;
  conditions.forEach((condition) => {
    if (condition instanceof Condition) {
      condition.base.orGroupId = orGroupId;
    }
  });

  return new OrCondition(conditions);
}
