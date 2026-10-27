import { randomUUID } from 'crypto';
import { Status } from 'nice-grpc';
import { createGrpcClient } from '@hatchet/util/grpc-helpers';
import {
  PublishStreamMessagesRequest,
  V1StreamsClient,
  V1StreamsDefinition,
} from '@hatchet/protoc/v1/streams';
import sleep from '@hatchet/util/sleep';
import { makeE2EClient, makeTestScope } from '../__e2e__/harness';

const encode = (text: string) => new TextEncoder().encode(text);
const decode = (payload: Uint8Array) => new TextDecoder().decode(payload);

// Reads the first `count` messages of a topic, then stops consuming.
async function collect(iter: AsyncIterable<{ payload: Uint8Array }>, count: number) {
  const out: string[] = [];
  for await (const item of iter) {
    out.push(decode(item.payload));
    if (out.length >= count) break;
  }
  return out;
}

const range = (n: number, prefix: string) => Array.from({ length: n }, (_, i) => `${prefix}-${i}`);

const MiB = 1024 * 1024;

// bytes that differ per message and per offset, so a swapped or truncated payload is caught
function payloadOf(index: number, size: number): Uint8Array {
  const bytes = new Uint8Array(size);
  for (let i = 0; i < size; i += 1) {
    bytes[i] = (index * 31 + i) % 251;
  }
  return bytes;
}

async function collectBytes(iter: AsyncIterable<{ payload: Uint8Array }>, count: number) {
  const out: Uint8Array[] = [];
  for await (const item of iter) {
    out.push(item.payload);
    if (out.length >= count) break;
  }
  return out;
}

function expectSamePayloads(received: Uint8Array[], sent: Uint8Array[]) {
  expect(received.map((p) => p.byteLength)).toEqual(sent.map((p) => p.byteLength));
  received.forEach((p, i) => {
    expect(Buffer.from(p).equals(Buffer.from(sent[i]))).toBe(true);
  });
}

describe('durable-streams-e2e publish batching: PublishBatch', () => {
  const hatchet = makeE2EClient();
  let raw: V1StreamsClient;

  beforeAll(() => {
    raw = createGrpcClient(hatchet.config, V1StreamsDefinition).client;
  });

  const batch = (topic: string, producerId: string, firstProducerSeq: number, messages: string[]) =>
    raw.publishBatch({
      namespace: '',
      topic,
      producerId,
      firstProducerSeq,
      entries: messages.map((m) => ({ payload: encode(m), payloadRef: '' })),
    });

  it('stores a batch in order, and a retried or overlapping batch adds only the entries not stored yet', async () => {
    const topic = makeTestScope('durable_streams_batch_retry');
    const producerId = randomUUID();

    await batch(topic, producerId, 0, ['a', 'b', 'c']);
    // the same request again, as after a lost response
    await batch(topic, producerId, 0, ['a', 'b', 'c']);
    // overlaps the stored seqs 0-2 at seq 2
    await batch(topic, producerId, 2, ['c', 'd', 'e']);
    // a single Publish continues the producer's sequence after a batch
    await raw.publish({
      namespace: '',
      topic,
      producerId,
      producerSeq: 5,
      payload: encode('f'),
      payloadRef: '',
    });

    expect(await collect(hatchet.streams.events(topic), 6)).toEqual(['a', 'b', 'c', 'd', 'e', 'f']);
  }, 30_000);

  it('rejects a batch that skips a sequence number and stores none of it', async () => {
    const topic = makeTestScope('durable_streams_batch_gap');
    const producerId = randomUUID();

    await batch(topic, producerId, 0, ['a']);
    await expect(batch(topic, producerId, 2, ['skipped-1', 'skipped-2'])).rejects.toMatchObject({
      code: Status.FAILED_PRECONDITION,
    });
    await batch(topic, producerId, 1, ['b']);

    expect(await collect(hatchet.streams.events(topic), 2)).toEqual(['a', 'b']);
  }, 30_000);

  it('rejects a batch with an invalid entry, naming the entry, and stores none of it', async () => {
    const topic = makeTestScope('durable_streams_batch_invalid');
    const producerId = randomUUID();

    const req: PublishStreamMessagesRequest = {
      namespace: '',
      topic,
      producerId,
      firstProducerSeq: 0,
      entries: [
        { payload: encode('valid'), payloadRef: '' },
        { payload: new Uint8Array(), payloadRef: '' },
      ],
    };
    const err = await raw.publishBatch(req).catch((e) => e);
    expect(err).toMatchObject({ code: Status.INVALID_ARGUMENT });
    expect(err.details).toContain('entry 1');

    await expect(batch(topic, producerId, 0, [])).rejects.toMatchObject({
      code: Status.INVALID_ARGUMENT,
    });

    // the rejected batch used no seq, so seq 0 is still free
    await batch(topic, producerId, 0, ['first']);
    expect(await collect(hatchet.streams.events(topic), 1)).toEqual(['first']);
  }, 30_000);
});

