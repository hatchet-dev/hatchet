// package: 
// file: workflows/workflows.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";
import * as google_protobuf_wrappers_pb from "google-protobuf/google/protobuf/wrappers_pb";

export class PutWorkflowRequest extends jspb.Message { 
    getTenantId(): string;
    setTenantId(value: string): PutWorkflowRequest;

    hasOpts(): boolean;
    clearOpts(): void;
    getOpts(): CreateWorkflowVersionOpts | undefined;
    setOpts(value?: CreateWorkflowVersionOpts): PutWorkflowRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): PutWorkflowRequest.AsObject;
    static toObject(includeInstance: boolean, msg: PutWorkflowRequest): PutWorkflowRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: PutWorkflowRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): PutWorkflowRequest;
    static deserializeBinaryFromReader(message: PutWorkflowRequest, reader: jspb.BinaryReader): PutWorkflowRequest;
}

export namespace PutWorkflowRequest {
    export type AsObject = {
        tenantId: string,
        opts?: CreateWorkflowVersionOpts.AsObject,
    }
}

export class CreateWorkflowVersionOpts extends jspb.Message { 
    getName(): string;
    setName(value: string): CreateWorkflowVersionOpts;
    getDescription(): string;
    setDescription(value: string): CreateWorkflowVersionOpts;
    getVersion(): string;
    setVersion(value: string): CreateWorkflowVersionOpts;
    clearEventTriggersList(): void;
    getEventTriggersList(): Array<string>;
    setEventTriggersList(value: Array<string>): CreateWorkflowVersionOpts;
    addEventTriggers(value: string, index?: number): string;
    clearCronTriggersList(): void;
    getCronTriggersList(): Array<string>;
    setCronTriggersList(value: Array<string>): CreateWorkflowVersionOpts;
    addCronTriggers(value: string, index?: number): string;
    clearScheduledTriggersList(): void;
    getScheduledTriggersList(): Array<google_protobuf_timestamp_pb.Timestamp>;
    setScheduledTriggersList(value: Array<google_protobuf_timestamp_pb.Timestamp>): CreateWorkflowVersionOpts;
    addScheduledTriggers(value?: google_protobuf_timestamp_pb.Timestamp, index?: number): google_protobuf_timestamp_pb.Timestamp;
    clearJobsList(): void;
    getJobsList(): Array<CreateWorkflowJobOpts>;
    setJobsList(value: Array<CreateWorkflowJobOpts>): CreateWorkflowVersionOpts;
    addJobs(value?: CreateWorkflowJobOpts, index?: number): CreateWorkflowJobOpts;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateWorkflowVersionOpts.AsObject;
    static toObject(includeInstance: boolean, msg: CreateWorkflowVersionOpts): CreateWorkflowVersionOpts.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateWorkflowVersionOpts, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateWorkflowVersionOpts;
    static deserializeBinaryFromReader(message: CreateWorkflowVersionOpts, reader: jspb.BinaryReader): CreateWorkflowVersionOpts;
}

export namespace CreateWorkflowVersionOpts {
    export type AsObject = {
        name: string,
        description: string,
        version: string,
        eventTriggersList: Array<string>,
        cronTriggersList: Array<string>,
        scheduledTriggersList: Array<google_protobuf_timestamp_pb.Timestamp.AsObject>,
        jobsList: Array<CreateWorkflowJobOpts.AsObject>,
    }
}

export class CreateWorkflowJobOpts extends jspb.Message { 
    getName(): string;
    setName(value: string): CreateWorkflowJobOpts;
    getDescription(): string;
    setDescription(value: string): CreateWorkflowJobOpts;
    getTimeout(): string;
    setTimeout(value: string): CreateWorkflowJobOpts;
    clearStepsList(): void;
    getStepsList(): Array<CreateWorkflowStepOpts>;
    setStepsList(value: Array<CreateWorkflowStepOpts>): CreateWorkflowJobOpts;
    addSteps(value?: CreateWorkflowStepOpts, index?: number): CreateWorkflowStepOpts;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateWorkflowJobOpts.AsObject;
    static toObject(includeInstance: boolean, msg: CreateWorkflowJobOpts): CreateWorkflowJobOpts.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateWorkflowJobOpts, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateWorkflowJobOpts;
    static deserializeBinaryFromReader(message: CreateWorkflowJobOpts, reader: jspb.BinaryReader): CreateWorkflowJobOpts;
}

