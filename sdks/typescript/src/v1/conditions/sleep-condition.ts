import { Condition, Action } from './base';

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
