import { Code, ConnectError } from '@connectrpc/connect';
import HatchetError from '@util/errors/hatchet-error';
import { AbortError } from '@hatchet/util/abort-error';
import type { EventsServiceClient, PutStreamEventResponse } from '@hatchet/protoc/events/events';
import type { V1StreamsClient } from '@hatchet/protoc/v1/streams';
import { StreamPublishQueue } from '@util/stream-publish-batch';
import type { StreamCallOptions, StreamTopicMetadata } from '../types';

/**
 * The largest payload one `publish` carries: the engine's limit for a payload sent inline
 * (`MaxStreamMessagePayloadBytes`). The Node client uploads larger payloads over its REST
 * client first; this client has none, so it rejects them.
 */
export const MAX_STREAM_MESSAGE_BYTES = 4 * 1024 * 1024 - 5 * 1024;

export interface StreamsClientDeps {
  events: EventsServiceClient;
  streams: V1StreamsClient;
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

/** The engine stored nothing, so the producer's sequence number is still unused. */
function publishRejectedBeforeStoring(err: unknown): boolean {
  return (
    err instanceof ConnectError &&
    (err.code === Code.InvalidArgument ||
      err.code === Code.ResourceExhausted ||
      err.code === Code.Unauthenticated ||
      err.code === Code.PermissionDenied)
  );
}

interface QueuedPublish {
  bytes: number;
  sendBy: number;
  payload: Uint8Array;
  signal: AbortSignal | undefined;
  resolve: () => void;
  reject: (err: unknown) => void;
}

/**
 * Task run streams and durable stream topics.
 *
 * `put` publishes a chunk on a task run's stream, the way `ctx.putStream` does from a worker.
 * `publish`, `reset` and `topicMetadata` are the Node client's durable topic calls
 * (`hatchet.streams`), with the same names and options. `events()` is not here: it is a
 * server stream, which the unary transport does not carry. Read a topic with the Node client
 * (`hatchet.streams.events(topic)`), or from a serverless task over the operator socket once
 * the operator relays the subscription.
 */
export class StreamsClient {
  /** Per (namespace, topic); the sequence number only advances after a publish succeeds. */
  private readonly producers = new Map<string, { producerId: string; seq: number }>();

  /**
   * Publishes waiting per (namespace, topic). One request per topic is in flight at a time, so
   * two never share a sequence number, and each carries everything queued behind the last.
   */
  private readonly publishQueues = new Map<string, StreamPublishQueue<QueuedPublish>>();

  private readonly events: EventsServiceClient;
  private readonly streams: V1StreamsClient | undefined;

  /**
   * Takes the events and durable streams services, or only the events service, which
   * serves `put` and leaves the durable stream calls unavailable.
   */
  constructor(deps: StreamsClientDeps | EventsServiceClient) {
    if (isEventsService(deps)) {
      this.events = deps;
      this.streams = undefined;
    } else {
      this.events = deps.events;
      this.streams = deps.streams;
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
    return this.events.putStreamEvent({
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

    if (payload.byteLength > MAX_STREAM_MESSAGE_BYTES) {
      throw new HatchetError(
        `stream payload is ${payload.byteLength} bytes, over the ${MAX_STREAM_MESSAGE_BYTES}-byte limit of one publish; publish it from the Node client, which uploads larger payloads`
      );
    }

    const signal = options?.signal;
    const sendBy = Date.now() + (options?.batchTimeoutMs ?? 0);
    const publishPromise = new Promise<void>((resolve, reject) => {
      const queued = { bytes: payload.byteLength, sendBy, payload, signal, resolve, reject };
      const queue = this.publishQueues.get(key);
      if (queue) {
        queue.push(queued);
        return;
      }
      const created = new StreamPublishQueue<QueuedPublish>();
      created.push(queued);
      this.publishQueues.set(key, created);
      void this.drainPublishes(key, namespace, topic, created);
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
   * Makes `events()` calls without a cursor start after the topic's latest message. Readers
   * already iterating, and any call with a cursor, are unaffected.
   * @param topic - the topic to reset
   * @param options - optional namespace override and abort signal
   */
  async reset(
    topic: string,
    options?: Pick<StreamCallOptions, 'namespace' | 'signal'>
  ): Promise<void> {
    await this.durable().reset(
      { namespace: options?.namespace ?? '', topic },
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
        try {
          await this.publishOrdered(
            key,
            namespace,
            topic,
            batch.map((p) => p.payload)
          );
          batch.forEach((p) => p.resolve());
        } catch (err) {
          batch.forEach((p) => p.reject(err));
        }
      }
    }

    this.publishQueues.delete(key);
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
      await streams.publishBatch({
        namespace,
        topic,
        producerId: producer.producerId,
        firstProducerSeq: producer.seq,
        entries: payloads.map((payload) => ({ payload, payloadRef: '' })),
      });
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
