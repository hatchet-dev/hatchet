// package: 
// file: dispatcher/dispatcher.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

export class WorkerRegisterRequest extends jspb.Message { 
    getTenantid(): string;
    setTenantid(value: string): WorkerRegisterRequest;
    getWorkername(): string;
    setWorkername(value: string): WorkerRegisterRequest;
    clearActionsList(): void;
    getActionsList(): Array<string>;
    setActionsList(value: Array<string>): WorkerRegisterRequest;
    addActions(value: string, index?: number): string;
    clearServicesList(): void;
    getServicesList(): Array<string>;
    setServicesList(value: Array<string>): WorkerRegisterRequest;
    addServices(value: string, index?: number): string;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): WorkerRegisterRequest.AsObject;
    static toObject(includeInstance: boolean, msg: WorkerRegisterRequest): WorkerRegisterRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: WorkerRegisterRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): WorkerRegisterRequest;
    static deserializeBinaryFromReader(message: WorkerRegisterRequest, reader: jspb.BinaryReader): WorkerRegisterRequest;
}

export namespace WorkerRegisterRequest {
    export type AsObject = {
        tenantid: string,
        workername: string,
        actionsList: Array<string>,
        servicesList: Array<string>,
    }
}

export class WorkerRegisterResponse extends jspb.Message { 
    getTenantid(): string;
    setTenantid(value: string): WorkerRegisterResponse;
    getWorkerid(): string;
    setWorkerid(value: string): WorkerRegisterResponse;
    getWorkername(): string;
    setWorkername(value: string): WorkerRegisterResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): WorkerRegisterResponse.AsObject;
    static toObject(includeInstance: boolean, msg: WorkerRegisterResponse): WorkerRegisterResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: WorkerRegisterResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): WorkerRegisterResponse;
    static deserializeBinaryFromReader(message: WorkerRegisterResponse, reader: jspb.BinaryReader): WorkerRegisterResponse;
}

export namespace WorkerRegisterResponse {
    export type AsObject = {
        tenantid: string,
        workerid: string,
        workername: string,
    }
}

export class AssignedAction extends jspb.Message { 
    getTenantid(): string;
    setTenantid(value: string): AssignedAction;
    getJobid(): string;
    setJobid(value: string): AssignedAction;
    getJobname(): string;
    setJobname(value: string): AssignedAction;
    getJobrunid(): string;
    setJobrunid(value: string): AssignedAction;
    getStepid(): string;
    setStepid(value: string): AssignedAction;
    getSteprunid(): string;
    setSteprunid(value: string): AssignedAction;
    getActionid(): string;
    setActionid(value: string): AssignedAction;
    getActiontype(): ActionType;
    setActiontype(value: ActionType): AssignedAction;
    getActionpayload(): string;
    setActionpayload(value: string): AssignedAction;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): AssignedAction.AsObject;
    static toObject(includeInstance: boolean, msg: AssignedAction): AssignedAction.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: AssignedAction, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): AssignedAction;
    static deserializeBinaryFromReader(message: AssignedAction, reader: jspb.BinaryReader): AssignedAction;
}

export namespace AssignedAction {
    export type AsObject = {
        tenantid: string,
        jobid: string,
        jobname: string,
        jobrunid: string,
        stepid: string,
        steprunid: string,
        actionid: string,
        actiontype: ActionType,
        actionpayload: string,
    }
}

export class WorkerListenRequest extends jspb.Message { 
    getTenantid(): string;
    setTenantid(value: string): WorkerListenRequest;
    getWorkerid(): string;
    setWorkerid(value: string): WorkerListenRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): WorkerListenRequest.AsObject;
    static toObject(includeInstance: boolean, msg: WorkerListenRequest): WorkerListenRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: WorkerListenRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): WorkerListenRequest;
    static deserializeBinaryFromReader(message: WorkerListenRequest, reader: jspb.BinaryReader): WorkerListenRequest;
}

export namespace WorkerListenRequest {
    export type AsObject = {
        tenantid: string,
        workerid: string,
    }
}