export namespace CreateWorkflowJobOpts {
    export type AsObject = {
        name: string,
        description: string,
        timeout: string,
        stepsList: Array<CreateWorkflowStepOpts.AsObject>,
    }
}

export class CreateWorkflowStepOpts extends jspb.Message { 
    getReadableId(): string;
    setReadableId(value: string): CreateWorkflowStepOpts;
    getAction(): string;
    setAction(value: string): CreateWorkflowStepOpts;
    getTimeout(): string;
    setTimeout(value: string): CreateWorkflowStepOpts;
    getInputs(): string;
    setInputs(value: string): CreateWorkflowStepOpts;
    clearParentsList(): void;
    getParentsList(): Array<string>;
    setParentsList(value: Array<string>): CreateWorkflowStepOpts;
    addParents(value: string, index?: number): string;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateWorkflowStepOpts.AsObject;
    static toObject(includeInstance: boolean, msg: CreateWorkflowStepOpts): CreateWorkflowStepOpts.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateWorkflowStepOpts, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateWorkflowStepOpts;
    static deserializeBinaryFromReader(message: CreateWorkflowStepOpts, reader: jspb.BinaryReader): CreateWorkflowStepOpts;
}

export namespace CreateWorkflowStepOpts {
    export type AsObject = {
        readableId: string,
        action: string,
        timeout: string,
        inputs: string,
        parentsList: Array<string>,
    }
}

export class ListWorkflowsRequest extends jspb.Message { 
    getTenantId(): string;
    setTenantId(value: string): ListWorkflowsRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ListWorkflowsRequest.AsObject;
    static toObject(includeInstance: boolean, msg: ListWorkflowsRequest): ListWorkflowsRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ListWorkflowsRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ListWorkflowsRequest;
    static deserializeBinaryFromReader(message: ListWorkflowsRequest, reader: jspb.BinaryReader): ListWorkflowsRequest;
}

export namespace ListWorkflowsRequest {
    export type AsObject = {
        tenantId: string,
    }
}

export class ScheduleWorkflowRequest extends jspb.Message { 
    getTenantId(): string;
    setTenantId(value: string): ScheduleWorkflowRequest;
    getWorkflowId(): string;
    setWorkflowId(value: string): ScheduleWorkflowRequest;
    clearSchedulesList(): void;
    getSchedulesList(): Array<google_protobuf_timestamp_pb.Timestamp>;
    setSchedulesList(value: Array<google_protobuf_timestamp_pb.Timestamp>): ScheduleWorkflowRequest;
    addSchedules(value?: google_protobuf_timestamp_pb.Timestamp, index?: number): google_protobuf_timestamp_pb.Timestamp;
    getInput(): string;
    setInput(value: string): ScheduleWorkflowRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ScheduleWorkflowRequest.AsObject;
    static toObject(includeInstance: boolean, msg: ScheduleWorkflowRequest): ScheduleWorkflowRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ScheduleWorkflowRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ScheduleWorkflowRequest;
    static deserializeBinaryFromReader(message: ScheduleWorkflowRequest, reader: jspb.BinaryReader): ScheduleWorkflowRequest;
}

export namespace ScheduleWorkflowRequest {
    export type AsObject = {
        tenantId: string,
        workflowId: string,
        schedulesList: Array<google_protobuf_timestamp_pb.Timestamp.AsObject>,
        input: string,
    }
}

export class ListWorkflowsResponse extends jspb.Message { 
    clearWorkflowsList(): void;
    getWorkflowsList(): Array<Workflow>;
    setWorkflowsList(value: Array<Workflow>): ListWorkflowsResponse;
    addWorkflows(value?: Workflow, index?: number): Workflow;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ListWorkflowsResponse.AsObject;
    static toObject(includeInstance: boolean, msg: ListWorkflowsResponse): ListWorkflowsResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ListWorkflowsResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ListWorkflowsResponse;
    static deserializeBinaryFromReader(message: ListWorkflowsResponse, reader: jspb.BinaryReader): ListWorkflowsResponse;
}

