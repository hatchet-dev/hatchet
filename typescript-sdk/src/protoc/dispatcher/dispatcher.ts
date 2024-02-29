/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "";

export enum ActionType {
  START_STEP_RUN = 0,
  CANCEL_STEP_RUN = 1,
  START_GET_GROUP_KEY = 2,
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
    case 2:
    case "START_GET_GROUP_KEY":
      return ActionType.START_GET_GROUP_KEY;
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
    case ActionType.START_GET_GROUP_KEY:
      return "START_GET_GROUP_KEY";
    case ActionType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum GroupKeyActionEventType {
  GROUP_KEY_EVENT_TYPE_UNKNOWN = 0,
  GROUP_KEY_EVENT_TYPE_STARTED = 1,
  GROUP_KEY_EVENT_TYPE_COMPLETED = 2,
  GROUP_KEY_EVENT_TYPE_FAILED = 3,
  UNRECOGNIZED = -1,
}

export function groupKeyActionEventTypeFromJSON(object: any): GroupKeyActionEventType {
  switch (object) {
    case 0:
    case "GROUP_KEY_EVENT_TYPE_UNKNOWN":
      return GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_UNKNOWN;
    case 1:
    case "GROUP_KEY_EVENT_TYPE_STARTED":
      return GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_STARTED;
    case 2:
    case "GROUP_KEY_EVENT_TYPE_COMPLETED":
      return GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_COMPLETED;
    case 3:
    case "GROUP_KEY_EVENT_TYPE_FAILED":
      return GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_FAILED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return GroupKeyActionEventType.UNRECOGNIZED;
  }
}

export function groupKeyActionEventTypeToJSON(object: GroupKeyActionEventType): string {
  switch (object) {
    case GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_UNKNOWN:
      return "GROUP_KEY_EVENT_TYPE_UNKNOWN";
    case GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_STARTED:
      return "GROUP_KEY_EVENT_TYPE_STARTED";
    case GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_COMPLETED:
      return "GROUP_KEY_EVENT_TYPE_COMPLETED";
    case GroupKeyActionEventType.GROUP_KEY_EVENT_TYPE_FAILED:
      return "GROUP_KEY_EVENT_TYPE_FAILED";
    case GroupKeyActionEventType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum StepActionEventType {
  STEP_EVENT_TYPE_UNKNOWN = 0,
  STEP_EVENT_TYPE_STARTED = 1,
  STEP_EVENT_TYPE_COMPLETED = 2,
  STEP_EVENT_TYPE_FAILED = 3,
  UNRECOGNIZED = -1,
}

export function stepActionEventTypeFromJSON(object: any): StepActionEventType {
  switch (object) {
    case 0:
    case "STEP_EVENT_TYPE_UNKNOWN":
      return StepActionEventType.STEP_EVENT_TYPE_UNKNOWN;
    case 1:
    case "STEP_EVENT_TYPE_STARTED":
      return StepActionEventType.STEP_EVENT_TYPE_STARTED;
    case 2:
    case "STEP_EVENT_TYPE_COMPLETED":
      return StepActionEventType.STEP_EVENT_TYPE_COMPLETED;
    case 3:
    case "STEP_EVENT_TYPE_FAILED":
      return StepActionEventType.STEP_EVENT_TYPE_FAILED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return StepActionEventType.UNRECOGNIZED;
  }
}

export function stepActionEventTypeToJSON(object: StepActionEventType): string {
  switch (object) {
    case StepActionEventType.STEP_EVENT_TYPE_UNKNOWN:
      return "STEP_EVENT_TYPE_UNKNOWN";
    case StepActionEventType.STEP_EVENT_TYPE_STARTED:
      return "STEP_EVENT_TYPE_STARTED";
    case StepActionEventType.STEP_EVENT_TYPE_COMPLETED:
      return "STEP_EVENT_TYPE_COMPLETED";
    case StepActionEventType.STEP_EVENT_TYPE_FAILED:
      return "STEP_EVENT_TYPE_FAILED";
    case StepActionEventType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum ResourceType {
  RESOURCE_TYPE_UNKNOWN = 0,
  RESOURCE_TYPE_STEP_RUN = 1,
  RESOURCE_TYPE_WORKFLOW_RUN = 2,
  UNRECOGNIZED = -1,
}

export function resourceTypeFromJSON(object: any): ResourceType {
  switch (object) {
    case 0:
    case "RESOURCE_TYPE_UNKNOWN":
      return ResourceType.RESOURCE_TYPE_UNKNOWN;
    case 1:
    case "RESOURCE_TYPE_STEP_RUN":
      return ResourceType.RESOURCE_TYPE_STEP_RUN;
    case 2:
    case "RESOURCE_TYPE_WORKFLOW_RUN":
      return ResourceType.RESOURCE_TYPE_WORKFLOW_RUN;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ResourceType.UNRECOGNIZED;
  }
}

export function resourceTypeToJSON(object: ResourceType): string {
  switch (object) {
    case ResourceType.RESOURCE_TYPE_UNKNOWN:
      return "RESOURCE_TYPE_UNKNOWN";
    case ResourceType.RESOURCE_TYPE_STEP_RUN:
      return "RESOURCE_TYPE_STEP_RUN";
    case ResourceType.RESOURCE_TYPE_WORKFLOW_RUN:
      return "RESOURCE_TYPE_WORKFLOW_RUN";
    case ResourceType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum ResourceEventType {
  RESOURCE_EVENT_TYPE_UNKNOWN = 0,
  RESOURCE_EVENT_TYPE_STARTED = 1,
  RESOURCE_EVENT_TYPE_COMPLETED = 2,
  RESOURCE_EVENT_TYPE_FAILED = 3,
  RESOURCE_EVENT_TYPE_CANCELLED = 4,
  RESOURCE_EVENT_TYPE_TIMED_OUT = 5,
  UNRECOGNIZED = -1,
}

export function resourceEventTypeFromJSON(object: any): ResourceEventType {
  switch (object) {
    case 0:
    case "RESOURCE_EVENT_TYPE_UNKNOWN":
      return ResourceEventType.RESOURCE_EVENT_TYPE_UNKNOWN;
    case 1:
    case "RESOURCE_EVENT_TYPE_STARTED":
      return ResourceEventType.RESOURCE_EVENT_TYPE_STARTED;
    case 2:
    case "RESOURCE_EVENT_TYPE_COMPLETED":
      return ResourceEventType.RESOURCE_EVENT_TYPE_COMPLETED;
    case 3:
    case "RESOURCE_EVENT_TYPE_FAILED":
      return ResourceEventType.RESOURCE_EVENT_TYPE_FAILED;
    case 4:
    case "RESOURCE_EVENT_TYPE_CANCELLED":
      return ResourceEventType.RESOURCE_EVENT_TYPE_CANCELLED;
    case 5:
    case "RESOURCE_EVENT_TYPE_TIMED_OUT":
      return ResourceEventType.RESOURCE_EVENT_TYPE_TIMED_OUT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ResourceEventType.UNRECOGNIZED;
  }
}

export function resourceEventTypeToJSON(object: ResourceEventType): string {
  switch (object) {
    case ResourceEventType.RESOURCE_EVENT_TYPE_UNKNOWN:
      return "RESOURCE_EVENT_TYPE_UNKNOWN";
    case ResourceEventType.RESOURCE_EVENT_TYPE_STARTED:
      return "RESOURCE_EVENT_TYPE_STARTED";
    case ResourceEventType.RESOURCE_EVENT_TYPE_COMPLETED:
      return "RESOURCE_EVENT_TYPE_COMPLETED";
    case ResourceEventType.RESOURCE_EVENT_TYPE_FAILED:
      return "RESOURCE_EVENT_TYPE_FAILED";
    case ResourceEventType.RESOURCE_EVENT_TYPE_CANCELLED:
      return "RESOURCE_EVENT_TYPE_CANCELLED";
    case ResourceEventType.RESOURCE_EVENT_TYPE_TIMED_OUT:
      return "RESOURCE_EVENT_TYPE_TIMED_OUT";
    case ResourceEventType.UNRECOGNIZED:
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
  /** (optional) the max number of runs this worker can handle */
  maxRuns?: number | undefined;
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
  /** the workflow run id (optional) */
  workflowRunId: string;
  /** the get group key run id (optional) */
  getGroupKeyRunId: string;
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
  /** the step name */
  stepName: string;
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

export interface GroupKeyActionEvent {
  /** the id of the worker */
  workerId: string;
  /** the id of the job */
  workflowRunId: string;
  getGroupKeyRunId: string;
  /** the action id */
  actionId: string;
  eventTimestamp:
    | Date
    | undefined;
  /** the step event type */
  eventType: GroupKeyActionEventType;
  /** the event payload */
  eventPayload: string;
}

export interface StepActionEvent {
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
  eventType: StepActionEventType;
  /** the event payload */
  eventPayload: string;
}

export interface ActionEventResponse {
  /** the tenant id */
  tenantId: string;
  /** the id of the worker */
  workerId: string;
}

export interface SubscribeToWorkflowEventsRequest {
  /** the id of the workflow run */
  workflowRunId: string;
}

export interface WorkflowEvent {
  /** the id of the workflow run */
  workflowRunId: string;
  resourceType: ResourceType;
  eventType: ResourceEventType;
  resourceId: string;
  eventTimestamp:
    | Date
    | undefined;
  /** the event payload */
  eventPayload: string;
  /**
   * whether this is the last event for the workflow run - server
   * will hang up the connection but clients might want to case
   */
  hangup: boolean;
}

export interface OverridesData {
  /** the step run id */
  stepRunId: string;
  /** the path of the data to set */
  path: string;
  /** the value to set */
  value: string;
  /** the filename of the caller */
  callerFilename: string;
}

export interface OverridesDataResponse {
}

function createBaseWorkerRegisterRequest(): WorkerRegisterRequest {
  return { workerName: "", actions: [], services: [], maxRuns: undefined };
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
    if (message.maxRuns !== undefined) {
      writer.uint32(32).int32(message.maxRuns);
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
        case 4:
          if (tag !== 32) {
            break;
          }

          message.maxRuns = reader.int32();
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
      maxRuns: isSet(object.maxRuns) ? globalThis.Number(object.maxRuns) : undefined,
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
    if (message.maxRuns !== undefined) {
      obj.maxRuns = Math.round(message.maxRuns);
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
    message.maxRuns = object.maxRuns ?? undefined;
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
    workflowRunId: "",
    getGroupKeyRunId: "",
    jobId: "",
    jobName: "",
    jobRunId: "",
    stepId: "",
    stepRunId: "",
    actionId: "",
    actionType: 0,
    actionPayload: "",
    stepName: "",
  };
}

export const AssignedAction = {
  encode(message: AssignedAction, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.tenantId !== "") {
      writer.uint32(10).string(message.tenantId);
    }
    if (message.workflowRunId !== "") {
      writer.uint32(18).string(message.workflowRunId);
    }
    if (message.getGroupKeyRunId !== "") {
      writer.uint32(26).string(message.getGroupKeyRunId);
    }
    if (message.jobId !== "") {
      writer.uint32(34).string(message.jobId);
    }
    if (message.jobName !== "") {
      writer.uint32(42).string(message.jobName);
    }
    if (message.jobRunId !== "") {
      writer.uint32(50).string(message.jobRunId);
    }
    if (message.stepId !== "") {
      writer.uint32(58).string(message.stepId);
    }
    if (message.stepRunId !== "") {
      writer.uint32(66).string(message.stepRunId);
    }
    if (message.actionId !== "") {
      writer.uint32(74).string(message.actionId);
    }
    if (message.actionType !== 0) {
      writer.uint32(80).int32(message.actionType);
    }
    if (message.actionPayload !== "") {
      writer.uint32(90).string(message.actionPayload);
    }
    if (message.stepName !== "") {
      writer.uint32(98).string(message.stepName);
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

          message.workflowRunId = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.getGroupKeyRunId = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.jobId = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.jobName = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.jobRunId = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.stepId = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.stepRunId = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.actionId = reader.string();
          continue;
        case 10:
          if (tag !== 80) {
            break;
          }

          message.actionType = reader.int32() as any;
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.actionPayload = reader.string();
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.stepName = reader.string();
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
      workflowRunId: isSet(object.workflowRunId) ? globalThis.String(object.workflowRunId) : "",
      getGroupKeyRunId: isSet(object.getGroupKeyRunId) ? globalThis.String(object.getGroupKeyRunId) : "",
      jobId: isSet(object.jobId) ? globalThis.String(object.jobId) : "",
      jobName: isSet(object.jobName) ? globalThis.String(object.jobName) : "",
      jobRunId: isSet(object.jobRunId) ? globalThis.String(object.jobRunId) : "",
      stepId: isSet(object.stepId) ? globalThis.String(object.stepId) : "",
      stepRunId: isSet(object.stepRunId) ? globalThis.String(object.stepRunId) : "",
      actionId: isSet(object.actionId) ? globalThis.String(object.actionId) : "",
      actionType: isSet(object.actionType) ? actionTypeFromJSON(object.actionType) : 0,
      actionPayload: isSet(object.actionPayload) ? globalThis.String(object.actionPayload) : "",
      stepName: isSet(object.stepName) ? globalThis.String(object.stepName) : "",
    };
  },

  toJSON(message: AssignedAction): unknown {
    const obj: any = {};
    if (message.tenantId !== "") {
      obj.tenantId = message.tenantId;
    }
    if (message.workflowRunId !== "") {
      obj.workflowRunId = message.workflowRunId;
    }
    if (message.getGroupKeyRunId !== "") {
      obj.getGroupKeyRunId = message.getGroupKeyRunId;
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
    if (message.stepName !== "") {
      obj.stepName = message.stepName;
    }
    return obj;
  },

  create(base?: DeepPartial<AssignedAction>): AssignedAction {
    return AssignedAction.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<AssignedAction>): AssignedAction {
    const message = createBaseAssignedAction();
    message.tenantId = object.tenantId ?? "";
    message.workflowRunId = object.workflowRunId ?? "";
    message.getGroupKeyRunId = object.getGroupKeyRunId ?? "";
    message.jobId = object.jobId ?? "";
    message.jobName = object.jobName ?? "";
    message.jobRunId = object.jobRunId ?? "";
    message.stepId = object.stepId ?? "";
    message.stepRunId = object.stepRunId ?? "";
    message.actionId = object.actionId ?? "";
    message.actionType = object.actionType ?? 0;
    message.actionPayload = object.actionPayload ?? "";
    message.stepName = object.stepName ?? "";
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

function createBaseGroupKeyActionEvent(): GroupKeyActionEvent {
  return {
    workerId: "",
    workflowRunId: "",
    getGroupKeyRunId: "",
    actionId: "",
    eventTimestamp: undefined,
    eventType: 0,
    eventPayload: "",
  };
}

export const GroupKeyActionEvent = {
  encode(message: GroupKeyActionEvent, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.workerId !== "") {
      writer.uint32(10).string(message.workerId);
    }
    if (message.workflowRunId !== "") {
      writer.uint32(18).string(message.workflowRunId);
    }
    if (message.getGroupKeyRunId !== "") {
      writer.uint32(26).string(message.getGroupKeyRunId);
    }
    if (message.actionId !== "") {
      writer.uint32(34).string(message.actionId);
    }
    if (message.eventTimestamp !== undefined) {
      Timestamp.encode(toTimestamp(message.eventTimestamp), writer.uint32(42).fork()).ldelim();
    }
    if (message.eventType !== 0) {
      writer.uint32(48).int32(message.eventType);
    }
    if (message.eventPayload !== "") {
      writer.uint32(58).string(message.eventPayload);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GroupKeyActionEvent {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGroupKeyActionEvent();
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

          message.workflowRunId = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.getGroupKeyRunId = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.actionId = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.eventTimestamp = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.eventType = reader.int32() as any;
          continue;
        case 7:
          if (tag !== 58) {
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

  fromJSON(object: any): GroupKeyActionEvent {
    return {
      workerId: isSet(object.workerId) ? globalThis.String(object.workerId) : "",
      workflowRunId: isSet(object.workflowRunId) ? globalThis.String(object.workflowRunId) : "",
      getGroupKeyRunId: isSet(object.getGroupKeyRunId) ? globalThis.String(object.getGroupKeyRunId) : "",
      actionId: isSet(object.actionId) ? globalThis.String(object.actionId) : "",
      eventTimestamp: isSet(object.eventTimestamp) ? fromJsonTimestamp(object.eventTimestamp) : undefined,
      eventType: isSet(object.eventType) ? groupKeyActionEventTypeFromJSON(object.eventType) : 0,
      eventPayload: isSet(object.eventPayload) ? globalThis.String(object.eventPayload) : "",
    };
  },

  toJSON(message: GroupKeyActionEvent): unknown {
    const obj: any = {};
    if (message.workerId !== "") {
      obj.workerId = message.workerId;
    }
    if (message.workflowRunId !== "") {
      obj.workflowRunId = message.workflowRunId;
    }
    if (message.getGroupKeyRunId !== "") {
      obj.getGroupKeyRunId = message.getGroupKeyRunId;
    }
    if (message.actionId !== "") {
      obj.actionId = message.actionId;
    }
    if (message.eventTimestamp !== undefined) {
      obj.eventTimestamp = message.eventTimestamp.toISOString();
    }
    if (message.eventType !== 0) {
      obj.eventType = groupKeyActionEventTypeToJSON(message.eventType);
    }
    if (message.eventPayload !== "") {
      obj.eventPayload = message.eventPayload;
    }
    return obj;
  },

  create(base?: DeepPartial<GroupKeyActionEvent>): GroupKeyActionEvent {
    return GroupKeyActionEvent.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GroupKeyActionEvent>): GroupKeyActionEvent {
    const message = createBaseGroupKeyActionEvent();
    message.workerId = object.workerId ?? "";
    message.workflowRunId = object.workflowRunId ?? "";
    message.getGroupKeyRunId = object.getGroupKeyRunId ?? "";
    message.actionId = object.actionId ?? "";
    message.eventTimestamp = object.eventTimestamp ?? undefined;
    message.eventType = object.eventType ?? 0;
    message.eventPayload = object.eventPayload ?? "";
    return message;
  },
};

function createBaseStepActionEvent(): StepActionEvent {
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

export const StepActionEvent = {
  encode(message: StepActionEvent, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): StepActionEvent {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStepActionEvent();
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

  fromJSON(object: any): StepActionEvent {
    return {
      workerId: isSet(object.workerId) ? globalThis.String(object.workerId) : "",
      jobId: isSet(object.jobId) ? globalThis.String(object.jobId) : "",
      jobRunId: isSet(object.jobRunId) ? globalThis.String(object.jobRunId) : "",
      stepId: isSet(object.stepId) ? globalThis.String(object.stepId) : "",
      stepRunId: isSet(object.stepRunId) ? globalThis.String(object.stepRunId) : "",
      actionId: isSet(object.actionId) ? globalThis.String(object.actionId) : "",
      eventTimestamp: isSet(object.eventTimestamp) ? fromJsonTimestamp(object.eventTimestamp) : undefined,
      eventType: isSet(object.eventType) ? stepActionEventTypeFromJSON(object.eventType) : 0,
      eventPayload: isSet(object.eventPayload) ? globalThis.String(object.eventPayload) : "",
    };
  },

  toJSON(message: StepActionEvent): unknown {
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
      obj.eventType = stepActionEventTypeToJSON(message.eventType);
    }
    if (message.eventPayload !== "") {
      obj.eventPayload = message.eventPayload;
    }
    return obj;
  },

  create(base?: DeepPartial<StepActionEvent>): StepActionEvent {
    return StepActionEvent.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<StepActionEvent>): StepActionEvent {
    const message = createBaseStepActionEvent();
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

function createBaseSubscribeToWorkflowEventsRequest(): SubscribeToWorkflowEventsRequest {
  return { workflowRunId: "" };
}

export const SubscribeToWorkflowEventsRequest = {
  encode(message: SubscribeToWorkflowEventsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.workflowRunId !== "") {
      writer.uint32(10).string(message.workflowRunId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SubscribeToWorkflowEventsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSubscribeToWorkflowEventsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.workflowRunId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SubscribeToWorkflowEventsRequest {
    return { workflowRunId: isSet(object.workflowRunId) ? globalThis.String(object.workflowRunId) : "" };
  },

  toJSON(message: SubscribeToWorkflowEventsRequest): unknown {
    const obj: any = {};
    if (message.workflowRunId !== "") {
      obj.workflowRunId = message.workflowRunId;
    }
    return obj;
  },

  create(base?: DeepPartial<SubscribeToWorkflowEventsRequest>): SubscribeToWorkflowEventsRequest {
    return SubscribeToWorkflowEventsRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SubscribeToWorkflowEventsRequest>): SubscribeToWorkflowEventsRequest {
    const message = createBaseSubscribeToWorkflowEventsRequest();
    message.workflowRunId = object.workflowRunId ?? "";
    return message;
  },
};

function createBaseWorkflowEvent(): WorkflowEvent {
  return {
    workflowRunId: "",
    resourceType: 0,
    eventType: 0,
    resourceId: "",
    eventTimestamp: undefined,
    eventPayload: "",
    hangup: false,
  };
}

export const WorkflowEvent = {
  encode(message: WorkflowEvent, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.workflowRunId !== "") {
      writer.uint32(10).string(message.workflowRunId);
    }
    if (message.resourceType !== 0) {
      writer.uint32(16).int32(message.resourceType);
    }
    if (message.eventType !== 0) {
      writer.uint32(24).int32(message.eventType);
    }
    if (message.resourceId !== "") {
      writer.uint32(34).string(message.resourceId);
    }
    if (message.eventTimestamp !== undefined) {
      Timestamp.encode(toTimestamp(message.eventTimestamp), writer.uint32(42).fork()).ldelim();
    }
    if (message.eventPayload !== "") {
      writer.uint32(50).string(message.eventPayload);
    }
    if (message.hangup === true) {
      writer.uint32(56).bool(message.hangup);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): WorkflowEvent {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWorkflowEvent();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.workflowRunId = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.resourceType = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.eventType = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.resourceId = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.eventTimestamp = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.eventPayload = reader.string();
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.hangup = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): WorkflowEvent {
    return {
      workflowRunId: isSet(object.workflowRunId) ? globalThis.String(object.workflowRunId) : "",
      resourceType: isSet(object.resourceType) ? resourceTypeFromJSON(object.resourceType) : 0,
      eventType: isSet(object.eventType) ? resourceEventTypeFromJSON(object.eventType) : 0,
      resourceId: isSet(object.resourceId) ? globalThis.String(object.resourceId) : "",
      eventTimestamp: isSet(object.eventTimestamp) ? fromJsonTimestamp(object.eventTimestamp) : undefined,
      eventPayload: isSet(object.eventPayload) ? globalThis.String(object.eventPayload) : "",
      hangup: isSet(object.hangup) ? globalThis.Boolean(object.hangup) : false,
    };
  },

  toJSON(message: WorkflowEvent): unknown {
    const obj: any = {};
    if (message.workflowRunId !== "") {
      obj.workflowRunId = message.workflowRunId;
    }
    if (message.resourceType !== 0) {
      obj.resourceType = resourceTypeToJSON(message.resourceType);
    }
    if (message.eventType !== 0) {
      obj.eventType = resourceEventTypeToJSON(message.eventType);
    }
    if (message.resourceId !== "") {
      obj.resourceId = message.resourceId;
    }
    if (message.eventTimestamp !== undefined) {
      obj.eventTimestamp = message.eventTimestamp.toISOString();
    }
    if (message.eventPayload !== "") {
      obj.eventPayload = message.eventPayload;
    }
    if (message.hangup === true) {
      obj.hangup = message.hangup;
    }
    return obj;
  },

  create(base?: DeepPartial<WorkflowEvent>): WorkflowEvent {
    return WorkflowEvent.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<WorkflowEvent>): WorkflowEvent {
    const message = createBaseWorkflowEvent();
    message.workflowRunId = object.workflowRunId ?? "";
    message.resourceType = object.resourceType ?? 0;
    message.eventType = object.eventType ?? 0;
    message.resourceId = object.resourceId ?? "";
    message.eventTimestamp = object.eventTimestamp ?? undefined;
    message.eventPayload = object.eventPayload ?? "";
    message.hangup = object.hangup ?? false;
    return message;
  },
};

function createBaseOverridesData(): OverridesData {
  return { stepRunId: "", path: "", value: "", callerFilename: "" };
}

export const OverridesData = {
  encode(message: OverridesData, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.stepRunId !== "") {
      writer.uint32(10).string(message.stepRunId);
    }
    if (message.path !== "") {
      writer.uint32(18).string(message.path);
    }
    if (message.value !== "") {
      writer.uint32(26).string(message.value);
    }
    if (message.callerFilename !== "") {
      writer.uint32(34).string(message.callerFilename);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): OverridesData {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseOverridesData();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.stepRunId = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.path = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.value = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.callerFilename = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): OverridesData {
    return {
      stepRunId: isSet(object.stepRunId) ? globalThis.String(object.stepRunId) : "",
      path: isSet(object.path) ? globalThis.String(object.path) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
      callerFilename: isSet(object.callerFilename) ? globalThis.String(object.callerFilename) : "",
    };
  },

  toJSON(message: OverridesData): unknown {
    const obj: any = {};
    if (message.stepRunId !== "") {
      obj.stepRunId = message.stepRunId;
    }
    if (message.path !== "") {
      obj.path = message.path;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    if (message.callerFilename !== "") {
      obj.callerFilename = message.callerFilename;
    }
    return obj;
  },

  create(base?: DeepPartial<OverridesData>): OverridesData {
    return OverridesData.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<OverridesData>): OverridesData {
    const message = createBaseOverridesData();
    message.stepRunId = object.stepRunId ?? "";
    message.path = object.path ?? "";
    message.value = object.value ?? "";
    message.callerFilename = object.callerFilename ?? "";
    return message;
  },
};

function createBaseOverridesDataResponse(): OverridesDataResponse {
  return {};
}

export const OverridesDataResponse = {
  encode(_: OverridesDataResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): OverridesDataResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseOverridesDataResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): OverridesDataResponse {
    return {};
  },

  toJSON(_: OverridesDataResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<OverridesDataResponse>): OverridesDataResponse {
    return OverridesDataResponse.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<OverridesDataResponse>): OverridesDataResponse {
    const message = createBaseOverridesDataResponse();
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
    subscribeToWorkflowEvents: {
      name: "SubscribeToWorkflowEvents",
      requestType: SubscribeToWorkflowEventsRequest,
      requestStream: false,
      responseType: WorkflowEvent,
      responseStream: true,
      options: {},
    },
    sendStepActionEvent: {
      name: "SendStepActionEvent",
      requestType: StepActionEvent,
      requestStream: false,
      responseType: ActionEventResponse,
      responseStream: false,
      options: {},
    },
    sendGroupKeyActionEvent: {
      name: "SendGroupKeyActionEvent",
      requestType: GroupKeyActionEvent,
      requestStream: false,
      responseType: ActionEventResponse,
      responseStream: false,
      options: {},
    },
    putOverridesData: {
      name: "PutOverridesData",
      requestType: OverridesData,
      requestStream: false,
      responseType: OverridesDataResponse,
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
  subscribeToWorkflowEvents(
    request: SubscribeToWorkflowEventsRequest,
    context: CallContext & CallContextExt,
  ): ServerStreamingMethodResult<DeepPartial<WorkflowEvent>>;
  sendStepActionEvent(
    request: StepActionEvent,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ActionEventResponse>>;
  sendGroupKeyActionEvent(
    request: GroupKeyActionEvent,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ActionEventResponse>>;
  putOverridesData(
    request: OverridesData,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<OverridesDataResponse>>;
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
  subscribeToWorkflowEvents(
    request: DeepPartial<SubscribeToWorkflowEventsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): AsyncIterable<WorkflowEvent>;
  sendStepActionEvent(
    request: DeepPartial<StepActionEvent>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ActionEventResponse>;
  sendGroupKeyActionEvent(
    request: DeepPartial<GroupKeyActionEvent>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ActionEventResponse>;
  putOverridesData(
    request: DeepPartial<OverridesData>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<OverridesDataResponse>;
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