export class WorkerUnsubscribeRequest extends jspb.Message { 
    getTenantid(): string;
    setTenantid(value: string): WorkerUnsubscribeRequest;
    getWorkerid(): string;
    setWorkerid(value: string): WorkerUnsubscribeRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): WorkerUnsubscribeRequest.AsObject;
    static toObject(includeInstance: boolean, msg: WorkerUnsubscribeRequest): WorkerUnsubscribeRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: WorkerUnsubscribeRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): WorkerUnsubscribeRequest;
    static deserializeBinaryFromReader(message: WorkerUnsubscribeRequest, reader: jspb.BinaryReader): WorkerUnsubscribeRequest;
}

export namespace WorkerUnsubscribeRequest {
    export type AsObject = {
        tenantid: string,
        workerid: string,
    }
}

export class WorkerUnsubscribeResponse extends jspb.Message { 
    getTenantid(): string;
    setTenantid(value: string): WorkerUnsubscribeResponse;
    getWorkerid(): string;
    setWorkerid(value: string): WorkerUnsubscribeResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): WorkerUnsubscribeResponse.AsObject;
    static toObject(includeInstance: boolean, msg: WorkerUnsubscribeResponse): WorkerUnsubscribeResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: WorkerUnsubscribeResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): WorkerUnsubscribeResponse;
    static deserializeBinaryFromReader(message: WorkerUnsubscribeResponse, reader: jspb.BinaryReader): WorkerUnsubscribeResponse;
}

export namespace WorkerUnsubscribeResponse {
    export type AsObject = {
        tenantid: string,
        workerid: string,
    }
}

export class ActionEvent extends jspb.Message { 
    getTenantid(): string;
    setTenantid(value: string): ActionEvent;
    getWorkerid(): string;
    setWorkerid(value: string): ActionEvent;
    getJobid(): string;
    setJobid(value: string): ActionEvent;
    getJobrunid(): string;
    setJobrunid(value: string): ActionEvent;
    getStepid(): string;
    setStepid(value: string): ActionEvent;
    getSteprunid(): string;
    setSteprunid(value: string): ActionEvent;
    getActionid(): string;
    setActionid(value: string): ActionEvent;

    hasEventtimestamp(): boolean;
    clearEventtimestamp(): void;
    getEventtimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setEventtimestamp(value?: google_protobuf_timestamp_pb.Timestamp): ActionEvent;
    getEventtype(): ActionEventType;
    setEventtype(value: ActionEventType): ActionEvent;
    getEventpayload(): string;
    setEventpayload(value: string): ActionEvent;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ActionEvent.AsObject;
    static toObject(includeInstance: boolean, msg: ActionEvent): ActionEvent.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ActionEvent, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ActionEvent;
    static deserializeBinaryFromReader(message: ActionEvent, reader: jspb.BinaryReader): ActionEvent;
}

export namespace ActionEvent {
    export type AsObject = {
        tenantid: string,
        workerid: string,
        jobid: string,
        jobrunid: string,
        stepid: string,
        steprunid: string,
        actionid: string,
        eventtimestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        eventtype: ActionEventType,
        eventpayload: string,
    }
}

export class ActionEventResponse extends jspb.Message { 
    getTenantid(): string;
    setTenantid(value: string): ActionEventResponse;
    getWorkerid(): string;
    setWorkerid(value: string): ActionEventResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ActionEventResponse.AsObject;
    static toObject(includeInstance: boolean, msg: ActionEventResponse): ActionEventResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ActionEventResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ActionEventResponse;
    static deserializeBinaryFromReader(message: ActionEventResponse, reader: jspb.BinaryReader): ActionEventResponse;
}

export namespace ActionEventResponse {
    export type AsObject = {
        tenantid: string,
        workerid: string,
    }
}

export enum ActionType {
    START_STEP_RUN = 0,
    CANCEL_STEP_RUN = 1,
}

export enum ActionEventType {
    STEP_EVENT_TYPE_UNKNOWN = 0,
    STEP_EVENT_TYPE_STARTED = 1,
    STEP_EVENT_TYPE_COMPLETED = 2,
    STEP_EVENT_TYPE_FAILED = 3,
}
