import { createRouterTransport } from '@connectrpc/connect';
import { EventsService } from '@hatchet-dev/typescript-sdk/protoc-es/events/events_pb.js';
import { V1Streams } from '@hatchet-dev/typescript-sdk/protoc-es/v1/streams_pb.js';
import { describe, expect, it, vi } from 'vitest';
import {
  HatchetCore,
  ServerlessLimitationError,
  StreamClosedError,
  StreamOpenError,
  clientFromContext,
  hatchet,
  type StreamEvent,
  type WorkflowEvent,
} from '../index';
import { StreamMessage } from '../generated/proto/v1/streams';
import type { ServerlessDurableFrame } from '../generated/proto/v1/serverless';
import { createSocketPair } from '../testing/socket-pair';
import { createTestOperator, publicStreamMessage } from '../testing';
import { decodeFrame, encodeFrame, frameKind } from './durable/frames';
import { SUBSCRIBE_STREAM, SocketStreams, StreamCloseCode } from './durable/streams';
import { createTaskClient } from './task-client';

const secret = 'test-secret-at-least-32-characters-long';
const quiet = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };
const TENANT_ID = '707d0855-80ab-4e1f-a156-f1c4546cbf52';
const decoder = new TextDecoder();

function makeToken(claims: Record<string, unknown>): string {
  const encode = (value: unknown) => Buffer.from(JSON.stringify(value)).toString('base64url');
  return `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode(claims)}.signature`;
}

const TOKEN = makeToken({ sub: TENANT_ID, grpc_broadcast_address: 'engine.example.com:7070' });

/** The engine's unary side: `publish` and `put` record what the task sent. */
function fakeEngine() {
  const published: Array<{ topic: string; payload: string; producerSeq: number }> = [];
  const put: Array<{ taskRunExternalId: string; message: string }> = [];
  const transport = createRouterTransport(({ service }) => {
    service(EventsService, {
      putStreamEvent: (req) => {
        put.push({
          taskRunExternalId: req.taskRunExternalId,
          message: decoder.decode(req.message),
        });
        return {};
      },
    });
    service(V1Streams, {
      publish: (req) => {
        published.push({
          topic: req.topic,
          payload: decoder.decode(req.payload),
          producerSeq: Number(req.producerSeq),
        });
        return {};
      },
    });
  });

  return { transport, published, put };
}

function client(engine = fakeEngine()): HatchetCore {
  return new HatchetCore({ token: TOKEN, transport: engine.transport, logLevel: 'OFF' });
}

/** A socket pair with the task's `SocketStreams` on the endpoint side and the operator's frames recorded. */
function harness(options: { maxQueuedBytes?: number } = {}) {
  const pair = createSocketPair();
  const received: ServerlessDurableFrame[] = [];

  pair.operator.onMessage((text) => received.push(decodeFrame(text)));

  const streams = new SocketStreams({ socket: pair.endpoint, console: quiet, ...options });
  pair.endpoint.onMessage((text) => streams.handleFrame(decodeFrame(text)));
  const fromOperator = (frame: ServerlessDurableFrame) => pair.operator.send(encodeFrame(frame));
  const settle = () => new Promise<void>((resolve) => setTimeout(resolve, 0));

  return { pair, received, streams, fromOperator, settle };
}

function message(id: string, entries: Array<{ payload: string; cursor: string }>, hangup = false) {
  return {
    streamMessage: {
      id,
      message: JSON.stringify(
        StreamMessage.toJSON(
          StreamMessage.create({
            entries: entries.map((entry) => ({
              payload: new TextEncoder().encode(entry.payload),
              cursor: entry.cursor,
              createdAt: new Date('2026-10-22T00:00:00Z'),
              payloadRef: '',
            })),
            hangup,
            cursor: entries.at(-1)?.cursor ?? '',
          })
        )
      ),
    },
  };
}

async function collect(events: AsyncIterable<StreamEvent>): Promise<Array<[string, string]>> {
  const seen: Array<[string, string]> = [];

  for await (const event of events) {
    seen.push([decoder.decode(event.payload), event.cursor]);
  }

  return seen;
}

