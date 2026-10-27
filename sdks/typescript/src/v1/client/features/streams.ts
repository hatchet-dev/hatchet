import { randomUUID } from 'crypto';
import { isAxiosError } from 'axios';
import { Status } from 'nice-grpc';
import { ClientConfig } from '@hatchet/clients/hatchet-client';
import { createGrpcClient } from '@hatchet/util/grpc-helpers';
import { getGrpcErrorCode } from '@hatchet/util/grpc-error';
import sleep from '@hatchet/util/sleep';
import { StreamPublishQueue, validateBatchTimeout } from '@hatchet/util/stream-publish-batch';
import {
  V1StreamsClient as PbV1StreamsClient,
  V1StreamsDefinition,
} from '@hatchet/protoc/v1/streams';
import type { StreamCallOptions, StreamTopicMetadata } from '@hatchet/core/types';
import { HatchetClient } from '../client';

// the server stored nothing, so the seq is unused
function publishRejectedBeforeStoring(err: unknown): boolean {
  const code = getGrpcErrorCode(err);
  return (
    code === Status.INVALID_ARGUMENT ||
    code === Status.RESOURCE_EXHAUSTED ||
    code === Status.UNAUTHENTICATED ||
    code === Status.PERMISSION_DENIED ||
    // an engine without the RPC
    code === Status.UNIMPLEMENTED
  );
}

type PublishEntry = { payload: Uint8Array; payloadRef: string };

// retrying a subscribe that failed with these can't succeed
const permanentSubscribeErrors = new Set<number>([
  Status.INVALID_ARGUMENT,
  Status.PERMISSION_DENIED,
  Status.UNAUTHENTICATED,
  Status.OUT_OF_RANGE,
  Status.FAILED_PRECONDITION,
  Status.NOT_FOUND,
]);

const resubscribeMinDelayMs = 250;
const resubscribeMaxDelayMs = 10_000;

// the server's limit for a payload sent over gRPC (MaxStreamMessagePayloadBytes)
const maxInlinePayloadBytes = 4 * 1024 * 1024 - 5 * 1024;

// the server's limit for an uploaded payload (MaxStreamUploadedPayloadBytes)
const maxUploadedPayloadBytes = 64 * 1024 * 1024;

const payloadRequestAttempts = 3;

// an upload's ref, which a batch carries in place of its payload, rounded up
const payloadRefBytes = 256;

function httpStatus(err: unknown): number | undefined {
  return isAxiosError(err) ? err.response?.status : undefined;
}

// a payload fetch that got no response, a 5xx or a 429 may succeed later
function isTemporaryHttpError(err: unknown): boolean {
  if (!isAxiosError(err)) {
    return false;
  }
  const status = httpStatus(err);
  return status === undefined || status >= 500 || status === 429;
}

function shouldResubscribe(err: unknown, attempt: number): boolean {
  if (isTemporaryHttpError(err)) {
    return true;
  }
  const code = getGrpcErrorCode(err);
  if (code === undefined || permanentSubscribeErrors.has(code)) {
    return false;
  }
  // proxies answer 404 while an engine restarts, but on a first attempt it means no streams support
  if (code === Status.UNIMPLEMENTED) {
    return attempt > 0;
  }
  return true;
}

export type StreamEvent = {
  /** the message payload */
  payload: Uint8Array;
  /** resumes events() right after this event */
  cursor: string;
  /** when the message was stored */
  createdAt?: Date;
};

export type { StreamCallOptions, StreamTopicMetadata } from '@hatchet/core/types';

interface QueuedPublish {
  payload: Uint8Array;
  // set instead of an inline payload for one too large for a gRPC message
  uploaded: Promise<string> | undefined;
  resolve: () => void;
  reject: (err: unknown) => void;
}

/**
 * StreamsClient publishes to and reads from durable topics. Unlike
 * `runs.subscribeToStream`, topics are stored, independent of any run, and
 * readers can resume from a cursor.
 */
export class StreamsClient {
  private _config: ClientConfig;
  private _client: HatchetClient;
  private _grpc: PbV1StreamsClient | undefined;

  // per (namespace, topic); seq only advances after a publish succeeds
  private producers = new Map<string, { producerId: string; seq: number }>();

  // per (namespace, topic); one request per topic is in flight at a time so two
  // never share a seq, and each carries everything queued behind the last
  private publishQueues = new Map<string, StreamPublishQueue<QueuedPublish>>();

  // set once the engine answers PublishBatch as unimplemented: it predates batching
  private publishBatchUnsupported = false;

