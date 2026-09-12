import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { REQUEST_MAX_AGE_SECONDS, hatchet } from '../index';
import { createTestOperator } from '../testing';
import { NonceSet } from './nonce-set';
import { ServerlessHealthcheckRequest } from '../generated/proto/v1/serverless';

const secret = 'test-secret-at-least-32-characters-long';
const quiet = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };

const echo = hatchet.task({
  name: 'echo',
  fn: async (input: { message: string }) => ({ echo: input.message }),
});

const sleeper = hatchet.durableTask({
  name: 'sleeper',
  fn: async (_input: {}, ctx) => {
    await ctx.sleepFor('1s');
    return {};
  },
});

const workflows = [echo, sleeper];

function operator(extra: Partial<Parameters<typeof createTestOperator>[0]> = {}) {
  return createTestOperator({ workflows, secret, console: quiet, ...extra });
}

const now = () => Math.floor(Date.now() / 1000);

describe('signed POST freshness', () => {
  const healthcheckWith = (
    op: ReturnType<typeof operator>,
    timestamp: number,
    endpointId?: string
  ) =>
    op.request(`${op.handler.basePath}/healthcheck`, {
      body: JSON.stringify(
        ServerlessHealthcheckRequest.toJSON({
          endpointId: endpointId ?? op.endpointId,
          namespace: op.namespace,
          timestamp,
        })
      ),
    });

  it('rejects a healthcheck whose timestamp is stale or from the future', async () => {
    const op = operator();

    expect((await healthcheckWith(op, now() - REQUEST_MAX_AGE_SECONDS - 1)).status).toBe(401);
    expect((await healthcheckWith(op, now() + REQUEST_MAX_AGE_SECONDS + 1)).status).toBe(401);
    expect(await (await healthcheckWith(op, 0)).json()).toEqual({
      error: 'stale or missing timestamp',
    });
  });

  it('accepts a healthcheck on either boundary of the window', async () => {
    const op = operator();

    expect((await healthcheckWith(op, now() - REQUEST_MAX_AGE_SECONDS + 1)).status).toBe(200);
    expect((await healthcheckWith(op, now() + REQUEST_MAX_AGE_SECONDS - 1)).status).toBe(200);
    expect((await healthcheckWith(op, now())).status).toBe(200);
  });

  it('accepts the timestamp as a number too', async () => {
    const op = operator();
    const response = await op.request(`${op.handler.basePath}/healthcheck`, {
      body: JSON.stringify({
        endpointId: op.endpointId,
        namespace: op.namespace,
        timestamp: now(),
      }),
    });

    expect(response.status).toBe(200);
  });

  it('rejects a mismatched endpoint id with 403 once the signature and timestamp hold', async () => {
    const op = operator();

    expect((await healthcheckWith(op, now(), 'someone-else')).status).toBe(403);
    // Stale takes precedence over the id: the checks run in the contract's order.
    expect((await healthcheckWith(op, 0, 'someone-else')).status).toBe(401);
  });

  it('applies the same rules to triggers', async () => {
    const op = operator();

    expect(await op.deliver(echo, { message: 'x' }, { timestamp: now() - 301 })).toMatchObject({
      status: 'failed',
      httpStatus: 401,
      retry: false,
      error: 'stale or missing timestamp',
    });
    expect(await op.deliver(echo, { message: 'x' }, { timestamp: now() + 301 })).toMatchObject({
      status: 'failed',
      httpStatus: 401,
    });
    expect(await op.deliver(echo, { message: 'x' }, { timestamp: now() - 299 })).toMatchObject({
      status: 'completed',
    });
    expect(await op.deliver(echo, { message: 'x' }, { endpointId: 'someone-else' })).toMatchObject({
      status: 'failed',
      httpStatus: 403,
      retry: false,
      error: 'unknown endpoint id',
    });
  });
});

