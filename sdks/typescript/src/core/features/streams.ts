import { Code, ConnectError } from '@connectrpc/connect';
import HatchetError from '@util/errors/hatchet-error';
import { AbortError } from '@hatchet/util/abort-error';
import type { EventsServiceClient, PutStreamEventResponse } from '@hatchet/protoc/events/events';
import type { V1StreamsClient } from '@hatchet/protoc/v1/streams';
import { StreamPublishQueue, validateBatchTimeout } from '@util/stream-publish-batch';
import sleep from '@hatchet/util/sleep';
import type { StreamCallOptions, StreamEvent, StreamTopicMetadata } from '../types';

/**
 * The largest payload one `publish` carries: the engine's limit for a payload sent inline
 * (`MaxStreamMessagePayloadBytes`). The Node client uploads larger payloads over its REST
 * client first; this client has none, so it rejects them.
 */
export const MAX_STREAM_MESSAGE_BYTES = 4 * 1024 * 1024 - 5 * 1024;

export interface StreamsClientDeps {
  events: EventsServiceClient;
  streams: V1StreamsClient;
  /**
   * Reads a payload the engine stored by ref (one published through an upload) from the REST
   * API. Without it, `subscribe` throws on reaching such a message.
   */
  fetchPayload?: (payloadRef: string, signal?: AbortSignal) => Promise<Uint8Array>;
}

/**
 * How many (namespace, topic) producers the client remembers. A topic past the cap starts
 * over with a new producer id, which the engine orders after the old one, so correctness
 * does not depend on the cap; it only bounds memory for clients that publish to many
 * short-lived topics.
 */
export const MAX_TRACKED_TOPICS = 1024;

function isEventsService(
  deps: StreamsClientDeps | EventsServiceClient
): deps is EventsServiceClient {
  return typeof (deps as EventsServiceClient).putStreamEvent === 'function';
}

/** Retrying a subscribe that failed with these can't succeed. */
const permanentSubscribeErrors = new Set<Code>([
  Code.InvalidArgument,
  Code.PermissionDenied,
  Code.Unauthenticated,
  Code.OutOfRange,
  Code.FailedPrecondition,
  Code.NotFound,
]);

const RESUBSCRIBE_MIN_DELAY_MS = 250;
const RESUBSCRIBE_MAX_DELAY_MS = 10_000;
const PAYLOAD_FETCH_ATTEMPTS = 3;

/** The status of an error response the fetch REST client threw, which is the response itself. */
function httpStatus(err: unknown): number | undefined {
  if (typeof err === 'object' && err !== null && 'status' in err && 'ok' in err) {
    return (err as { status: number }).status;
  }
  return undefined;
}

function shouldResubscribe(err: unknown, attempt: number): boolean {
  if (err instanceof ConnectError) {
    if (permanentSubscribeErrors.has(err.code)) {
      return false;
    }
    // proxies answer 404 while an engine restarts, but on a first attempt it means no streams support
    if (err.code === Code.Unimplemented) {
      return attempt > 0;
    }
    return true;
  }
  // a payload fetch: a 5xx or 429 may succeed later, and fetch rejects with a TypeError when no
  // response arrives
  const status = httpStatus(err);
  if (status !== undefined) {
    return status >= 500 || status === 429;
  }
  return err instanceof TypeError;
}

/** The engine stored nothing, so the producer's sequence number is still unused. */
function publishRejectedBeforeStoring(err: unknown): boolean {
  return (
    err instanceof ConnectError &&
    (err.code === Code.InvalidArgument ||
      err.code === Code.ResourceExhausted ||
      err.code === Code.Unauthenticated ||
      err.code === Code.PermissionDenied ||
      // an engine without the RPC
      err.code === Code.Unimplemented)
  );
}

interface QueuedPublish {
  payload: Uint8Array;
  signal: AbortSignal | undefined;
  resolve: () => void;
  reject: (err: unknown) => void;
}

/**
 * Task run streams and durable stream topics.
 *
 * `put` publishes a chunk on a task run's stream, the way `ctx.putStream` does from a worker.
 * `publish`, `subscribe`, `reset` and `topicMetadata` are the Node client's durable topic calls
 * (`hatchet.streams`), with the same names and options.
 */