  constructor(client: HatchetClient) {
    this._config = client.config;
    this._client = client;
  }

  private get grpc(): PbV1StreamsClient {
    if (!this._grpc) {
      const { client } = createGrpcClient(this._config, V1StreamsDefinition);
      this._grpc = client;
    }
    return this._grpc;
  }

  // retries a 5xx or network failure; a 4xx won't succeed on retry
  private static async withRetries<T>(request: () => Promise<T>): Promise<T> {
    for (let attempt = 1; ; attempt += 1) {
      try {
        return await request();
      } catch (err) {
        const status = httpStatus(err);
        if ((status !== undefined && status < 500) || attempt >= payloadRequestAttempts) {
          throw err;
        }
      }
    }
  }

  private async uploadPayload(payload: Uint8Array): Promise<string> {
    const res = await StreamsClient.withRetries(() =>
      this._client.api.v1StreamPayloadUpload(this._config.tenant_id, {
        payload: new Blob([payload]) as File,
      })
    );
    return res.data.ref;
  }

  private async fetchPayload(payloadRef: string, signal?: AbortSignal): Promise<Uint8Array> {
    const res = await StreamsClient.withRetries(() =>
      this._client.api.v1StreamPayloadGet(
        this._config.tenant_id,
        { ref: payloadRef },
        { format: 'arraybuffer', signal }
      )
    );
    return new Uint8Array(res.data as unknown as ArrayBuffer);
  }

  private async drainPublishes(
    key: string,
    namespace: string,
    topic: string,
    queue: StreamPublishQueue<QueuedPublish>
  ): Promise<void> {
    while (queue.size > 0) {
      const batch = await queue.next();
      const refs = await Promise.allSettled(batch.map((p) => p.uploaded));
      const sent: QueuedPublish[] = [];
      const entries: PublishEntry[] = [];

      // a failed upload fails only its own publish, as the rest never depended on it
      batch.forEach((p, i) => {
        const ref = refs[i];
        if (ref.status === 'rejected') {
          p.reject(ref.reason);
          return;
        }
        sent.push(p);
        entries.push(
          ref.value === undefined
            ? { payload: p.payload, payloadRef: '' }
            : { payload: new Uint8Array(), payloadRef: ref.value }
        );
      });

      if (entries.length > 0) {
        await this.sendPublishes(key, namespace, topic, sent, entries);
      }
    }

    this.publishQueues.delete(key);
  }

