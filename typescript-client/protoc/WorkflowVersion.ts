// Original file: ../api-contracts/workflows/workflows.proto

import type { Timestamp as _google_protobuf_Timestamp, Timestamp__Output as _google_protobuf_Timestamp__Output } from './google/protobuf/Timestamp';
import type { WorkflowTriggers as _WorkflowTriggers, WorkflowTriggers__Output as _WorkflowTriggers__Output } from './WorkflowTriggers';
import type { Job as _Job, Job__Output as _Job__Output } from './Job';

export interface WorkflowVersion {
  'id'?: (string);
  'createdAt'?: (_google_protobuf_Timestamp | null);
  'updatedAt'?: (_google_protobuf_Timestamp | null);
  'version'?: (string);
  'order'?: (number);
  'workflowId'?: (string);
  'triggers'?: (_WorkflowTriggers | null);
  'jobs'?: (_Job)[];
}

export interface WorkflowVersion__Output {
  'id'?: (string);
  'createdAt'?: (_google_protobuf_Timestamp__Output);
  'updatedAt'?: (_google_protobuf_Timestamp__Output);
  'version'?: (string);
  'order'?: (number);
  'workflowId'?: (string);
  'triggers'?: (_WorkflowTriggers__Output);
  'jobs'?: (_Job__Output)[];
}
