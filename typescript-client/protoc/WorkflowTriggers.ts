// Original file: ../api-contracts/workflows/workflows.proto

import type { Timestamp as _google_protobuf_Timestamp, Timestamp__Output as _google_protobuf_Timestamp__Output } from './google/protobuf/Timestamp';
import type { WorkflowTriggerEventRef as _WorkflowTriggerEventRef, WorkflowTriggerEventRef__Output as _WorkflowTriggerEventRef__Output } from './WorkflowTriggerEventRef';
import type { WorkflowTriggerCronRef as _WorkflowTriggerCronRef, WorkflowTriggerCronRef__Output as _WorkflowTriggerCronRef__Output } from './WorkflowTriggerCronRef';

export interface WorkflowTriggers {
  'id'?: (string);
  'createdAt'?: (_google_protobuf_Timestamp | null);
  'updatedAt'?: (_google_protobuf_Timestamp | null);
  'workflowVersionId'?: (string);
  'tenantId'?: (string);
  'events'?: (_WorkflowTriggerEventRef)[];
  'crons'?: (_WorkflowTriggerCronRef)[];
}

export interface WorkflowTriggers__Output {
  'id'?: (string);
  'createdAt'?: (_google_protobuf_Timestamp__Output);
  'updatedAt'?: (_google_protobuf_Timestamp__Output);
  'workflowVersionId'?: (string);
  'tenantId'?: (string);
  'events'?: (_WorkflowTriggerEventRef__Output)[];
  'crons'?: (_WorkflowTriggerCronRef__Output)[];
}
