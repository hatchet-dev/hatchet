// Original file: ../api-contracts/dispatcher/dispatcher.proto

import type { ActionType as _ActionType, ActionType__Output as _ActionType__Output } from './ActionType';

export interface AssignedAction {
  'tenantId'?: (string);
  'jobId'?: (string);
  'jobName'?: (string);
  'jobRunId'?: (string);
  'stepId'?: (string);
  'stepRunId'?: (string);
  'actionId'?: (string);
  'actionType'?: (_ActionType);
  'actionPayload'?: (string);
}

export interface AssignedAction__Output {
  'tenantId'?: (string);
  'jobId'?: (string);
  'jobName'?: (string);
  'jobRunId'?: (string);
  'stepId'?: (string);
  'stepRunId'?: (string);
  'actionId'?: (string);
  'actionType'?: (_ActionType__Output);
  'actionPayload'?: (string);
}