export class StreamsClient {
  /** Per (namespace, topic); the sequence number only advances after a publish succeeds. */
  private readonly producers = new Map<string, { producerId: string; seq: number }>();

  /**
   * Publishes waiting per (namespace, topic). One request per topic is in flight at a time, so
   * two never share a sequence number, and each carries everything queued behind the last.
   */
  private readonly publishQueues = new Map<string, StreamPublishQueue<QueuedPublish>>();

  /** Set once the engine answers `PublishBatch` as unimplemented: it predates batching. */
  private publishBatchUnsupported = false;

  private readonly eventsService: EventsServiceClient;
  private readonly streams: V1StreamsClient | undefined;
  private readonly payloads: StreamsClientDeps['fetchPayload'];

  /**
   * Takes the events and durable streams services, or only the events service, which
   * serves `put` and leaves the durable stream calls unavailable.
   */
  constructor(deps: StreamsClientDeps | EventsServiceClient) {
    if (isEventsService(deps)) {
      this.eventsService = deps;
      this.streams = undefined;
    } else {
      this.eventsService = deps.events;
      this.streams = deps.streams;
      this.payloads = deps.fetchPayload;
    }
  }

  private durable(): V1StreamsClient {
    if (!this.streams) {
      throw new HatchetError(
        'durable streams are unavailable: this StreamsClient was built with the events service only; build it with { events, streams }'
      );
    }
    return this.streams;
  }

  /**
   * Publishes one chunk on a task run's stream. `index` orders chunks that are published
   * concurrently. Resolves once the engine has the chunk.
   */
  put(
    taskRunExternalId: string,
    data: string | Uint8Array,
    index?: number
  ): Promise<PutStreamEventResponse> {
    const message = typeof data === 'string' ? new TextEncoder().encode(data) : data;
    return this.eventsService.putStreamEvent({
      taskRunExternalId,
      createdAt: new Date(),
      message,
      eventIndex: index,
    });
  }

  /**
   * Publishes a message to a durable topic and resolves once it is stored. A client's
   * messages to a topic are delivered in the order `publish` was called. A payload over
   * `MAX_STREAM_MESSAGE_BYTES` rejects with a `HatchetError` before anything is sent.
   * Publishes to a topic share requests: everything queued while one is in flight goes out in
   * the next, and `batchTimeoutMs` holds a message up to that long for more to join it.
   * @param topic - the topic to publish to
   * @param message - the message payload
   * @param options - optional namespace override, abort signal and batch timeout
   */
  async publish(
    topic: string,
    message: Uint8Array | string,
    options?: StreamCallOptions
  ): Promise<void> {
    const payload = typeof message === 'string' ? new TextEncoder().encode(message) : message;
    const namespace = options?.namespace ?? '';
    const key = `${namespace}\0${topic}`;

    // the engine would reject the whole request it shares, failing other publishes too
    if (payload.byteLength === 0) {
      throw new HatchetError('stream payload is empty; a message must carry at least one byte');
    }

    if (payload.byteLength > MAX_STREAM_MESSAGE_BYTES) {
      throw new HatchetError(
        `stream payload is ${payload.byteLength} bytes, over the ${MAX_STREAM_MESSAGE_BYTES}-byte limit of one publish; publish it from the Node client, which uploads larger payloads`
      );
    }

    validateBatchTimeout(options?.batchTimeoutMs);

    const signal = options?.signal;
    const publishPromise = new Promise<void>((resolve, reject) => {
      const queue =
        this.publishQueues.get(key) ??
        new StreamPublishQueue<QueuedPublish>((p) => p.payload.byteLength);
      queue.push({ payload, signal, resolve, reject }, options?.batchTimeoutMs);

      // pushed first, since the drain stops at an empty queue
      if (!this.publishQueues.has(key)) {
        this.publishQueues.set(key, queue);
        void this.drainPublishes(key, namespace, topic, queue);
      }
    });

    if (!signal) {
      return publishPromise;
    }

    // The caller is released as soon as its signal aborts, while the queue keeps its order
    // and the aborted publish is skipped when its turn comes.
    return new Promise<void>((resolve, reject) => {
      const onAbort = () => reject(new AbortError());
      if (signal.aborted) {
        onAbort();
        return;
      }
      signal.addEventListener('abort', onAbort, { once: true });
      publishPromise.then(resolve, reject).finally(() => {
        signal.removeEventListener('abort', onAbort);
      });
    });
  }

