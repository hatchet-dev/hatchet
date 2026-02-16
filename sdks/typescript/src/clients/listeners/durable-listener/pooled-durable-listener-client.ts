// eslint-disable-next-line max-classes-per-file
import { EventEmitter, on } from 'events';
import {
  DurableEvent,
  ListenForDurableEventRequest,
  RegisterDurableEventRequest,
  RegisterDurableEventResponse,
} from '@hatchet/protoc/v1/dispatcher';
import { isAbortError } from 'abort-controller-x';
import sleep from '@hatchet/util/sleep';
import { createAbortError } from '@hatchet/util/abort-error';
import {
  DurableEventListenerConditions,
  SleepMatchCondition,
  UserEventMatchCondition,
} from '@hatchet/protoc/v1/shared/condition';
import { DurableListenerClient } from './durable-listener-client';

export class DurableEventStreamable {
  listener: AsyncIterable<DurableEvent>;
  taskId: string;
  signalKey: string;
  subscriptionId: string;
  onCleanup: () => void;

  responseEmitter = new EventEmitter();

  constructor(
    listener: AsyncIterable<DurableEvent>,
    taskId: string,
    signalKey: string,
    subscriptionId: string,
    onCleanup: () => void
  ) {
    this.listener = listener;
    this.taskId = taskId;
    this.signalKey = signalKey;
    this.subscriptionId = subscriptionId;
    this.onCleanup = onCleanup;
  }

