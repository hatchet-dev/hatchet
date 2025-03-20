/* eslint-disable no-shadow */
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
