// eslint-disable-next-line max-classes-per-file
import { EventEmitter, on } from 'events';
import {
  WorkflowRunEvent,
  SubscribeToWorkflowRunsRequest,
  WorkflowRunEventType,
} from '@hatchet/protoc/dispatcher';
import { isAbortError } from 'abort-controller-x';
import sleep from '@hatchet/util/sleep';
import { createAbortError } from '@hatchet/util/abort-error';
import { RunListenerClient } from './child-listener-client';

export class Streamable {
  listener: AsyncIterable<WorkflowRunEvent>;
  id: string;
  onCleanup: () => void;
  private cleanedUp = false;

  responseEmitter = new EventEmitter();

  constructor(listener: AsyncIterable<WorkflowRunEvent>, id: string, onCleanup: () => void) {
    this.listener = listener;
    this.id = id;
    this.onCleanup = onCleanup;
  }

  private cleanupOnce() {
    if (this.cleanedUp) return;
    this.cleanedUp = true;
    this.onCleanup();
  }

  async get(opts?: { signal?: AbortSignal }): Promise<WorkflowRunEvent> {
    const signal = opts?.signal;

    return new Promise((resolve, reject) => {
      const cleanupListeners = () => {
        this.responseEmitter.removeListener('response', onResponse);
        if (signal) {
          signal.removeEventListener('abort', onAbort);
        }
      };

      const onResponse = (event: WorkflowRunEvent) => {
        cleanupListeners();
        resolve(event);
      };

      const onAbort = () => {
        cleanupListeners();
        this.cleanupOnce();
        reject(createAbortError('Operation cancelled by AbortSignal'));
      };

      if (signal?.aborted) {
        onAbort();
        return;
      }

      this.responseEmitter.once('response', onResponse);
      if (signal) {
        signal.addEventListener('abort', onAbort, { once: true });
      }
    });
  }

  async *stream(opts?: { signal?: AbortSignal }): AsyncGenerator<WorkflowRunEvent, void, unknown> {
    while (true) {
      const event = await this.get(opts);
      yield event;

      if (event.eventType === WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_FINISHED) {
        this.cleanupOnce();
        break;
      }
    }
  }
}

export class RunGrpcPooledListener {
  listener: AsyncIterable<WorkflowRunEvent> | undefined;
  requestEmitter = new EventEmitter();
  signal: AbortController = new AbortController();
  client: RunListenerClient;

  subscribers: Record<string, Streamable> = {};
  onFinish: () => void = () => {};

  constructor(client: RunListenerClient, onFinish: () => void) {
    this.client = client;
    this.init();
    this.onFinish = onFinish;
  }

  private async init(retries = 0) {
    let retryCount = retries;
    const MAX_RETRY_INTERVAL = 5000; // 5 seconds in milliseconds
    const BASE_RETRY_INTERVAL = 100; // 0.1 seconds in milliseconds

    if (retries > 0) {
      const backoffTime = Math.min(BASE_RETRY_INTERVAL * 2 ** (retries - 1), MAX_RETRY_INTERVAL);
      this.client.logger.info(`Retrying in ... ${backoffTime / 1000} seconds`);
      await sleep(backoffTime);
    }

    try {
      this.client.logger.debug('Initializing child-listener');

      this.signal = new AbortController();
      this.listener = this.client.client.subscribeToWorkflowRuns(this.request(), {
        signal: this.signal.signal,
      });

      if (retries > 0) setTimeout(() => this.replayRequests(), 100);

      for await (const event of this.listener) {
        retryCount = 0;

        const emitter = this.subscribers[event.workflowRunId];
        if (emitter) {
          emitter.responseEmitter.emit('response', event);
          if (event.eventType === WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_FINISHED) {
            delete this.subscribers[event.workflowRunId];
          }
        }
      }

      this.client.logger.debug('Child listener finished');
    } catch (e: any) {
      if (isAbortError(e)) {
        this.client.logger.debug('Child Listener aborted');
        return;
      }
      this.client.logger.error(`Error in child-listener: ${e.message}`);
    } finally {
      // it is possible the server hangs up early,
      // restart the listener if we still have subscribers
      this.client.logger.debug(
        `Child listener loop exited with ${Object.keys(this.subscribers).length} subscribers`
      );
      this.client.logger.debug(`Restarting child listener retry ${retryCount + 1}`);
      this.init(retryCount + 1);
    }
  }

  subscribe(request: SubscribeToWorkflowRunsRequest) {
    if (!this.listener) throw new Error('listener not initialized');

    this.subscribers[request.workflowRunId] = new Streamable(
      this.listener,
      request.workflowRunId,
      () => {
        delete this.subscribers[request.workflowRunId];
      }
    );
    this.requestEmitter.emit('subscribe', request);
    return this.subscribers[request.workflowRunId];
  }

  replayRequests() {
    const subs = Object.values(this.subscribers);
    this.client.logger.debug(`Replaying ${subs.length} requests...`);

    for (const subscriber of subs) {
      this.requestEmitter.emit('subscribe', { workflowRunId: subscriber.id });
    }
  }

  private async *request(): AsyncIterable<SubscribeToWorkflowRunsRequest> {
    for await (const e of on(this.requestEmitter, 'subscribe')) {
      yield e[0];
    }
  }
}
