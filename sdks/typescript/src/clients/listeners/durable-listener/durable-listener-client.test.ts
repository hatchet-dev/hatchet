/* eslint-disable require-yield */
import sleep from '@hatchet/util/sleep';
import { DurableListenerClient } from './durable-listener-client';

jest.mock('@hatchet/util/sleep', () => ({
  __esModule: true,
  default: jest.fn(() => Promise.resolve()),
}));

const mockedSleep = jest.mocked(sleep);

function noopLogger() {
  return { info: jest.fn(), warn: jest.fn(), error: jest.fn(), debug: jest.fn() };
}

function mockConfig(): any {
  return { logger: () => noopLogger(), log_level: 'OFF' };
}

async function settle(ms = 50): Promise<void> {
  await new Promise<void>((r) => {
    setTimeout(r, ms);
  });
}

function emptyStream(): AsyncIterable<any> {
  return (async function* empty() {})();
}

function errorStream(err: Error): AsyncIterable<any> {
  return (async function* throwErr() {
    throw err;
  })();
}

function hangingStream(): { stream: AsyncIterable<any>; end: () => void } {
  let resolver!: () => void;
  const gate = new Promise<void>((r) => {
    resolver = r;
  });
  const stream = (async function* hang() {
    await gate;
  })();
  return { stream, end: () => resolver() };
}

function controllableStream() {
  const buffer: any[] = [];
  let waiter: ((v: { response?: any; done?: boolean; error?: Error }) => void) | null = null;
  let ended = false;

  return {
    push(response: any) {
      if (waiter) {
        const w = waiter;
        waiter = null;
        w({ response });
      } else {
        buffer.push(response);
      }
    },
    end() {
      ended = true;
      if (waiter) {
        const w = waiter;
        waiter = null;
        w({ done: true });
      }
    },
    error(err: Error) {
      if (waiter) {
        const w = waiter;
        waiter = null;
        w({ error: err });
      }
    },
    stream: {
      async *[Symbol.asyncIterator]() {
        while (true) {
          if (buffer.length > 0) {
            yield buffer.shift()!;

            continue;
          }
          if (ended) return;
          const result = await new Promise<{ response?: any; done?: boolean; error?: Error }>(
            (r) => {
              waiter = r;
            }
          );
          if (result.error) throw result.error;
          if (result.done) return;
          if (result.response !== undefined) yield result.response;
        }
      },
    },
  };
}

