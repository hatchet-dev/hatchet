// Original file: ../api-contracts/workflows/workflows.proto

import type { Timestamp as _google_protobuf_Timestamp, Timestamp__Output as _google_protobuf_Timestamp__Output } from './google/protobuf/Timestamp';
import type { StringValue as _google_protobuf_StringValue, StringValue__Output as _google_protobuf_StringValue__Output } from './google/protobuf/StringValue';
import type { Step as _Step, Step__Output as _Step__Output } from './Step';

export interface Job {
  'id'?: (string);
  'createdAt'?: (_google_protobuf_Timestamp | null);
  'updatedAt'?: (_google_protobuf_Timestamp | null);
  'tenantId'?: (string);
  'workflowVersionId'?: (string);
  'name'?: (string);
  'description'?: (_google_protobuf_StringValue | null);
  'steps'?: (_Step)[];
  'timeout'?: (_google_protobuf_StringValue | null);
}

export interface Job__Output {
  'id'?: (string);
  'createdAt'?: (_google_protobuf_Timestamp__Output);
  'updatedAt'?: (_google_protobuf_Timestamp__Output);
  'tenantId'?: (string);
  'workflowVersionId'?: (string);
  'name'?: (string);
  'description'?: (_google_protobuf_StringValue__Output);
  'steps'?: (_Step__Output)[];
  'timeout'?: (_google_protobuf_StringValue__Output);
}
