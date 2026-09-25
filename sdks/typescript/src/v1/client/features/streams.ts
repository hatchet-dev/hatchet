import { randomUUID } from 'crypto';
import { ClientConfig } from '@hatchet/clients/hatchet-client';
import { createGrpcClient } from '@hatchet/util/grpc-helpers';
import {
  V1StreamsClient as PbV1StreamsClient,
  V1StreamsDefinition,
} from '@hatchet/protoc/v1/streams';
import { HatchetClient } from '../client';

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
};

/**
 * StreamsClient provides methods for publishing to and reading from durable,
 * topic-based streams. This is distinct from the ephemeral, per-run streaming
 * exposed by `runs.subscribeToStream`, which is fanout-only and never
 * persisted: a durable stream topic is independent of any workflow run, and a
 * late-connecting reader can resume from a cursor.
 */
export class StreamsClient {
  private _config: ClientConfig;
  private _grpc: PbV1StreamsClient | undefined;

  // producerId identifies this client instance to the server for ordering
  // purposes (see api-contracts/v1/streams.proto).
  private producerId: string;

  // nextSeq is the next producer_seq to use per (namespace, topic), only
  // advanced after a publish succeeds -- see publishChains.
  private nextSeq = new Map<string, number>();

  // publishChains serializes publish() calls per (namespace, topic): each
  // call waits for the previous one on the same key to settle before reading
  // nextSeq, so a failed call never advances it (a retry reuses the same
  // seq) without risking two calls ever using the same seq for different
  // payloads. The stored promise always resolves, even when the publish it
  // chains from failed, so one failure doesn't block every later call on
  // that key.
  private publishChains = new Map<string, Promise<void>>();

  constructor(client: HatchetClient) {
    this._config = client.config;
    this.producerId = randomUUID();
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
    const producerSeq = this.nextSeq.get(key) ?? 0;

    await this.grpc.publish({
      namespace,
      topic,
      payload,
      producerId: this.producerId,
      producerSeq,
    });

    this.nextSeq.set(key, producerSeq + 1);
  }

  /**
   * Durably publishes a message to a topic. Topics are created implicitly on
   * first publish. Messages from this client instance are delivered to
   * events() in the order publish() was called, even under concurrent calls
   * or network/queue reordering. A failed publish never consumes its
   * producer_seq, so retrying with the same arguments picks it back up.
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
   * the given cursor (options.cursor) or from "now" if none is supplied.
   * Yields one message at a time -- the server may batch several messages
   * into one wire frame while catching up from an old cursor, transparently
   * to this iterable.
   * @param topic - the topic to read from
   * @param options - optional namespace override and resume cursor
   */
  async *events(topic: string, options?: StreamCallOptions): AsyncIterable<StreamEvent> {
    const stream = this.grpc.subscribe({
      namespace: options?.namespace ?? '',
      topic,
      cursor: options?.cursor,
    });

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
  }
}
