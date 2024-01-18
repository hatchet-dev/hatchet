// Original file: ../api-contracts/events/events.proto

import type { Timestamp as _google_protobuf_Timestamp, Timestamp__Output as _google_protobuf_Timestamp__Output } from './google/protobuf/Timestamp';

export interface PushEventRequest {
  'tenantId'?: (string);
  'key'?: (string);
  'payload'?: (string);
  'eventTimestamp'?: (_google_protobuf_Timestamp | null);
}

export interface PushEventRequest__Output {
  'tenantId'?: (string);
  'key'?: (string);
  'payload'?: (string);
  'eventTimestamp'?: (_google_protobuf_Timestamp__Output);
}
