// eslint-disable-next-line max-classes-per-file
import { Channel, ClientFactory, Status } from 'nice-grpc';
import { EventEmitter, on } from 'events';
import {
  DispatcherClient as PbDispatcherClient,
  DispatcherDefinition,
  ResourceEventType,
  ResourceType,
  DispatcherClient,
  WorkflowEvent,
} from '@hatchet/protoc/dispatcher';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import HatchetError from '@util/errors/hatchet-error';
import { Logger } from '@hatchet/util/logger';
import sleep from '@hatchet/util/sleep';
import { Api } from '../rest';
import { WorkflowRunStatus } from '../rest/generated/data-contracts';
import { GrpcPooledListener } from './child-listener-client';

const DEFAULT_EVENT_LISTENER_RETRY_INTERVAL = 5; // seconds
const DEFAULT_EVENT_LISTENER_RETRY_COUNT = 5;
const DEFAULT_EVENT_LISTENER_POLL_INTERVAL = 5000; // milliseconds

// eslint-disable-next-line no-shadow
export enum RunEventType {
  STEP_RUN_EVENT_TYPE_STARTED = 'STEP_RUN_EVENT_TYPE_STARTED',
  STEP_RUN_EVENT_TYPE_COMPLETED = 'STEP_RUN_EVENT_TYPE_COMPLETED',
  STEP_RUN_EVENT_TYPE_FAILED = 'STEP_RUN_EVENT_TYPE_FAILED',
  STEP_RUN_EVENT_TYPE_CANCELLED = 'STEP_RUN_EVENT_TYPE_CANCELLED',
  STEP_RUN_EVENT_TYPE_TIMED_OUT = 'STEP_RUN_EVENT_TYPE_TIMED_OUT',
  STEP_RUN_EVENT_TYPE_STREAM = 'STEP_RUN_EVENT_TYPE_STREAM',
  WORKFLOW_RUN_EVENT_TYPE_STARTED = 'WORKFLOW_RUN_EVENT_TYPE_STARTED',
  WORKFLOW_RUN_EVENT_TYPE_COMPLETED = 'WORKFLOW_RUN_EVENT_TYPE_COMPLETED',
  WORKFLOW_RUN_EVENT_TYPE_FAILED = 'WORKFLOW_RUN_EVENT_TYPE_FAILED',
  WORKFLOW_RUN_EVENT_TYPE_CANCELLED = 'WORKFLOW_RUN_EVENT_TYPE_CANCELLED',
  WORKFLOW_RUN_EVENT_TYPE_TIMED_OUT = 'WORKFLOW_RUN_EVENT_TYPE_TIMED_OUT',
}

const stepEventTypeMap: Record<ResourceEventType, RunEventType | undefined> = {
  [ResourceEventType.RESOURCE_EVENT_TYPE_STARTED]: RunEventType.STEP_RUN_EVENT_TYPE_STARTED,
  [ResourceEventType.RESOURCE_EVENT_TYPE_COMPLETED]: RunEventType.STEP_RUN_EVENT_TYPE_COMPLETED,
  [ResourceEventType.RESOURCE_EVENT_TYPE_FAILED]: RunEventType.STEP_RUN_EVENT_TYPE_FAILED,
  [ResourceEventType.RESOURCE_EVENT_TYPE_CANCELLED]: RunEventType.STEP_RUN_EVENT_TYPE_CANCELLED,
  [ResourceEventType.RESOURCE_EVENT_TYPE_TIMED_OUT]: RunEventType.STEP_RUN_EVENT_TYPE_TIMED_OUT,
  [ResourceEventType.RESOURCE_EVENT_TYPE_STREAM]: RunEventType.STEP_RUN_EVENT_TYPE_STREAM,
  [ResourceEventType.RESOURCE_EVENT_TYPE_UNKNOWN]: undefined,
  [ResourceEventType.UNRECOGNIZED]: undefined,
};

const workflowEventTypeMap: Record<ResourceEventType, RunEventType | undefined> = {
  [ResourceEventType.RESOURCE_EVENT_TYPE_STARTED]: RunEventType.WORKFLOW_RUN_EVENT_TYPE_STARTED,
  [ResourceEventType.RESOURCE_EVENT_TYPE_COMPLETED]: RunEventType.WORKFLOW_RUN_EVENT_TYPE_COMPLETED,
  [ResourceEventType.RESOURCE_EVENT_TYPE_FAILED]: RunEventType.WORKFLOW_RUN_EVENT_TYPE_FAILED,
  [ResourceEventType.RESOURCE_EVENT_TYPE_CANCELLED]: RunEventType.WORKFLOW_RUN_EVENT_TYPE_CANCELLED,
  [ResourceEventType.RESOURCE_EVENT_TYPE_TIMED_OUT]: RunEventType.WORKFLOW_RUN_EVENT_TYPE_TIMED_OUT,
  [ResourceEventType.RESOURCE_EVENT_TYPE_UNKNOWN]: undefined,
  [ResourceEventType.RESOURCE_EVENT_TYPE_STREAM]: undefined,
  [ResourceEventType.UNRECOGNIZED]: undefined,
};

const resourceTypeMap: Record<
  ResourceType,
  Record<ResourceEventType, RunEventType | undefined> | undefined
