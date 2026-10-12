/**
 * Event and acknowledgement shapes exchanged between a durable task and the durable
 * transport it runs on. Runtime-neutral: nothing here imports from Node.
 */
import type { DurableEventListenerConditions } from '@hatchet/protoc/v1/shared/condition';
import type { TriggerWorkflowRequest } from '@hatchet/protoc/v1/shared/trigger';

export interface DurableTaskRunAckEntryResult {
  nodeId: number;
  branchId: number;
  workflowRunExternalId: string;
}

export interface DurableTaskEventRunAck {
  ackType: 'run';
  invocationCount: number;
  durableTaskExternalId: string;
  runEntries: DurableTaskRunAckEntryResult[];
}

export interface DurableTaskEventMemoAck {
  ackType: 'memo';
  invocationCount: number;
  durableTaskExternalId: string;
  branchId: number;
  nodeId: number;
  memoAlreadyExisted: boolean;
  memoResultPayload?: Uint8Array;
}

export interface DurableTaskEventWaitForAck {
  ackType: 'waitFor';
  invocationCount: number;
  durableTaskExternalId: string;
  branchId: number;
  nodeId: number;
}

export type DurableTaskEventAck =
  DurableTaskEventRunAck | DurableTaskEventMemoAck | DurableTaskEventWaitForAck;

export interface DurableTaskEventLogEntryResult {
  durableTaskExternalId: string;
  nodeId: number;
  payload: Record<string, unknown> | undefined;
  isFailure: boolean;
  errorMessage: string | undefined;
}

export interface WaitForEvent {
  kind: 'waitFor';
  waitForConditions: DurableEventListenerConditions;
  label?: string;
}

export interface RunChildrenEvent {
  kind: 'runChildren';
  triggerOpts: TriggerWorkflowRequest[];
}

export interface MemoEvent {
  kind: 'memo';
  memoKey: Uint8Array;
  payload?: Uint8Array;
}

export type DurableTaskSendEvent = WaitForEvent | RunChildrenEvent | MemoEvent;
