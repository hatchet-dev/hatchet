import { Condition, Action } from './base';

export interface UserEvent {
  /**
   * The unique key identifying the specific user event to monitor.
   * This should match the event key that will be emitted from your application.
   * @example "button:clicked", "page:viewed", "form:submitted"
   */
  eventKey: string;

  /**
   * Optional CEL expression to evaluate against the event data.
   * When provided, the condition will only trigger if this expression evaluates to true.
   * @example "input.quantity > 5", "input.status == 'completed'"
   */
  expression?: string;

  /**
   * Optional unique identifier for the event data in the readable stream.
   * When multiple conditions are listening to the same event type,
   * using a custom readableDataKey prevents duplicate data processing
   * by differentiating between the conditions in the data store.
   * If not specified, the eventKey will be used as the default identifier.
   */
  readableDataKey?: string;
}

/**
 * Represents a condition that is triggered based on a specific user event.
 * This condition monitors for events with the specified key and evaluates
 * any provided expression against the event data.
 *
 * @param eventKey The key identifying the specific user event to monitor
 * @param expression The CEL expression to evaluate against the event data
 * @param readableDataKey Optional parameter that provides a unique identifier for the data.
 *                        When multiple conditions are listening to the same event, using a custom
 *                        readableDataKey prevents duplicate data by differentiating between the conditions.
 *                        If not provided, defaults to the eventKey.
 * @param action Optional action to execute when the condition is met
 *
 * @example
 * // Create a condition that triggers when a "purchase" event occurs with amount > 100
 * const purchaseCondition = new UserEventCondition(
 *   "purchase",
 *   "data.amount > 100",
 *   "high_value_purchase",
 *   () => console.log("High value purchase detected!")
 * );
 */
export class UserEventCondition extends Condition {
  eventKey: string;
  expression: string;

  constructor(eventKey: string, expression: string, readableDataKey?: string, action?: Action) {
    super({
      readableDataKey: readableDataKey || eventKey,
      action,
      orGroupId: '',
      expression: '',
    });
    this.eventKey = eventKey;
    this.expression = expression;
  }
}