> = {
  [ResourceType.RESOURCE_TYPE_STEP_RUN]: stepEventTypeMap,
  [ResourceType.RESOURCE_TYPE_WORKFLOW_RUN]: workflowEventTypeMap,
  [ResourceType.RESOURCE_TYPE_UNKNOWN]: undefined,
  [ResourceType.UNRECOGNIZED]: undefined,
};

const workflowStatusMap: Record<WorkflowRunStatus, RunEventType | undefined> = {
  [WorkflowRunStatus.SUCCEEDED]: RunEventType.WORKFLOW_RUN_EVENT_TYPE_COMPLETED,
  [WorkflowRunStatus.FAILED]: RunEventType.WORKFLOW_RUN_EVENT_TYPE_FAILED,
  [WorkflowRunStatus.CANCELLED]: RunEventType.WORKFLOW_RUN_EVENT_TYPE_CANCELLED,
  [WorkflowRunStatus.PENDING]: undefined,
  [WorkflowRunStatus.RUNNING]: undefined,
  [WorkflowRunStatus.QUEUED]: undefined,
};

export interface StepRunEvent {
  type: RunEventType;
  payload: string;
  resourceId: string;
  workflowRunId: string;
}

export class RunEventListener {
  client: DispatcherClient;

  q: Array<StepRunEvent> = [];
  eventEmitter = new EventEmitter();

  pollInterval: any;

  constructor(client: DispatcherClient) {
    this.client = client;
  }

  static forRunId(workflowRunId: string, client: DispatcherClient): RunEventListener {
    const listener = new RunEventListener(client);
    listener.listenForRunId(workflowRunId);
    return listener;
  }

  static forAdditionalMeta(key: string, value: string, client: DispatcherClient): RunEventListener {
    const listener = new RunEventListener(client);
    listener.listenForAdditionalMeta(key, value);
    return listener;
  }

  emit(event: StepRunEvent) {
    this.q.push(event);
    this.eventEmitter.emit('event');
  }

  async listenForRunId(workflowRunId: string) {
    const listenerFactory = () =>
      this.client.subscribeToWorkflowEvents({
        workflowRunId,
      });

    return this.listenLoop(listenerFactory);
  }

  async listenForAdditionalMeta(key: string, value: string) {
    const listenerFactory = () =>
      this.client.subscribeToWorkflowEvents({
        additionalMetaKey: key,
        additionalMetaValue: value,
      });

    return this.listenLoop(listenerFactory);
  }

  async listenLoop(listenerFactory: () => AsyncIterable<WorkflowEvent>) {
    let listener = listenerFactory();

    try {
      for await (const workflowEvent of listener) {
        const eventType = resourceTypeMap[workflowEvent.resourceType]?.[workflowEvent.eventType];
        if (eventType) {
          this.emit({
            type: eventType,
            payload: workflowEvent.eventPayload,
            resourceId: workflowEvent.resourceId,
            workflowRunId: workflowEvent.workflowRunId,
          });
        }
      }
    } catch (e: any) {
      if (e.code === Status.CANCELLED) {
        return;
      }
      if (e.code === Status.UNAVAILABLE) {
        listener = await this.retrySubscribe(listenerFactory);
      }
    }
  }

  async retrySubscribe(listenerFactory: () => AsyncIterable<WorkflowEvent>) {
    let retries = 0;

    while (retries < DEFAULT_EVENT_LISTENER_RETRY_COUNT) {
      try {
        await sleep(DEFAULT_EVENT_LISTENER_RETRY_INTERVAL);
        return listenerFactory();
      } catch (e: any) {
        retries += 1;
      }
    }

    throw new HatchetError(
      `Could not subscribe to the worker after ${DEFAULT_EVENT_LISTENER_RETRY_COUNT} retries`
    );
  }

  async *stream(): AsyncGenerator<StepRunEvent, void, unknown> {
    for await (const _ of on(this.eventEmitter, 'event')) {
      while (this.q.length > 0) {
        const r = this.q.shift();
        if (r) {
          yield r;
        }
      }
    }
  }
}

export class ListenerClient {
  config: ClientConfig;
  client: PbDispatcherClient;
  logger: Logger;
  api: Api;

  pooledListener: GrpcPooledListener | undefined;

  constructor(config: ClientConfig, channel: Channel, factory: ClientFactory, api: Api) {
    this.config = config;
    this.client = factory.create(DispatcherDefinition, channel);
    this.logger = config.logger(`Listener`, config.log_level);
    this.api = api;
  }

  get(workflowRunId: string) {
    if (!this.pooledListener) {
      this.pooledListener = new GrpcPooledListener(this, () => {
        this.pooledListener = undefined;
      });
    }

    return this.pooledListener.subscribe({
      workflowRunId,
    });
  }

  async stream(workflowRunId: string): Promise<AsyncGenerator<StepRunEvent, void, unknown>> {
    const listener = RunEventListener.forRunId(workflowRunId, this.client);
    return listener.stream();
  }

  async streamByRunId(workflowRunId: string): Promise<AsyncGenerator<StepRunEvent, void, unknown>> {
    return this.stream(workflowRunId);
  }

  async streamByAdditionalMeta(
    key: string,
    value: string
  ): Promise<AsyncGenerator<StepRunEvent, void, unknown>> {
    const listener = RunEventListener.forAdditionalMeta(key, value, this.client);
    return listener.stream();
  }
}
