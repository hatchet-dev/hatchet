import { Condition, Action } from './base';
import { CreateWorkflowTaskOpts } from '../task';

export interface Parent {
  /**
   * The parent workflow task this condition is associated with.
   * This establishes a relationship between this condition and a specific workflow task.
   */
  parent: CreateWorkflowTaskOpts<any, any>;

  /**
   * Optional CEL expression to evaluate against the parent task's data.
   * When provided, the condition will only trigger if this expression evaluates to true.
   * @example "input.status == 'completed'", "input.result > 0"
   */
  expression?: string;
}

/**
 * Represents a condition that is triggered based on a parent workflow task.
 * This condition monitors the specified parent task and evaluates
 * any provided expression against the task's data.
 *
 * @example
 * // Create a condition that triggers when a parent task completes successfully
 * const parentCondition = new ParentCondition(
 *   parentTaskOpts,
 *   "input.status == 'success'",
 *   () => console.log("Parent task completed successfully!")
 * );
 */
export class ParentCondition extends Condition {
  /** The parent workflow task this condition is associated with */
  parent: CreateWorkflowTaskOpts<any, any>;

  /**
   * Creates a new condition that is triggered based on a parent workflow task.
   *
   * @param parent The parent workflow task this condition is associated with
   * @param expression Optional CEL expression to evaluate against the parent task's data
   * @param action Optional action to execute when the condition is met
   */
  constructor(
    parent: CreateWorkflowTaskOpts<any, any>,
    expression?: string,
    readableDataKey?: string,
    action?: Action
  ) {
    super({
      readableDataKey: readableDataKey || `parent-${parent.name || Date.now().toString()}`,
      action,
      orGroupId: '',
      expression: expression || '',
    });
    this.parent = parent;
  }
}
