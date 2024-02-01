/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "";

export enum ActionType {
  START_STEP_RUN = 0,
  CANCEL_STEP_RUN = 1,
  UNRECOGNIZED = -1,
}

export function actionTypeFromJSON(object: any): ActionType {
  switch (object) {
    case 0:
    case "START_STEP_RUN":
      return ActionType.START_STEP_RUN;
    case 1:
    case "CANCEL_STEP_RUN":
      return ActionType.CANCEL_STEP_RUN;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ActionType.UNRECOGNIZED;
  }
}

export function actionTypeToJSON(object: ActionType): string {
  switch (object) {
    case ActionType.START_STEP_RUN:
      return "START_STEP_RUN";
    case ActionType.CANCEL_STEP_RUN:
      return "CANCEL_STEP_RUN";
    case ActionType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum ActionEventType {
  STEP_EVENT_TYPE_UNKNOWN = 0,
  STEP_EVENT_TYPE_STARTED = 1,
  STEP_EVENT_TYPE_COMPLETED = 2,
  STEP_EVENT_TYPE_FAILED = 3,
  UNRECOGNIZED = -1,
}

export function actionEventTypeFromJSON(object: any): ActionEventType {
  switch (object) {
    case 0:
    case "STEP_EVENT_TYPE_UNKNOWN":
      return ActionEventType.STEP_EVENT_TYPE_UNKNOWN;
    case 1:
    case "STEP_EVENT_TYPE_STARTED":
      return ActionEventType.STEP_EVENT_TYPE_STARTED;
    case 2:
    case "STEP_EVENT_TYPE_COMPLETED":
      return ActionEventType.STEP_EVENT_TYPE_COMPLETED;
    case 3:
    case "STEP_EVENT_TYPE_FAILED":
      return ActionEventType.STEP_EVENT_TYPE_FAILED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ActionEventType.UNRECOGNIZED;
  }
}

export function actionEventTypeToJSON(object: ActionEventType): string {
  switch (object) {
    case ActionEventType.STEP_EVENT_TYPE_UNKNOWN:
      return "STEP_EVENT_TYPE_UNKNOWN";
    case ActionEventType.STEP_EVENT_TYPE_STARTED:
      return "STEP_EVENT_TYPE_STARTED";
    case ActionEventType.STEP_EVENT_TYPE_COMPLETED:
      return "STEP_EVENT_TYPE_COMPLETED";
    case ActionEventType.STEP_EVENT_TYPE_FAILED:
      return "STEP_EVENT_TYPE_FAILED";
    case ActionEventType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface WorkerRegisterRequest {
  /** the name of the worker */
  workerName: string;
  /** a list of actions that this worker can run */
  actions: string[];
  /** (optional) the services for this worker */
  services: string[];
}

export interface WorkerRegisterResponse {
  /** the tenant id */
  tenantId: string;
  /** the id of the worker */
  workerId: string;
  /** the name of the worker */
  workerName: string;
}

export interface AssignedAction {
  /** the tenant id */
  tenantId: string;
  /** the job id */
  jobId: string;
  /** the job name */
  jobName: string;
  /** the job run id */
  jobRunId: string;
  /** the step id */
  stepId: string;
  /** the step run id */
  stepRunId: string;
  /** the action id */
  actionId: string;
  /** the action type */
  actionType: ActionType;
  /** the action payload */
  actionPayload: string;
}

export interface WorkerListenRequest {
  /** the id of the worker */
  workerId: string;
}

export interface WorkerUnsubscribeRequest {
  /** the id of the worker */
  workerId: string;
}

export interface WorkerUnsubscribeResponse {
  /** the tenant id to unsubscribe from */
  tenantId: string;
  /** the id of the worker */
  workerId: string;
}

export interface ActionEvent {
  /** the id of the worker */
  workerId: string;
  /** the id of the job */
  jobId: string;
  /** the job run id */
  jobRunId: string;
  /** the id of the step */
  stepId: string;
  /** the step run id */
  stepRunId: string;
  /** the action id */
  actionId: string;
  eventTimestamp:
    | Date
    | undefined;
  /** the step event type */
  eventType: ActionEventType;
  /** the event payload */
  eventPayload: string;
}

export interface ActionEventResponse {
  /** the tenant id */
  tenantId: string;
  /** the id of the worker */
  workerId: string;
}

function createBaseWorkerRegisterRequest(): WorkerRegisterRequest {
  return { workerName: "", actions: [], services: [] };
}

export const WorkerRegisterRequest = {
  encode(message: WorkerRegisterRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.workerName !== "") {
      writer.uint32(10).string(message.workerName);
    }
    for (const v of message.actions) {
      writer.uint32(18).string(v!);
    }
    for (const v of message.services) {
      writer.uint32(26).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): WorkerRegisterRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWorkerRegisterRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.workerName = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.actions.push(reader.string());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.services.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): WorkerRegisterRequest {
    return {
      workerName: isSet(object.workerName) ? globalThis.String(object.workerName) : "",
      actions: globalThis.Array.isArray(object?.actions) ? object.actions.map((e: any) => globalThis.String(e)) : [],
      services: globalThis.Array.isArray(object?.services) ? object.services.map((e: any) => globalThis.String(e)) : [],
    };
  },

  toJSON(message: WorkerRegisterRequest): unknown {
    const obj: any = {};
    if (message.workerName !== "") {
      obj.workerName = message.workerName;
    }
    if (message.actions?.length) {
      obj.actions = message.actions;
    }
    if (message.services?.length) {
      obj.services = message.services;
    }
    return obj;
  },

  create(base?: DeepPartial<WorkerRegisterRequest>): WorkerRegisterRequest {
    return WorkerRegisterRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<WorkerRegisterRequest>): WorkerRegisterRequest {
    const message = createBaseWorkerRegisterRequest();
    message.workerName = object.workerName ?? "";
    message.actions = object.actions?.map((e) => e) || [];
    message.services = object.services?.map((e) => e) || [];
    return message;
  },
};

function createBaseWorkerRegisterResponse(): WorkerRegisterResponse {
  return { tenantId: "", workerId: "", workerName: "" };
}

export const WorkerRegisterResponse = {
  encode(message: WorkerRegisterResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.tenantId !== "") {
      writer.uint32(10).string(message.tenantId);
    }
    if (message.workerId !== "") {
      writer.uint32(18).string(message.workerId);
    }
    if (message.workerName !== "") {
      writer.uint32(26).string(message.workerName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): WorkerRegisterResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWorkerRegisterResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.tenantId = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.workerId = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.workerName = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): WorkerRegisterResponse {
    return {
      tenantId: isSet(object.tenantId) ? globalThis.String(object.tenantId) : "",
      workerId: isSet(object.workerId) ? globalThis.String(object.workerId) : "",
      workerName: isSet(object.workerName) ? globalThis.String(object.workerName) : "",
    };
  },

  toJSON(message: WorkerRegisterResponse): unknown {
    const obj: any = {};
    if (message.tenantId !== "") {
      obj.tenantId = message.tenantId;
    }
    if (message.workerId !== "") {
      obj.workerId = message.workerId;
    }
    if (message.workerName !== "") {
      obj.workerName = message.workerName;
    }
    return obj;
  },

  create(base?: DeepPartial<WorkerRegisterResponse>): WorkerRegisterResponse {
    return WorkerRegisterResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<WorkerRegisterResponse>): WorkerRegisterResponse {
    const message = createBaseWorkerRegisterResponse();
    message.tenantId = object.tenantId ?? "";
    message.workerId = object.workerId ?? "";
    message.workerName = object.workerName ?? "";
    return message;
  },
};

function createBaseAssignedAction(): AssignedAction {
  return {
    tenantId: "",
    jobId: "",
    jobName: "",
    jobRunId: "",
    stepId: "",
    stepRunId: "",
    actionId: "",
    actionType: 0,
    actionPayload: "",
  };
}

export const AssignedAction = {
  encode(message: AssignedAction, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.tenantId !== "") {
      writer.uint32(10).string(message.tenantId);
    }
    if (message.jobId !== "") {
      writer.uint32(18).string(message.jobId);
    }
    if (message.jobName !== "") {
      writer.uint32(26).string(message.jobName);
    }
    if (message.jobRunId !== "") {
      writer.uint32(34).string(message.jobRunId);
    }
    if (message.stepId !== "") {
      writer.uint32(42).string(message.stepId);
    }
    if (message.stepRunId !== "") {
      writer.uint32(50).string(message.stepRunId);
    }
    if (message.actionId !== "") {
      writer.uint32(58).string(message.actionId);
    }
    if (message.actionType !== 0) {
      writer.uint32(64).int32(message.actionType);
    }
    if (message.actionPayload !== "") {
      writer.uint32(74).string(message.actionPayload);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AssignedAction {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAssignedAction();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.tenantId = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.jobId = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.jobName = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.jobRunId = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.stepId = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.stepRunId = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.actionId = reader.string();
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.actionType = reader.int32() as any;
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.actionPayload = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AssignedAction {
    return {
      tenantId: isSet(object.tenantId) ? globalThis.String(object.tenantId) : "",
      jobId: isSet(object.jobId) ? globalThis.String(object.jobId) : "",
      jobName: isSet(object.jobName) ? globalThis.String(object.jobName) : "",
      jobRunId: isSet(object.jobRunId) ? globalThis.String(object.jobRunId) : "",
      stepId: isSet(object.stepId) ? globalThis.String(object.stepId) : "",
      stepRunId: isSet(object.stepRunId) ? globalThis.String(object.stepRunId) : "",
      actionId: isSet(object.actionId) ? globalThis.String(object.actionId) : "",
      actionType: isSet(object.actionType) ? actionTypeFromJSON(object.actionType) : 0,
      actionPayload: isSet(object.actionPayload) ? globalThis.String(object.actionPayload) : "",
    };
  },

  toJSON(message: AssignedAction): unknown {
    const obj: any = {};
    if (message.tenantId !== "") {
      obj.tenantId = message.tenantId;
    }
    if (message.jobId !== "") {
      obj.jobId = message.jobId;
    }
    if (message.jobName !== "") {
      obj.jobName = message.jobName;
    }
    if (message.jobRunId !== "") {
      obj.jobRunId = message.jobRunId;
    }
    if (message.stepId !== "") {
      obj.stepId = message.stepId;
    }
    if (message.stepRunId !== "") {
      obj.stepRunId = message.stepRunId;
    }
    if (message.actionId !== "") {
      obj.actionId = message.actionId;
    }
    if (message.actionType !== 0) {
      obj.actionType = actionTypeToJSON(message.actionType);
    }
    if (message.actionPayload !== "") {
      obj.actionPayload = message.actionPayload;
    }
    return obj;
  },

  create(base?: DeepPartial<AssignedAction>): AssignedAction {
    return AssignedAction.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<AssignedAction>): AssignedAction {
    const message = createBaseAssignedAction();
    message.tenantId = object.tenantId ?? "";
    message.jobId = object.jobId ?? "";
    message.jobName = object.jobName ?? "";
    message.jobRunId = object.jobRunId ?? "";
    message.stepId = object.stepId ?? "";
    message.stepRunId = object.stepRunId ?? "";
    message.actionId = object.actionId ?? "";
    message.actionType = object.actionType ?? 0;
    message.actionPayload = object.actionPayload ?? "";
    return message;
  },
};

function createBaseWorkerListenRequest(): WorkerListenRequest {
  return { workerId: "" };
}

export const WorkerListenRequest = {
  encode(message: WorkerListenRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.workerId !== "") {
      writer.uint32(10).string(message.workerId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): WorkerListenRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWorkerListenRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.workerId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): WorkerListenRequest {
    return { workerId: isSet(object.workerId) ? globalThis.String(object.workerId) : "" };
  },

  toJSON(message: WorkerListenRequest): unknown {
    const obj: any = {};
    if (message.workerId !== "") {
      obj.workerId = message.workerId;
    }
    return obj;
  },

  create(base?: DeepPartial<WorkerListenRequest>): WorkerListenRequest {
    return WorkerListenRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<WorkerListenRequest>): WorkerListenRequest {
    const message = createBaseWorkerListenRequest();
    message.workerId = object.workerId ?? "";
    return message;
  },
};

function createBaseWorkerUnsubscribeRequest(): WorkerUnsubscribeRequest {
  return { workerId: "" };
}

export const WorkerUnsubscribeRequest = {
  encode(message: WorkerUnsubscribeRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.workerId !== "") {
      writer.uint32(10).string(message.workerId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): WorkerUnsubscribeRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWorkerUnsubscribeRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.workerId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): WorkerUnsubscribeRequest {
    return { workerId: isSet(object.workerId) ? globalThis.String(object.workerId) : "" };
  },

  toJSON(message: WorkerUnsubscribeRequest): unknown {
    const obj: any = {};
    if (message.workerId !== "") {
      obj.workerId = message.workerId;
    }
    return obj;
  },

  create(base?: DeepPartial<WorkerUnsubscribeRequest>): WorkerUnsubscribeRequest {
    return WorkerUnsubscribeRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<WorkerUnsubscribeRequest>): WorkerUnsubscribeRequest {
    const message = createBaseWorkerUnsubscribeRequest();
    message.workerId = object.workerId ?? "";
    return message;
  },
};

function createBaseWorkerUnsubscribeResponse(): WorkerUnsubscribeResponse {
  return { tenantId: "", workerId: "" };
}

export const WorkerUnsubscribeResponse = {
  encode(message: WorkerUnsubscribeResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.tenantId !== "") {
      writer.uint32(10).string(message.tenantId);
    }
    if (message.workerId !== "") {
      writer.uint32(18).string(message.workerId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): WorkerUnsubscribeResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWorkerUnsubscribeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.tenantId = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.workerId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): WorkerUnsubscribeResponse {
    return {
      tenantId: isSet(object.tenantId) ? globalThis.String(object.tenantId) : "",
      workerId: isSet(object.workerId) ? globalThis.String(object.workerId) : "",
    };
  },

  toJSON(message: WorkerUnsubscribeResponse): unknown {
    const obj: any = {};
    if (message.tenantId !== "") {
      obj.tenantId = message.tenantId;
    }
    if (message.workerId !== "") {
      obj.workerId = message.workerId;
    }
    return obj;
  },

  create(base?: DeepPartial<WorkerUnsubscribeResponse>): WorkerUnsubscribeResponse {
    return WorkerUnsubscribeResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<WorkerUnsubscribeResponse>): WorkerUnsubscribeResponse {
    const message = createBaseWorkerUnsubscribeResponse();
    message.tenantId = object.tenantId ?? "";
    message.workerId = object.workerId ?? "";
    return message;
  },
};

function createBaseActionEvent(): ActionEvent {
  return {
    workerId: "",
    jobId: "",
    jobRunId: "",
    stepId: "",
    stepRunId: "",
    actionId: "",
    eventTimestamp: undefined,
    eventType: 0,
    eventPayload: "",
  };
}

export const ActionEvent = {
  encode(message: ActionEvent, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.workerId !== "") {
      writer.uint32(10).string(message.workerId);
    }
    if (message.jobId !== "") {
      writer.uint32(18).string(message.jobId);
    }
    if (message.jobRunId !== "") {
      writer.uint32(26).string(message.jobRunId);
    }
    if (message.stepId !== "") {
      writer.uint32(34).string(message.stepId);
    }
    if (message.stepRunId !== "") {
      writer.uint32(42).string(message.stepRunId);
    }
    if (message.actionId !== "") {
      writer.uint32(50).string(message.actionId);
    }
    if (message.eventTimestamp !== undefined) {
      Timestamp.encode(toTimestamp(message.eventTimestamp), writer.uint32(58).fork()).ldelim();
    }
    if (message.eventType !== 0) {
      writer.uint32(64).int32(message.eventType);
    }
    if (message.eventPayload !== "") {
      writer.uint32(74).string(message.eventPayload);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ActionEvent {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActionEvent();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.workerId = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.jobId = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.jobRunId = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.stepId = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.stepRunId = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.actionId = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.eventTimestamp = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.eventType = reader.int32() as any;
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.eventPayload = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ActionEvent {
    return {
      workerId: isSet(object.workerId) ? globalThis.String(object.workerId) : "",
      jobId: isSet(object.jobId) ? globalThis.String(object.jobId) : "",
      jobRunId: isSet(object.jobRunId) ? globalThis.String(object.jobRunId) : "",
      stepId: isSet(object.stepId) ? globalThis.String(object.stepId) : "",
      stepRunId: isSet(object.stepRunId) ? globalThis.String(object.stepRunId) : "",
      actionId: isSet(object.actionId) ? globalThis.String(object.actionId) : "",
      eventTimestamp: isSet(object.eventTimestamp) ? fromJsonTimestamp(object.eventTimestamp) : undefined,
      eventType: isSet(object.eventType) ? actionEventTypeFromJSON(object.eventType) : 0,
      eventPayload: isSet(object.eventPayload) ? globalThis.String(object.eventPayload) : "",
    };
  },

  toJSON(message: ActionEvent): unknown {
    const obj: any = {};
    if (message.workerId !== "") {
      obj.workerId = message.workerId;
    }
    if (message.jobId !== "") {
      obj.jobId = message.jobId;
    }
    if (message.jobRunId !== "") {
      obj.jobRunId = message.jobRunId;
    }
    if (message.stepId !== "") {
      obj.stepId = message.stepId;
    }
    if (message.stepRunId !== "") {
      obj.stepRunId = message.stepRunId;
    }
    if (message.actionId !== "") {
      obj.actionId = message.actionId;
    }
    if (message.eventTimestamp !== undefined) {
      obj.eventTimestamp = message.eventTimestamp.toISOString();
    }
    if (message.eventType !== 0) {
      obj.eventType = actionEventTypeToJSON(message.eventType);
    }
    if (message.eventPayload !== "") {
      obj.eventPayload = message.eventPayload;
    }
    return obj;
  },

  create(base?: DeepPartial<ActionEvent>): ActionEvent {
    return ActionEvent.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ActionEvent>): ActionEvent {
    const message = createBaseActionEvent();
    message.workerId = object.workerId ?? "";
    message.jobId = object.jobId ?? "";
    message.jobRunId = object.jobRunId ?? "";
    message.stepId = object.stepId ?? "";
    message.stepRunId = object.stepRunId ?? "";
    message.actionId = object.actionId ?? "";
    message.eventTimestamp = object.eventTimestamp ?? undefined;
    message.eventType = object.eventType ?? 0;
    message.eventPayload = object.eventPayload ?? "";
    return message;
  },
};

function createBaseActionEventResponse(): ActionEventResponse {
  return { tenantId: "", workerId: "" };
}

export const ActionEventResponse = {
  encode(message: ActionEventResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.tenantId !== "") {
      writer.uint32(10).string(message.tenantId);
    }
    if (message.workerId !== "") {
      writer.uint32(18).string(message.workerId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ActionEventResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActionEventResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.tenantId = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.workerId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ActionEventResponse {
    return {
      tenantId: isSet(object.tenantId) ? globalThis.String(object.tenantId) : "",
      workerId: isSet(object.workerId) ? globalThis.String(object.workerId) : "",
    };
  },

  toJSON(message: ActionEventResponse): unknown {
    const obj: any = {};
    if (message.tenantId !== "") {
      obj.tenantId = message.tenantId;
    }
    if (message.workerId !== "") {
      obj.workerId = message.workerId;
    }
    return obj;
  },

  create(base?: DeepPartial<ActionEventResponse>): ActionEventResponse {
    return ActionEventResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ActionEventResponse>): ActionEventResponse {
    const message = createBaseActionEventResponse();
    message.tenantId = object.tenantId ?? "";
    message.workerId = object.workerId ?? "";
    return message;
  },
};

export type DispatcherDefinition = typeof DispatcherDefinition;
export const DispatcherDefinition = {
  name: "Dispatcher",
  fullName: "Dispatcher",
  methods: {
    register: {
      name: "Register",
      requestType: WorkerRegisterRequest,
      requestStream: false,
      responseType: WorkerRegisterResponse,
      responseStream: false,
      options: {},
    },
    listen: {
      name: "Listen",
      requestType: WorkerListenRequest,
      requestStream: false,
      responseType: AssignedAction,
      responseStream: true,
      options: {},
    },
    sendActionEvent: {
      name: "SendActionEvent",
      requestType: ActionEvent,
      requestStream: false,
      responseType: ActionEventResponse,
      responseStream: false,
      options: {},
    },
    unsubscribe: {
      name: "Unsubscribe",
      requestType: WorkerUnsubscribeRequest,
      requestStream: false,
      responseType: WorkerUnsubscribeResponse,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface DispatcherServiceImplementation<CallContextExt = {}> {
  register(
    request: WorkerRegisterRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<WorkerRegisterResponse>>;
  listen(
    request: WorkerListenRequest,
    context: CallContext & CallContextExt,
  ): ServerStreamingMethodResult<DeepPartial<AssignedAction>>;
  sendActionEvent(
    request: ActionEvent,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ActionEventResponse>>;
  unsubscribe(
    request: WorkerUnsubscribeRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<WorkerUnsubscribeResponse>>;
}

export interface DispatcherClient<CallOptionsExt = {}> {
  register(
    request: DeepPartial<WorkerRegisterRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<WorkerRegisterResponse>;
  listen(
    request: DeepPartial<WorkerListenRequest>,
    options?: CallOptions & CallOptionsExt,
  ): AsyncIterable<AssignedAction>;
  sendActionEvent(
    request: DeepPartial<ActionEvent>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ActionEventResponse>;
  unsubscribe(
    request: DeepPartial<WorkerUnsubscribeRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<WorkerUnsubscribeResponse>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function toTimestamp(date: Date): Timestamp {
  const seconds = Math.trunc(date.getTime() / 1_000);
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = (t.seconds || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
  return new globalThis.Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof globalThis.Date) {
    return o;
  } else if (typeof o === "string") {
    return new globalThis.Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}

export type ServerStreamingMethodResult<Response> = { [Symbol.asyncIterator](): AsyncIterator<Response, void> };
