/**
 * Runtime-neutral helpers for the actions a worker receives from the dispatcher.
 *
 * This module imports nothing from Node so it can be used from the edge entry point.
 */
import type { AssignedAction } from '@hatchet/protoc/dispatcher';

export type ActionKey = `${string}/${number}` | `${string}/${number}/${number}`;

export type Action = AssignedAction & { readonly key: ActionKey };

/**
 * Builds the canonical action id for a task: `<workflow name>:<task name>`, fully
 * lowercased. The engine lowercases action ids on registration and the Go SDK sends
 * them lowercased, so this is the one form used for registration, the worker's
 * action registries and the ids the dispatcher assigns.
 */
export function createActionId(workflowName: string, taskName: string): string {
  return `${workflowName}:${taskName}`.toLowerCase();
}

export function workflowNameFromAction(
  action: Pick<AssignedAction, 'actionId' | 'jobName'>
): string {
  const separatorIndex = action.actionId.lastIndexOf(':');
  return separatorIndex === -1 ? action.jobName : action.actionId.substring(0, separatorIndex);
}

export function createAction(assignedAction: AssignedAction): Action {
  const action = assignedAction as Action;
  Object.defineProperty(action, 'key', {
    get(): ActionKey {
      // Durable task invocations each get a distinct key so a restored
      // invocation never collides with the one it replaces in contexts,
      // futures, or eviction state.
      if (this.durableTaskInvocationCount !== undefined) {
        return `${this.taskRunExternalId}/${this.retryCount}/${this.durableTaskInvocationCount}`;
      }
      return `${this.taskRunExternalId}/${this.retryCount}`;
    },
    enumerable: true,
    configurable: true,
  });
  return action;
}