describe('durable-streams-e2e publish batching: StreamsClient.publish', () => {
  const hatchet = makeE2EClient();
  let publishBatch: jest.SpyInstance;
  let send: V1StreamsClient['publishBatch'];

  beforeEach(() => {
    const { grpc } = hatchet.streams as unknown as { grpc: V1StreamsClient };
    send = grpc.publishBatch.bind(grpc);
    publishBatch = jest.spyOn(grpc, 'publishBatch');
  });

  afterEach(() => {
    publishBatch.mockRestore();
  });

  // entries per request to the topic, in send order
  const requestsTo = (topic: string): number[] =>
    publishBatch.mock.calls
      .map(([req]) => req as PublishStreamMessagesRequest)
      .filter((req) => req.topic === topic)
      .map((req) => req.entries.length);

  const requestBodiesTo = (topic: string): PublishStreamMessagesRequest[] =>
    publishBatch.mock.calls
      .map(([req]) => req as PublishStreamMessagesRequest)
      .filter((req) => req.topic === topic);

  // holds the next request to the engine for `ms`, so publishes made meanwhile queue behind it
  const delayNextRequest = (ms: number) => {
    publishBatch.mockImplementationOnce(async (...args: Parameters<typeof send>) => {
      await sleep(ms);
      return send(...args);
    });
  };

  it('sends publishes that are not awaited one by one in fewer requests, in order', async () => {
    const topic = makeTestScope('durable_streams_publish_unawaited');
    const messages = range(1200, 'm');

    await Promise.all(messages.map((m) => hatchet.streams.publish(topic, m)));

    const requests = requestsTo(topic);
    expect(requests.reduce((a, b) => a + b, 0)).toBe(messages.length);
    expect(requests.length).toBeLessThan(messages.length / 10);

    expect(await collect(hatchet.streams.events(topic), messages.length)).toEqual(messages);
  }, 60_000);

  it('holds publishes for their batch timeout and sends the ones made within it as one request', async () => {
    const topic = makeTestScope('durable_streams_publish_batch_timeout');
    const messages = range(5, 'm');

    const startedAt = Date.now();
    await Promise.all(
      messages.map((m) => hatchet.streams.publish(topic, m, { batchTimeoutMs: 300 }))
    );

    expect(Date.now() - startedAt).toBeGreaterThanOrEqual(250);
    expect(requestsTo(topic)).toEqual([5]);
    expect(await collect(hatchet.streams.events(topic), messages.length)).toEqual(messages);
  }, 30_000);

  it('sends a publish without a timeout at once, carrying the ones still waiting out their timeout', async () => {
    const topic = makeTestScope('durable_streams_publish_flush_waiting');

    const startedAt = Date.now();
    const waiting = range(3, 'waiting').map((m) =>
      hatchet.streams.publish(topic, m, { batchTimeoutMs: 60_000 })
    );
    await sleep(100);
    expect(requestsTo(topic)).toEqual([]);

    await hatchet.streams.publish(topic, 'immediate');
    await Promise.all(waiting);

    expect(Date.now() - startedAt).toBeLessThan(10_000);
    expect(requestsTo(topic)).toEqual([4]);
    expect(await collect(hatchet.streams.events(topic), 4)).toEqual([
      'waiting-0',
      'waiting-1',
      'waiting-2',
      'immediate',
    ]);
  }, 30_000);

  it('a single publish made while a batch is in flight waits behind it and is delivered after it', async () => {
    const topic = makeTestScope('durable_streams_publish_single_behind_batch');
    const batched = range(200, 'batched');
    const resolved: string[] = [];

    delayNextRequest(1_000);
    const batchPublishes = batched.map((m) =>
      hatchet.streams.publish(topic, m, { batchTimeoutMs: 50 }).then(() => resolved.push(m))
    );

    // the batch's request is now held in flight
    await sleep(300);
    expect(requestsTo(topic)).toEqual([batched.length]);

    const single = hatchet.streams.publish(topic, 'single').then(() => resolved.push('single'));
    await Promise.all([...batchPublishes, single]);

    expect(requestsTo(topic)).toEqual([batched.length, 1]);
    expect(resolved[resolved.length - 1]).toBe('single');
    expect(await collect(hatchet.streams.events(topic), batched.length + 1)).toEqual([
      ...batched,
      'single',
    ]);
  }, 30_000);

  it('publishes made while a single message is in flight go out together right after it', async () => {
    const topic = makeTestScope('durable_streams_publish_batch_behind_single');
    const queued = range(50, 'queued');

    delayNextRequest(1_000);
    const single = hatchet.streams.publish(topic, 'single');
    await sleep(300);
    expect(requestsTo(topic)).toEqual([1]);

    const queuedPublishes = queued.map((m) => hatchet.streams.publish(topic, m));
    await Promise.all([single, ...queuedPublishes]);

    expect(requestsTo(topic)).toEqual([1, queued.length]);
    expect(await collect(hatchet.streams.events(topic), queued.length + 1)).toEqual([
      'single',
      ...queued,
    ]);
  }, 30_000);

  it('batches each topic on its own, so one topic waiting out a timeout does not hold up another', async () => {
    const waitingTopic = makeTestScope('durable_streams_publish_topic_waiting');
    const otherTopic = makeTestScope('durable_streams_publish_topic_other');

    let waitingDone = false;
    const waiting = hatchet.streams
      .publish(waitingTopic, 'waiting', { batchTimeoutMs: 60_000 })
      .then(() => {
        waitingDone = true;
      });

    await hatchet.streams.publish(otherTopic, 'other');
    expect(waitingDone).toBe(false);
    expect(requestsTo(waitingTopic)).toEqual([]);

    // flushes the waiting topic so the test does not wait out its timeout
    await hatchet.streams.publish(waitingTopic, 'flush');
    await waiting;
    expect(requestsTo(waitingTopic)).toEqual([2]);
  }, 30_000);

  it('splits large inline payloads across requests at the byte limit, each under the gRPC message limit', async () => {
    const topic = makeTestScope('durable_streams_publish_large_inline');
    // three fit in one request, a fourth does not
    const sent = Array.from({ length: 10 }, (_, i) => payloadOf(i, MiB));

    await Promise.all(sent.map((p) => hatchet.streams.publish(topic, p, { batchTimeoutMs: 200 })));

    const requests = requestBodiesTo(topic);
    expect(requests.map((r) => r.entries.length)).toEqual([3, 3, 3, 1]);
    requests.forEach((r) => {
      expect(PublishStreamMessagesRequest.encode(r).finish().byteLength).toBeLessThanOrEqual(
        4 * MiB
      );
    });

    expectSamePayloads(await collectBytes(hatchet.streams.events(topic), sent.length), sent);
  }, 60_000);

  it('sends a max-size inline payload on its own, between batches of small ones', async () => {
    const topic = makeTestScope('durable_streams_publish_max_inline');
    const maxInline = 4 * MiB - 5 * 1024;
    const sent = [
      payloadOf(0, 10),
      payloadOf(1, 10),
      payloadOf(2, maxInline),
      payloadOf(3, 10),
      payloadOf(4, 10),
    ];

    await Promise.all(sent.map((p) => hatchet.streams.publish(topic, p, { batchTimeoutMs: 200 })));

    expect(requestsTo(topic)).toEqual([2, 1, 2]);
    expectSamePayloads(await collectBytes(hatchet.streams.events(topic), sent.length), sent);
  }, 60_000);

  it('batches uploaded payloads by ref alongside inline ones, and readers get every payload back whole', async () => {
    const topic = makeTestScope('durable_streams_publish_uploaded');
    // over the inline limit, so these are uploaded and published by ref
    const sent = [
      payloadOf(0, 10),
      payloadOf(1, 5 * MiB),
      payloadOf(2, 10),
      payloadOf(3, 6 * MiB),
      payloadOf(4, 10),
    ];

    await Promise.all(sent.map((p) => hatchet.streams.publish(topic, p, { batchTimeoutMs: 200 })));

    const requests = requestBodiesTo(topic);
    expect(requests).toHaveLength(1);
    expect(requests[0].entries.map((e) => e.payloadRef !== '')).toEqual([
      false,
      true,
      false,
      true,
      false,
    ]);

    expectSamePayloads(await collectBytes(hatchet.streams.events(topic), sent.length), sent);
  }, 120_000);
});
