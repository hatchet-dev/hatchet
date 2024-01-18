// Original file: ../api-contracts/workflows/workflows.proto

import type { Timestamp as _google_protobuf_Timestamp, Timestamp__Output as _google_protobuf_Timestamp__Output } from './google/protobuf/Timestamp';
import type { StringValue as _google_protobuf_StringValue, StringValue__Output as _google_protobuf_StringValue__Output } from './google/protobuf/StringValue';

export interface Step {
  'id'?: (string);
  'createdAt'?: (_google_protobuf_Timestamp | null);
  'updatedAt'?: (_google_protobuf_Timestamp | null);
  'readableId'?: (_google_protobuf_StringValue | null);
  'tenantId'?: (string);
  'jobId'?: (string);
  'action'?: (string);
  'timeout'?: (_google_protobuf_StringValue | null);
  'parents'?: (string)[];
  'children'?: (string)[];
}

export interface Step__Output {
  'id'?: (string);
  'createdAt'?: (_google_protobuf_Timestamp__Output);
  'updatedAt'?: (_google_protobuf_Timestamp__Output);
  'readableId'?: (_google_protobuf_StringValue__Output);
  'tenantId'?: (string);
  'jobId'?: (string);
  'action'?: (string);
  'timeout'?: (_google_protobuf_StringValue__Output);
  'parents'?: (string)[];
  'children'?: (string)[];
}
