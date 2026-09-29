import { randomUUID } from 'crypto';
import { Status } from 'nice-grpc';
import { ClientConfig } from '@hatchet/clients/hatchet-client';
import { createGrpcClient } from '@hatchet/util/grpc-helpers';
import { getGrpcErrorCode } from '@hatchet/util/grpc-error';
import {
  V1StreamsClient as PbV1StreamsClient,
  V1StreamsDefinition,
} from '@hatchet/protoc/v1/streams';
import { HatchetClient } from '../client';

// errors the server returns before the message could have reached the queue,
// so its producer_seq was never used
function publishRejectedBeforeEnqueue(err: unknown): boolean {
  const code = getGrpcErrorCode(err);
  return (
    code === Status.INVALID_ARGUMENT ||
    code === Status.RESOURCE_EXHAUSTED ||
    code === Status.UNAUTHENTICATED ||
    code === Status.PERMISSION_DENIED
  );
}

export type StreamEvent = {
  /** the message payload */
  payload: Uint8Array;
  /** this event's own cursor -- checkpoint after any event, not just at the start of a call to events() */
  cursor: string;
  /** when the message was durably persisted */
  createdAt?: Date;
};

export type StreamCallOptions = {
  namespace?: string;
  cursor?: string;
  /** aborts an in-progress events() call; the iteration then ends cleanly instead of throwing */
  signal?: AbortSignal;
};

/**
 * StreamsClient provides methods for publishing to and reading from durable,
 * topic-based streams. This is distinct from the ephemeral, per-run streaming
 * exposed by `runs.subscribeToStream`, which is not durable, and tied to specific workflow runs.
 * A durable stream topic is independent of any workflow run, and can be started from any point using cursors.
 */
export class StreamsClient {
  private _config: ClientConfig;
  private _grpc: PbV1StreamsClient | undefined;

  // producer identity and next producer_seq per (namespace, topic), only
  // advanced after a publish succeeds -- see publishChains.
  private producers = new Map<string, { producerId: string; seq: number }>();

  // publishChains serializes publish() calls per (namespace, topic): each
  // call waits for the previous one on the same key to settle before reading
  // producers, so two calls never use the same seq. The stored promise always resolves, even when the publish it
  // chains from failed, so one failure doesn't block every later call on
  // that key.
  private publishChains = new Map<string, Promise<void>>();

  constructor(client: HatchetClient) {
    this._config = client.config;
  }

  private get grpc(): PbV1StreamsClient {
    if (!this._grpc) {
      const { client } = createGrpcClient(this._config, V1StreamsDefinition);
      this._grpc = client;
    }
    return this._grpc;
  }

  private async publishOrdered(
    key: string,
    namespace: string,
    topic: string,
    payload: Uint8Array
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
        payload,
        producerId: producer.producerId,
        producerSeq: producer.seq,
      });
    } catch (err) {
      // the message may still land, so reusing its seq for a different
      // payload would get that payload dropped as a duplicate
      if (!publishRejectedBeforeEnqueue(err)) {
        this.producers.set(key, { producerId: randomUUID(), seq: 0 });
      }
      throw err;
    }

    producer.seq += 1;
  }

  /**
   * Durably publishes a message to a topic. Messages from this client instance are delivered to
   * events() in the order publish() was called, even under concurrent calls
   * or network/queue reordering.
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

    const previous = this.publishChains.get(key) ?? Promise.resolve();
    const publishPromise = previous.then(() => this.publishOrdered(key, namespace, topic, payload));

    this.publishChains.set(
      key,
      publishPromise.catch(() => {})
    );

    return publishPromise;
  }

  /**
   * Returns an async iterable of messages published to topic, starting from
   * the given cursor (options.cursor) or from the beginning of the topic if
   * none is supplied. Yields one message at a time.
   * Stops when the server hangs up, options.signal aborts, or
   * the caller stops iterating (e.g. `break`ing a `for await` loop).
   * @param topic - the topic to read from
   * @param options - optional namespace override, resume cursor, and abort signal
   */
  async *events(topic: string, options?: StreamCallOptions): AsyncIterable<StreamEvent> {
    const stream = this.grpc.subscribe(
      {
        namespace: options?.namespace ?? '',
        topic,
        cursor: options?.cursor,
      },
      { signal: options?.signal }
    );

    try {
      for await (const msg of stream) {
        if (msg.hangup) {
          return;
        }

        for (const entry of msg.entries) {
          yield {
            payload: entry.payload,
            cursor: entry.cursor,
            createdAt: entry.createdAt,
          };
        }
      }
    } catch (err) {
      // an aborted signal is an intentional stop, not a failure
      if (options?.signal?.aborted) {
        return;
      }
      throw err;
    }
  }
}
