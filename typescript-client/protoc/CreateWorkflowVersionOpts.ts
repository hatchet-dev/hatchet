// Original file: ../api-contracts/workflows/workflows.proto

import type { Timestamp as _google_protobuf_Timestamp, Timestamp__Output as _google_protobuf_Timestamp__Output } from './google/protobuf/Timestamp';
import type { CreateWorkflowJobOpts as _CreateWorkflowJobOpts, CreateWorkflowJobOpts__Output as _CreateWorkflowJobOpts__Output } from './CreateWorkflowJobOpts';

export interface CreateWorkflowVersionOpts {
  'name'?: (string);
  'description'?: (string);
  'version'?: (string);
  'eventTriggers'?: (string)[];
  'cronTriggers'?: (string)[];
  'scheduledTriggers'?: (_google_protobuf_Timestamp)[];
  'jobs'?: (_CreateWorkflowJobOpts)[];
}

export interface CreateWorkflowVersionOpts__Output {
  'name'?: (string);
  'description'?: (string);
  'version'?: (string);
  'eventTriggers'?: (string)[];
  'cronTriggers'?: (string)[];
  'scheduledTriggers'?: (_google_protobuf_Timestamp__Output)[];
  'jobs'?: (_CreateWorkflowJobOpts__Output)[];
}
