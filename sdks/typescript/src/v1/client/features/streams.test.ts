import { Status } from 'nice-grpc';
import { createGrpcClient } from '@hatchet/util/grpc-helpers';
import { StreamsClient } from './streams';

jest.mock('@hatchet/util/grpc-helpers', () => ({
  createGrpcClient: jest.fn(),
}));

const mockedCreateGrpcClient = jest.mocked(createGrpcClient);

function fakeHatchetClient(): any {
  return { config: {} };
}

function grpcError(code: Status) {
  return Object.assign(new Error('failed'), { code });
}

function entry(byte: number, cursor: string) {
  return { payload: new Uint8Array([byte]), cursor, createdAt: undefined };
}

describe('StreamsClient.events cancellation', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('stops yielding, without error, as soon as the server sends a hangup message', async () => {
    async function* subscribeStub() {
      yield { entries: [entry(1, 'c1')], hangup: false };
      yield { entries: [], hangup: true };
      // events() must never reach this -- a hangup ends the iteration
      yield { entries: [entry(2, 'c2')], hangup: false };
    }

    const subscribe = jest.fn(() => subscribeStub());
    mockedCreateGrpcClient.mockReturnValue({ client: { subscribe } } as any);

    const streams = new StreamsClient(fakeHatchetClient());

    const received: number[] = [];
    for await (const event of streams.events('topic')) {
      received.push(event.payload[0]);
    }

    expect(received).toEqual([1]);
  });

  it('ends the iteration cleanly on abort instead of rejecting the caller', async () => {
    async function* subscribeStub(_req: unknown, options: { signal?: AbortSignal }) {
      yield { entries: [entry(1, 'c1')], hangup: false };

      // stands in for a real network call left pending until either the next
      // message arrives or the caller cancels -- exactly what a quiet topic
      // looks like without a cancel signal (see subscribeIdleHangupTimeout)
      await new Promise((_resolve, reject) => {
        options.signal?.addEventListener('abort', () => reject(new Error('call aborted')));
      });
    }

    const subscribe = jest.fn((req: unknown, options: { signal?: AbortSignal }) =>
      subscribeStub(req, options)
    );
    mockedCreateGrpcClient.mockReturnValue({ client: { subscribe } } as any);

    const streams = new StreamsClient(fakeHatchetClient());
    const controller = new AbortController();
    const iterator = streams.events('topic', { signal: controller.signal })[Symbol.asyncIterator]();

    const first = await iterator.next();
    expect(first.done).toBe(false);
    expect(first.value?.payload[0]).toBe(1);

    const pending = iterator.next();
    controller.abort();

    const result = await pending;
    expect(result.done).toBe(true);
  });

  it('still throws a non-abort error from the underlying stream', async () => {
    async function* subscribeStub() {
      yield { entries: [entry(1, 'c1')], hangup: false };
      throw new Error('transport failure');
    }

    const subscribe = jest.fn(() => subscribeStub());
    mockedCreateGrpcClient.mockReturnValue({ client: { subscribe } } as any);

    const streams = new StreamsClient(fakeHatchetClient());

    await expect(async () => {
      for await (const _event of streams.events('topic')) {
        // drain
      }
    }).rejects.toThrow('transport failure');
  });

  it('breaking a for-await loop stops consuming further events', async () => {
    let yieldedAfterBreak = false;

    async function* subscribeStub() {
      yield { entries: [entry(1, 'c1'), entry(2, 'c2')], hangup: false };
      yieldedAfterBreak = true;
      yield { entries: [entry(3, 'c3')], hangup: false };
    }

    const subscribe = jest.fn(() => subscribeStub());
    mockedCreateGrpcClient.mockReturnValue({ client: { subscribe } } as any);

    const streams = new StreamsClient(fakeHatchetClient());

    const received: number[] = [];
    for await (const event of streams.events('topic')) {
      received.push(event.payload[0]);
      if (received.length === 1) {
        break;
      }
    }

    expect(received).toEqual([1]);
    expect(yieldedAfterBreak).toBe(false);
  });
});

