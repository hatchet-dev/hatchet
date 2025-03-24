import { Duration } from '../client/duration';
import { Condition, Action } from './base';

export interface Sleep {
  /**
   * Amount of time to sleep for.
   *
   * Go duration string format:
   * @example "10s", "1m", "1m5s"
   */
  sleepFor: Duration;
}

export class SleepCondition extends Condition {
  sleepFor: Duration;

  constructor(sleepFor: Duration, action?: Action) {
    super({
      // TODO readableDataKey
      readableDataKey: `sleep-${sleepFor}`,
      action,
      orGroupId: '',
      expression: '',
    });
    this.sleepFor = sleepFor;
  }
}