describe('upgrade replay protection', () => {
  const upgrade = vi.fn(() => new Response(null, { status: 200 }));

  it('refuses a future timestamp and a replayed nonce', async () => {
    const op = operator();
    const url = `https://endpoint.test${op.handler.basePath}/trigger`;
    const taskId = crypto.randomUUID();

    const future = await op.upgradeHeaders(taskId, 1, { timestamp: String(now() + 301) });
    expect(
      (await op.handler.fetch(new Request(url, { headers: future }), undefined, { upgrade })).status
    ).toBe(401);

    const headers = await op.upgradeHeaders(taskId, 1);
    expect(
      (await op.handler.fetch(new Request(url, { headers }), undefined, { upgrade })).status
    ).toBe(200);

    const replay = await op.handler.fetch(new Request(url, { headers }), undefined, { upgrade });
    expect(replay.status).toBe(401);
    expect(await replay.json()).toEqual({ error: 'missing or replayed nonce', retry: false });
  });

  it('does not consume a nonce from an unsigned upgrade', async () => {
    const op = operator();
    const url = `https://endpoint.test${op.handler.basePath}/trigger`;
    const taskId = crypto.randomUUID();
    const nonce = 'reused-nonce';

    const forged = await op.upgradeHeaders(taskId, 1, { nonce, signature: 'nope' });
    expect(
      (await op.handler.fetch(new Request(url, { headers: forged }), undefined, { upgrade })).status
    ).toBe(401);

    const genuine = await op.upgradeHeaders(taskId, 1, { nonce });
    expect(
      (await op.handler.fetch(new Request(url, { headers: genuine }), undefined, { upgrade }))
        .status
    ).toBe(200);
  });

  it('lets the seenNonce option replace the in-memory set', async () => {
    const seen = vi.fn(() => true);
    const op = operator({ seenNonce: seen });
    const url = `https://endpoint.test${op.handler.basePath}/trigger`;
    const headers = await op.upgradeHeaders(crypto.randomUUID(), 1);

    expect(
      (await op.handler.fetch(new Request(url, { headers }), undefined, { upgrade })).status
    ).toBe(401);
    expect(seen).toHaveBeenCalledTimes(1);
  });

  it('answers 503 retryable when the nonce store is full', async () => {
    const op = operator({ seenNonce: () => 'full' });
    const url = `https://endpoint.test${op.handler.basePath}/trigger`;
    const headers = await op.upgradeHeaders(crypto.randomUUID(), 1);
    const response = await op.handler.fetch(new Request(url, { headers }), undefined, {
      upgrade,
    });

    expect(response.status).toBe(503);
    expect(await response.json()).toEqual({ error: 'nonce store full', retry: true });
    expect(upgrade).not.toHaveBeenCalled();
  });
});

describe('NonceSet', () => {
  it('remembers a nonce within the window and forgets it after', () => {
    const set = new NonceSet(4096, 300);

    expect(set.consume('a', 1000)).toBe('accepted');
    expect(set.consume('a', 1299)).toBe('replayed');
    expect(set.consume('a', 1300)).toBe('accepted');
  });

  it('refuses new nonces at capacity and keeps every unexpired one', () => {
    const set = new NonceSet(2, 300);

    expect(set.consume('a', 1000)).toBe('accepted');
    expect(set.consume('b', 1001)).toBe('accepted');
    expect(set.consume('c', 1002)).toBe('full');
    expect(set.size).toBe(2);
    // The refused nonce was not recorded; the accepted ones still are.
    expect(set.consume('a', 1003)).toBe('replayed');
    expect(set.consume('b', 1003)).toBe('replayed');
    // Room returns only once an entry expires.
    expect(set.consume('c', 1300)).toBe('accepted');
    expect(set.consume('d', 1300)).toBe('full');
  });
});

describe('first durable frame identity', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    expect(vi.getTimerCount()).toBe(0);
    vi.useRealTimers();
  });

  it('closes with 1008 when the first frame names another task', async () => {
    const op = operator();
    const result = await op.invokeDurable(
      sleeper,
      {},
      {
        firstFrame: (frame) => ({
          ...frame,
          action: { ...frame.action!, taskRunExternalId: crypto.randomUUID() },
        }),
      }
    );

    expect(result.closeCode).toBe(1008);
    expect(result.status).toBe('failed');
    expect(result.endpointFrames).toEqual([]);
  });

  it('closes with 1008 when the first frame names another invocation', async () => {
    const op = operator();
    const result = await op.invokeDurable(
      sleeper,
      {},
      {
        firstFrame: (frame) => ({ ...frame, invocationCount: 2 }),
      }
    );

    expect(result.closeCode).toBe(1008);
    expect(result.endpointFrames).toEqual([]);
  });
});
