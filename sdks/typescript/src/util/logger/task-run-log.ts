import { ActionType } from '../../protoc/dispatcher';

export function actionMap(action: ActionType): string {
  switch (action) {
    case ActionType.START_STEP_RUN:
      return 'starting...';
    case ActionType.CANCEL_STEP_RUN:
      return 'cancelling...';
    case ActionType.START_GET_GROUP_KEY:
      return 'starting to get group key...';
    default:
      return 'unknown';
  }
}

export function taskRunLog(taskName: string, taskRunExternalId: string, action: string): string {
  return `Task run ${action} \t ${taskName}/${taskRunExternalId} `;
}
