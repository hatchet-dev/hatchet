// Original file: ../api-contracts/dispatcher/dispatcher.proto

import type * as grpc from '@grpc/grpc-js'
import type { MethodDefinition } from '@grpc/proto-loader'
import type { ActionEvent as _ActionEvent, ActionEvent__Output as _ActionEvent__Output } from './ActionEvent';
import type { ActionEventResponse as _ActionEventResponse, ActionEventResponse__Output as _ActionEventResponse__Output } from './ActionEventResponse';
import type { AssignedAction as _AssignedAction, AssignedAction__Output as _AssignedAction__Output } from './AssignedAction';
import type { WorkerListenRequest as _WorkerListenRequest, WorkerListenRequest__Output as _WorkerListenRequest__Output } from './WorkerListenRequest';
import type { WorkerRegisterRequest as _WorkerRegisterRequest, WorkerRegisterRequest__Output as _WorkerRegisterRequest__Output } from './WorkerRegisterRequest';
import type { WorkerRegisterResponse as _WorkerRegisterResponse, WorkerRegisterResponse__Output as _WorkerRegisterResponse__Output } from './WorkerRegisterResponse';
import type { WorkerUnsubscribeRequest as _WorkerUnsubscribeRequest, WorkerUnsubscribeRequest__Output as _WorkerUnsubscribeRequest__Output } from './WorkerUnsubscribeRequest';
import type { WorkerUnsubscribeResponse as _WorkerUnsubscribeResponse, WorkerUnsubscribeResponse__Output as _WorkerUnsubscribeResponse__Output } from './WorkerUnsubscribeResponse';

export interface DispatcherClient extends grpc.Client {
  Listen(argument: _WorkerListenRequest, metadata: grpc.Metadata, options?: grpc.CallOptions): grpc.ClientReadableStream<_AssignedAction__Output>;
  Listen(argument: _WorkerListenRequest, options?: grpc.CallOptions): grpc.ClientReadableStream<_AssignedAction__Output>;
  listen(argument: _WorkerListenRequest, metadata: grpc.Metadata, options?: grpc.CallOptions): grpc.ClientReadableStream<_AssignedAction__Output>;
  listen(argument: _WorkerListenRequest, options?: grpc.CallOptions): grpc.ClientReadableStream<_AssignedAction__Output>;
  
  Register(argument: _WorkerRegisterRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkerRegisterResponse__Output>): grpc.ClientUnaryCall;
  Register(argument: _WorkerRegisterRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_WorkerRegisterResponse__Output>): grpc.ClientUnaryCall;
  Register(argument: _WorkerRegisterRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkerRegisterResponse__Output>): grpc.ClientUnaryCall;
  Register(argument: _WorkerRegisterRequest, callback: grpc.requestCallback<_WorkerRegisterResponse__Output>): grpc.ClientUnaryCall;
  register(argument: _WorkerRegisterRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkerRegisterResponse__Output>): grpc.ClientUnaryCall;
  register(argument: _WorkerRegisterRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_WorkerRegisterResponse__Output>): grpc.ClientUnaryCall;
  register(argument: _WorkerRegisterRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkerRegisterResponse__Output>): grpc.ClientUnaryCall;
  register(argument: _WorkerRegisterRequest, callback: grpc.requestCallback<_WorkerRegisterResponse__Output>): grpc.ClientUnaryCall;
  
  SendActionEvent(argument: _ActionEvent, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_ActionEventResponse__Output>): grpc.ClientUnaryCall;
  SendActionEvent(argument: _ActionEvent, metadata: grpc.Metadata, callback: grpc.requestCallback<_ActionEventResponse__Output>): grpc.ClientUnaryCall;
  SendActionEvent(argument: _ActionEvent, options: grpc.CallOptions, callback: grpc.requestCallback<_ActionEventResponse__Output>): grpc.ClientUnaryCall;
  SendActionEvent(argument: _ActionEvent, callback: grpc.requestCallback<_ActionEventResponse__Output>): grpc.ClientUnaryCall;
  sendActionEvent(argument: _ActionEvent, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_ActionEventResponse__Output>): grpc.ClientUnaryCall;
  sendActionEvent(argument: _ActionEvent, metadata: grpc.Metadata, callback: grpc.requestCallback<_ActionEventResponse__Output>): grpc.ClientUnaryCall;
  sendActionEvent(argument: _ActionEvent, options: grpc.CallOptions, callback: grpc.requestCallback<_ActionEventResponse__Output>): grpc.ClientUnaryCall;
  sendActionEvent(argument: _ActionEvent, callback: grpc.requestCallback<_ActionEventResponse__Output>): grpc.ClientUnaryCall;
  
  Unsubscribe(argument: _WorkerUnsubscribeRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkerUnsubscribeResponse__Output>): grpc.ClientUnaryCall;
  Unsubscribe(argument: _WorkerUnsubscribeRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_WorkerUnsubscribeResponse__Output>): grpc.ClientUnaryCall;
  Unsubscribe(argument: _WorkerUnsubscribeRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkerUnsubscribeResponse__Output>): grpc.ClientUnaryCall;
  Unsubscribe(argument: _WorkerUnsubscribeRequest, callback: grpc.requestCallback<_WorkerUnsubscribeResponse__Output>): grpc.ClientUnaryCall;
  unsubscribe(argument: _WorkerUnsubscribeRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkerUnsubscribeResponse__Output>): grpc.ClientUnaryCall;
  unsubscribe(argument: _WorkerUnsubscribeRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_WorkerUnsubscribeResponse__Output>): grpc.ClientUnaryCall;
  unsubscribe(argument: _WorkerUnsubscribeRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_WorkerUnsubscribeResponse__Output>): grpc.ClientUnaryCall;
  unsubscribe(argument: _WorkerUnsubscribeRequest, callback: grpc.requestCallback<_WorkerUnsubscribeResponse__Output>): grpc.ClientUnaryCall;
  
}

export interface DispatcherHandlers extends grpc.UntypedServiceImplementation {
  Listen: grpc.handleServerStreamingCall<_WorkerListenRequest__Output, _AssignedAction>;
  
  Register: grpc.handleUnaryCall<_WorkerRegisterRequest__Output, _WorkerRegisterResponse>;
  
  SendActionEvent: grpc.handleUnaryCall<_ActionEvent__Output, _ActionEventResponse>;
  
  Unsubscribe: grpc.handleUnaryCall<_WorkerUnsubscribeRequest__Output, _WorkerUnsubscribeResponse>;
  
}

export interface DispatcherDefinition extends grpc.ServiceDefinition {
  Listen: MethodDefinition<_WorkerListenRequest, _AssignedAction, _WorkerListenRequest__Output, _AssignedAction__Output>
  Register: MethodDefinition<_WorkerRegisterRequest, _WorkerRegisterResponse, _WorkerRegisterRequest__Output, _WorkerRegisterResponse__Output>
  SendActionEvent: MethodDefinition<_ActionEvent, _ActionEventResponse, _ActionEvent__Output, _ActionEventResponse__Output>
  Unsubscribe: MethodDefinition<_WorkerUnsubscribeRequest, _WorkerUnsubscribeResponse, _WorkerUnsubscribeRequest__Output, _WorkerUnsubscribeResponse__Output>
}
