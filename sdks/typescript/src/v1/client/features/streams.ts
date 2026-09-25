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
  // purposes (see api-contracts/v1/streams.proto). nextSeq assigns a strictly
  // increasing sequence per (namespace, topic), synchronously at call time --
  // before any network I/O -- so the assigned order reflects the order
  // publish() was actually called in, regardless of what happens to each
  // request afterward (network reordering, concurrent queue processing,
  // retries).
  private producerId: string;
  private nextSeq = new Map<string, number>();

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

  private nextProducerSeq(namespace: string, topic: string): number {
    const key = `${namespace}\u0000${topic}`;
    const seq = this.nextSeq.get(key) ?? 0;
    this.nextSeq.set(key, seq + 1);
    return seq;
  }

  /**
   * Durably publishes a message to a topic. Topics are created implicitly on
   * first publish. Messages from this client instance are delivered to
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
    const producerSeq = this.nextProducerSeq(namespace, topic);

    await this.grpc.publish({
      namespace,
      topic,
      payload,
      producerId: this.producerId,
      producerSeq,
    });
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