  async get(opts?: { signal?: AbortSignal }): Promise<DurableEvent> {
    const signal = opts?.signal;

    return new Promise((resolve, reject) => {
      let cleanedUp = false;

      const cleanup = () => {
        if (cleanedUp) return;
        cleanedUp = true;
        this.responseEmitter.removeListener('response', onResponse);
        if (signal) {
          signal.removeEventListener('abort', onAbort);
        }
        this.onCleanup();
      };

      const onResponse = (event: DurableEvent) => {
        cleanup();
        resolve(event);
      };

      const onAbort = () => {
        cleanup();
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
}

export class DurableEventGrpcPooledListener {
  listener: AsyncIterable<DurableEvent> | undefined;
  requestEmitter = new EventEmitter();
  signal: AbortController = new AbortController();
  client: DurableListenerClient;

  subscribers: Record<string, DurableEventStreamable> = {};
  taskSignalKeyToSubscriptionIds: Record<string, string[]> = {};
  onFinish: () => void = () => {};

  private subscriptionCounter = 0;
  private currRequester = 0;
  private readonly DEFAULT_INTERRUPT_INTERVAL = 15 * 60 * 1000; // 15 minutes in milliseconds

  constructor(client: DurableListenerClient, onFinish: () => void) {
    this.client = client;
    this.init();
    this.onFinish = onFinish;
    this.scheduleInterrupt();
  }

  private scheduleInterrupt() {
    setTimeout(() => {
      if (this.signal) {
        this.signal.abort();
        this.init(0);
      }
      this.scheduleInterrupt();
    }, this.DEFAULT_INTERRUPT_INTERVAL);
  }

  private async init(retries = 0) {
    let retryCount = retries;
    const MAX_RETRY_INTERVAL = 5000; // 5 seconds in milliseconds
    const BASE_RETRY_INTERVAL = 100; // 0.1 seconds in milliseconds
    const MAX_RETRY_COUNT = 5;

    if (retries > 0) {
      const backoffTime = Math.min(BASE_RETRY_INTERVAL * 2 ** (retries - 1), MAX_RETRY_INTERVAL);
      this.client.logger.info(`Retrying in ... ${backoffTime / 1000} seconds`);
      await sleep(backoffTime);
    }

    if (retries > MAX_RETRY_COUNT) {
      this.client.logger.error('Max retry count exceeded for durable event listener');
      return;
    }

    try {
      this.client.logger.debug('Initializing durable-event-listener');

      this.signal = new AbortController();
      // eslint-disable-next-line no-plusplus
      this.currRequester++;

      this.listener = this.client.client.listenForDurableEvent(this.request(), {
        signal: this.signal.signal,
      });

      if (retries > 0) setTimeout(() => this.replayRequests(), 100);

      for await (const event of this.listener) {
        retryCount = 0;

        const subscriptionKey = keyHelper(event.taskId, event.signalKey);
        const subscriptionIds = this.taskSignalKeyToSubscriptionIds[subscriptionKey] || [];

        for (const subId of subscriptionIds) {
          const emitter = this.subscribers[subId];
          if (emitter) {
            emitter.responseEmitter.emit('response', event);
            this.cleanupSubscription(subId);
          }
        }
      }

      this.client.logger.debug('Durable event listener finished');
    } catch (e: any) {
      if (isAbortError(e)) {
        this.client.logger.debug('Durable event listener aborted');
        return;
      }
      this.client.logger.error(`Error in durable-event-listener: ${e.message}`);
    } finally {
      const subscriberCount = Object.keys(this.subscribers).length;
      if (subscriberCount > 0) {
        this.client.logger.debug(
          `Durable event listener loop exited with ${subscriberCount} subscribers`
        );
        this.client.logger.debug(`Restarting durable event listener retry ${retryCount + 1}`);
        this.init(retryCount + 1);
      }
    }
  }

  private cleanupSubscription(subscriptionId: string) {
    const emitter = this.subscribers[subscriptionId];
    if (!emitter) {
      return;
    }

    const subscriptionKey = keyHelper(emitter.taskId, emitter.signalKey);

    delete this.subscribers[subscriptionId];

    // Remove from the mapping
    if (this.taskSignalKeyToSubscriptionIds[subscriptionKey]) {
      this.taskSignalKeyToSubscriptionIds[subscriptionKey] = this.taskSignalKeyToSubscriptionIds[
        subscriptionKey
      ].filter((id) => id !== subscriptionId);

      if (this.taskSignalKeyToSubscriptionIds[subscriptionKey].length === 0) {
        delete this.taskSignalKeyToSubscriptionIds[subscriptionKey];
      }
    }
  }

  subscribe(request: { taskId: string; signalKey: string }): DurableEventStreamable {
    const { taskId, signalKey } = request;

    if (!this.listener) throw new Error('listener not initialized');

    // eslint-disable-next-line no-plusplus
    const subscriptionId = (this.subscriptionCounter++).toString();
    const subscriber = new DurableEventStreamable(
      this.listener,
      taskId,
      signalKey,
      subscriptionId,
      () => this.cleanupSubscription(subscriptionId)
    );

    this.subscribers[subscriptionId] = subscriber;

    const key = keyHelper(taskId, signalKey);
    if (!this.taskSignalKeyToSubscriptionIds[key]) {
      this.taskSignalKeyToSubscriptionIds[key] = [];
    }
    this.taskSignalKeyToSubscriptionIds[key].push(subscriptionId);

    this.requestEmitter.emit('subscribe', { taskId, signalKey });
    return subscriber;
  }

  async result(
    request: { taskId: string; signalKey: string },
    opts?: { signal?: AbortSignal }
  ): Promise<DurableEvent> {
    const subscriber = this.subscribe(request);
    const event = await subscriber.get({ signal: opts?.signal });
    return event;
  }

  async registerDurableEvent(request: {
    taskId: string;
    signalKey: string;
    sleepConditions: Array<SleepMatchCondition>;
    userEventConditions: Array<UserEventMatchCondition>;
  }): Promise<RegisterDurableEventResponse> {
    const conditions: DurableEventListenerConditions = {
      sleepConditions: request.sleepConditions,
      userEventConditions: request.userEventConditions,
    };

    const registerRequest: RegisterDurableEventRequest = {
      taskId: request.taskId,
      signalKey: request.signalKey,
      conditions,
    };

    return this.client.client.registerDurableEvent(registerRequest);
  }

  replayRequests() {
    const subscriptionEntries = Object.entries(this.taskSignalKeyToSubscriptionIds);
    this.client.logger.debug(`Replaying ${subscriptionEntries.length} requests...`);

    for (const [key] of subscriptionEntries) {
      const [taskId, signalKey] = key.split('|');
      this.requestEmitter.emit('subscribe', { taskId, signalKey });
    }
  }

  private async *request(): AsyncIterable<ListenForDurableEventRequest> {
    const { currRequester } = this;

    // Replay existing subscriptions
    const existingSubscriptions = new Set<string>();

    for (const key in this.taskSignalKeyToSubscriptionIds) {
      if (this.taskSignalKeyToSubscriptionIds[key].length > 0) {
        const [taskId, signalKey] = key.split('|');
        existingSubscriptions.add(key);
        yield { taskId, signalKey };
      }
    }

    for await (const e of on(this.requestEmitter, 'subscribe')) {
      // Stop if this requester is outdated
      if (currRequester !== this.currRequester) break;

      const request = e[0] as ListenForDurableEventRequest;
      const key = keyHelper(request.taskId, request.signalKey);

      // Only send unique subscriptions
      if (!existingSubscriptions.has(key)) {
        existingSubscriptions.add(key);
        yield request;
      }
    }
  }
}

const keyHelper = (taskId: string, signalKey: string) => `${taskId}|${signalKey}`;
