import { Duration } from '../client/duration';
import { Condition, Action } from './base';

export interface Sleep {
  /**
   * Amount of time to sleep for.
   * Specifies how long this condition should wait before proceeding.
   * Uses Go duration string format.
   *
   * @example "10s", "1m", "1m5s", "24h"
   */
  sleepFor: Duration;

  /**
   * Optional unique identifier for the sleep condition in the readable stream.
   * When multiple conditions have the same sleep duration,
   * using a custom readableDataKey prevents duplicate data processing
   * by differentiating between the conditions in the data store.
   * If not specified, a default identifier based on the sleep duration will be used.
   */
  readableDataKey?: string;
}

/**
 * Represents a condition that waits for a specified duration before proceeding.
 * This condition is useful for implementing time delays in workflows.
 *
 * @example
 * // Create a condition that waits for 5 minutes
 * const waitCondition = new SleepCondition(
 *   "5m",
 *   "reminder_delay",
 *   () => console.log("Wait period completed!")
 * );
 */
export class SleepCondition extends Condition {
  /** The duration to sleep for in Go duration string format */
  sleepFor: Duration;

  /**
   * Creates a new sleep condition that waits for the specified duration.
   *
   * @param sleepFor Duration to wait in Go duration string format (e.g., "30s", "5m")
   * @param readableDataKey Optional unique identifier for the condition data.
   *                        When multiple sleep conditions have the same duration, using a custom
   *                        readableDataKey prevents duplicate data by differentiating between them.
   *                        If not provided, defaults to `sleep-${sleepFor}`.
   * @param action Optional action to execute when the sleep duration completes
   */
  constructor(sleepFor: Duration, readableDataKey?: string, action?: Action) {
    super({
      readableDataKey: readableDataKey || `sleep-${sleepFor}`,
      action,
      orGroupId: '',
      expression: '',
    });
    this.sleepFor = sleepFor;
  }
}
