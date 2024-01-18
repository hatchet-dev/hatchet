// Original file: ../api-contracts/events/events.proto

import type * as grpc from '@grpc/grpc-js'
import type { MethodDefinition } from '@grpc/proto-loader'
import type { Event as _Event, Event__Output as _Event__Output } from './Event';
import type { ListEventRequest as _ListEventRequest, ListEventRequest__Output as _ListEventRequest__Output } from './ListEventRequest';
import type { ListEventResponse as _ListEventResponse, ListEventResponse__Output as _ListEventResponse__Output } from './ListEventResponse';
import type { PushEventRequest as _PushEventRequest, PushEventRequest__Output as _PushEventRequest__Output } from './PushEventRequest';
import type { ReplayEventRequest as _ReplayEventRequest, ReplayEventRequest__Output as _ReplayEventRequest__Output } from './ReplayEventRequest';

export interface EventsServiceClient extends grpc.Client {
  List(argument: _ListEventRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_ListEventResponse__Output>): grpc.ClientUnaryCall;
  List(argument: _ListEventRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_ListEventResponse__Output>): grpc.ClientUnaryCall;
  List(argument: _ListEventRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_ListEventResponse__Output>): grpc.ClientUnaryCall;
  List(argument: _ListEventRequest, callback: grpc.requestCallback<_ListEventResponse__Output>): grpc.ClientUnaryCall;
  list(argument: _ListEventRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_ListEventResponse__Output>): grpc.ClientUnaryCall;
  list(argument: _ListEventRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_ListEventResponse__Output>): grpc.ClientUnaryCall;
  list(argument: _ListEventRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_ListEventResponse__Output>): grpc.ClientUnaryCall;
  list(argument: _ListEventRequest, callback: grpc.requestCallback<_ListEventResponse__Output>): grpc.ClientUnaryCall;
  
  Push(argument: _PushEventRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  Push(argument: _PushEventRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  Push(argument: _PushEventRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  Push(argument: _PushEventRequest, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  push(argument: _PushEventRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  push(argument: _PushEventRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  push(argument: _PushEventRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  push(argument: _PushEventRequest, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  
  ReplaySingleEvent(argument: _ReplayEventRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  ReplaySingleEvent(argument: _ReplayEventRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  ReplaySingleEvent(argument: _ReplayEventRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  ReplaySingleEvent(argument: _ReplayEventRequest, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  replaySingleEvent(argument: _ReplayEventRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  replaySingleEvent(argument: _ReplayEventRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  replaySingleEvent(argument: _ReplayEventRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  replaySingleEvent(argument: _ReplayEventRequest, callback: grpc.requestCallback<_Event__Output>): grpc.ClientUnaryCall;
  
}

export interface EventsServiceHandlers extends grpc.UntypedServiceImplementation {
  List: grpc.handleUnaryCall<_ListEventRequest__Output, _ListEventResponse>;
  
  Push: grpc.handleUnaryCall<_PushEventRequest__Output, _Event>;
  
  ReplaySingleEvent: grpc.handleUnaryCall<_ReplayEventRequest__Output, _Event>;
  
}

export interface EventsServiceDefinition extends grpc.ServiceDefinition {
  List: MethodDefinition<_ListEventRequest, _ListEventResponse, _ListEventRequest__Output, _ListEventResponse__Output>
  Push: MethodDefinition<_PushEventRequest, _Event, _PushEventRequest__Output, _Event__Output>
  ReplaySingleEvent: MethodDefinition<_ReplayEventRequest, _Event, _ReplayEventRequest__Output, _Event__Output>
}
