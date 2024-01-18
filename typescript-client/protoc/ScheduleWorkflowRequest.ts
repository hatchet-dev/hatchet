// Original file: ../api-contracts/workflows/workflows.proto

import type { Timestamp as _google_protobuf_Timestamp, Timestamp__Output as _google_protobuf_Timestamp__Output } from './google/protobuf/Timestamp';

export interface ScheduleWorkflowRequest {
  'tenantId'?: (string);
  'workflowId'?: (string);
  'schedules'?: (_google_protobuf_Timestamp)[];
  'input'?: (string);
}

export interface ScheduleWorkflowRequest__Output {
  'tenantId'?: (string);
  'workflowId'?: (string);
  'schedules'?: (_google_protobuf_Timestamp__Output)[];
  'input'?: (string);
}
