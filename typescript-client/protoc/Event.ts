// Original file: ../api-contracts/events/events.proto

import type { Timestamp as _google_protobuf_Timestamp, Timestamp__Output as _google_protobuf_Timestamp__Output } from './google/protobuf/Timestamp';

export interface Event {
  'tenantId'?: (string);
  'eventId'?: (string);
  'key'?: (string);
  'payload'?: (string);
  'eventTimestamp'?: (_google_protobuf_Timestamp | null);
}

export interface Event__Output {
  'tenantId'?: (string);
  'eventId'?: (string);
  'key'?: (string);
  'payload'?: (string);
  'eventTimestamp'?: (_google_protobuf_Timestamp__Output);
}
