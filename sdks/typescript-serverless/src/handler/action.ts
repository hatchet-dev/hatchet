import {
  AssignedAction as SdkAssignedAction,
  createAction,
} from '@hatchet-dev/typescript-sdk/edge/index.js';
import { AssignedAction } from '../generated/proto/dispatcher';
import { stripNamespace } from './contract';

/**
 * Re-reads the package's AssignedAction as the SDK's (two generated copies of the same
 * message) and strips the namespace from the names user code sees: `ctx.workflowName()`,
 * `ctx.workflowNameV1()` and the action id are the ones the user declared.
 */
export function toSdkAction(action: AssignedAction, namespace: string) {
  const sdkAction = SdkAssignedAction.fromJSON(AssignedAction.toJSON(action));

  sdkAction.actionId = stripNamespace(action.actionId, namespace);
  sdkAction.jobName = stripNamespace(action.jobName, namespace);

  if (!sdkAction.actionPayload) {
    sdkAction.actionPayload = '{}';
  }

  return createAction(sdkAction);
}
