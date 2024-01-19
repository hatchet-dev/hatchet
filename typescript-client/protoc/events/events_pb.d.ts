// package: 
// file: events/events.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

export class Event extends jspb.Message { 
    getTenantid(): string;
    setTenantid(value: string): Event;
    getEventid(): string;
    setEventid(value: string): Event;
    getKey(): string;
    setKey(value: string): Event;
    getPayload(): string;
    setPayload(value: string): Event;

    hasEventtimestamp(): boolean;
    clearEventtimestamp(): void;
    getEventtimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setEventtimestamp(value?: google_protobuf_timestamp_pb.Timestamp): Event;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Event.AsObject;
    static toObject(includeInstance: boolean, msg: Event): Event.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Event, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Event;
    static deserializeBinaryFromReader(message: Event, reader: jspb.BinaryReader): Event;
}

export namespace Event {
    export type AsObject = {
        tenantid: string,
        eventid: string,
        key: string,
        payload: string,
        eventtimestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    }
}

export class PushEventRequest extends jspb.Message { 
    getTenantid(): string;
    setTenantid(value: string): PushEventRequest;
    getKey(): string;
    setKey(value: string): PushEventRequest;
    getPayload(): string;
    setPayload(value: string): PushEventRequest;

    hasEventtimestamp(): boolean;
    clearEventtimestamp(): void;
    getEventtimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setEventtimestamp(value?: google_protobuf_timestamp_pb.Timestamp): PushEventRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): PushEventRequest.AsObject;
    static toObject(includeInstance: boolean, msg: PushEventRequest): PushEventRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: PushEventRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): PushEventRequest;
    static deserializeBinaryFromReader(message: PushEventRequest, reader: jspb.BinaryReader): PushEventRequest;
}

export namespace PushEventRequest {
    export type AsObject = {
        tenantid: string,
        key: string,
        payload: string,
        eventtimestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    }
}

export class ListEventRequest extends jspb.Message { 
    getTenantid(): string;
    setTenantid(value: string): ListEventRequest;
    getOffset(): number;
    setOffset(value: number): ListEventRequest;
    getKey(): string;
    setKey(value: string): ListEventRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ListEventRequest.AsObject;
    static toObject(includeInstance: boolean, msg: ListEventRequest): ListEventRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ListEventRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ListEventRequest;
    static deserializeBinaryFromReader(message: ListEventRequest, reader: jspb.BinaryReader): ListEventRequest;
}

export namespace ListEventRequest {
    export type AsObject = {
        tenantid: string,
        offset: number,
        key: string,
    }
}

export class ListEventResponse extends jspb.Message { 
    clearEventsList(): void;
    getEventsList(): Array<Event>;
    setEventsList(value: Array<Event>): ListEventResponse;
    addEvents(value?: Event, index?: number): Event;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ListEventResponse.AsObject;
    static toObject(includeInstance: boolean, msg: ListEventResponse): ListEventResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ListEventResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ListEventResponse;
    static deserializeBinaryFromReader(message: ListEventResponse, reader: jspb.BinaryReader): ListEventResponse;
}

export namespace ListEventResponse {
    export type AsObject = {
        eventsList: Array<Event.AsObject>,
    }
}

export class ReplayEventRequest extends jspb.Message { 
    getTenantid(): string;
    setTenantid(value: string): ReplayEventRequest;
    getEventid(): string;
    setEventid(value: string): ReplayEventRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ReplayEventRequest.AsObject;
    static toObject(includeInstance: boolean, msg: ReplayEventRequest): ReplayEventRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ReplayEventRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ReplayEventRequest;
    static deserializeBinaryFromReader(message: ReplayEventRequest, reader: jspb.BinaryReader): ReplayEventRequest;
}

export namespace ReplayEventRequest {
    export type AsObject = {
        tenantid: string,
        eventid: string,
    }
}