function makeDeferred<T = any>() {
  let resolve!: (v: T) => void;
  let reject!: (r: any) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe('DurableListenerClient reconnection', () => {
  let grpcClient: any;
  let listener: DurableListenerClient;
  const openStreams: { end: () => void }[] = [];

  function tracked(s: ReturnType<typeof hangingStream>) {
    openStreams.push(s);
    return s;
  }

  beforeEach(() => {
    jest.clearAllMocks();
    grpcClient = { durableTask: jest.fn() };
    const factory = { create: jest.fn(() => grpcClient) };
    listener = new DurableListenerClient(mockConfig(), {} as any, factory as any);
  });

  afterEach(async () => {
    await listener.stop();
    for (const s of openStreams) s.end();
    openStreams.length = 0;
    await settle(10);
  });

  // ── reconnection on stream EOF ──

  describe('reconnects on stream EOF', () => {
    it('opens a new stream after the first stream ends', async () => {
      const h = tracked(hangingStream());
      let call = 0;
      grpcClient.durableTask.mockImplementation(() => (++call === 1 ? emptyStream() : h.stream));

      await listener.start('w1');
      await settle();

      expect(grpcClient.durableTask).toHaveBeenCalledTimes(2);
    });

    it('sleeps DEFAULT_RECONNECT_INTERVAL before reconnecting', async () => {
      const h = tracked(hangingStream());
      let call = 0;
      grpcClient.durableTask.mockImplementation(() => (++call === 1 ? emptyStream() : h.stream));

      await listener.start('w1');
      await settle();

      expect(mockedSleep).toHaveBeenCalledWith(3000);
    });
  });

  // ── reconnection on stream error ──

  describe('reconnects on stream error', () => {
    it('opens a new stream after a non-abort error', async () => {
      const h = tracked(hangingStream());
      let call = 0;
      grpcClient.durableTask.mockImplementation(() =>
        ++call === 1 ? errorStream(new Error('network reset')) : h.stream
      );

      await listener.start('w1');
      await settle();

      expect(grpcClient.durableTask).toHaveBeenCalledTimes(2);
    });

    it('sleeps before reconnecting on error', async () => {
      const h = tracked(hangingStream());
      let call = 0;
      grpcClient.durableTask.mockImplementation(() =>
        ++call === 1 ? errorStream(new Error('fail')) : h.stream
      );

      await listener.start('w1');
      await settle();

      expect(mockedSleep).toHaveBeenCalledWith(3000);
    });
  });

  // ── no reconnect when stopped ──

  describe('does not reconnect when stopped', () => {
    it('does not open a new stream after stop()', async () => {
      const h = tracked(hangingStream());
      grpcClient.durableTask.mockReturnValue(h.stream);

      await listener.start('w1');
      await listener.stop();
      await settle();

      expect(grpcClient.durableTask).toHaveBeenCalledTimes(1);
    });
  });

  // ── multiple sequential reconnects ──

  describe('multiple sequential reconnects', () => {
    it('recovers through several consecutive EOFs', async () => {
      const h = tracked(hangingStream());
      let call = 0;
      grpcClient.durableTask.mockImplementation(() => (++call <= 3 ? emptyStream() : h.stream));

      await listener.start('w1');
      await settle(150);

      expect(grpcClient.durableTask.mock.calls.length).toBeGreaterThanOrEqual(4);
    });

    it('recovers through several consecutive errors', async () => {
      const h = tracked(hangingStream());
      let call = 0;
      grpcClient.durableTask.mockImplementation(() =>
        ++call <= 3 ? errorStream(new Error(`err-${call}`)) : h.stream
      );

      await listener.start('w1');
      await settle(150);

      expect(grpcClient.durableTask.mock.calls.length).toBeGreaterThanOrEqual(4);
    });
  });

  // ── worker re-registration ──

  describe('worker re-registration on reconnect', () => {
    it('enqueues a registerWorker request for each connection', async () => {
      const registrations: any[] = [];
      const h = tracked(hangingStream());
      let call = 0;

      grpcClient.durableTask.mockImplementation((reqIter: AsyncIterable<any>) => {
        call++;
        (async () => {
          const iter = reqIter[Symbol.asyncIterator]();
          const first = await iter.next();
          if (!first.done) registrations.push(first.value);
        })();
        return call === 1 ? emptyStream() : h.stream;
      });

      await listener.start('w1');
      await settle(100);

      expect(registrations.length).toBeGreaterThanOrEqual(2);
      for (const reg of registrations) {
        expect(reg).toHaveProperty('registerWorker');
        expect(reg.registerWorker.workerId).toBe('w1');
      }
    });
  });

  // ── _failPendingAcks correctness ──

  describe('_failPendingAcks', () => {
    beforeEach(async () => {
      const h = tracked(hangingStream());
      grpcClient.durableTask.mockReturnValue(h.stream);
      await listener.start('w1');
    });

    it('rejects all pending event acks and clears the map', () => {
      const l = listener as any;
      const d = makeDeferred();
      l._pendingEventAcks.set('task:1', d);

      l._failPendingAcks(new Error('disconnected'));

      expect(l._pendingEventAcks.size).toBe(0);
      return expect(d.promise).rejects.toThrow('disconnected');
    });

    it('preserves pending callbacks (server-side state survives reconnection)', () => {
      const l = listener as any;
      const d = makeDeferred();
      // Swallow the rejection that stop() will produce in afterEach
      d.promise.catch(() => {});
      l._pendingCallbacks.set('task:1:0:1', d);

      l._failPendingAcks(new Error('disconnected'));

      expect(l._pendingCallbacks.size).toBe(1);
      expect(l._pendingCallbacks.get('task:1:0:1')).toBe(d);
    });

    it('rejects all pending eviction acks and clears the map', () => {
      const l = listener as any;
      const d = makeDeferred();
      l._pendingEvictionAcks.set('task:1', d);

      l._failPendingAcks(new Error('disconnected'));

      expect(l._pendingEvictionAcks.size).toBe(0);
      return expect(d.promise).rejects.toThrow('disconnected');
    });

    it('preserves buffered completions (server-side state survives reconnection)', () => {
      const l = listener as any;
      const completion = {
        durableTaskExternalId: 'task',
        nodeId: 1,
        payload: {},
      };
      l._bufferedCompletions.set('task:1:0:1', completion);

      l._failPendingAcks(new Error('disconnected'));

      expect(l._bufferedCompletions.size).toBe(1);
      expect(l._bufferedCompletions.get('task:1:0:1')).toBe(completion);
    });
  });

  // ── _failAllPending correctness (used on stop) ──

  describe('_failAllPending', () => {
    beforeEach(async () => {
      const h = tracked(hangingStream());
      grpcClient.durableTask.mockReturnValue(h.stream);
      await listener.start('w1');
    });

    it('rejects pending callbacks and clears the map', () => {
      const l = listener as any;
      const d = makeDeferred();
      l._pendingCallbacks.set('task:1:0:1', d);

      l._failAllPending(new Error('stopped'));

      expect(l._pendingCallbacks.size).toBe(0);
      return expect(d.promise).rejects.toThrow('stopped');
    });

    it('clears buffered completions', () => {
      const l = listener as any;
      l._bufferedCompletions.set('task:1:0:1', {
        durableTaskExternalId: 'task',
        nodeId: 1,
        payload: {},
      });

      l._failAllPending(new Error('stopped'));

      expect(l._bufferedCompletions.size).toBe(0);
    });

    it('also rejects pending event acks and eviction acks', () => {
      const l = listener as any;
      const ackD = makeDeferred();
      const evD = makeDeferred();
      l._pendingEventAcks.set('task:1', ackD);
      l._pendingEvictionAcks.set('task:1', evD);

      l._failAllPending(new Error('stopped'));

      expect(l._pendingEventAcks.size).toBe(0);
      expect(l._pendingEvictionAcks.size).toBe(0);
      return Promise.all([
        expect(ackD.promise).rejects.toThrow('stopped'),
        expect(evD.promise).rejects.toThrow('stopped'),
      ]);
    });
  });

  // ── pending state rejected on stream disconnect ──

  describe('pending state is rejected on stream disconnect', () => {
    it('rejects pending event acks when stream ends (EOF)', async () => {
      const ctrl = controllableStream();
      const h = tracked(hangingStream());
      let call = 0;
      grpcClient.durableTask.mockImplementation(() => (++call === 1 ? ctrl.stream : h.stream));

      await listener.start('w1');
      await settle();

      const d = makeDeferred();
      (listener as any)._pendingEventAcks.set('task:1', d);

      const assertion = expect(d.promise).rejects.toThrow('durable stream disconnected');
      ctrl.end();
      await settle();
      await assertion;
    });

    it('preserves pending callbacks when stream ends (EOF)', async () => {
      const ctrl = controllableStream();
      const h = tracked(hangingStream());
      let call = 0;
      grpcClient.durableTask.mockImplementation(() => (++call === 1 ? ctrl.stream : h.stream));

      await listener.start('w1');
      await settle();

      const d = makeDeferred();
      // Swallow the rejection that stop() will produce in afterEach
      d.promise.catch(() => {});
      (listener as any)._pendingCallbacks.set('task:1:0:1', d);

      ctrl.end();
      await settle();

      expect((listener as any)._pendingCallbacks.size).toBe(1);
      expect((listener as any)._pendingCallbacks.get('task:1:0:1')).toBe(d);
    });

    it('rejects pending eviction acks when stream ends (EOF)', async () => {
      const ctrl = controllableStream();
      const h = tracked(hangingStream());
      let call = 0;
      grpcClient.durableTask.mockImplementation(() => (++call === 1 ? ctrl.stream : h.stream));

      await listener.start('w1');
      await settle();

      const d = makeDeferred();
      (listener as any)._pendingEvictionAcks.set('task:1', d);

      const assertion = expect(d.promise).rejects.toThrow('durable stream disconnected');
      ctrl.end();
      await settle();
      await assertion;
    });

    it('rejects pending event acks when stream errors', async () => {
      const ctrl = controllableStream();
      const h = tracked(hangingStream());
      let call = 0;
      grpcClient.durableTask.mockImplementation(() => (++call === 1 ? ctrl.stream : h.stream));

      await listener.start('w1');
      await settle();

      const d = makeDeferred();
      (listener as any)._pendingEventAcks.set('task:1', d);

      const assertion = expect(d.promise).rejects.toThrow('durable stream error');
      ctrl.error(new Error('transport failure'));
      await settle();
      await assertion;
    });
  });

  // ── listener remains operational after reconnect ──

  describe('listener state after reconnect', () => {
    it('is still running after reconnect', async () => {
      const h = tracked(hangingStream());
      let call = 0;
      grpcClient.durableTask.mockImplementation(() => (++call === 1 ? emptyStream() : h.stream));

      await listener.start('w1');
      await settle();

      expect((listener as any)._running).toBe(true);
    });

    it('creates a fresh AbortController for the new stream', async () => {
      const h = tracked(hangingStream());
      let call = 0;
      grpcClient.durableTask.mockImplementation(() => (++call === 1 ? emptyStream() : h.stream));

      await listener.start('w1');
      await settle();

      const abort = (listener as any)._receiveAbort as AbortController;
      expect(abort).toBeDefined();
      expect(abort.signal.aborted).toBe(false);
    });
  });
});
