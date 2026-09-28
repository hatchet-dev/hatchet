import {
  AssignedAction as SdkAssignedAction,
  createAction,
} from '@hatchet-dev/typescript-sdk/edge/index.js';
import { AssignedAction } from '../generated/proto/dispatcher';

/**
 * Re-reads the package's AssignedAction as the SDK's (two generated copies of the same
 * message). The operator registers workflows and actions under the names the endpoint
 * declared, so `ctx.workflowName()`, `ctx.workflowNameV1()` and the action id arrive as the
 * user wrote them.
 */
export function toSdkAction(action: AssignedAction) {
  const sdkAction = SdkAssignedAction.fromJSON(AssignedAction.toJSON(action));

  if (!sdkAction.actionPayload) {
    sdkAction.actionPayload = '{}';
  }

  return createAction(sdkAction);
}
