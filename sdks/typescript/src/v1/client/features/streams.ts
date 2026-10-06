import { randomUUID } from 'crypto';
import { isAxiosError } from 'axios';
import { Status } from 'nice-grpc';
import { ClientConfig } from '@hatchet/clients/hatchet-client';
import { createGrpcClient } from '@hatchet/util/grpc-helpers';
import { getGrpcErrorCode } from '@hatchet/util/grpc-error';
import sleep from '@hatchet/util/sleep';
import {
  V1StreamsClient as PbV1StreamsClient,
  V1StreamsDefinition,
} from '@hatchet/protoc/v1/streams';
import { HatchetClient } from '../client';

// the server stored nothing, so the seq is unused
function publishRejectedBeforeStoring(err: unknown): boolean {
  const code = getGrpcErrorCode(err);
  return (
    code === Status.INVALID_ARGUMENT ||
    code === Status.RESOURCE_EXHAUSTED ||
    code === Status.UNAUTHENTICATED ||
    code === Status.PERMISSION_DENIED
  );
}

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

/** Covers only messages within the tenant's retention. */
export type StreamTopicMetadata = {
  namespace: string;
  topic: string;
  tenantId: string;
  messageCount: number;
  /** resumes events() after the newest retained message; undefined when none is retained */
  latestCursor?: string;
  /** when the newest retained message was stored; undefined when none is retained */
  lastPublishedAt?: Date;
};

export type StreamCallOptions = {
  namespace?: string;
  cursor?: string;
  /** ends an events() iteration cleanly, without throwing */
  signal?: AbortSignal;
};

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

  // serializes publishes per (namespace, topic) so two never share a seq; the
  // stored promise always resolves, so one failure doesn't block later calls
  private publishChains = new Map<string, Promise<void>>();

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

  private async publishOrdered(
    key: string,
    namespace: string,
    topic: string,
    body: { payload: Uint8Array } | { payloadRef: string },
    retryGap = true
  ): Promise<void> {
    let producer = this.producers.get(key);
    if (!producer) {
      producer = { producerId: randomUUID(), seq: 0 };
      this.producers.set(key, producer);
    }

    try {
      await this.grpc.publish({
        namespace,
        topic,
        ...body,
        producerId: producer.producerId,
        producerSeq: producer.seq,
      });
    } catch (err) {
      // it may still land, and a different payload under its seq would be dropped as a duplicate
      if (!publishRejectedBeforeStoring(err)) {
        this.producers.set(key, { producerId: randomUUID(), seq: 0 });
      }
      // a gap stored nothing (e.g. the watermark passed cursor retention), so resend as the new producer
      if (retryGap && getGrpcErrorCode(err) === Status.FAILED_PRECONDITION) {
        return this.publishOrdered(key, namespace, topic, body, false);
      }
      throw err;
    }

    producer.seq += 1;
  }

  /**
   * Resolves once the message is stored. A client's messages to a topic are
   * delivered in the order publish() was called.
   * @param topic - the topic to publish to
   * @param message - the message payload
   * @param options - optional namespace override
   */
  async publish(
    topic: string,
    message: Uint8Array | string,
    options?: StreamCallOptions
  ): Promise<void> {
    const payload = typeof message === 'string' ? new TextEncoder().encode(message) : message;
    const namespace = options?.namespace ?? '';
    const key = `${namespace}\u0000${topic}`;

    if (payload.byteLength > maxUploadedPayloadBytes) {
      throw new Error(`payload exceeds maximum size of ${maxUploadedPayloadBytes} bytes`);
    }

    // uploaded before taking its place in the topic's order, so uploads overlap
    const uploaded =
      payload.byteLength > maxInlinePayloadBytes ? this.uploadPayload(payload) : undefined;
    // awaited in order below; this only keeps an early failure from reporting as unhandled
    uploaded?.catch(() => {});

    const previous = this.publishChains.get(key) ?? Promise.resolve();
    const publishPromise = previous.then(async () =>
      this.publishOrdered(
        key,
        namespace,
        topic,
        uploaded ? { payloadRef: await uploaded } : { payload }
      )
    );

    this.publishChains.set(
      key,
      publishPromise.catch(() => {})
    );

    return publishPromise;
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
   * Yields topic's messages after options.cursor, or from the oldest retained.
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
