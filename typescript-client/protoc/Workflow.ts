// Original file: ../api-contracts/workflows/workflows.proto

import type { Timestamp as _google_protobuf_Timestamp, Timestamp__Output as _google_protobuf_Timestamp__Output } from './google/protobuf/Timestamp';
import type { StringValue as _google_protobuf_StringValue, StringValue__Output as _google_protobuf_StringValue__Output } from './google/protobuf/StringValue';
import type { WorkflowVersion as _WorkflowVersion, WorkflowVersion__Output as _WorkflowVersion__Output } from './WorkflowVersion';

export interface Workflow {
  'id'?: (string);
  'createdAt'?: (_google_protobuf_Timestamp | null);
  'updatedAt'?: (_google_protobuf_Timestamp | null);
  'tenantId'?: (string);
  'name'?: (string);
  'description'?: (_google_protobuf_StringValue | null);
  'versions'?: (_WorkflowVersion)[];
}

export interface Workflow__Output {
  'id'?: (string);
  'createdAt'?: (_google_protobuf_Timestamp__Output);
  'updatedAt'?: (_google_protobuf_Timestamp__Output);
  'tenantId'?: (string);
  'name'?: (string);
  'description'?: (_google_protobuf_StringValue__Output);
  'versions'?: (_WorkflowVersion__Output)[];
}
