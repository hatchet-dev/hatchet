// package: 
// file: dispatcher/dispatcher.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as dispatcher_dispatcher_pb from "../dispatcher/dispatcher_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

interface IDispatcherService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    register: IDispatcherService_IRegister;
    listen: IDispatcherService_IListen;
    sendActionEvent: IDispatcherService_ISendActionEvent;
    unsubscribe: IDispatcherService_IUnsubscribe;
}

interface IDispatcherService_IRegister extends grpc.MethodDefinition<dispatcher_dispatcher_pb.WorkerRegisterRequest, dispatcher_dispatcher_pb.WorkerRegisterResponse> {
    path: "/Dispatcher/Register";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<dispatcher_dispatcher_pb.WorkerRegisterRequest>;
    requestDeserialize: grpc.deserialize<dispatcher_dispatcher_pb.WorkerRegisterRequest>;
    responseSerialize: grpc.serialize<dispatcher_dispatcher_pb.WorkerRegisterResponse>;
    responseDeserialize: grpc.deserialize<dispatcher_dispatcher_pb.WorkerRegisterResponse>;
}
interface IDispatcherService_IListen extends grpc.MethodDefinition<dispatcher_dispatcher_pb.WorkerListenRequest, dispatcher_dispatcher_pb.AssignedAction> {
    path: "/Dispatcher/Listen";
    requestStream: false;
    responseStream: true;
    requestSerialize: grpc.serialize<dispatcher_dispatcher_pb.WorkerListenRequest>;
    requestDeserialize: grpc.deserialize<dispatcher_dispatcher_pb.WorkerListenRequest>;
    responseSerialize: grpc.serialize<dispatcher_dispatcher_pb.AssignedAction>;
    responseDeserialize: grpc.deserialize<dispatcher_dispatcher_pb.AssignedAction>;
}
interface IDispatcherService_ISendActionEvent extends grpc.MethodDefinition<dispatcher_dispatcher_pb.ActionEvent, dispatcher_dispatcher_pb.ActionEventResponse> {
    path: "/Dispatcher/SendActionEvent";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<dispatcher_dispatcher_pb.ActionEvent>;
    requestDeserialize: grpc.deserialize<dispatcher_dispatcher_pb.ActionEvent>;
    responseSerialize: grpc.serialize<dispatcher_dispatcher_pb.ActionEventResponse>;
    responseDeserialize: grpc.deserialize<dispatcher_dispatcher_pb.ActionEventResponse>;
}
interface IDispatcherService_IUnsubscribe extends grpc.MethodDefinition<dispatcher_dispatcher_pb.WorkerUnsubscribeRequest, dispatcher_dispatcher_pb.WorkerUnsubscribeResponse> {
    path: "/Dispatcher/Unsubscribe";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<dispatcher_dispatcher_pb.WorkerUnsubscribeRequest>;
    requestDeserialize: grpc.deserialize<dispatcher_dispatcher_pb.WorkerUnsubscribeRequest>;
    responseSerialize: grpc.serialize<dispatcher_dispatcher_pb.WorkerUnsubscribeResponse>;
    responseDeserialize: grpc.deserialize<dispatcher_dispatcher_pb.WorkerUnsubscribeResponse>;
}

export const DispatcherService: IDispatcherService;

export interface IDispatcherServer extends grpc.UntypedServiceImplementation {
    register: grpc.handleUnaryCall<dispatcher_dispatcher_pb.WorkerRegisterRequest, dispatcher_dispatcher_pb.WorkerRegisterResponse>;
    listen: grpc.handleServerStreamingCall<dispatcher_dispatcher_pb.WorkerListenRequest, dispatcher_dispatcher_pb.AssignedAction>;
    sendActionEvent: grpc.handleUnaryCall<dispatcher_dispatcher_pb.ActionEvent, dispatcher_dispatcher_pb.ActionEventResponse>;
    unsubscribe: grpc.handleUnaryCall<dispatcher_dispatcher_pb.WorkerUnsubscribeRequest, dispatcher_dispatcher_pb.WorkerUnsubscribeResponse>;
}