  /**
   * Describes a topic's retained messages. Rejects with a `ConnectError` of code `NotFound`
   * for a topic nothing was published to.
   * @param topic - the topic to describe
   * @param options - optional namespace override and abort signal
   */
  async topicMetadata(
    topic: string,
    options?: Pick<StreamCallOptions, 'namespace' | 'signal'>
  ): Promise<StreamTopicMetadata> {
    const res = await this.durable().getTopicMetadata(
      { namespace: options?.namespace ?? '', topic },
      { signal: options?.signal }
    );

    return {
      namespace: res.namespace,
      topic: res.topic,
      tenantId: res.tenantId,
      messageCount: res.messageCount,
      latestCursor: res.latestCursor,
      lastPublishedAt: res.lastPublishedAt,
    };
  }

  /**
   * Yields topic's messages after `options.cursor`, or from the oldest retained (or, once the
   * topic is reset, after the reset). If the connection drops, it resubscribes from the last
   * message yielded, so nothing is skipped or repeated. Ends when the server hangs up an idle
   * subscription, `options.signal` aborts, or the caller stops iterating; throws on errors a
   * retry can't fix, such as an expired cursor.
   * @param topic - the topic to read from
   * @param options - optional namespace override, resume cursor and abort signal
   */
  async *subscribe(topic: string, options?: StreamCallOptions): AsyncIterable<StreamEvent> {
    const streams = this.durable();
    const namespace = options?.namespace ?? '';
    const signal = options?.signal;
    let { cursor } = options ?? {};
    let failures = 0;

    for (let attempt = 0; ; attempt += 1) {
      try {
        for await (const msg of streams.subscribe({ namespace, topic, cursor }, { signal })) {
          if (msg.hangup) {
            return;
          }

          for (const entry of msg.entries) {
            // aborting cancels the request, but a message's remaining entries are already here
            if (signal?.aborted) {
              return;
            }
            const payload = entry.payloadRef
              ? await this.fetchPayload(entry.payloadRef, signal)
              : entry.payload;
            if (signal?.aborted) {
              return;
            }
            ({ cursor } = entry);
            // reset only on delivery, so a payload fetch failing on every reconnect still backs off
            failures = 0;
            yield { payload, cursor: entry.cursor, createdAt: entry.createdAt };
          }
        }
        // ending without a hangup means the engine went away, e.g. it is shutting down
      } catch (err) {
        if (signal?.aborted) {
          return;
        }
        if (!shouldResubscribe(err, attempt)) {
          throw err;
        }
      }

      const delayMs = Math.min(RESUBSCRIBE_MIN_DELAY_MS * 2 ** failures, RESUBSCRIBE_MAX_DELAY_MS);
      failures += 1;

      try {
        await sleep(delayMs * (0.5 + Math.random() / 2), signal);
      } catch {
        return;
      }
    }
  }

  /** Retries a 5xx or network failure; a 4xx won't succeed on retry. */
  private async fetchPayload(payloadRef: string, signal?: AbortSignal): Promise<Uint8Array> {
    if (!this.payloads) {
      throw new HatchetError(
        'the message payload is stored by ref, which this StreamsClient cannot read; build it with fetchPayload'
      );
    }

    for (let attempt = 1; ; attempt += 1) {
      try {
        return await this.payloads(payloadRef, signal);
      } catch (err) {
        const status = httpStatus(err);
        if (
          signal?.aborted ||
          (status !== undefined && status < 500) ||
          attempt >= PAYLOAD_FETCH_ATTEMPTS
        ) {
          throw err;
        }
      }
    }
  }

