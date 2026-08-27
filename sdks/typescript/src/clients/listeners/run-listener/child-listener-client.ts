import { Channel, ClientFactory, Status } from 'nice-grpc';
import { getGrpcErrorCode } from '@util/grpc-error';
import { isAbortError } from 'abort-controller-x';
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
import { getErrorMessage } from '@util/errors/hatchet-error';
import { Logger } from '@hatchet/util/logger';
import sleep from '@hatchet/util/sleep';
import { bindAbortSignalHandler, throwIfAborted } from '@hatchet/util/abort-error';
import { classifyListenerFailure } from '@clients/dispatcher/listener-severity';
import { Api } from '../../rest';
import { RunGrpcPooledListener } from './pooled-child-listener-client';

// Reconnect backoff for the run event stream. Mirrors the pooled workflow-run
// listener (RunGrpcPooledListener) so `.stream()` recovers from engine
// redeploys/restarts the same way `.result()` does.
const BASE_RETRY_INTERVAL_MS = 100;
const MAX_RETRY_INTERVAL_MS = 5000;

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

export interface StepRunEvent {
  type: RunEventType;
  payload: string;
  resourceId: string;
  workflowRunId: string;
}

export interface RunEventListenerOpts {
  /**
   * Optional AbortSignal. When aborted, the listener stops reconnecting, cancels
   * the underlying gRPC stream, and surfaces the abort to `stream()` consumers as
   * an error — matching the pooled `.result()` path.
   */
  signal?: AbortSignal;
  logger?: Logger;
}

export class RunEventListener {
  client: DispatcherClient;

  q: Array<StepRunEvent> = [];
  eventEmitter = new EventEmitter();

  private signal?: AbortSignal;
  logger?: Logger;

  /** AbortController for the in-flight gRPC subscription; recreated per attempt. */
  private abortController?: AbortController;

  /** Set once the listener should stop for good (completed, aborted, or hung up). */
  private done = false;

  /** True when termination was caused by the caller-supplied AbortSignal. */
  private externallyAborted = false;

  constructor(client: DispatcherClient, opts?: RunEventListenerOpts) {
    this.client = client;
    this.signal = opts?.signal;
    this.logger = opts?.logger;

    if (this.signal) {
      if (this.signal.aborted) {
        this.externallyAborted = true;
        this.done = true;
      } else {
        bindAbortSignalHandler(this.signal, () => {
          this.externallyAborted = true;
          this.close();
        });
      }
    }
  }

  static forRunId(
    workflowRunId: string,
    client: DispatcherClient,
    opts?: RunEventListenerOpts
  ): RunEventListener {
    const listener = new RunEventListener(client, opts);
    listener.start(() => listener.listenForRunId(workflowRunId));
    return listener;
  }

  static forAdditionalMeta(
    key: string,
    value: string,
    client: DispatcherClient,
    opts?: RunEventListenerOpts
  ): RunEventListener {
    const listener = new RunEventListener(client, opts);
    listener.start(() => listener.listenForAdditionalMeta(key, value));
    return listener;
  }

  /**
   * Kicks off the detached listen loop and guarantees an unexpected rejection can
   * never surface as an unhandled promise rejection — it terminates the stream
   * cleanly instead, the same way the pooled listener's `init()` does.
   */
  private start(run: () => Promise<void>) {
    run().catch((e) => {
      this.logger?.error(`Run event listener stopped unexpectedly: ${getErrorMessage(e)}`);
      this.close();
      this.eventEmitter.emit('complete');
    });
  }

  /**
   * Stop reconnecting and cancel the in-flight gRPC stream. Idempotent.
   */
  close() {
    this.done = true;
    this.abortController?.abort();
  }

  emit(event: StepRunEvent) {
    this.q.push(event);
    this.eventEmitter.emit('event');
  }

  async listenForRunId(workflowRunId: string) {
    return this.listenLoop((signal) =>
      this.client.subscribeToWorkflowEvents({ workflowRunId }, { signal })
    );
  }

  async listenForAdditionalMeta(key: string, value: string) {
    return this.listenLoop((signal) =>
      this.client.subscribeToWorkflowEvents(
        { additionalMetaKey: key, additionalMetaValue: value },
        { signal }
      )
    );
  }