describe('StreamsClient.publish producer sequencing', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  function setup(results: Array<Error | undefined>) {
    const publish = jest.fn(async (_req: any) => {
      const err = results.shift();
      if (err) throw err;
      return {};
    });
    mockedCreateGrpcClient.mockReturnValue({ client: { publish } } as any);
    return { streams: new StreamsClient(fakeHatchetClient()), publish };
  }

  it('rotates the producer after an ambiguous failure so its seq is never reused', async () => {
    const { streams, publish } = setup([undefined, grpcError(Status.DEADLINE_EXCEEDED)]);

    await streams.publish('topic', 'a');
    await expect(streams.publish('topic', 'b')).rejects.toThrow();
    await streams.publish('topic', 'c');

    const reqs = publish.mock.calls.map(([req]) => req);
    expect(reqs[1].producerId).toBe(reqs[0].producerId);
    expect(reqs[1].producerSeq).toBe(1);
    expect(reqs[2].producerId).not.toBe(reqs[1].producerId);
    expect(reqs[2].producerSeq).toBe(0);
  });

  it('retries a sequence gap once as a new producer instead of failing', async () => {
    const { streams, publish } = setup([undefined, grpcError(Status.FAILED_PRECONDITION)]);

    await streams.publish('topic', 'a');
    await streams.publish('topic', 'b');

    const reqs = publish.mock.calls.map(([req]) => req);
    expect(reqs).toHaveLength(3);
    expect(reqs[1].producerSeq).toBe(1);
    expect(reqs[2].producerId).not.toBe(reqs[1].producerId);
    expect(reqs[2].producerSeq).toBe(0);
  });

  it('surfaces a second consecutive sequence gap', async () => {
    const { streams, publish } = setup([
      grpcError(Status.FAILED_PRECONDITION),
      grpcError(Status.FAILED_PRECONDITION),
    ]);

    await expect(streams.publish('topic', 'a')).rejects.toThrow();
    expect(publish).toHaveBeenCalledTimes(2);
  });

  it('reuses the seq when the server rejected the publish before storing it', async () => {
    const { streams, publish } = setup([grpcError(Status.RESOURCE_EXHAUSTED)]);

    await expect(streams.publish('topic', 'a')).rejects.toThrow();
    await streams.publish('topic', 'a');

    const reqs = publish.mock.calls.map(([req]) => req);
    expect(reqs[1].producerId).toBe(reqs[0].producerId);
    expect(reqs[1].producerSeq).toBe(0);
  });
});

describe('StreamsClient without the durable streams entitlement', () => {
  const denied = () =>
    Object.assign(new Error('durable streams are not enabled for this tenant'), {
      code: Status.PERMISSION_DENIED,
    });

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('rejects a publish without retrying, and keeps its seq for once the tenant is entitled', async () => {
    const results: Array<Error | undefined> = [denied()];
    const publish = jest.fn(async (_req: any) => {
      const err = results.shift();
      if (err) throw err;
      return {};
    });
    mockedCreateGrpcClient.mockReturnValue({ client: { publish } } as any);
    const streams = new StreamsClient(fakeHatchetClient());

    await expect(streams.publish('topic', 'a')).rejects.toMatchObject({
      code: Status.PERMISSION_DENIED,
      message: 'durable streams are not enabled for this tenant',
    });
    expect(publish).toHaveBeenCalledTimes(1);

    await streams.publish('topic', 'a');

    const reqs = publish.mock.calls.map(([req]) => req);
    expect(reqs[1].producerId).toBe(reqs[0].producerId);
    expect(reqs[1].producerSeq).toBe(0);
  });

  it('throws from events() instead of ending the iteration as if the topic were quiet', async () => {
    // eslint-disable-next-line require-yield
    async function* subscribeStub(): AsyncGenerator<never> {
      throw denied();
    }
    mockedCreateGrpcClient.mockReturnValue({ client: { subscribe: subscribeStub } } as any);
    const streams = new StreamsClient(fakeHatchetClient());

    const received: unknown[] = [];
    await expect(
      (async () => {
        for await (const event of streams.events('topic')) {
          received.push(event);
        }
      })()
    ).rejects.toMatchObject({ code: Status.PERMISSION_DENIED });
    expect(received).toEqual([]);
  });
});
