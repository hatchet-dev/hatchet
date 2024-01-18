import type * as grpc from '@grpc/grpc-js';
import type { MessageTypeDefinition } from '@grpc/proto-loader';

import type { EventsServiceClient as _EventsServiceClient, EventsServiceDefinition as _EventsServiceDefinition } from './EventsService';

type SubtypeConstructor<Constructor extends new (...args: any) => any, Subtype> = {
  new(...args: ConstructorParameters<Constructor>): Subtype;
};

export interface ProtoGrpcType {
  Event: MessageTypeDefinition
  EventsService: SubtypeConstructor<typeof grpc.Client, _EventsServiceClient> & { service: _EventsServiceDefinition }
  ListEventRequest: MessageTypeDefinition
  ListEventResponse: MessageTypeDefinition
  PushEventRequest: MessageTypeDefinition
  ReplayEventRequest: MessageTypeDefinition
  google: {
    protobuf: {
      Timestamp: MessageTypeDefinition
    }
  }
}