export interface IDispatcherClient {
    register(request: dispatcher_dispatcher_pb.WorkerRegisterRequest, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.WorkerRegisterResponse) => void): grpc.ClientUnaryCall;
    register(request: dispatcher_dispatcher_pb.WorkerRegisterRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.WorkerRegisterResponse) => void): grpc.ClientUnaryCall;
    register(request: dispatcher_dispatcher_pb.WorkerRegisterRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.WorkerRegisterResponse) => void): grpc.ClientUnaryCall;
    listen(request: dispatcher_dispatcher_pb.WorkerListenRequest, options?: Partial<grpc.CallOptions>): grpc.ClientReadableStream<dispatcher_dispatcher_pb.AssignedAction>;
    listen(request: dispatcher_dispatcher_pb.WorkerListenRequest, metadata?: grpc.Metadata, options?: Partial<grpc.CallOptions>): grpc.ClientReadableStream<dispatcher_dispatcher_pb.AssignedAction>;
    sendActionEvent(request: dispatcher_dispatcher_pb.ActionEvent, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.ActionEventResponse) => void): grpc.ClientUnaryCall;
    sendActionEvent(request: dispatcher_dispatcher_pb.ActionEvent, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.ActionEventResponse) => void): grpc.ClientUnaryCall;
    sendActionEvent(request: dispatcher_dispatcher_pb.ActionEvent, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.ActionEventResponse) => void): grpc.ClientUnaryCall;
    unsubscribe(request: dispatcher_dispatcher_pb.WorkerUnsubscribeRequest, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.WorkerUnsubscribeResponse) => void): grpc.ClientUnaryCall;
    unsubscribe(request: dispatcher_dispatcher_pb.WorkerUnsubscribeRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.WorkerUnsubscribeResponse) => void): grpc.ClientUnaryCall;
    unsubscribe(request: dispatcher_dispatcher_pb.WorkerUnsubscribeRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.WorkerUnsubscribeResponse) => void): grpc.ClientUnaryCall;
}

export class DispatcherClient extends grpc.Client implements IDispatcherClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public register(request: dispatcher_dispatcher_pb.WorkerRegisterRequest, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.WorkerRegisterResponse) => void): grpc.ClientUnaryCall;
    public register(request: dispatcher_dispatcher_pb.WorkerRegisterRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.WorkerRegisterResponse) => void): grpc.ClientUnaryCall;
    public register(request: dispatcher_dispatcher_pb.WorkerRegisterRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.WorkerRegisterResponse) => void): grpc.ClientUnaryCall;
    public listen(request: dispatcher_dispatcher_pb.WorkerListenRequest, options?: Partial<grpc.CallOptions>): grpc.ClientReadableStream<dispatcher_dispatcher_pb.AssignedAction>;
    public listen(request: dispatcher_dispatcher_pb.WorkerListenRequest, metadata?: grpc.Metadata, options?: Partial<grpc.CallOptions>): grpc.ClientReadableStream<dispatcher_dispatcher_pb.AssignedAction>;
    public sendActionEvent(request: dispatcher_dispatcher_pb.ActionEvent, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.ActionEventResponse) => void): grpc.ClientUnaryCall;
    public sendActionEvent(request: dispatcher_dispatcher_pb.ActionEvent, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.ActionEventResponse) => void): grpc.ClientUnaryCall;
    public sendActionEvent(request: dispatcher_dispatcher_pb.ActionEvent, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.ActionEventResponse) => void): grpc.ClientUnaryCall;
    public unsubscribe(request: dispatcher_dispatcher_pb.WorkerUnsubscribeRequest, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.WorkerUnsubscribeResponse) => void): grpc.ClientUnaryCall;
    public unsubscribe(request: dispatcher_dispatcher_pb.WorkerUnsubscribeRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.WorkerUnsubscribeResponse) => void): grpc.ClientUnaryCall;
    public unsubscribe(request: dispatcher_dispatcher_pb.WorkerUnsubscribeRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: dispatcher_dispatcher_pb.WorkerUnsubscribeResponse) => void): grpc.ClientUnaryCall;
}