export namespace ListWorkflowsResponse {
    export type AsObject = {
        workflowsList: Array<Workflow.AsObject>,
    }
}

export class ListWorkflowsForEventRequest extends jspb.Message { 
    getTenantId(): string;
    setTenantId(value: string): ListWorkflowsForEventRequest;
    getEventKey(): string;
    setEventKey(value: string): ListWorkflowsForEventRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ListWorkflowsForEventRequest.AsObject;
    static toObject(includeInstance: boolean, msg: ListWorkflowsForEventRequest): ListWorkflowsForEventRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ListWorkflowsForEventRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ListWorkflowsForEventRequest;
    static deserializeBinaryFromReader(message: ListWorkflowsForEventRequest, reader: jspb.BinaryReader): ListWorkflowsForEventRequest;
}

export namespace ListWorkflowsForEventRequest {
    export type AsObject = {
        tenantId: string,
        eventKey: string,
    }
}

export class Workflow extends jspb.Message { 
    getId(): string;
    setId(value: string): Workflow;

    hasCreatedAt(): boolean;
    clearCreatedAt(): void;
    getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): Workflow;

    hasUpdatedAt(): boolean;
    clearUpdatedAt(): void;
    getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): Workflow;
    getTenantId(): string;
    setTenantId(value: string): Workflow;
    getName(): string;
    setName(value: string): Workflow;

    hasDescription(): boolean;
    clearDescription(): void;
    getDescription(): google_protobuf_wrappers_pb.StringValue | undefined;
    setDescription(value?: google_protobuf_wrappers_pb.StringValue): Workflow;
    clearVersionsList(): void;
    getVersionsList(): Array<WorkflowVersion>;
    setVersionsList(value: Array<WorkflowVersion>): Workflow;
    addVersions(value?: WorkflowVersion, index?: number): WorkflowVersion;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Workflow.AsObject;
    static toObject(includeInstance: boolean, msg: Workflow): Workflow.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Workflow, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Workflow;
    static deserializeBinaryFromReader(message: Workflow, reader: jspb.BinaryReader): Workflow;
}

export namespace Workflow {
    export type AsObject = {
        id: string,
        createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        tenantId: string,
        name: string,
        description?: google_protobuf_wrappers_pb.StringValue.AsObject,
        versionsList: Array<WorkflowVersion.AsObject>,
    }
}

export class WorkflowVersion extends jspb.Message { 
    getId(): string;
    setId(value: string): WorkflowVersion;

    hasCreatedAt(): boolean;
    clearCreatedAt(): void;
    getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): WorkflowVersion;

    hasUpdatedAt(): boolean;
    clearUpdatedAt(): void;
    getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): WorkflowVersion;
    getVersion(): string;
    setVersion(value: string): WorkflowVersion;
    getOrder(): number;
    setOrder(value: number): WorkflowVersion;
    getWorkflowId(): string;
    setWorkflowId(value: string): WorkflowVersion;

    hasTriggers(): boolean;
    clearTriggers(): void;
    getTriggers(): WorkflowTriggers | undefined;
    setTriggers(value?: WorkflowTriggers): WorkflowVersion;
    clearJobsList(): void;
    getJobsList(): Array<Job>;
    setJobsList(value: Array<Job>): WorkflowVersion;
    addJobs(value?: Job, index?: number): Job;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): WorkflowVersion.AsObject;
    static toObject(includeInstance: boolean, msg: WorkflowVersion): WorkflowVersion.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: WorkflowVersion, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): WorkflowVersion;
    static deserializeBinaryFromReader(message: WorkflowVersion, reader: jspb.BinaryReader): WorkflowVersion;
}

export namespace WorkflowVersion {
    export type AsObject = {
        id: string,
        createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        version: string,
        order: number,
        workflowId: string,
        triggers?: WorkflowTriggers.AsObject,
        jobsList: Array<Job.AsObject>,
    }
}

