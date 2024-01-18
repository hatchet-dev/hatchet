// Original file: ../api-contracts/dispatcher/dispatcher.proto

import type { Timestamp as _google_protobuf_Timestamp, Timestamp__Output as _google_protobuf_Timestamp__Output } from './google/protobuf/Timestamp';
import type { ActionEventType as _ActionEventType, ActionEventType__Output as _ActionEventType__Output } from './ActionEventType';

export interface ActionEvent {
  'tenantId'?: (string);
  'workerId'?: (string);
  'jobId'?: (string);
  'jobRunId'?: (string);
  'stepId'?: (string);
  'stepRunId'?: (string);
  'actionId'?: (string);
  'eventTimestamp'?: (_google_protobuf_Timestamp | null);
  'eventType'?: (_ActionEventType);
  'eventPayload'?: (string);
}

export interface ActionEvent__Output {
  'tenantId'?: (string);
  'workerId'?: (string);
  'jobId'?: (string);
  'jobRunId'?: (string);
  'stepId'?: (string);
  'stepRunId'?: (string);
  'actionId'?: (string);
  'eventTimestamp'?: (_google_protobuf_Timestamp__Output);
  'eventType'?: (_ActionEventType__Output);
  'eventPayload'?: (string);
}
