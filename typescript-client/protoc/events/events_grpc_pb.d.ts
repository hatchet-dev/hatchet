// package: 
// file: events/events.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as events_events_pb from "../events/events_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

interface IEventsServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    push: IEventsServiceService_IPush;
    list: IEventsServiceService_IList;
    replaySingleEvent: IEventsServiceService_IReplaySingleEvent;
}

interface IEventsServiceService_IPush extends grpc.MethodDefinition<events_events_pb.PushEventRequest, events_events_pb.Event> {
    path: "/EventsService/Push";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<events_events_pb.PushEventRequest>;
    requestDeserialize: grpc.deserialize<events_events_pb.PushEventRequest>;
    responseSerialize: grpc.serialize<events_events_pb.Event>;
    responseDeserialize: grpc.deserialize<events_events_pb.Event>;
}
interface IEventsServiceService_IList extends grpc.MethodDefinition<events_events_pb.ListEventRequest, events_events_pb.ListEventResponse> {
    path: "/EventsService/List";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<events_events_pb.ListEventRequest>;
    requestDeserialize: grpc.deserialize<events_events_pb.ListEventRequest>;
    responseSerialize: grpc.serialize<events_events_pb.ListEventResponse>;
    responseDeserialize: grpc.deserialize<events_events_pb.ListEventResponse>;
}
interface IEventsServiceService_IReplaySingleEvent extends grpc.MethodDefinition<events_events_pb.ReplayEventRequest, events_events_pb.Event> {
    path: "/EventsService/ReplaySingleEvent";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<events_events_pb.ReplayEventRequest>;
    requestDeserialize: grpc.deserialize<events_events_pb.ReplayEventRequest>;
    responseSerialize: grpc.serialize<events_events_pb.Event>;
    responseDeserialize: grpc.deserialize<events_events_pb.Event>;
}

export const EventsServiceService: IEventsServiceService;

export interface IEventsServiceServer extends grpc.UntypedServiceImplementation {
    push: grpc.handleUnaryCall<events_events_pb.PushEventRequest, events_events_pb.Event>;
    list: grpc.handleUnaryCall<events_events_pb.ListEventRequest, events_events_pb.ListEventResponse>;
    replaySingleEvent: grpc.handleUnaryCall<events_events_pb.ReplayEventRequest, events_events_pb.Event>;
}

export interface IEventsServiceClient {
    push(request: events_events_pb.PushEventRequest, callback: (error: grpc.ServiceError | null, response: events_events_pb.Event) => void): grpc.ClientUnaryCall;
    push(request: events_events_pb.PushEventRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: events_events_pb.Event) => void): grpc.ClientUnaryCall;
    push(request: events_events_pb.PushEventRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: events_events_pb.Event) => void): grpc.ClientUnaryCall;
    list(request: events_events_pb.ListEventRequest, callback: (error: grpc.ServiceError | null, response: events_events_pb.ListEventResponse) => void): grpc.ClientUnaryCall;
    list(request: events_events_pb.ListEventRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: events_events_pb.ListEventResponse) => void): grpc.ClientUnaryCall;
    list(request: events_events_pb.ListEventRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: events_events_pb.ListEventResponse) => void): grpc.ClientUnaryCall;
    replaySingleEvent(request: events_events_pb.ReplayEventRequest, callback: (error: grpc.ServiceError | null, response: events_events_pb.Event) => void): grpc.ClientUnaryCall;
    replaySingleEvent(request: events_events_pb.ReplayEventRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: events_events_pb.Event) => void): grpc.ClientUnaryCall;
    replaySingleEvent(request: events_events_pb.ReplayEventRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: events_events_pb.Event) => void): grpc.ClientUnaryCall;
}

export class EventsServiceClient extends grpc.Client implements IEventsServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public push(request: events_events_pb.PushEventRequest, callback: (error: grpc.ServiceError | null, response: events_events_pb.Event) => void): grpc.ClientUnaryCall;
    public push(request: events_events_pb.PushEventRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: events_events_pb.Event) => void): grpc.ClientUnaryCall;
    public push(request: events_events_pb.PushEventRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: events_events_pb.Event) => void): grpc.ClientUnaryCall;
    public list(request: events_events_pb.ListEventRequest, callback: (error: grpc.ServiceError | null, response: events_events_pb.ListEventResponse) => void): grpc.ClientUnaryCall;
    public list(request: events_events_pb.ListEventRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: events_events_pb.ListEventResponse) => void): grpc.ClientUnaryCall;
    public list(request: events_events_pb.ListEventRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: events_events_pb.ListEventResponse) => void): grpc.ClientUnaryCall;
    public replaySingleEvent(request: events_events_pb.ReplayEventRequest, callback: (error: grpc.ServiceError | null, response: events_events_pb.Event) => void): grpc.ClientUnaryCall;
    public replaySingleEvent(request: events_events_pb.ReplayEventRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: events_events_pb.Event) => void): grpc.ClientUnaryCall;
    public replaySingleEvent(request: events_events_pb.ReplayEventRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: events_events_pb.Event) => void): grpc.ClientUnaryCall;
}