export class WorkflowTriggers extends jspb.Message { 
    getId(): string;
    setId(value: string): WorkflowTriggers;

    hasCreatedAt(): boolean;
    clearCreatedAt(): void;
    getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): WorkflowTriggers;

    hasUpdatedAt(): boolean;
    clearUpdatedAt(): void;
    getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): WorkflowTriggers;
    getWorkflowVersionId(): string;
    setWorkflowVersionId(value: string): WorkflowTriggers;
    getTenantId(): string;
    setTenantId(value: string): WorkflowTriggers;
    clearEventsList(): void;
    getEventsList(): Array<WorkflowTriggerEventRef>;
    setEventsList(value: Array<WorkflowTriggerEventRef>): WorkflowTriggers;
    addEvents(value?: WorkflowTriggerEventRef, index?: number): WorkflowTriggerEventRef;
    clearCronsList(): void;
    getCronsList(): Array<WorkflowTriggerCronRef>;
    setCronsList(value: Array<WorkflowTriggerCronRef>): WorkflowTriggers;
    addCrons(value?: WorkflowTriggerCronRef, index?: number): WorkflowTriggerCronRef;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): WorkflowTriggers.AsObject;
    static toObject(includeInstance: boolean, msg: WorkflowTriggers): WorkflowTriggers.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: WorkflowTriggers, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): WorkflowTriggers;
    static deserializeBinaryFromReader(message: WorkflowTriggers, reader: jspb.BinaryReader): WorkflowTriggers;
}

export namespace WorkflowTriggers {
    export type AsObject = {
        id: string,
        createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        workflowVersionId: string,
        tenantId: string,
        eventsList: Array<WorkflowTriggerEventRef.AsObject>,
        cronsList: Array<WorkflowTriggerCronRef.AsObject>,
    }
}

export class WorkflowTriggerEventRef extends jspb.Message { 
    getParentId(): string;
    setParentId(value: string): WorkflowTriggerEventRef;
    getEventKey(): string;
    setEventKey(value: string): WorkflowTriggerEventRef;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): WorkflowTriggerEventRef.AsObject;
    static toObject(includeInstance: boolean, msg: WorkflowTriggerEventRef): WorkflowTriggerEventRef.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: WorkflowTriggerEventRef, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): WorkflowTriggerEventRef;
    static deserializeBinaryFromReader(message: WorkflowTriggerEventRef, reader: jspb.BinaryReader): WorkflowTriggerEventRef;
}

export namespace WorkflowTriggerEventRef {
    export type AsObject = {
        parentId: string,
        eventKey: string,
    }
}

export class WorkflowTriggerCronRef extends jspb.Message { 
    getParentId(): string;
    setParentId(value: string): WorkflowTriggerCronRef;
    getCron(): string;
    setCron(value: string): WorkflowTriggerCronRef;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): WorkflowTriggerCronRef.AsObject;
    static toObject(includeInstance: boolean, msg: WorkflowTriggerCronRef): WorkflowTriggerCronRef.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: WorkflowTriggerCronRef, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): WorkflowTriggerCronRef;
    static deserializeBinaryFromReader(message: WorkflowTriggerCronRef, reader: jspb.BinaryReader): WorkflowTriggerCronRef;
}

export namespace WorkflowTriggerCronRef {
    export type AsObject = {
        parentId: string,
        cron: string,
    }
}

export class Job extends jspb.Message { 
    getId(): string;
    setId(value: string): Job;

    hasCreatedAt(): boolean;
    clearCreatedAt(): void;
    getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): Job;

    hasUpdatedAt(): boolean;
    clearUpdatedAt(): void;
    getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): Job;
    getTenantId(): string;
    setTenantId(value: string): Job;
    getWorkflowVersionId(): string;
    setWorkflowVersionId(value: string): Job;
    getName(): string;
    setName(value: string): Job;

    hasDescription(): boolean;
    clearDescription(): void;
    getDescription(): google_protobuf_wrappers_pb.StringValue | undefined;
    setDescription(value?: google_protobuf_wrappers_pb.StringValue): Job;
    clearStepsList(): void;
    getStepsList(): Array<Step>;
    setStepsList(value: Array<Step>): Job;
    addSteps(value?: Step, index?: number): Step;

    hasTimeout(): boolean;
    clearTimeout(): void;
    getTimeout(): google_protobuf_wrappers_pb.StringValue | undefined;
    setTimeout(value?: google_protobuf_wrappers_pb.StringValue): Job;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Job.AsObject;
    static toObject(includeInstance: boolean, msg: Job): Job.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Job, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Job;
    static deserializeBinaryFromReader(message: Job, reader: jspb.BinaryReader): Job;
}