  /**
   * Makes `subscribe()` calls without a cursor start after `options.cursor`, or the topic's latest
   * message. Readers already iterating, and any call with a cursor, are unaffected.
   * @param topic - the topic to reset
   * @param options - optional namespace override, cursor to reset to and abort signal
   */
  async reset(
    topic: string,
    options?: Pick<StreamCallOptions, 'namespace' | 'cursor' | 'signal'>
  ): Promise<void> {
    await this.durable().reset(
      { namespace: options?.namespace ?? '', topic, cursor: options?.cursor },
      { signal: options?.signal }
    );
  }

  /** Sends key's queued publishes in batches until none are left, then forgets the queue. */
  private async drainPublishes(
    key: string,
    namespace: string,
    topic: string,
    queue: StreamPublishQueue<QueuedPublish>
  ): Promise<void> {
    while (queue.size > 0) {
      // An abort while a publish waited behind earlier ones sends nothing.
      const batch = (await queue.next()).filter((p) => {
        if (p.signal?.aborted) {
          p.reject(new AbortError());
          return false;
        }
        return true;
      });

      if (batch.length > 0) {
        await this.sendPublishes(key, namespace, topic, batch);
      }
    }

    this.publishQueues.delete(key);
  }

  /**
   * Sends batch as one request, or as one request per message to an engine that predates
   * `PublishBatch`, and settles each publish in it.
   */
  private async sendPublishes(
    key: string,
    namespace: string,
    topic: string,
    batch: QueuedPublish[]
  ): Promise<void> {
    if (!this.publishBatchUnsupported) {
      try {
        await this.publishOrdered(
          key,
          namespace,
          topic,
          batch.map((p) => p.payload)
        );
        batch.forEach((p) => p.resolve());
        return;
      } catch (err) {
        if (!(err instanceof ConnectError && err.code === Code.Unimplemented)) {
          batch.forEach((p) => p.reject(err));
          return;
        }
        // nothing was stored, so the batch goes out again below
        this.publishBatchUnsupported = true;
      }
    }

    // Settled one by one: a failure partway leaves the publishes before it stored.
    for (const p of batch) {
      try {
        await this.publishOrdered(key, namespace, topic, [p.payload]);
        p.resolve();
      } catch (err) {
        p.reject(err);
      }
    }
  }

  private async publishOrdered(
    key: string,
    namespace: string,
    topic: string,
    payloads: Uint8Array[],
    retryGap = true
  ): Promise<void> {
    const streams = this.durable();
    let producer = this.producers.get(key);
    if (producer) {
      // re-inserting keeps the map in last-use order for the eviction below
      this.producers.delete(key);
    } else {
      producer = { producerId: crypto.randomUUID(), seq: 0 };
    }
    this.producers.set(key, producer);
    while (this.producers.size > MAX_TRACKED_TOPICS) {
      const oldest = this.producers.keys().next().value;
      if (oldest === undefined) {
        break;
      }
      this.producers.delete(oldest);
    }

    try {
      // not one caller's signal: the request carries other callers' publishes too
      if (this.publishBatchUnsupported && payloads.length === 1) {
        await streams.publish({
          namespace,
          topic,
          payload: payloads[0],
          producerId: producer.producerId,
          producerSeq: producer.seq,
        });
      } else {
        await streams.publishBatch({
          namespace,
          topic,
          producerId: producer.producerId,
          firstProducerSeq: producer.seq,
          entries: payloads.map((payload) => ({ payload, payloadRef: '' })),
        });
      }
    } catch (err) {
      // The message may still land, and a different payload under its sequence number
      // would be dropped as a duplicate, so the next publish comes from a new producer.
      if (!publishRejectedBeforeStoring(err)) {
        this.producers.set(key, { producerId: crypto.randomUUID(), seq: 0 });
      }
      // A gap stored nothing (the watermark passed cursor retention, say): resend as the
      // new producer.
      if (retryGap && err instanceof ConnectError && err.code === Code.FailedPrecondition) {
        return this.publishOrdered(key, namespace, topic, payloads, false);
      }
      throw err;
    }

    producer.seq += payloads.length;
  }
}