describe('createTaskClient', () => {
  it('is the client with streams and runs views, and publish keeps the client producer state', async () => {
    const engine = fakeEngine();
    const core = client(engine);
    const { streams } = harness();
    const task = createTaskClient(core, streams);

    expect(task).toBeInstanceOf(HatchetCore);
    expect(task.tenantId).toBe(TENANT_ID);
    expect(typeof task.streams.events).toBe('function');
    expect(typeof task.runs.events).toBe('function');
    expect(typeof task.runs.cancel).toBe('function');
    expect(typeof core.streams).not.toHaveProperty('events');

    await task.streams.publish('orders', 'one');
    await core.streams.publish('orders', 'two');
    await task.streams.publish('orders', 'three');

    expect(engine.published.map((entry) => [entry.payload, entry.producerSeq])).toEqual([
      ['one', 0],
      ['two', 1],
      ['three', 2],
    ]);
  });

  it('runs the inherited methods against the client, so put() reaches the engine', async () => {
    const engine = fakeEngine();
    const task = createTaskClient(client(engine), harness().streams);

    await task.streams.put('run-1', 'progress');
    await task.streams.put('run-1', new TextEncoder().encode('bytes'));

    expect(engine.put).toEqual([
      { taskRunExternalId: 'run-1', message: 'progress' },
      { taskRunExternalId: 'run-1', message: 'bytes' },
    ]);
    expect(task.streams.events).not.toBe(undefined);
    expect(typeof task.runs.cancel).toBe('function');
  });

  it('closes the stream with RESOURCE_EXHAUSTED when the reader falls behind the bound', async () => {
    // Room for one and a half relayed messages: one unread message fits, a second does not.
    const one = message('x', [{ payload: 'one', cursor: '1' }]).streamMessage.message.length;
    const { streams, received, fromOperator, settle } = harness({
      maxQueuedBytes: Math.floor(one * 1.5),
    });
    const task = createTaskClient(client(), streams);
    const iterator = task.streams.events('orders')[Symbol.asyncIterator]();
    const first = iterator.next();
    await settle();
    const { id } = received[0].streamOpen!;

    fromOperator(message(id, [{ payload: 'one', cursor: '1' }]));
    await expect(first).resolves.toMatchObject({ value: { cursor: '1' } });

    // Two messages arrive before the task reads again; the second crosses the bound.
    fromOperator(message(id, [{ payload: 'two', cursor: '2' }]));
    fromOperator(message(id, [{ payload: 'three', cursor: '3' }]));

    const behind = iterator.next();
    await expect(behind).rejects.toBeInstanceOf(StreamClosedError);
    await expect(behind).rejects.toMatchObject({
      code: StreamCloseCode.RESOURCE_EXHAUSTED,
      cursor: '1',
    });
    await expect(behind).rejects.toThrow(/fell behind reading the stream/);
    expect(streams.size).toBe(0);
  });

  it('reads a topic over the socket and ends when the engine finishes the stream', async () => {
    const { received, streams, fromOperator, settle } = harness();
    const task = createTaskClient(client(), streams);

    const reading = collect(task.streams.events('orders', { namespace: 'e2e', cursor: 'c0' }));
    await settle();

    expect(received).toEqual([
      {
        streamOpen: {
          id: '1',
          procedure: SUBSCRIBE_STREAM,
          request: '{"namespace":"e2e","topic":"orders","cursor":"c0"}',
        },
      },
    ]);

    fromOperator(
      message('1', [
        { payload: '{"n":1}', cursor: 'c1' },
        { payload: '{"n":2}', cursor: 'c2' },
      ])
    );
    fromOperator(message('1', [{ payload: '{"n":3}', cursor: 'c3' }]));
    fromOperator({ streamClose: { id: '1', code: StreamCloseCode.OK, message: '' } });

    expect(await reading).toEqual([
      ['{"n":1}', 'c1'],
      ['{"n":2}', 'c2'],
      ['{"n":3}', 'c3'],
    ]);
    await settle();
    // The engine closed it, so the task sends no stream_close of its own.
    expect(received).toHaveLength(1);
    expect(streams.size).toBe(0);
  });

  it('yields createdAt as a Date and ends on the engine hanging up an idle subscription', async () => {
    const { received, streams, fromOperator, settle } = harness();
    const task = createTaskClient(client(), streams);
    const seen: StreamEvent[] = [];

    const reading = (async () => {
      for await (const event of task.streams.events('orders')) {
        seen.push(event);
      }
    })();
    await settle();

    fromOperator(message('1', [{ payload: 'a', cursor: 'c1' }]));
    fromOperator(message('1', [], true));
    await reading;
    await settle();

    expect(seen).toHaveLength(1);
    expect(seen[0].createdAt).toEqual(new Date('2026-10-22T00:00:00Z'));
    // A hang-up ends the iteration on the task's side, which closes the stream.
    expect(received.at(-1)).toEqual({ streamClose: { id: '1', code: 0, message: '' } });
    expect(streams.size).toBe(0);
  });

  it('closes the stream and frees the slot when the task stops iterating', async () => {
    const { received, streams, fromOperator, settle } = harness();
    const task = createTaskClient(client(), streams);
    const seen: string[] = [];

    const reading = (async () => {
      for await (const event of task.streams.events('orders')) {
        seen.push(decoder.decode(event.payload));

        if (seen.length === 2) {
          break;
        }
      }
    })();
    await settle();

    fromOperator(
      message('1', [
        { payload: 'a', cursor: 'c1' },
        { payload: 'b', cursor: 'c2' },
        { payload: 'c', cursor: 'c3' },
      ])
    );
    await reading;
    await settle();

    expect(seen).toEqual(['a', 'b']);
    expect(received.map(frameKind)).toEqual(['streamOpen', 'streamClose']);
    expect(streams.size).toBe(0);

    // A message crossing the close is dropped; the next read is a new stream.
    fromOperator(message('1', [{ payload: 'late', cursor: 'c4' }]));
    const controller = new AbortController();
    const next = task.streams
      .events('orders', { signal: controller.signal })
      [Symbol.asyncIterator]();
    const pending = next.next();
    await settle();
    expect(received.at(-1)?.streamOpen?.id).toBe('2');
    controller.abort();
    await expect(pending).resolves.toEqual({ done: true, value: undefined });
    expect(streams.size).toBe(0);
  });

  it('ends cleanly on the signal, whether it aborts before the read or while it waits', async () => {
    const { received, streams, fromOperator, settle } = harness();
    const task = createTaskClient(client(), streams);

    expect(await collect(task.streams.events('orders', { signal: AbortSignal.abort() }))).toEqual(
      []
    );
    expect(received).toEqual([]);

    const controller = new AbortController();
    const reading = collect(task.streams.events('orders', { signal: controller.signal }));
    await settle();
    fromOperator(message('1', [{ payload: 'a', cursor: 'c1' }]));
    await settle();

    controller.abort();
    expect(await reading).toEqual([['a', 'c1']]);
    await settle();
    expect(received.at(-1)).toEqual({ streamClose: { id: '1', code: 0, message: '' } });
    expect(streams.size).toBe(0);
  });

  it('throws StreamClosedError with the cursor to resume from when the engine ends the stream', async () => {
    const { streams, fromOperator, settle } = harness();
    const task = createTaskClient(client(), streams);
    const seen: string[] = [];
    let failure: unknown;

    const reading = (async () => {
      try {
        for await (const event of task.streams.events('orders')) {
          seen.push(event.cursor);
        }
      } catch (err) {
        failure = err;
      }
    })();
    await settle();

    fromOperator(message('1', [{ payload: 'a', cursor: 'c1' }]));
    fromOperator({
      streamClose: {
        id: '1',
        code: StreamCloseCode.UNAVAILABLE,
        message: publicStreamMessage(StreamCloseCode.UNAVAILABLE),
      },
    });
    await reading;

    expect(seen).toEqual(['c1']);
    expect(failure).toBeInstanceOf(StreamClosedError);
    expect(failure).toMatchObject({ code: 14, cursor: 'c1' });
    expect((failure as Error).message).toBe(
      'the subscription to topic "orders" closed with code 14 (unavailable: the engine stream was closed)'
    );
    expect(streams.size).toBe(0);
  });

  it('reads code 7 as the entitlement refusal and code 12 as an operator without the relay', async () => {
    const { streams, fromOperator, settle } = harness();
    const task = createTaskClient(client(), streams);

    const denied = collect(task.streams.events('orders'));
    await settle();
    fromOperator({
      streamClose: {
        id: '1',
        code: StreamCloseCode.PERMISSION_DENIED,
        message: publicStreamMessage(StreamCloseCode.PERMISSION_DENIED),
      },
    });
    await expect(denied).rejects.toThrow(
      'the subscription to topic "orders" closed with code 7 (permission_denied: the engine denied the request): durable streams are not enabled for this tenant; the durable_streams entitlement enables them'
    );

    const unserved = collect(task.streams.events('orders'));
    await settle();
    fromOperator({
      streamClose: {
        id: '2',
        code: StreamCloseCode.UNIMPLEMENTED,
        message: 'procedure "/v1.V1Streams/Subscribe" is not served over the socket',
      },
    });
    await expect(unserved).rejects.toThrow(
      /code 12 \(unimplemented: procedure .* is not served over the socket\): the operator does not relay durable stream subscriptions/
    );
    expect(streams.size).toBe(0);
  });

  it('refuses an uploaded payload, which only the Node client can fetch', async () => {
    const { received, streams, fromOperator, settle } = harness();
    const task = createTaskClient(client(), streams);

    const reading = collect(task.streams.events('orders'));
    await settle();
    fromOperator({
      streamMessage: {
        id: '1',
        message: JSON.stringify({ entries: [{ cursor: 'c1', payloadRef: 'ref-1' }], cursor: 'c1' }),
      },
    });
    await expect(reading).rejects.toThrow(
      /message c1 of topic "orders" is an uploaded payload .* read the topic with the Node client/
    );
    await settle();
    expect(received.at(-1)).toEqual({ streamClose: { id: '1', code: 0, message: '' } });
  });

  it('reads a run events stream, decoded, until the hang-up event', async () => {
    const { received, streams, fromOperator, settle } = harness();
    const task = createTaskClient(client(), streams);
    const seen: WorkflowEvent[] = [];

    const reading = (async () => {
      for await (const event of task.runs.events('run-1')) {
        seen.push(event);
      }
    })();
    await settle();

    expect(received[0]).toEqual({
      streamOpen: {
        id: '1',
        procedure: '/Dispatcher/SubscribeToWorkflowEvents',
        request: '{"workflowRunId":"run-1"}',
      },
    });

    fromOperator({
      streamMessage: {
        id: '1',
        message: JSON.stringify({
          workflowRunId: 'run-1',
          resourceType: 'RESOURCE_TYPE_STEP_RUN',
          eventType: 'RESOURCE_EVENT_TYPE_STARTED',
          resourceId: 'step-1',
          eventPayload: '',
        }),
      },
    });
    fromOperator({
      streamMessage: {
        id: '1',
        message: JSON.stringify({
          workflowRunId: 'run-1',
          resourceType: 'RESOURCE_TYPE_WORKFLOW_RUN',
          eventType: 'RESOURCE_EVENT_TYPE_COMPLETED',
          resourceId: 'run-1',
          eventPayload: '{"ok":true}',
          hangup: true,
        }),
      },
    });
    await reading;

    expect(seen.map((event) => [event.resourceId, event.eventPayload, event.hangup])).toEqual([
      ['step-1', '', false],
      ['run-1', '{"ok":true}', true],
    ]);
    expect(streams.size).toBe(0);
  });

  it('without a socket, both iterators throw ServerlessLimitationError at the call', () => {
    const task = createTaskClient(client(), undefined);

    expect(() => task.streams.events('orders')).toThrow(ServerlessLimitationError);
    expect(() => task.streams.events('orders')).toThrow(
      /clientFromContext\(ctx\)\.streams\.events\(topic\) is not available in the serverless runtime: this task was invoked over a POST and has no invocation socket/
    );
    expect(() => task.runs.events('run-1')).toThrow(
      /clientFromContext\(ctx\)\.runs\.events\(runId\) .* invoked over a POST/
    );
  });

  it('surfaces the stream cap as a StreamOpenError on the read that crosses it', async () => {
    const { streams, settle } = harness();
    const task = createTaskClient(client(), streams);
    const controllers = Array.from({ length: 16 }, () => new AbortController());
    // Each read waits for its first message; the signal is what ends a waiting read.
    const pending = controllers.map((controller) =>
      task.streams.events('orders', { signal: controller.signal })[Symbol.asyncIterator]().next()
    );
    await settle();
    expect(streams.size).toBe(16);

    const over = task.streams.events('orders')[Symbol.asyncIterator]();
    const refused = over.next();
    await expect(refused).rejects.toBeInstanceOf(StreamOpenError);
    await expect(refused).rejects.toThrow(/already holds 16 streams, the operator's limit/);
    await expect(over.next()).resolves.toEqual({ done: true, value: undefined });

    controllers[0].abort();
    await expect(pending[0]).resolves.toEqual({ done: true, value: undefined });
    expect(streams.size).toBe(15);

    const fits = task.streams.events('orders', { signal: AbortSignal.abort() });
    expect(await collect(fits)).toEqual([]);

    for (const controller of controllers.slice(1)) {
      controller.abort();
    }
    await Promise.all(pending);
    expect(streams.size).toBe(0);
  });
});

/** Publishes to a topic, then reads it back over the socket and returns what it read. */
const streamEcho = hatchet.task({
  name: 'stream-echo',
  fn: async (input: { topic: string; messages: string[]; cursor?: string; take?: number }, ctx) => {
    const client = clientFromContext(ctx);

    for (const message of input.messages) {
      await client.streams.publish(input.topic, message);
    }

    const read: string[] = [];
    let cursor: string | undefined;

    for await (const event of client.streams.events(input.topic, { cursor: input.cursor })) {
      read.push(decoder.decode(event.payload));
      ({ cursor } = event);

      if (input.take !== undefined && read.length >= input.take) {
        break;
      }
    }

    return { read, cursor };
  },
});

/** Stops reading when its signal fires after `until` messages. */
const abortingReader = hatchet.task({
  name: 'aborting-reader',
  fn: async (input: { topic: string; until: number }, ctx) => {
    const controller = new AbortController();
    const read: string[] = [];

    for await (const event of clientFromContext(ctx).streams.events(input.topic, {
      signal: controller.signal,
    })) {
      read.push(decoder.decode(event.payload));

      if (read.length === input.until) {
        controller.abort();
      }
    }

    return { read, aborted: controller.signal.aborted };
  },
});

const runEventsReader = hatchet.task({
  name: 'run-events-reader',
  fn: async (input: { runId: string }, ctx) => {
    const events: string[] = [];

    for await (const event of clientFromContext(ctx).runs.events(input.runId)) {
      events.push(`${event.resourceId}:${event.eventPayload}`);
    }

    return { events };
  },
});

const workflows = [streamEcho, abortingReader, runEventsReader];

describe('streams.events over the test operator', () => {
  it('publishes through the client and reads the topic back over the socket', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
      runStream: {
        topic: (request) => ({
          entries: engine.published
            .filter((entry) => entry.topic === request.topic)
            .map((entry) => entry.payload),
          batch: 2,
          close: 0,
        }),
      },
    });

    const run = op.startDurable(streamEcho, {
      topic: 'orders',
      messages: ['{"n":1}', '{"n":2}', '{"n":3}'],
    });
    const result = await run.result;

    expect(result.status).toBe('completed');
    expect(result.output).toEqual({ read: ['{"n":1}', '{"n":2}', '{"n":3}'], cursor: 'c3' });
    expect(result.endpointFrames).toEqual(['streamOpen', 'done:output']);
    expect(result.operatorFrames).toEqual([
      'first',
      'streamMessage',
      'streamMessage',
      'streamClose',
    ]);
    expect(result.frames[1].frame.streamOpen).toEqual({
      id: '1',
      procedure: '/v1.V1Streams/Subscribe',
      request: '{"topic":"orders"}',
    });
    expect(run.openStreams()).toEqual([]);
  });

  it('resumes after a cursor, and a cursor the topic does not hold is out of range', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
      runStream: { topic: () => ({ entries: ['a', 'b', 'c'], close: 0 }) },
    });

    expect(await op.invoke(streamEcho, { topic: 'orders', messages: [], cursor: 'c1' })).toEqual({
      read: ['b', 'c'],
      cursor: 'c3',
    });

    const outcome = await op.deliver(streamEcho, { topic: 'orders', messages: [], cursor: 'gone' });
    expect(outcome).toMatchObject({ status: 'failed', retry: true });
    expect((outcome as { error: string }).error).toBe(
      'the subscription to topic "orders" closed with code 11 (out_of_range: the request was out of range)'
    );
  });

  it('reports the entitlement refusal as code 7 and fails the task with it', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
      runStream: {
        topic: () => Promise.reject(Object.assign(new Error(''), { code: 7 })),
      },
    });

    const result = await op.invokeDurable(streamEcho, { topic: 'orders', messages: ['a'] });

    expect(result.status).toBe('failed');
    expect(result.error).toBe(
      'the subscription to topic "orders" closed with code 7 (permission_denied: the engine denied the request): durable streams are not enabled for this tenant; the durable_streams entitlement enables them'
    );
    expect(result.endpointFrames).toEqual(['streamOpen', 'done:error']);
    expect(result.operatorFrames).toEqual(['first', 'streamClose']);
    expect(quiet.error.mock.calls.at(-1)?.[1]).toBeInstanceOf(StreamClosedError);
  });

  it('stopping early closes the stream from the task and frees its slot', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
      runStream: { topic: () => ({ entries: ['a', 'b', 'c'] }) },
    });

    const run = op.startDurable(streamEcho, { topic: 'orders', messages: [], take: 2 });
    const result = await run.result;

    expect(result.status).toBe('completed');
    expect(result.output).toEqual({ read: ['a', 'b'], cursor: 'c2' });
    expect(result.endpointFrames).toEqual(['streamOpen', 'streamClose', 'done:output']);
    expect(result.frames.find((f) => f.frame.streamClose)?.frame.streamClose).toEqual({
      id: '1',
      code: 0,
      message: '',
    });
    expect(run.openStreams()).toEqual([]);
  });

  it('an aborted signal ends the read while the stream waits for more', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
      runStream: { topic: () => ({ entries: ['a', 'b'] }) },
    });

    const result = await op.invokeDurable(abortingReader, { topic: 'orders', until: 2 });

    expect(result.status).toBe('completed');
    expect(result.output).toEqual({ read: ['a', 'b'], aborted: true });
    expect(result.endpointFrames).toEqual(['streamOpen', 'streamClose', 'done:output']);
  });

  it('over a POST the read throws ServerlessLimitationError naming the missing socket', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      durable: false,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
    });

    const outcome = await op.deliver(streamEcho, { topic: 'orders', messages: ['a'] });

    expect(outcome).toMatchObject({ status: 'failed', retry: false, httpStatus: 422 });
    expect((outcome as { error: string }).error).toMatch(
      /clientFromContext\(ctx\)\.streams\.events\(topic\) is not available .* invoked over a POST and has no invocation socket/
    );
    // The publish before the read went through: the unary client works over a POST.
    expect(engine.published).toEqual([{ topic: 'orders', payload: 'a', producerSeq: 0 }]);
  });

  it('reads a run events stream the test operator scripts', async () => {
    const engine = fakeEngine();
    const op = createTestOperator({
      workflows,
      secret,
      console: quiet,
      client: { token: TOKEN, transport: engine.transport, logLevel: 'OFF' },
      runStream: {
        workflowEvents: (request) =>
          [
            {
              workflowRunId: request.workflowRunId!,
              resourceId: 'step-1',
              eventPayload: 'started',
            },
            {
              workflowRunId: request.workflowRunId!,
              resourceId: 'run-1',
              eventPayload: 'done',
              hangup: true,
            },
          ].map((event) => ({
            resourceType: 0,
            eventType: 0,
            eventTimestamp: undefined,
            hangup: false,
            ...event,
          })),
      },
    });

    const result = await op.invokeDurable(runEventsReader, { runId: 'run-1' });

    expect(result.status).toBe('completed');
    expect(result.output).toEqual({ events: ['step-1:started', 'run-1:done'] });
    expect(result.frames[1].frame.streamOpen).toEqual({
      id: '1',
      procedure: '/Dispatcher/SubscribeToWorkflowEvents',
      request: '{"workflowRunId":"run-1"}',
    });
  });
});