export namespace Job {
    export type AsObject = {
        id: string,
        createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        tenantId: string,
        workflowVersionId: string,
        name: string,
        description?: google_protobuf_wrappers_pb.StringValue.AsObject,
        stepsList: Array<Step.AsObject>,
        timeout?: google_protobuf_wrappers_pb.StringValue.AsObject,
    }
}

export class Step extends jspb.Message { 
    getId(): string;
    setId(value: string): Step;

    hasCreatedAt(): boolean;
    clearCreatedAt(): void;
    getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): Step;

    hasUpdatedAt(): boolean;
    clearUpdatedAt(): void;
    getUpdatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setUpdatedAt(value?: google_protobuf_timestamp_pb.Timestamp): Step;

    hasReadableId(): boolean;
    clearReadableId(): void;
    getReadableId(): google_protobuf_wrappers_pb.StringValue | undefined;
    setReadableId(value?: google_protobuf_wrappers_pb.StringValue): Step;
    getTenantId(): string;
    setTenantId(value: string): Step;
    getJobId(): string;
    setJobId(value: string): Step;
    getAction(): string;
    setAction(value: string): Step;

    hasTimeout(): boolean;
    clearTimeout(): void;
    getTimeout(): google_protobuf_wrappers_pb.StringValue | undefined;
    setTimeout(value?: google_protobuf_wrappers_pb.StringValue): Step;
    clearParentsList(): void;
    getParentsList(): Array<string>;
    setParentsList(value: Array<string>): Step;
    addParents(value: string, index?: number): string;
    clearChildrenList(): void;
    getChildrenList(): Array<string>;
    setChildrenList(value: Array<string>): Step;
    addChildren(value: string, index?: number): string;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Step.AsObject;
    static toObject(includeInstance: boolean, msg: Step): Step.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Step, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Step;
    static deserializeBinaryFromReader(message: Step, reader: jspb.BinaryReader): Step;
}

export namespace Step {
    export type AsObject = {
        id: string,
        createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        updatedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        readableId?: google_protobuf_wrappers_pb.StringValue.AsObject,
        tenantId: string,
        jobId: string,
        action: string,
        timeout?: google_protobuf_wrappers_pb.StringValue.AsObject,
        parentsList: Array<string>,
        childrenList: Array<string>,
    }
}

export class DeleteWorkflowRequest extends jspb.Message { 
    getTenantId(): string;
    setTenantId(value: string): DeleteWorkflowRequest;
    getWorkflowId(): string;
    setWorkflowId(value: string): DeleteWorkflowRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteWorkflowRequest.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteWorkflowRequest): DeleteWorkflowRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteWorkflowRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteWorkflowRequest;
    static deserializeBinaryFromReader(message: DeleteWorkflowRequest, reader: jspb.BinaryReader): DeleteWorkflowRequest;
}

export namespace DeleteWorkflowRequest {
    export type AsObject = {
        tenantId: string,
        workflowId: string,
    }
}

export class GetWorkflowByNameRequest extends jspb.Message { 
    getTenantId(): string;
    setTenantId(value: string): GetWorkflowByNameRequest;
    getName(): string;
    setName(value: string): GetWorkflowByNameRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetWorkflowByNameRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetWorkflowByNameRequest): GetWorkflowByNameRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetWorkflowByNameRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetWorkflowByNameRequest;
    static deserializeBinaryFromReader(message: GetWorkflowByNameRequest, reader: jspb.BinaryReader): GetWorkflowByNameRequest;
}

export namespace GetWorkflowByNameRequest {
    export type AsObject = {
        tenantId: string,
        name: string,
    }
}