  // sends entries as one request, or one request per entry to an engine that
  // predates PublishBatch, and settles each publish
  private async sendPublishes(
    key: string,
    namespace: string,
    topic: string,
    sent: QueuedPublish[],
    entries: PublishEntry[]
  ): Promise<void> {
    if (!this.publishBatchUnsupported) {
      try {
        await this.publishOrdered(key, namespace, topic, entries);
        sent.forEach((p) => p.resolve());
        return;
      } catch (err) {
        if (getGrpcErrorCode(err) !== Status.UNIMPLEMENTED) {
          sent.forEach((p) => p.reject(err));
          return;
        }
        // nothing was stored, so the batch goes out again below
        this.publishBatchUnsupported = true;
      }
    }

    // settled one by one: a failure partway leaves the publishes before it stored
    for (const [i, p] of sent.entries()) {
      try {
        await this.publishOrdered(key, namespace, topic, [entries[i]]);
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
    entries: PublishEntry[],
    retryGap = true
  ): Promise<void> {
    let producer = this.producers.get(key);
    if (!producer) {
      producer = { producerId: randomUUID(), seq: 0 };
      this.producers.set(key, producer);
    }

    try {
      if (this.publishBatchUnsupported && entries.length === 1) {
        await this.grpc.publish({
          namespace,
          topic,
          ...entries[0],
          producerId: producer.producerId,
          producerSeq: producer.seq,
        });
      } else {
        await this.grpc.publishBatch({
          namespace,
          topic,
          producerId: producer.producerId,
          firstProducerSeq: producer.seq,
          entries,
        });
      }
    } catch (err) {
      // it may still land, and a different payload under its seq would be dropped as a duplicate
      if (!publishRejectedBeforeStoring(err)) {
        this.producers.set(key, { producerId: randomUUID(), seq: 0 });
      }
      // a gap stored nothing (e.g. the watermark passed cursor retention), so resend as the new producer
      if (retryGap && getGrpcErrorCode(err) === Status.FAILED_PRECONDITION) {
        return this.publishOrdered(key, namespace, topic, entries, false);
      }
      throw err;
    }

    producer.seq += entries.length;
  }

  /**
   * Resolves once the message is stored. A client's messages to a topic are
   * delivered in the order publish() was called. Publishes to a topic share
   * requests: everything queued while one is in flight goes out in the next,
   * and `batchTimeoutMs` holds a message up to that long for more to join it.
   * @param topic - the topic to publish to
   * @param message - the message payload
   * @param options - optional namespace override and batch timeout
   */
  async publish(
    topic: string,
    message: Uint8Array | string,
    options?: StreamCallOptions
  ): Promise<void> {
    const payload = typeof message === 'string' ? new TextEncoder().encode(message) : message;
    const namespace = options?.namespace ?? '';
    const key = `${namespace}\u0000${topic}`;

    // the engine would reject the whole request it shares, failing other publishes too
    if (payload.byteLength === 0) {
      throw new Error('payload is empty; a message must carry at least one byte');
    }

    if (payload.byteLength > maxUploadedPayloadBytes) {
      throw new Error(`payload exceeds maximum size of ${maxUploadedPayloadBytes} bytes`);
    }

    validateBatchTimeout(options?.batchTimeoutMs);

    // uploaded before taking its place in the topic's order, so uploads overlap
    const uploaded =
      payload.byteLength > maxInlinePayloadBytes ? this.uploadPayload(payload) : undefined;
    // awaited when its batch is sent; this only keeps an early failure from reporting as unhandled
    uploaded?.catch(() => {});

    return new Promise<void>((resolve, reject) => {
      const queue =
        this.publishQueues.get(key) ??
        new StreamPublishQueue<QueuedPublish>((p) =>
          p.uploaded ? payloadRefBytes : p.payload.byteLength
        );
      queue.push({ payload, uploaded, resolve, reject }, options?.batchTimeoutMs);

      // pushed first, since the drain stops at an empty queue
      if (!this.publishQueues.has(key)) {
        this.publishQueues.set(key, queue);
        void this.drainPublishes(key, namespace, topic, queue);
      }
    });
  }

  /**
   * Describes topic's retained messages. Rejects with a NOT_FOUND gRPC error
   * for a topic nothing was published to.
   * @param topic - the topic to describe
   * @param options - optional namespace override
   */
  async topicMetadata(
    topic: string,
    options?: Pick<StreamCallOptions, 'namespace'>
  ): Promise<StreamTopicMetadata> {
    const res = await this.grpc.getTopicMetadata({ namespace: options?.namespace ?? '', topic });

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
   * Makes events() calls without a cursor start after options.cursor, or the
   * topic's latest message. Readers already iterating, and any call with a
   * cursor, are unaffected.
   * @param topic - the topic to reset
   * @param options - optional namespace override and cursor to reset to
   */
  async reset(
    topic: string,
    options?: Pick<StreamCallOptions, 'namespace' | 'cursor'>
  ): Promise<void> {
    await this.grpc.reset({ namespace: options?.namespace ?? '', topic, cursor: options?.cursor });
  }

  /**
   * Yields topic's messages after options.cursor, or from the oldest retained
   * (or, once the topic is reset, after the reset).
   * If the connection drops, it resubscribes from the last message yielded, so
   * nothing is skipped or repeated. Ends when the server hangs up an idle
   * subscription, options.signal aborts, or the caller stops iterating; throws
   * on errors a retry can't fix, such as an expired cursor.
   * @param topic - the topic to read from
   * @param options - optional namespace override, resume cursor, and abort signal
   */
  async *events(topic: string, options?: StreamCallOptions): AsyncIterable<StreamEvent> {
    const namespace = options?.namespace ?? '';
    const signal = options?.signal;
    let { cursor } = options ?? {};
    let failures = 0;

    for (let attempt = 0; ; attempt += 1) {
      try {
        const stream = this.grpc.subscribe({ namespace, topic, cursor }, { signal });

        for await (const msg of stream) {
          if (msg.hangup) {
            return;
          }

          for (const entry of msg.entries) {
            const payload = entry.payloadRef
              ? await this.fetchPayload(entry.payloadRef, signal)
              : entry.payload;
            ({ cursor } = entry);
            // reset only on delivery, so a payload fetch failing on every reconnect still backs off
            failures = 0;
            yield {
              payload,
              cursor: entry.cursor,
              createdAt: entry.createdAt,
            };
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

      const delayMs = Math.min(resubscribeMinDelayMs * 2 ** failures, resubscribeMaxDelayMs);
      failures += 1;

      try {
        await sleep(delayMs * (0.5 + Math.random() / 2), signal);
      } catch {
        return;
      }
    }
  }
}