  async listenLoop(listenerFactory: (signal: AbortSignal) => AsyncIterable<WorkflowEvent>) {
    let retries = 0;

    while (!this.done) {
      if (retries > 0) {
        const backoff = Math.min(
          BASE_RETRY_INTERVAL_MS * 2 ** (retries - 1),
          MAX_RETRY_INTERVAL_MS
        );
        this.logger?.info(`Retrying run event listener in ${backoff / 1000}s...`);
        await sleep(backoff);
        if (this.done) break;
      }

      this.abortController = new AbortController();

      try {
        for await (const workflowEvent of listenerFactory(this.abortController.signal)) {
          // A successful message means the connection is healthy again.
          retries = 0;

          const eventType = resourceTypeMap[workflowEvent.resourceType]?.[workflowEvent.eventType];
          if (eventType) {
            this.emit({
              type: eventType,
              payload: workflowEvent.eventPayload,
              resourceId: workflowEvent.resourceId,
              workflowRunId: workflowEvent.workflowRunId,
            });
          }

          if (workflowEvent.hangup) {
            this.done = true;
            break;
          }
        }

        // Stream ended without an error and without an explicit hangup. Preserve
        // the long-standing behavior of treating a clean EOF as "server is done".
        this.done = true;
      } catch (e: unknown) {
        if (isAbortError(e) || this.done || getGrpcErrorCode(e) === Status.CANCELLED) {
          this.done = true;
          break;
        }

        // Every other error (UNAVAILABLE from an engine redeploy, RST_STREAM,
        // transient network failures, ...) is retryable: keep reconnecting with
        // exponential backoff, exactly like the pooled `.result()` listener.
        retries += 1;
        const severity = classifyListenerFailure(e, retries);
        if (severity !== 'silent') {
          this.logger?.[severity](`Run event listener error: ${getErrorMessage(e)}`);
        }
      }
    }

    this.eventEmitter.emit('complete');
  }

  async *stream(): AsyncGenerator<StepRunEvent, void, unknown> {
    let completed = this.done;

    const onComplete = () => {
      completed = true;
      this.eventEmitter.emit('event');
    };
    this.eventEmitter.once('complete', onComplete);

    try {
      // Skip waiting on 'event' entirely if the loop already finished before we
      // subscribed (e.g. a signal that was aborted before `stream()` was called).
      if (!completed) {
        for await (const _ of on(this.eventEmitter, 'event')) {
          while (this.q.length > 0) {
            const r = this.q.shift();
            if (r) {
              yield r;
            }
          }

          if (completed && this.q.length === 0) {
            break;
          }
        }
      }

      // Flush anything buffered before the loop terminated.
      while (this.q.length > 0) {
        const r = this.q.shift();
        if (r) {
          yield r;
        }
      }
    } finally {
      // Consumer stopped iterating (return/break/throw) or we finished: tear down
      // the reconnect loop and the underlying gRPC stream so we don't leak a
      // subscription that keeps reconnecting across redeploys.
      this.eventEmitter.removeListener('complete', onComplete);
      this.close();
    }

    // Only reached when iteration completed on its own (not via consumer return).
    // Surface a caller-triggered abort as an error, mirroring `.result()`.
    if (this.externallyAborted) {
      throwIfAborted(this.signal, 'Run event stream aborted');
    }
  }
}

export class RunListenerClient {
  config: ClientConfig;
  client: PbDispatcherClient;
  logger: Logger;
  api: Api;

  pooledListener: RunGrpcPooledListener | undefined;

  constructor(config: ClientConfig, channel: Channel, factory: ClientFactory, api: Api) {
    this.config = config;
    this.client = factory.create(DispatcherDefinition, channel);
    this.logger = config.logger(`Listener`, config.log_level);
    this.api = api;
  }

  get(workflowRunId: string) {
    if (!this.pooledListener) {
      this.pooledListener = new RunGrpcPooledListener(this, () => {
        this.pooledListener = undefined;
      });
    }

    return this.pooledListener.subscribe({
      workflowRunId,
    });
  }

  async stream(
    workflowRunId: string,
    opts?: { signal?: AbortSignal }
  ): Promise<AsyncGenerator<StepRunEvent, void, unknown>> {
    const listener = RunEventListener.forRunId(workflowRunId, this.client, {
      signal: opts?.signal,
      logger: this.logger,
    });
    return listener.stream();
  }

  async streamByRunId(
    workflowRunId: string,
    opts?: { signal?: AbortSignal }
  ): Promise<AsyncGenerator<StepRunEvent, void, unknown>> {
    return this.stream(workflowRunId, opts);
  }

  async streamByAdditionalMeta(
    key: string,
    value: string,
    opts?: { signal?: AbortSignal }
  ): Promise<AsyncGenerator<StepRunEvent, void, unknown>> {
    const listener = RunEventListener.forAdditionalMeta(key, value, this.client, {
      signal: opts?.signal,
      logger: this.logger,
    });
    return listener.stream();
  }
}
